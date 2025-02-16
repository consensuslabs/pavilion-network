package video

import (
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// VideoHandler handles HTTP requests for video operations
type VideoHandler struct {
	app *App
}

// NewVideoHandler creates a new video handler
func NewVideoHandler(app *App) *VideoHandler {
	return &VideoHandler{app: app}
}

// HandleUpload handles video upload requests
func (h *VideoHandler) HandleUpload(c *gin.Context) {
	requestID := c.GetString("request_id")

	file, fileHeader, err := c.Request.FormFile("video")
	if err != nil {
		h.app.Logger.LogInfo("No video file received", map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "ERR_NO_FILE", "No video file received", err)
		return
	}
	defer file.Close()

	title := c.PostForm("title")
	description := c.PostForm("description")

	if err := h.validateVideoUpload(fileHeader, title, description); err != nil {
		h.app.Logger.LogInfo("Video upload validation failed", map[string]interface{}{
			"request_id": requestID,
			"filename":   fileHeader.Filename,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "ERR_VALIDATION", err.Error(), err)
		return
	}

	// Create initial upload record first
	upload, err := h.app.Video.InitializeUpload(title, description, fileHeader.Size)
	if err != nil {
		h.app.Logger.LogInfo("Failed to initialize upload", map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "UPLOAD_FAILED", "Failed to initialize upload", err)
		return
	}

	// Return the upload record to client immediately
	response := UploadResponse{
		FileID:      upload.TempFileId,
		Title:       upload.Title,
		Description: upload.Description,
		Status:      string(upload.UploadStatus),
	}

	h.app.Logger.LogInfo("Upload initialized", map[string]interface{}{
		"request_id": requestID,
		"filename":   fileHeader.Filename,
		"file_id":    upload.TempFileId,
	})

	// Start processing in background
	go func() {
		if err := h.app.Video.ProcessUpload(upload, file, fileHeader); err != nil {
			h.app.Logger.LogInfo("Video processing failed", map[string]interface{}{
				"request_id": requestID,
				"filename":   fileHeader.Filename,
				"error":      err.Error(),
			})
		}
	}()

	h.app.ResponseHandler.SuccessResponse(c, response, "Upload initiated successfully")
}

// HandleWatch handles the video watch endpoint
func (h *VideoHandler) HandleWatch(c *gin.Context) {
	cid := c.Query("cid")
	file := c.Query("file")

	if cid != "" {
		ipfsURL := h.app.IPFS.GetGatewayURL(cid)
		c.Redirect(http.StatusTemporaryRedirect, ipfsURL)
		return
	}

	if file != "" {
		c.File(filepath.Join(h.app.Config.Storage.UploadDir, file))
		return
	}

	h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "ERR_NO_PARAM", "No 'cid' or 'file' parameter provided", nil)
}

// HandleList handles video list requests
func (h *VideoHandler) HandleList(c *gin.Context) {
	uploads, err := h.app.Video.GetVideoList()
	if err != nil {
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "LIST_FAILED", "Failed to retrieve video list", err)
		return
	}

	var response []StatusResponse
	for _, upload := range uploads {
		response = append(response, h.buildStatusResponse(&upload))
	}

	h.app.ResponseHandler.SuccessResponse(c, response, "Video list retrieved successfully")
}

// HandleStatus handles video status requests
func (h *VideoHandler) HandleStatus(c *gin.Context) {
	fileID := c.Param("fileId")
	if fileID == "" {
		h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "File ID is required", nil)
		return
	}

	upload, err := h.app.Video.GetVideoStatus(fileID)
	if err != nil {
		h.app.ResponseHandler.ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", "Video upload not found", err)
		return
	}

	response := h.buildStatusResponse(upload)
	h.app.ResponseHandler.SuccessResponse(c, response, "Video status retrieved successfully")
}

