package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/auth"
	"github.com/consensuslabs/pavilion-network/backend/internal/cache"
	"github.com/consensuslabs/pavilion-network/backend/internal/comment"
	"github.com/consensuslabs/pavilion-network/backend/internal/config"
	"github.com/consensuslabs/pavilion-network/backend/internal/database"
	"github.com/consensuslabs/pavilion-network/backend/internal/database/scylladb"
	"github.com/consensuslabs/pavilion-network/backend/internal/health"
	httpHandler "github.com/consensuslabs/pavilion-network/backend/internal/http"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification"
	"github.com/consensuslabs/pavilion-network/backend/internal/storage"
	"github.com/consensuslabs/pavilion-network/backend/internal/storage/ipfs"
	"github.com/consensuslabs/pavilion-network/backend/internal/storage/s3"
	videostorage "github.com/consensuslabs/pavilion-network/backend/internal/storage/video"
	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/consensuslabs/pavilion-network/backend/internal/video/ffmpeg"
	"github.com/consensuslabs/pavilion-network/backend/internal/video/tempfile"
	"github.com/consensuslabs/pavilion-network/backend/migrations"
	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// App holds all application dependencies
type App struct {
	ctx                 context.Context
	Config              *config.Config
	db                  *gorm.DB
	dbService           database.Service
	cache               cache.Service
	router              *gin.Engine
	auth                *auth.Service
	jwtService          auth.TokenService
	refreshTokens       auth.RefreshTokenService
	logger              logger.Logger
	ipfsService         storage.IPFSService
	s3Service           storage.S3Service
	videoHandler        *video.VideoHandler
	healthHandler       *health.Handler
	httpHandler         httpHandler.ResponseHandler
	authHandler         *auth.Handler
	commentHandler      *comment.Handler
	notificationService notification.NotificationService
	notificationHandler *notification.Handler
	scyllaSession       *gocql.Session
	scyllaManager       *scylladb.SchemaManager
}

