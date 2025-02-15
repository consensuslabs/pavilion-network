package video

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// VideoHandler handles video-related HTTP requests
type VideoHandler struct {
	app *App
}

// NewVideoHandler creates a new video handler instance
func NewVideoHandler(app *App) *VideoHandler {
	return &VideoHandler{app: app}
}

// HandleUpload handles the video upload endpoint
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

	h.app.Logger.LogInfo("Received video upload request", map[string]interface{}{
		"request_id":         requestID,
		"filename":           fileHeader.Filename,
		"filesize":           fileHeader.Size,
		"title":              title,
		"description_length": len(description),
		"content_type":       fileHeader.Header.Get("Content-Type"),
	})

	if err := h.validateVideoUpload(fileHeader, title, description); err != nil {
		h.app.Logger.LogInfo("Video upload validation failed", map[string]interface{}{
			"request_id": requestID,
			"filename":   fileHeader.Filename,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "ERR_VALIDATION", err.Error(), err)
		return
	}

	video, err := h.app.Video.ProcessVideo(file, fileHeader, title, description)
	if err != nil {
		h.app.Logger.LogInfo("Video processing failed", map[string]interface{}{
			"request_id": requestID,
			"filename":   fileHeader.Filename,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "ERR_UPLOAD", "Failed to process video upload", err)
		return
	}

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

	h.app.Logger.LogInfo("Video upload processed successfully", map[string]interface{}{
		"request_id": requestID,
		"filename":   fileHeader.Filename,
		"video_id":   video.ID,
		"file_id":    video.FileId,
		"ipfs_cid":   video.IPFSCID,
		"status":     video.UploadStatus,
	})

	h.app.ResponseHandler.SuccessResponse(c, gin.H{
		"video":    videoResponse,
		"filePath": video.FilePath,
		"ipfsCid":  video.IPFSCID,
	}, video.GetUploadStatusMessage())
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

// HandleList handles the video list endpoint
func (h *VideoHandler) HandleList(c *gin.Context) {
	videos, err := h.app.Video.GetVideoList()
	if err != nil {
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "ERR_DB", "Failed to list videos", err)
		return
	}
	h.app.ResponseHandler.SuccessResponse(c, gin.H{"videos": videos}, "Videos retrieved successfully")
}

// HandleStatus handles the video status endpoint
func (h *VideoHandler) HandleStatus(c *gin.Context) {
	fileId := c.Param("fileId")
	if fileId == "" {
		h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "ERR_NO_FILE_ID", "No file ID provided", nil)
		return
	}

	video, err := h.app.Video.GetVideoStatus(fileId)
	if err != nil {
		h.app.ResponseHandler.ErrorResponse(c, http.StatusNotFound, "ERR_NOT_FOUND", "Video not found", err)
		return
	}

	h.app.ResponseHandler.SuccessResponse(c, gin.H{
		"status":    video.UploadStatus,
		"message":   video.GetUploadStatusMessage(),
		"fileId":    video.FileId,
		"title":     video.Title,
		"fileSize":  video.FileSize,
		"ipfsCid":   video.IPFSCID,
		"updatedAt": video.UpdatedAt,
	}, "Video status retrieved successfully")
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
