package main

import (
	"context"
	"fmt"

	// Import your models package

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// App holds all application dependencies
type App struct {
	ctx       context.Context
	Config    *Config
	db        *gorm.DB
	cache     *RedisClient
	ipfs      *IPFSService
	p2p       *P2P
	router    *gin.Engine
	video     *VideoService
	auth      *AuthService
	transcode *TranscodeService
	logger    *Logger
}

// NewApp creates a new application instance with all dependencies
func NewApp(ctx context.Context) (*App, error) {
	app := &App{
		ctx:    ctx,
		router: gin.Default(),
	}

	// Load configuration
	if err := app.initConfig(); err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	// Initialize logger
	logger, err := NewLogger(app.Config.Logging)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %v", err)
	}
	app.logger = logger
	app.logger.LogInfo("Logger initialized", nil)

	// Initialize components
	if err := app.initDatabase(); err != nil {
		return nil, app.logger.LogError(err, "failed to initialize database")
	}

	if err := app.initCache(); err != nil {
		return nil, app.logger.LogError(err, "failed to initialize cache")
	}

	if err := app.initIPFS(); err != nil {
		return nil, app.logger.LogError(err, "failed to initialize IPFS")
	}

	if err := app.initP2P(); err != nil {
		return nil, app.logger.LogError(err, "failed to initialize P2P")
	}

	// Initialize services
	app.initServices()
	app.logger.LogInfo("Services initialized", nil)

	// Setup routes
	app.setupRoutes()
	app.logger.LogInfo("Routes configured", nil)

	return app, nil
}

func (a *App) initConfig() error {
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}
	a.Config = config
	return nil
}

func (a *App) initDatabase() error {
	db, err := initDatabase(a.Config.Database)
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
	return nil
}

func (a *App) initServices() {
	a.video = NewVideoService(a.db, a.ipfs, a.Config)
	a.auth = NewAuthService(a.db)
	a.transcode = NewTranscodeService(a.db, a.ipfs, a.Config)
}

func (a *App) setupRoutes() {
	SetupRoutes(a.router, a)
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
	a.logger.LogInfo("Shutting down application", nil)

	// Close P2P connections
	if err := a.p2p.Close(); err != nil {
		a.logger.LogWarn("Error closing P2P connections", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Close cache connections
	if err := a.cache.Close(); err != nil {
		a.logger.LogWarn("Error closing cache connections", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Add any other cleanup needed here

	a.logger.LogInfo("Application shutdown complete", nil)
	return nil
}