// NewApp creates a new application instance
func NewApp(ctx context.Context, cfg *config.Config) (*App, error) {
	// Convert config.LoggingConfig to logger.Config
	loggerConfig := &logger.Config{
		Level:       logger.Level(cfg.Logging.Level),
		Format:      cfg.Logging.Format,
		Output:      cfg.Logging.Output,
		Development: cfg.Logging.Development,
		File: struct {
			Enabled bool   `mapstructure:"enabled" yaml:"enabled"`
			Path    string `mapstructure:"path" yaml:"path"`
			Rotate  bool   `mapstructure:"rotate" yaml:"rotate"`
			MaxSize string `mapstructure:"maxSize" yaml:"maxSize"`
			MaxAge  string `mapstructure:"maxAge" yaml:"maxAge"`
		}{
			Enabled: cfg.Logging.File.Enabled,
			Path:    cfg.Logging.File.Path,
			Rotate:  cfg.Logging.File.Rotate,
			MaxSize: cfg.Logging.File.MaxSize,
			MaxAge:  cfg.Logging.File.MaxAge,
		},
		Sampling: struct {
			Initial    int `mapstructure:"initial" yaml:"initial"`
			Thereafter int `mapstructure:"thereafter" yaml:"thereafter"`
		}{
			Initial:    cfg.Logging.Sampling.Initial,
			Thereafter: cfg.Logging.Sampling.Thereafter,
		},
	}

	loggerService, err := logger.NewLogger(loggerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %v", err)
	}

	// Initialize response handler
	responseHandler := httpHandler.NewResponseHandler(loggerService)

	// Initialize database service
	dbService := database.NewDatabaseService(&cfg.Database, loggerService)
	db, err := dbService.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Run migrations
	if err := migrations.RunMigrations(db, "up"); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %v", err)
	}
	loggerService.LogInfo("Database migrations completed successfully", nil)

	// Initialize cache service
	redisConfig := &cache.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	cacheService, err := cache.NewRedisService(redisConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Redis service: %v", err)
	}

	// Initialize IPFS service
	ipfsConfig := &storage.IPFSConfig{
		APIAddress: cfg.Storage.IPFS.APIAddress,
		Gateway:    cfg.Storage.IPFS.Gateway,
	}
	ipfsService := ipfs.NewService(ipfsConfig, loggerService)
	ipfsAdapter := storage.NewVideoIPFSAdapter(ipfsService)

	// Initialize S3 service
	s3Config := &videostorage.Config{
		Endpoint:        cfg.Storage.S3.Endpoint,
		AccessKeyID:     cfg.Storage.S3.AccessKeyID,
		SecretAccessKey: cfg.Storage.S3.SecretAccessKey,
		UseSSL:          cfg.Storage.S3.UseSSL,
		Region:          cfg.Storage.S3.Region,
		Bucket:          cfg.Storage.S3.Bucket,
	}

	// Debug log for S3 configuration
	loggerService.LogInfo("S3 Configuration in App", map[string]interface{}{
		"endpoint":        cfg.Storage.S3.Endpoint,
		"region":          cfg.Storage.S3.Region,
		"bucket":          cfg.Storage.S3.Bucket,
		"useSSL":          cfg.Storage.S3.UseSSL,
		"accessKeyID":     cfg.Storage.S3.AccessKeyID,
		"secretAccessKey": cfg.Storage.S3.SecretAccessKey != "", // Don't log the actual secret
		"accessKeyLength": len(cfg.Storage.S3.AccessKeyID),
		"secretKeyLength": len(cfg.Storage.S3.SecretAccessKey),
	})

	s3Service, err := s3.NewService(s3Config, loggerService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 service: %v", err)
	}

	// Initialize temporary file manager
	tempConfig := &tempfile.Config{
		BaseDir:     "/tmp/videos",
		Permissions: 0755,
	}
	tempManager, err := tempfile.NewManager(tempConfig, loggerService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize temporary file manager: %v", err)
	}

	// Initialize FFmpeg service
	ffmpegConfig := &ffmpeg.Config{
		Path:        cfg.Ffmpeg.Path,
		ProbePath:   cfg.Ffmpeg.ProbePath,
		VideoCodec:  cfg.Ffmpeg.VideoCodec,
		AudioCodec:  cfg.Ffmpeg.AudioCodec,
		Preset:      cfg.Ffmpeg.Preset,
		OutputPath:  cfg.Ffmpeg.OutputPath,
		Resolutions: cfg.Ffmpeg.Resolutions,
	}
	ffmpegService := ffmpeg.NewService(ffmpegConfig, loggerService)

	// Initialize video service
	videoService := video.NewVideoService(
		db,
		ipfsAdapter,
		s3Service,
		ffmpegService,
		tempManager,
		video.NewLoggerAdapter(loggerService),
	)

	// Initialize video app context
	videoApp := &video.App{
		Config: &video.Config{
			Video: struct {
				MaxFileSize    int64    `yaml:"max_file_size"`
				MinTitleLength int      `yaml:"min_title_length"`
				MaxTitleLength int      `yaml:"max_title_length"`
				MaxDescLength  int      `yaml:"max_desc_length"`
				AllowedFormats []string `yaml:"allowed_formats"`
			}{
				MaxFileSize:    cfg.Video.MaxSize,
				MinTitleLength: cfg.Video.MinTitleLength,
				MaxTitleLength: cfg.Video.MaxTitleLength,
				MaxDescLength:  cfg.Video.MaxDescLength,
				AllowedFormats: cfg.Video.AllowedFormats,
			},
			FFmpeg: video.FfmpegConfig{
				Path:        cfg.Ffmpeg.Path,
				ProbePath:   cfg.Ffmpeg.ProbePath,
				VideoCodec:  cfg.Ffmpeg.VideoCodec,
				AudioCodec:  cfg.Ffmpeg.AudioCodec,
				Preset:      cfg.Ffmpeg.Preset,
				OutputPath:  cfg.Ffmpeg.OutputPath,
				Resolutions: cfg.Ffmpeg.Resolutions,
			},
		},
		Logger:              video.NewLoggerAdapter(loggerService),
		IPFS:                ipfsAdapter,
		ResponseHandler:     responseHandler,
		Video:               videoService,
		NotificationService: nil, // Will be set later after notification service is initialized
	}

	// Initialize video handler
	videoHandler := video.NewVideoHandler(videoApp)

	// Initialize health handler
	healthHandler := health.NewHandler(responseHandler)

	// Initialize router
	router := gin.Default()

	// Initialize JWT service
	authConfig := auth.NewConfigFromAuthConfig(&cfg.Auth)
	jwtService := auth.NewJWTService(authConfig)

	// Initialize refresh token service
	refreshTokens := auth.NewRefreshTokenRepository(db, loggerService)

	// Initialize auth service
	authService := auth.NewService(db, jwtService, refreshTokens, authConfig, loggerService)

	// Initialize auth handler
	authHandler := auth.NewHandler(authService, responseHandler)

	// Create app instance
	app := &App{
		ctx:           ctx,
		Config:        cfg,
		db:            db,
		dbService:     dbService,
		cache:         cacheService,
		router:        router,
		auth:          authService,
		jwtService:    jwtService,
		refreshTokens: refreshTokens,
		logger:        loggerService,
		ipfsService:   ipfsService,
		s3Service:     s3Service,
		videoHandler:  videoHandler,
		healthHandler: healthHandler,
		httpHandler:   responseHandler,
		authHandler:   authHandler,
	}

	// Initialize ScyllaDB connection
	scyllaConfig := scylladb.Config{
		Hosts:          cfg.ScyllaDB.Hosts,
		Port:           cfg.ScyllaDB.Port,
		Keyspace:       cfg.ScyllaDB.Keyspace,
		Username:       cfg.ScyllaDB.Username,
		Password:       cfg.ScyllaDB.Password,
		Consistency:    cfg.ScyllaDB.Consistency,
		Timeout:        cfg.ScyllaDB.Timeout,
		ConnectTimeout: cfg.ScyllaDB.ConnectTimeout,
		Replication: scylladb.Replication{
			Class:             cfg.ScyllaDB.Replication.Class,
			ReplicationFactor: cfg.ScyllaDB.Replication.ReplicationFactor,
		},
	}

	// Create logger adapter to convert between logger interfaces
	loggerAdapter := &loggerAdapter{logger: loggerService}

	scyllaClient := scylladb.NewClient(scyllaConfig, loggerAdapter)
	if err := scyllaClient.Connect(); err != nil {
		loggerService.LogError(fmt.Errorf("failed to connect to ScyllaDB: %w", err), "ScyllaDB connection error")
		return nil, fmt.Errorf("failed to connect to ScyllaDB: %w", err)
	}

	app.scyllaSession = scyllaClient.Session()

	// Initialize schema manager
	app.scyllaManager = scylladb.NewSchemaManager(app.scyllaSession, scyllaConfig, loggerAdapter)

	// Get migration config to check if we should auto-migrate
	migrationConfig := app.dbService.GetMigrationConfig()

	// Initialize schema only if auto-migrate is enabled
	if migrationConfig.ShouldAutoMigrate() {
		loggerService.LogInfo("Auto-migrate enabled, initializing ScyllaDB schema", nil)
		if err := app.scyllaManager.InitializeSchema(); err != nil {
			loggerService.LogError(fmt.Errorf("failed to initialize ScyllaDB schema: %w", err), "ScyllaDB schema error")
			return nil, fmt.Errorf("failed to initialize ScyllaDB schema: %w", err)
		}
	} else {
		loggerService.LogInfo("Auto-migrate disabled, skipping ScyllaDB schema initialization", nil)
	}

	// Initialize comment repository
	commentRepo := scylladb.NewCommentRepository(app.scyllaSession, loggerAdapter)

	// Initialize comment service
	commentService := comment.NewService(commentRepo)

	// Initialize comment handler
	app.commentHandler = comment.NewHandler(commentService, responseHandler, loggerAdapter)

	// Initialize notification repository
	notificationRepo := scylladb.NewNotificationRepository(app.scyllaSession, loggerService)

	// Initialize notification schema manager
	notificationSchemaManager := notification.NewSchemaManager(app.scyllaSession, scyllaConfig.Keyspace, loggerService)

	// Initialize notification schema if auto-migrate is enabled
	if migrationConfig.ShouldAutoMigrate() {
		loggerService.LogInfo("Auto-migrate enabled, initializing notification schema", nil)
		if err := notificationSchemaManager.CreateTables(); err != nil {
			loggerService.LogError(fmt.Errorf("failed to initialize notification schema: %w", err), "Notification schema error")
			loggerService.LogWarn("Continuing without notification service", nil)
		}
	}

	// Initialize notification service config
	notificationConfig := notification.NewServiceConfigFromConfig(cfg)

	// Initialize notification service
	notificationService, err := notification.NewService(ctx, notificationConfig, loggerService, notificationRepo)
	if err != nil {
		loggerService.LogError(fmt.Errorf("failed to initialize notification service: %w", err), "Notification service error")
		loggerService.LogWarn("Continuing without notification service", nil)
	} else {
		app.notificationService = notificationService
		
		// Initialize notification handler only if service is successfully created
		app.notificationHandler = notification.NewHandler(notificationService, responseHandler, loggerService)
		
		// Create adapter and inject notification service into video app for video events
		notificationAdapter := notification.NewVideoNotificationAdapter(notificationService)
		videoApp.NotificationService = notificationAdapter
		
		loggerService.LogInfo("Notification service and handler initialized successfully", nil)
	}

	loggerService.LogInfo("ScyllaDB, comment and notification services initialized successfully", nil)

	return app, nil
}

