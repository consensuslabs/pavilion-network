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

// handleLogin handles the OAuth login endpoint
func (a *App) handleLogin(c *gin.Context) {
	user, err := a.auth.Login("test@example.com")
	if err != nil {
		a.errorResponse(c, http.StatusInternalServerError, "ERR_DB", "Failed to save user", err)
		return
	}
	a.successResponse(c, gin.H{"user": user}, "OAuth login stub - token: dummy-token")
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

// handleVideoUpload handles the video upload endpoint
func (a *App) handleVideoUpload(c *gin.Context) {
	file, fileHeader, err := c.Request.FormFile("video")
	if err != nil {
		a.errorResponse(c, http.StatusBadRequest, "ERR_NO_FILE", "No video file received", err)
		return
	}
	defer file.Close()

	// Get title and description from form
	title := c.PostForm("title")
	description := c.PostForm("description")

	// Validate the upload
	if err := a.video.validateVideoUpload(fileHeader, title, description); err != nil {
		a.errorResponse(c, http.StatusBadRequest, "ERR_VALIDATION", err.Error(), err)
		return
	}

	// Process the video (upload to IPFS and S3)
	video, err := a.video.ProcessVideo(file, fileHeader, title, description)
	if err != nil {
		a.errorResponse(c, http.StatusInternalServerError, "ERR_UPLOAD", "Failed to process video upload", err)
		return
	}

	// Create a response with progress information
	videoResponse := gin.H{
		"id":          video.ID,
		"fileId":      video.FileId,
		"title":       video.Title,
		"description": video.Description,
		"filePath":    video.FilePath,
		"ipfsCid":     video.IPFSCID,
		"status":      video.Status,
		"statusMsg":   video.StatusMsg,
		"fileSize":    video.FileSize,
		"createdAt":   video.CreatedAt,
		"updatedAt":   video.UpdatedAt,
	}

	a.successResponse(c, gin.H{
		"video":    videoResponse,
		"filePath": video.FilePath,
		"ipfsCid":  video.IPFSCID,
	}, video.StatusMsg)
}

// handleVideoWatch handles the video watch endpoint
func (a *App) handleVideoWatch(c *gin.Context) {
	cid := c.Query("cid")
	file := c.Query("file")

	if cid != "" {
		ipfsURL := a.ipfs.GetGatewayURL(cid)
		c.Redirect(http.StatusTemporaryRedirect, ipfsURL)
		return
	}

	if file != "" {
		c.File(filepath.Join(a.Config.Storage.UploadDir, file))
		return
	}

	a.errorResponse(c, http.StatusBadRequest, "ERR_NO_PARAM", "No 'cid' or 'file' parameter provided", nil)
}

// handleVideoList handles the video list endpoint
func (a *App) handleVideoList(c *gin.Context) {
	videos, err := a.video.GetVideoList()
	if err != nil {
		a.errorResponse(c, http.StatusInternalServerError, "ERR_DB", "Failed to list videos", err)
		return
	}
	a.successResponse(c, gin.H{"videos": videos}, "Videos retrieved successfully")
}

// handleVideoStatus handles the video status endpoint
func (a *App) handleVideoStatus(c *gin.Context) {
	fileId := c.Param("fileId")
	if fileId == "" {
		a.errorResponse(c, http.StatusBadRequest, "ERR_NO_FILE_ID", "No file ID provided", nil)
		return
	}

	video, err := a.video.GetVideoStatus(fileId)
	if err != nil {
		a.errorResponse(c, http.StatusNotFound, "ERR_NOT_FOUND", "Video not found", err)
		return
	}

	a.successResponse(c, gin.H{
		"status":    video.Status,
		"message":   video.StatusMsg,
		"fileId":    video.FileId,
		"title":     video.Title,
		"fileSize":  video.FileSize,
		"ipfsCid":   video.IPFSCID,
		"updatedAt": video.UpdatedAt,
	}, "Video status retrieved successfully")
}

// UploadVideoHandler handles video uploads
func (a *App) UploadVideoHandler(c *gin.Context) {
	// Get the uploaded file
	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		a.logger.LogError(err, "Failed to get file from request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file"})
		return
	}
	defer file.Close()

	// Get title and description from form
	title := c.PostForm("title")
	description := c.PostForm("description")

	// Process the video (upload to IPFS and S3)
	video, err := a.VideoService.ProcessVideo(file, fileHeader, title, description)
	if err != nil {
		a.logger.LogError(err, "Failed to process video")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload video"})
		return
	}

	// Return the IPFS CID and S3 URL without checksum
	c.JSON(http.StatusCreated, gin.H{
		"message":   "Video uploaded successfully",
		"fileId":    video.FileId,
		"ipfsCid":   video.IPFSCID,
		"cdnUrl":    video.FilePath,
		"createdAt": video.CreatedAt,
	})
}
