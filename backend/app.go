package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/auth"
	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// App holds all application dependencies
type App struct {
	ctx          context.Context
	Config       *Config
	db           *gorm.DB
	cache        *RedisClient
	ipfs         *IPFSService
	p2p          *P2P
	router       *gin.Engine
	auth         *auth.Service
	transcode    *TranscodeService
	logger       *Logger
	IPFSService  *IPFSService
	AuthService  *auth.Service
	S3Service    *S3Service
	videoHandler *video.VideoHandler
}

// NewApp creates a new application instance with all dependencies
func NewApp(ctx context.Context, config *Config) (*App, error) {
	// Initialize logger
	logger, err := NewLogger(config.Logging)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %v", err)
	}

	// Initialize database
	db, err := initDatabase(&config.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to setup database: %v", err)
	}

	// Initialize IPFS service
	ipfs := NewIPFSService(config)

	// Initialize S3 service
	s3Service, err := NewS3Service(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 service: %v", err)
	}

	// Initialize auth service
	authService := auth.NewService(db)

	// Create app instance
	app := &App{
		ctx:         ctx,
		Config:      config,
		db:          db,
		ipfs:        ipfs,
		logger:      logger,
		IPFSService: ipfs,
		S3Service:   s3Service,
		auth:        authService,
	}

	// Initialize video service
	videoService := video.NewService(db, ipfs, s3Service, &video.Config{
		Storage: struct{ UploadDir string }{
			UploadDir: config.Storage.UploadDir,
		},
		Video: struct {
			MaxSize        int64
			MinTitleLength int
			MaxTitleLength int
			MaxDescLength  int
			AllowedFormats []string
		}{
			MaxSize:        config.Video.MaxSize,
			MinTitleLength: config.Video.MinTitleLength,
			MaxTitleLength: config.Video.MaxTitleLength,
			MaxDescLength:  config.Video.MaxDescLength,
			AllowedFormats: config.Video.AllowedFormats,
		},
	})

	// Initialize video handler app context
	videoApp := &video.App{
		Config: &video.Config{
			Storage: struct{ UploadDir string }{
				UploadDir: config.Storage.UploadDir,
			},
			Video: struct {
				MaxSize        int64
				MinTitleLength int
				MaxTitleLength int
				MaxDescLength  int
				AllowedFormats []string
			}{
				MaxSize:        config.Video.MaxSize,
				MinTitleLength: config.Video.MinTitleLength,
				MaxTitleLength: config.Video.MaxTitleLength,
				MaxDescLength:  config.Video.MaxDescLength,
				AllowedFormats: config.Video.AllowedFormats,
			},
		},
		Logger:          logger,
		Video:           videoService,
		IPFS:            ipfs,
		ResponseHandler: app,
	}

	// Initialize video handler
	app.videoHandler = video.NewVideoHandler(videoApp)

	// Initialize router
	app.router = gin.Default()

	// Setup routes
	app.setupRoutes()

	return app, nil
}

func (a *App) initConfig() error {
	config, err := LoadConfig(".")
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}
	a.Config = config
	return nil
}

func (a *App) initDatabase() error {
	db, err := initDatabase(&a.Config.Database)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}
	a.db = db
	return nil
}

func (a *App) initCache() error {
	cache, err := initRedis(a.Config.Redis)
	if err != nil {
		return fmt.Errorf("failed to initialize Redis: %v", err)
	}
	a.cache = cache
	return nil
}

func (a *App) initIPFS() error {
	ipfs := NewIPFSService(a.Config)
	a.ipfs = ipfs
	return nil
}

func (a *App) initP2P() error {
	// Temporarily disabled P2P functionality
	/*
		p2p, err := NewP2PNode(a.ctx, a.Config.P2P.Port, a.Config.P2P.Rendezvous)
		if err != nil {
			return fmt.Errorf("failed to create P2P node: %v", err)
		}

		// Subscribe to default topics
		defaultTopics := []string{"videos", "transcodes"}
		for _, topic := range defaultTopics {
			if _, _, err := p2p.Subscribe(topic); err != nil {
				return fmt.Errorf("failed to subscribe to topic %s: %v", topic, err)
			}
			a.logger.LogInfo(fmt.Sprintf("Subscribed to topic: %s", topic), nil)
		}

		a.p2p = p2p
	*/
	return nil
}

func (a *App) initServices() {
	a.auth = auth.NewService(a.db)
	a.transcode = NewTranscodeService(a.db, a.ipfs, a.S3Service, a.Config, a.logger)
}

func (a *App) setupRoutes() {
	// Static file serving
	a.router.Static("/public", "../frontend/public")
	a.router.Static("/uploads", a.Config.Storage.UploadDir)

	// Health check
	a.router.GET("/health", a.handleHealthCheck)

	// Video routes
	a.router.POST("/video/upload", a.videoHandler.HandleUpload)
	a.router.GET("/video/watch", a.videoHandler.HandleWatch)
	a.router.GET("/video/list", a.videoHandler.HandleList)
	a.router.GET("/video/status/:fileId", a.videoHandler.HandleStatus)
	a.router.POST("/video/transcode", a.videoHandler.HandleTranscode)
}

// Run starts the application
func (a *App) Run() error {
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

	// Close P2P connections if enabled
	if a.p2p != nil {
		if err := a.p2p.Close(); err != nil {
			a.logger.LogWarn("Error closing P2P connections", map[string]interface{}{
				"error": err.Error(),
			})
		}
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
