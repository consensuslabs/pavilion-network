package main

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// Standardized response helpers
func (a *App) successResponse(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"message": message,
	})
}

func (a *App) errorResponse(c *gin.Context, status int, code, message string, err error) {
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

// handleHealthCheck handles the health check endpoint
func (a *App) handleHealthCheck(c *gin.Context) {
	a.successResponse(c, nil, "Health check successful")
}

// validateVideoUpload validates the video upload request against configuration settings
func (a *App) validateVideoUpload(file *multipart.FileHeader, title, description string) error {
	// Check file size
	if file.Size > a.Config.Video.MaxSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d MB", a.Config.Video.MaxSize/1024/1024)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validExt := false
	for _, format := range a.Config.Video.AllowedFormats {
		if ext == format {
			validExt = true
			break
		}
	}
	if !validExt {
		return fmt.Errorf("unsupported file type: %s. Allowed types: %v", ext, a.Config.Video.AllowedFormats)
	}

	// Validate title
	title = strings.TrimSpace(title)
	if len(title) < a.Config.Video.MinTitleLength {
		return fmt.Errorf("title must be at least %d characters", a.Config.Video.MinTitleLength)
	}
	if len(title) > a.Config.Video.MaxTitleLength {
		return fmt.Errorf("title must not exceed %d characters", a.Config.Video.MaxTitleLength)
	}

	// Validate description
	if len(description) > a.Config.Video.MaxDescLength {
		return fmt.Errorf("description must not exceed %d characters", a.Config.Video.MaxDescLength)
	}

	return nil
}
