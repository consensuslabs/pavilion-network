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
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Pavilion Network API
// @version         1.0
// @description     API Server for Pavilion Network Application
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.basic  BasicAuth
func main() {
	ctx := context.Background()

	// Initialize logger for bootstrapping
	loggerService, err := logger.NewService(&logger.Config{Level: "debug"})
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

	// Add Swagger documentation route
	app.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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
