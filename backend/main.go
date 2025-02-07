package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create new app instance
	app, err := NewApp(ctx)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

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