// HandleTranscode handles the video transcode endpoint
func (h *VideoHandler) HandleTranscode(c *gin.Context) {
	var jsonInput struct {
		CID string `json:"cid"`
	}
	if err := c.ShouldBindJSON(&jsonInput); err != nil {
		h.app.Logger.LogInfo("Invalid JSON input", map[string]interface{}{
			"error": err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "ERR_INVALID_JSON", "Invalid JSON input. Please provide cid in request body", err)
		return
	}

	if jsonInput.CID == "" {
		h.app.Logger.LogInfo("Missing CID in request body", map[string]interface{}{
			"error": "missing_cid",
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "ERR_MISSING_CID", "Missing CID in request body", nil)
		return
	}
}

// buildStatusResponse creates a detailed status response from a video upload
func (h *VideoHandler) buildStatusResponse(upload *VideoUpload) StatusResponse {
	response := StatusResponse{
		FileID:        upload.TempFileId,
		Title:         upload.Title,
		Status:        string(upload.UploadStatus),
		CurrentPhase:  upload.CurrentPhase,
		TotalSize:     upload.FileSize,
		TotalProgress: upload.GetUploadProgress(),
		ErrorMessage:  upload.ErrorMessage,
	}

	// Add IPFS progress if available
	if upload.IPFSStartTime != nil {
		ipfsProgress := &Progress{
			BytesUploaded: upload.IPFSBytesUploaded,
			StartTime:     upload.IPFSStartTime,
			EndTime:       upload.IPFSEndTime,
		}
		if upload.FileSize > 0 {
			ipfsProgress.Percentage = float64(upload.IPFSBytesUploaded) / float64(upload.FileSize) * 100
		}
		if upload.IPFSStartTime != nil && upload.IPFSEndTime != nil {
			ipfsProgress.Duration = upload.IPFSEndTime.Sub(*upload.IPFSStartTime).String()
		}
		response.IPFSProgress = ipfsProgress
	}

	// Add S3 progress if available
	if upload.S3StartTime != nil {
		s3Progress := &Progress{
			BytesUploaded: upload.S3BytesUploaded,
			StartTime:     upload.S3StartTime,
			EndTime:       upload.S3EndTime,
		}
		if upload.FileSize > 0 {
			s3Progress.Percentage = float64(upload.S3BytesUploaded) / float64(upload.FileSize) * 100
		}
		if upload.S3StartTime != nil && upload.S3EndTime != nil {
			s3Progress.Duration = upload.S3EndTime.Sub(*upload.S3StartTime).String()
		}
		response.S3Progress = s3Progress
	}

	// Calculate estimated duration considering both phases
	if upload.UploadStatus != StatusCompleted && upload.UploadStatus != StatusFailed {
		var totalElapsedTime time.Duration
		var totalBytesUploaded int64

		// Get the initial start time from IPFS phase
		if upload.IPFSStartTime != nil {
			totalBytesUploaded += upload.IPFSBytesUploaded

			// If IPFS phase is completed, add its duration
			if upload.IPFSEndTime != nil {
				totalElapsedTime += upload.IPFSEndTime.Sub(*upload.IPFSStartTime)
			} else {
				totalElapsedTime += time.Since(*upload.IPFSStartTime)
			}
		}

		// Add S3 phase progress if started
		if upload.S3StartTime != nil {
			totalBytesUploaded += upload.S3BytesUploaded
			if upload.S3EndTime != nil {
				totalElapsedTime += upload.S3EndTime.Sub(*upload.S3StartTime)
			} else {
				totalElapsedTime += time.Since(*upload.S3StartTime)
			}
		}

		// Calculate total progress and estimate remaining time
		if totalBytesUploaded > 0 && upload.FileSize > 0 {
			// We need to upload the file twice (once for IPFS, once for S3)
			totalProgress := float64(totalBytesUploaded) / (float64(upload.FileSize) * 2)
			if totalProgress > 0 {
				estimatedTotal := float64(totalElapsedTime) / totalProgress
				response.EstimatedDuration = time.Duration(estimatedTotal).String()
			}
		}
	}

	// Set completed time if upload is finished
	if upload.UploadStatus == StatusCompleted && upload.S3EndTime != nil {
		response.CompletedAt = upload.S3EndTime
	}

	return response
}
