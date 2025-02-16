package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/auth"
	"github.com/consensuslabs/pavilion-network/backend/internal/cache"
	"github.com/consensuslabs/pavilion-network/backend/internal/config"
	"github.com/consensuslabs/pavilion-network/backend/internal/database"
	httpHandler "github.com/consensuslabs/pavilion-network/backend/internal/http"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// App holds all application dependencies
type App struct {
	ctx          context.Context
	Config       *config.Config
	db           *gorm.DB
	dbService    database.Service
	cache        cache.Service
	ipfs         *IPFSService
	router       *gin.Engine
	auth         *auth.Service
	logger       logger.Logger
	IPFSService  *IPFSService
	AuthService  *auth.Service
	S3Service    *S3Service
	videoHandler *video.VideoHandler
}

// NewApp creates a new application instance
func NewApp(ctx context.Context, cfg *config.Config) (*App, error) {
	loggerService, err := logger.NewService(&cfg.Logging)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %v", err)
	}

	// Initialize database service
	dbService := database.NewDatabaseService(&cfg.Database, loggerService)
	db, err := dbService.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

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
	ipfsService := NewIPFSService(cfg)

	// Initialize S3 service
	s3Config := &config.Config{
		Storage: config.StorageConfig{
			S3: cfg.Storage.S3,
		},
	}
	s3Service, err := NewS3Service(s3Config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 service: %v", err)
	}

	// Initialize auth service
	authService := auth.NewService(db)

	// Initialize router
	router := gin.Default()

	// Initialize video app context
	videoApp := &video.App{
		Config: &video.Config{
			Storage: struct{ UploadDir string }{
				UploadDir: cfg.Storage.UploadDir,
			},
			Video: struct {
				MaxSize        int64
				MinTitleLength int
				MaxTitleLength int
				MaxDescLength  int
				AllowedFormats []string
			}{
				MaxSize:        cfg.Video.MaxSize,
				MinTitleLength: cfg.Video.MinTitleLength,
				MaxTitleLength: cfg.Video.MaxTitleLength,
				MaxDescLength:  cfg.Video.MaxDescLength,
				AllowedFormats: cfg.Video.AllowedFormats,
			},
			TempDir: cfg.Storage.TempDir,
			Ffmpeg:  cfg.Ffmpeg,
		},
		Logger:          loggerService,
		IPFS:            ipfsService,
		ResponseHandler: nil, // Will be set after app creation
	}

	// Initialize video handler
	videoHandler := video.NewVideoHandler(videoApp)

	app := &App{
		ctx:          ctx,
		Config:       cfg,
		db:           db,
		dbService:    dbService,
		cache:        cacheService,
		ipfs:         ipfsService,
		router:       router,
		auth:         authService,
		logger:       loggerService,
		IPFSService:  ipfsService,
		AuthService:  authService,
		S3Service:    s3Service,
		videoHandler: videoHandler,
	}

	// Set the response handler for video app
	videoApp.ResponseHandler = app

	// Initialize P2P
	if err := app.initP2P(); err != nil {
		return nil, fmt.Errorf("failed to initialize P2P: %v", err)
	}

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
	ipfs := NewIPFSService(a.Config)
	a.ipfs = ipfs
	return nil
}

func (a *App) initP2P() error {
	return nil
}

func (a *App) initServices() {
	a.auth = auth.NewService(a.db)
}

func (a *App) setupRoutes() error {
	// Configure static file serving
	if err := httpHandler.ServeStaticFiles(a.router, []httpHandler.StaticFileConfig{
		{
			URLPath:   "/public",
			FilePath:  "../frontend/public",
			IndexFile: "index.html",
		},
	}); err != nil {
		return fmt.Errorf("failed to configure static files: %v", err)
	}

	// Health check
	a.router.GET("/health", a.handleHealthCheck)

	// Video routes
	a.router.POST("/video/upload", a.videoHandler.HandleUpload)
	a.router.GET("/video/watch", a.videoHandler.HandleWatch)
	a.router.GET("/video/list", a.videoHandler.HandleList)
	a.router.GET("/video/status/:fileId", a.videoHandler.HandleStatus)
	a.router.POST("/video/transcode", a.videoHandler.HandleTranscode)

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
	if err := a.router.Run(fmt.Sprintf(":%d", port)); err != nil {
		return a.logger.LogError(err, "server failed to start")
	}
	return nil
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown() error {
	a.logger.LogInfo("Initiating graceful shutdown", nil)

	// Create a timeout context for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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
	if a.ipfs != nil {
		if err := a.ipfs.Close(); err != nil {
			a.logger.LogWarn("Error closing IPFS connections", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Wait for context timeout or completion
	<-ctx.Done()
	if err := ctx.Err(); err != nil && err != context.Canceled {
		a.logger.LogWarn("Shutdown timed out", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	a.logger.LogInfo("Application shutdown complete", nil)
	return nil
}

// Add response handler methods to App
func (a *App) SuccessResponse(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"message": message,
	})
}

func (a *App) ErrorResponse(c *gin.Context, status int, code, message string, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	requestID, _ := c.Get("request_id")

	c.JSON(status, gin.H{
		"success": false,
		"error": gin.H{
			"code":       code,
			"message":    message,
			"details":    errMsg,
			"request_id": requestID,
		},
	})

	// Log error with request ID and context
	a.logger.LogInfo(message, map[string]interface{}{
		"request_id": requestID,
		"code":       code,
		"status":     status,
	})
	if err != nil {
		a.logger.LogError(err, message)
	}
}
