package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/consensuslabs/pavilion-network/backend/docs/api"
	"github.com/consensuslabs/pavilion-network/backend/internal/config"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
)

// @title           Pavilion Network API
// @version         1.0
// @description     API Server for Pavilion Network Application - A decentralized video platform
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT token for authentication

// @tag.name auth
// @tag.description Authentication endpoints

// @tag.name video
// @tag.description Video management endpoints

// @tag.name health
// @tag.description Health check endpoints

// @securityDefinitions.basic  BasicAuth
func main() {
	ctx := context.Background()

	// Initialize logger first
	loggerConfig := &logger.Config{
		Level:       logger.Level("info"),
		Format:      "json",
		Output:      "stdout",
		Development: false,
		File: struct {
			Enabled bool   `mapstructure:"enabled" yaml:"enabled"`
			Path    string `mapstructure:"path" yaml:"path"`
			Rotate  bool   `mapstructure:"rotate" yaml:"rotate"`
			MaxSize string `mapstructure:"maxSize" yaml:"maxSize"`
			MaxAge  string `mapstructure:"maxAge" yaml:"maxAge"`
		}{
			Enabled: false,
			Path:    "/var/log/pavilion",
			Rotate:  true,
			MaxSize: "100MB",
			MaxAge:  "30d",
		},
		Sampling: struct {
			Initial    int `mapstructure:"initial" yaml:"initial"`
			Thereafter int `mapstructure:"thereafter" yaml:"thereafter"`
		}{
			Initial:    100,
			Thereafter: 100,
		},
	}

	loggerService, err := logger.NewLogger(loggerConfig)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Load configuration
	configService := config.NewConfigService(loggerService)
	cfg, err := configService.Load(".")
	if err != nil {
		loggerService.LogFatal(err, "Failed to load configuration")
	}

	// Create a context that will be canceled on interrupt
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Create and run application
	app, err := NewApp(ctx, cfg)
	if err != nil {
		loggerService.LogFatal(err, "Failed to initialize application")
	}

	// Start the application
	if err := app.Run(); err != nil {
		log.Printf("Application error: %v", err)
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Perform graceful shutdown
	if err := app.Shutdown(); err != nil {
		fmt.Printf("Error during shutdown: %v\n", err)
		os.Exit(1)
	}
}
