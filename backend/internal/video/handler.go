package video

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// VideoHandler handles HTTP requests for video operations
type VideoHandler struct {
	app *App
}

// NewVideoHandler creates a new VideoHandler instance
func NewVideoHandler(app *App) *VideoHandler {
	return &VideoHandler{app: app}
}

// @Summary Upload video
// @Description Upload a new video file
// @Tags video
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param video formData file true "Video file to upload (.mp4, .mov)"
// @Param title formData string true "Video title (3-100 characters)" minLength(3) maxLength(100)
// @Param description formData string false "Video description (max 1000 characters)" maxLength(1000)
// @Success 200 {object} APIResponse{data=UploadResponse} "Upload completed successfully"
// @Failure 400 {object} APIResponse "Invalid request format or validation error"
// @Failure 401 {object} APIResponse "Unauthorized"
// @Failure 500 {object} APIResponse "Processing error"
// @Router /video/upload [post]
func (h *VideoHandler) HandleUpload(c *gin.Context) {
	requestID := c.GetString("request_id")

	// Check authentication
	if !isAuthenticated(c) {
		h.app.ResponseHandler.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

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

	// Create initial upload record
	upload, err := h.app.Video.InitializeUpload(title, description, fileHeader.Size)
	if err != nil {
		h.app.Logger.LogInfo("Failed to initialize upload", map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "UPLOAD_FAILED", "Failed to initialize upload", err)
		return
	}

	// Process upload synchronously
	if err := h.app.Video.ProcessUpload(upload, file, fileHeader); err != nil {
		h.app.Logger.LogInfo("Video processing failed", map[string]interface{}{
			"request_id": requestID,
			"filename":   fileHeader.Filename,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "TRANSCODE_FAILED", "Video transcoding failed", err)
		return
	}

	// Get updated video record with transcodes
	video, err := h.app.Video.GetVideo(upload.VideoID)
	if err != nil {
		h.app.Logger.LogInfo("Failed to get video details", map[string]interface{}{
			"request_id": requestID,
			"video_id":   upload.VideoID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "VIDEO_NOT_FOUND", "Failed to get video details", err)
		return
	}

	// Build response
	response := UploadResponse{
		ID:          video.ID.String(),
		FileID:      video.FileID,
		StoragePath: video.StoragePath,
		IPFSCID:     video.IPFSCID,
		Status:      string(upload.Status),
		Transcodes:  make([]TranscodeInfo, 0),
	}

	// Add transcodes to response
	for _, t := range video.Transcodes {
		segments := make([]TranscodeSegmentInfo, 0, len(t.Segments))
		resolution := "original" // Default resolution

		for _, s := range t.Segments {
			segments = append(segments, TranscodeSegmentInfo{
				ID:          s.ID.String(),
				StoragePath: s.StoragePath,
				IPFSCID:     s.IPFSCID,
				Duration:    s.Duration,
			})

			// Extract resolution from storage path (e.g., videos/{video_id}/720p.mp4)
			if s.StoragePath != "" {
				parts := strings.Split(s.StoragePath, "/")
				if len(parts) > 0 {
					lastPart := parts[len(parts)-1]
					if strings.HasSuffix(lastPart, ".mp4") {
						res := strings.TrimSuffix(lastPart, ".mp4")
						if res == "720p" || res == "480p" || res == "360p" {
							resolution = res
						}
					}
				}
			}
		}

		response.Transcodes = append(response.Transcodes, TranscodeInfo{
			ID:         t.ID.String(),
			Format:     t.Format,
			Resolution: resolution,
			Segments:   segments,
			CreatedAt:  t.CreatedAt,
		})
	}

	h.app.Logger.LogInfo("Video upload completed successfully", map[string]interface{}{
		"request_id": requestID,
		"video_id":   upload.VideoID,
		"file_path":  upload.Video.StoragePath,
	})

	h.app.ResponseHandler.SuccessResponse(c, APIResponse{
		Message: "Upload completed successfully",
		Status:  "success",
		Data:    response,
	}, "")
}

// Helper function to check authentication
func isAuthenticated(c *gin.Context) bool {
	// Get the Authorization header
	authHeader := c.GetHeader("Authorization")
	return authHeader != "" && strings.HasPrefix(authHeader, "Bearer ")
}

// validateVideoUpload validates the video upload request
func (h *VideoHandler) validateVideoUpload(fileHeader *multipart.FileHeader, title, description string) error {
	if fileHeader == nil {
		return errors.New("no video file provided")
	}

	// Validate file size
	if fileHeader.Size > h.app.Config.Video.MaxFileSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", h.app.Config.Video.MaxFileSize)
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	validExts := make(map[string]bool)
	for _, format := range h.app.Config.Video.AllowedFormats {
		validExts[format] = true
	}
	if !validExts[ext] {
		return fmt.Errorf("invalid file type, allowed formats: %v", h.app.Config.Video.AllowedFormats)
	}

	// Validate title
	if len(title) < h.app.Config.Video.MinTitleLength || len(title) > h.app.Config.Video.MaxTitleLength {
		return fmt.Errorf("title must be between %d and %d characters",
			h.app.Config.Video.MinTitleLength,
			h.app.Config.Video.MaxTitleLength)
	}

	// Validate description if provided
	if description != "" && len(description) > h.app.Config.Video.MaxDescLength {
		return fmt.Errorf("description cannot exceed %d characters", h.app.Config.Video.MaxDescLength)
	}

	return nil
}
