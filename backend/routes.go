package main

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all the routes for our application
func SetupRoutes(router *gin.Engine, app *App) {
	// Static file serving
	router.Static("/public", "../frontend/public")
	router.Static("/uploads", app.Config.Storage.UploadDir)

	// Health check
	router.GET("/health", app.handleHealthCheck)

	// Video routes
	router.POST("/video/upload", app.handleVideoUpload)
	router.GET("/video/watch", app.handleVideoWatch)
	router.GET("/video/list", app.handleVideoList)
	router.GET("/video/status/:fileId", app.handleVideoStatus)
	router.POST("/video/transcode", app.handleVideoTranscode)
}
