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
	videoHandler := app.videoHandler
	router.POST("/video/upload", videoHandler.HandleUpload)
	router.GET("/video/watch", videoHandler.HandleWatch)
	router.GET("/video/list", videoHandler.HandleList)
	router.GET("/video/status/:fileId", videoHandler.HandleStatus)
	router.POST("/video/transcode", videoHandler.HandleTranscode)
}
