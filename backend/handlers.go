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
	file, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No file uploaded or invalid form data",
			"code":  "ERR_INVALID_REQUEST",
		})
		return
	}

	title := strings.TrimSpace(c.PostForm("title"))
	description := strings.TrimSpace(c.PostForm("description"))

	// Validate the upload
	if err := a.validateVideoUpload(file, title, description); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"code":  "ERR_VALIDATION",
		})
		return
	}

	uploadedFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to open uploaded file",
			"code":  "ERR_FILE_OPEN",
		})
		return
	}
	defer uploadedFile.Close()

	video, err := a.video.UploadVideo(uploadedFile, file.Filename, title, description)
	if err != nil {
		switch e := err.(type) {
		case *ValidationError:
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": e.Error(),
				"code":  "ERR_VALIDATION",
				"field": e.Field,
			})
		case *StorageError:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to store video file",
				"code":  "ERR_STORAGE",
			})
		case *ProcessingError:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to process video",
				"code":  "ERR_PROCESSING",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "An unexpected error occurred",
				"code":  "ERR_INTERNAL",
			})
		}
		return
	}

	a.successResponse(c, gin.H{
		"video":    video,
		"filePath": video.FilePath,
		"ipfsCid":  video.IPFSCID,
	}, "Video uploaded and stored in IPFS successfully")
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
		"checksum":  video.Checksum,
		"ipfsCid":   video.IPFSCID,
		"updatedAt": video.UpdatedAt,
	}, "Video status retrieved successfully")
}
