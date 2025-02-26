package main

import (
	"github.com/consensuslabs/pavilion-network/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all the routes for our application
func SetupRoutes(router *gin.Engine, app *App) {
	// Static file serving
	router.Static("/public", "../frontend/public")
	router.Static("/uploads", app.Config.Storage.UploadDir)

	// Health check
	router.GET("/health", app.healthHandler.HandleHealthCheck)

	// Register auth routes
	app.authHandler.RegisterRoutes(router)

	// Protected routes group
	protected := router.Group("")
	protected.Use(auth.AuthMiddleware(app.auth, app.httpHandler))
	{
		// Video routes that require authentication
		protected.POST("/video/upload", app.videoHandler.HandleUpload)
		protected.GET("/videos", app.videoHandler.ListVideos)
		protected.GET("/video/:id", app.videoHandler.GetVideo)
		protected.GET("/video/:id/status", app.videoHandler.GetVideoStatus)
		protected.PATCH("/video/:id", app.videoHandler.UpdateVideo)
		protected.DELETE("/video/:id", app.videoHandler.DeleteVideo)
	}
}
