package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/consensuslabs/pavilion-network/backend/docs"
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
	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	config, err := LoadConfig(".")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create new app instance with both context and config
	app, err := NewApp(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Add Swagger documentation route
	app.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Received shutdown signal")

		if err := app.Shutdown(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
		cancel()
	}()

	// Start the application
	if err := app.Run(); err != nil {
		log.Fatalf("Error running app: %v", err)
	}
}