func (a *App) initConfig() error {
	configService := config.NewConfigService(a.logger)
	cfg, err := configService.Load(".")
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}
	a.Config = cfg
	return nil
}

func (a *App) initDatabase() error {
	db, err := a.dbService.Connect()
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}
	a.db = db
	return nil
}

func (a *App) initCache() error {
	cacheService, err := cache.NewRedisService(&cache.Config{
		Addr:     a.Config.Redis.Addr,
		Password: a.Config.Redis.Password,
		DB:       a.Config.Redis.DB,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize Redis service: %v", err)
	}
	a.cache = cacheService
	return nil
}

func (a *App) initIPFS() error {
	ipfsConfig := &storage.IPFSConfig{
		APIAddress: a.Config.Storage.IPFS.APIAddress,
		Gateway:    a.Config.Storage.IPFS.Gateway,
	}
	ipfsService := ipfs.NewService(ipfsConfig, a.logger)
	a.ipfsService = ipfsService
	return nil
}

func (a *App) initP2P() error {
	return nil
}

func (a *App) initServices() {
	// Initialize JWT service
	authConfig := auth.NewConfigFromAuthConfig(&a.Config.Auth)
	a.jwtService = auth.NewJWTService(authConfig)

	// Initialize refresh token service
	a.refreshTokens = auth.NewRefreshTokenRepository(a.db, a.logger)

	// Initialize auth service
	a.auth = auth.NewService(a.db, a.jwtService, a.refreshTokens, authConfig, a.logger)

	// Initialize auth handler
	a.authHandler = auth.NewHandler(a.auth, a.httpHandler)
}

func (a *App) setupRoutes() error {
	// Add CORS middleware
	a.router.Use(httpHandler.CORSMiddleware())

	// Add request logging middleware
	a.router.Use(httpHandler.RequestLoggerMiddleware(a.logger))

	// Add recovery middleware
	a.router.Use(httpHandler.RecoveryMiddleware(a.httpHandler, a.logger))

	// Set up routes
	SetupRoutes(a.router, a)

	// Set up Swagger documentation
	a.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return nil
}

// Run starts the application
func (a *App) Run() error {
	// Setup routes
	if err := a.setupRoutes(); err != nil {
		return fmt.Errorf("failed to setup routes: %v", err)
	}

	port := a.Config.Server.Port
	a.logger.LogInfo(fmt.Sprintf("Starting server on port %d", port), nil)

	// Create an http.Server instance
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: a.router,
		// Add BaseContext to ensure the server uses our context
		BaseContext: func(net.Listener) context.Context {
			return a.ctx
		},
	}

	// Create a TCP listener with SO_REUSEADDR option
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			if err := c.Control(func(fd uintptr) {
				opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			}); err != nil {
				return err
			}
			return opErr
		},
	}

	listener, err := lc.Listen(a.ctx, "tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %v", err)
	}

	// Start the server in a goroutine
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			a.logger.LogError(err, "server failed to start")
		}
	}()

	// Store the server in the app context for shutdown
	a.ctx = context.WithValue(a.ctx, "server", srv)

	// Block until context is canceled
	<-a.ctx.Done()
	return nil
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown() error {
	a.logger.LogInfo("Initiating graceful shutdown", nil)

	// Create a timeout context for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the server from context
	if srv, ok := a.ctx.Value("server").(*http.Server); ok {
		// First shutdown the HTTP server
		if err := srv.Shutdown(ctx); err != nil {
			a.logger.LogError(err, "Error shutting down HTTP server")
			return err
		}
		a.logger.LogInfo("HTTP server shutdown complete", nil)
	}

	// Close cache connections
	if a.cache != nil {
		if err := a.cache.Close(); err != nil {
			a.logger.LogWarn("Error closing cache connections", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Close database connections
	if a.db != nil {
		sqlDB, err := a.db.DB()
		if err != nil {
			a.logger.LogWarn("Error getting underlying database instance", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			if err := sqlDB.Close(); err != nil {
				a.logger.LogWarn("Error closing database connections", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	}

	// Close IPFS connections if any
	if a.ipfsService != nil {
		if err := a.ipfsService.Close(); err != nil {
			a.logger.LogWarn("Error closing IPFS connections", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Close notification service connections if any
	if a.notificationService != nil {
		if err := a.notificationService.Close(); err != nil {
			a.logger.LogWarn("Error closing notification service connections", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	a.logger.LogInfo("Application shutdown complete", nil)
	return nil
}

// loggerAdapter adapts the logger.Logger interface to the video.Logger interface
type loggerAdapter struct {
	logger logger.Logger
}

// LogInfo implements the video.Logger interface
func (a *loggerAdapter) LogInfo(msg string, fields map[string]interface{}) {
	a.logger.LogInfo(msg, fields)
}

// LogError implements the video.Logger interface
func (a *loggerAdapter) LogError(msg string, fields map[string]interface{}) {
	if err, ok := fields["error"]; ok {
		if errStr, ok := err.(string); ok {
			a.logger.LogError(fmt.Errorf(errStr), msg)
		} else {
			a.logger.LogError(fmt.Errorf("unknown error"), msg)
		}
	} else {
		a.logger.LogError(fmt.Errorf("no error details"), msg)
	}
}
