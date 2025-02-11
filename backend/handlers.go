package main

import (
	"bytes"
	"fmt"
	"io"
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
	requestID := c.GetString("request_id")

	file, fileHeader, err := c.Request.FormFile("video")
	if err != nil {
		a.logger.LogInfo("No video file received", map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		})
		a.errorResponse(c, http.StatusBadRequest, "ERR_NO_FILE", "No video file received", err)
		return
	}
	defer file.Close()

	// Get title and description from form
	title := c.PostForm("title")
	description := c.PostForm("description")

	// Log request parameters
	a.logger.LogInfo("Received video upload request", map[string]interface{}{
		"request_id":         requestID,
		"filename":           fileHeader.Filename,
		"filesize":           fileHeader.Size,
		"title":              title,
		"description_length": len(description),
		"content_type":       fileHeader.Header.Get("Content-Type"),
	})

	// Validate the upload
	if err := a.video.validateVideoUpload(fileHeader, title, description); err != nil {
		a.logger.LogInfo("Video upload validation failed", map[string]interface{}{
			"request_id": requestID,
			"filename":   fileHeader.Filename,
			"error":      err.Error(),
		})
		a.errorResponse(c, http.StatusBadRequest, "ERR_VALIDATION", err.Error(), err)
		return
	}

	// Process the video (upload to IPFS and S3)
	video, err := a.video.ProcessVideo(file, fileHeader, title, description)
	if err != nil {
		a.logger.LogInfo("Video processing failed", map[string]interface{}{
			"request_id": requestID,
			"filename":   fileHeader.Filename,
			"error":      err.Error(),
		})
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
		"status":      video.UploadStatus,
		"statusMsg":   video.GetUploadStatusMessage(),
		"fileSize":    video.FileSize,
		"createdAt":   video.CreatedAt,
		"updatedAt":   video.UpdatedAt,
	}

	// Log successful response
	a.logger.LogInfo("Video upload processed successfully", map[string]interface{}{
		"request_id": requestID,
		"filename":   fileHeader.Filename,
		"video_id":   video.ID,
		"file_id":    video.FileId,
		"ipfs_cid":   video.IPFSCID,
		"status":     video.UploadStatus,
	})

	a.successResponse(c, gin.H{
		"video":    videoResponse,
		"filePath": video.FilePath,
		"ipfsCid":  video.IPFSCID,
	}, video.GetUploadStatusMessage())
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
		"status":    video.UploadStatus,
		"message":   video.GetUploadStatusMessage(),
		"fileId":    video.FileId,
		"title":     video.Title,
		"fileSize":  video.FileSize,
		"ipfsCid":   video.IPFSCID,
		"updatedAt": video.UpdatedAt,
	}, "Video status retrieved successfully")
}

// handleVideoTranscode handles the /video/transcode endpoint
func (a *App) handleVideoTranscode(c *gin.Context) {
	// Log raw request body for debugging
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		a.logger.LogInfo("Error reading raw request body", map[string]interface{}{"error": err.Error()})
	} else {
		a.logger.LogInfo("Raw request body received", map[string]interface{}{"body": string(bodyBytes)})
	}
	// Restore the request body so that it can be read again by ShouldBindJSON
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var jsonInput struct {
		CID string `json:"cid"`
	}
	if err := c.ShouldBindJSON(&jsonInput); err != nil {
		a.logger.LogInfo("Invalid JSON input", map[string]interface{}{"error": err.Error()})
		a.errorResponse(c, http.StatusBadRequest, "ERR_INVALID_JSON", "Invalid JSON input. Please provide cid in request body", err)
		return
	}

	// Check if CID is empty
	if jsonInput.CID == "" {
		a.logger.LogInfo("Missing CID in request body", map[string]interface{}{
			"error": "missing_cid",
		})
		a.errorResponse(c, http.StatusBadRequest, "ERR_MISSING_CID", "Missing CID in request body", nil)
		return
	}

	// Start transcoding process
	a.logger.LogInfo("Starting transcoding process", map[string]interface{}{
		"cid": jsonInput.CID,
	})

	result, err := a.transcode.ProcessTranscode(jsonInput.CID)
	if err != nil {
		a.logger.LogInfo("Transcoding failed", map[string]interface{}{
			"error": err.Error(),
			"cid":   jsonInput.CID,
		})
		a.errorResponse(c, http.StatusInternalServerError, "ERR_TRANSCODE", fmt.Sprintf("Failed to process transcode: %v", err), err)
		return
	}

	// Format response
	response := formatTranscodeResponse(result)
	a.successResponse(c, response, "Transcoding started successfully")
}

type TranscodeResponse struct {
	Success bool                 `json:"success"`
	Formats map[string][]Version `json:"formats"`
}

type Version struct {
	Resolution  string   `json:"resolution"`
	StorageType string   `json:"storage_type"`
	URL         string   `json:"url"`
	CID         string   `json:"cid,omitempty"`
	Segments    []string `json:"segments,omitempty"`
}

func formatTranscodeResponse(result *TranscodeResult) TranscodeResponse {
	response := TranscodeResponse{
		Success: true,
		Formats: make(map[string][]Version),
	}

	// Group transcodes by format
	for _, transcode := range result.Transcodes {
		version := Version{
			Resolution:  transcode.Resolution,
			StorageType: transcode.StorageType,
			URL:         transcode.FilePath,
			CID:         transcode.FileCID,
		}

		// For HLS format, add segment URLs
		if transcode.Format == "hls" {
			segmentURLs := make([]string, 0)
			for _, segment := range result.TranscodeSegments {
				if segment.TranscodeID == transcode.ID {
					segmentURLs = append(segmentURLs, segment.FilePath)
				}
			}
			version.Segments = segmentURLs
		}

		response.Formats[transcode.Format] = append(response.Formats[transcode.Format], version)
	}

	return response
}
