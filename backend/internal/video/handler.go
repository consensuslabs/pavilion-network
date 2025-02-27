package video

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve video details", err)
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

	// Directly pass the UploadResponse to SuccessResponse without wrapping it in APIResponse
	h.app.ResponseHandler.SuccessResponse(c, response, "Upload completed successfully")
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

// @Summary Get video details
// @Description Retrieve detailed information about a specific video
// @Tags video
// @Produce json
// @Security BearerAuth
// @Param id path string true "Video ID (UUID)"
// @Success 200 {object} APIResponse{data=VideoDetailsResponse} "Video details retrieved successfully"
// @Failure 401 {object} APIResponse "Unauthorized"
// @Failure 404 {object} APIResponse "Video not found"
// @Failure 500 {object} APIResponse "Internal server error"
// @Router /video/{id} [get]
func (h *VideoHandler) GetVideo(c *gin.Context) {
	requestID := c.GetString("request_id")
	videoID := c.Param("id")

	// Check authentication
	if !isAuthenticated(c) {
		h.app.ResponseHandler.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	// Parse UUID from string
	uuid, err := parseUUID(videoID)
	if err != nil {
		h.app.Logger.LogInfo("Invalid video ID format", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "INVALID_ID", "Invalid video ID format", err)
		return
	}

	// Get video details
	video, err := h.app.Video.GetVideo(uuid)
	if err != nil {
		h.app.Logger.LogInfo("Failed to get video details", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve video details", err)
		return
	}

	if video == nil {
		h.app.Logger.LogInfo("Video not found", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusNotFound, "VIDEO_NOT_FOUND", "Video not found", nil)
		return
	}

	// Convert to API response
	response := video.ToVideoDetailsResponse()

	h.app.Logger.LogInfo("Video details retrieved successfully", map[string]interface{}{
		"request_id": requestID,
		"video_id":   videoID,
	})

	// Directly pass the VideoDetailsResponse to SuccessResponse without wrapping it in APIResponse
	h.app.ResponseHandler.SuccessResponse(c, response, "Video details retrieved successfully")
}

// Helper function to parse UUID from string
func parseUUID(id string) (uuid.UUID, error) {
	return uuid.Parse(id)
}

// @Summary List videos
// @Description Retrieve a paginated list of videos with detailed information including transcodes
// @Tags video
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Number of videos to return (default: 10, max: 50)"
// @Param page query int false "Page number for pagination (default: 1)"
// @Success 200 {object} APIResponse{data=VideoListResponse} "Videos retrieved successfully with detailed information"
// @Failure 400 {object} APIResponse "Invalid request parameters"
// @Failure 401 {object} APIResponse "Unauthorized"
// @Failure 500 {object} APIResponse "Internal server error"
// @Router /videos [get]
func (h *VideoHandler) ListVideos(c *gin.Context) {
	requestID := c.GetString("request_id")

	// Check authentication
	if !isAuthenticated(c) {
		h.app.ResponseHandler.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	// Parse pagination parameters
	limit := 10 // Default limit
	page := 1   // Default page

	if limitParam := c.Query("limit"); limitParam != "" {
		parsedLimit, err := strconv.Atoi(limitParam)
		if err != nil || parsedLimit <= 0 {
			h.app.Logger.LogInfo("Invalid limit parameter", map[string]interface{}{
				"request_id": requestID,
				"limit":      limitParam,
			})
			h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "INVALID_PARAMETER", "Invalid limit parameter, must be a positive integer", nil)
			return
		}
		// Cap limit to prevent excessive queries
		if parsedLimit > 50 {
			parsedLimit = 50
		}
		limit = parsedLimit
	}

	if pageParam := c.Query("page"); pageParam != "" {
		parsedPage, err := strconv.Atoi(pageParam)
		if err != nil || parsedPage <= 0 {
			h.app.Logger.LogInfo("Invalid page parameter", map[string]interface{}{
				"request_id": requestID,
				"page":       pageParam,
			})
			h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "INVALID_PARAMETER", "Invalid page parameter, must be a positive integer", nil)
			return
		}
		page = parsedPage
	}

	// Query videos
	videos, err := h.app.Video.ListVideos(page, limit)
	if err != nil {
		h.app.Logger.LogInfo("Failed to list videos", map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve videos", err)
		return
	}

	// Build response with detailed video information
	videoDetails := make([]VideoDetailsResponse, 0, len(videos))
	for _, video := range videos {
		videoDetails = append(videoDetails, video.ToVideoDetailsResponse())
	}

	response := VideoListResponse{
		Videos: videoDetails,
		Page:   page,
		Limit:  limit,
		Total:  int64(len(videoDetails)), // For MVP, this is just the current page count - note we're converting to int64
	}

	h.app.Logger.LogInfo("Videos retrieved successfully", map[string]interface{}{
		"request_id": requestID,
		"count":      len(videos),
		"page":       page,
		"limit":      limit,
	})

	// Directly pass the VideoListResponse to SuccessResponse without wrapping it in APIResponse
	h.app.ResponseHandler.SuccessResponse(c, response, "Videos retrieved successfully")
}

// @Summary Get video upload status
// @Description Retrieve the current upload status of a specific video
// @Tags video
// @Produce json
// @Security BearerAuth
// @Param id path string true "Video ID (UUID)"
// @Success 200 {object} APIResponse{data=map[string]string} "Video status retrieved successfully"
// @Failure 401 {object} APIResponse "Unauthorized"
// @Failure 404 {object} APIResponse "Video not found"
// @Failure 500 {object} APIResponse "Internal server error"
// @Router /video/{id}/status [get]
func (h *VideoHandler) GetVideoStatus(c *gin.Context) {
	requestID := c.GetString("request_id")
	videoID := c.Param("id")

	// Check authentication
	if !isAuthenticated(c) {
		h.app.ResponseHandler.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	// Parse UUID from string
	uuid, err := parseUUID(videoID)
	if err != nil {
		h.app.Logger.LogInfo("Invalid video ID format", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "INVALID_ID", "Invalid video ID format", err)
		return
	}

	// Get video details
	video, err := h.app.Video.GetVideo(uuid)
	if err != nil {
		h.app.Logger.LogInfo("Failed to get video status", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve video status", err)
		return
	}

	if video == nil {
		h.app.Logger.LogInfo("Video not found", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusNotFound, "VIDEO_NOT_FOUND", "Video not found", nil)
		return
	}

	// Get upload status
	status := "unknown"
	if video.Upload != nil {
		status = string(video.Upload.Status)
	}

	h.app.Logger.LogInfo("Video status retrieved successfully", map[string]interface{}{
		"request_id": requestID,
		"video_id":   videoID,
		"status":     status,
	})

	// Directly pass the status map to SuccessResponse without wrapping it in APIResponse
	h.app.ResponseHandler.SuccessResponse(c, map[string]string{"status": status}, "Video status retrieved successfully")
}

// @Summary Update video details
// @Description Update a video's title and/or description
// @Tags video
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Video ID (UUID)"
// @Param request body VideoUpdateRequest true "Update request"
// @Success 200 {object} APIResponse{data=VideoDetailsResponse} "Video updated successfully"
// @Failure 400 {object} APIResponse "Invalid request format or validation error"
// @Failure 401 {object} APIResponse "Unauthorized"
// @Failure 404 {object} APIResponse "Video not found"
// @Failure 500 {object} APIResponse "Internal server error"
// @Router /video/{id} [patch]
func (h *VideoHandler) UpdateVideo(c *gin.Context) {
	requestID := c.GetString("request_id")
	videoID := c.Param("id")

	// Check authentication
	if !isAuthenticated(c) {
		h.app.ResponseHandler.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	// Parse UUID from string
	uuid, err := parseUUID(videoID)
	if err != nil {
		h.app.Logger.LogInfo("Invalid video ID format", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "INVALID_ID", "Invalid video ID format", err)
		return
	}

	// Parse request body
	var request VideoUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.app.Logger.LogInfo("Invalid update request format", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request format", err)
		return
	}

	// Validate the request
	if err := h.validateUpdateRequest(&request); err != nil {
		h.app.Logger.LogInfo("Video update validation failed", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), err)
		return
	}

	// Get the current video to check if it exists
	video, err := h.app.Video.GetVideo(uuid)
	if err != nil {
		h.app.Logger.LogInfo("Failed to get video for update", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve video for update", err)
		return
	}

	if video == nil {
		h.app.Logger.LogInfo("Video not found for update", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusNotFound, "VIDEO_NOT_FOUND", "Video not found", nil)
		return
	}

	// Extract values to update
	title := video.Title
	description := video.Description

	if request.Title != nil {
		title = *request.Title
	}
	if request.Description != nil {
		description = *request.Description
	}

	// Update the video
	if err := h.app.Video.UpdateVideo(uuid, title, description); err != nil {
		h.app.Logger.LogInfo("Failed to update video", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update video", err)
		return
	}

	// Get updated video
	updatedVideo, err := h.app.Video.GetVideo(uuid)
	if err != nil {
		h.app.Logger.LogInfo("Failed to get updated video", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve updated video", err)
		return
	}

	h.app.Logger.LogInfo("Video updated successfully", map[string]interface{}{
		"request_id": requestID,
		"video_id":   videoID,
	})

	// Directly pass the VideoDetailsResponse to SuccessResponse without wrapping it in APIResponse
	h.app.ResponseHandler.SuccessResponse(c, updatedVideo.ToVideoDetailsResponse(), "Video updated successfully")
}

// validateUpdateRequest validates the video update request
func (h *VideoHandler) validateUpdateRequest(request *VideoUpdateRequest) error {
	// Check if at least one field is being updated
	if request.Title == nil && request.Description == nil {
		return errors.New("at least one field (title or description) must be provided")
	}

	// Validate title if provided
	if request.Title != nil {
		if len(*request.Title) < h.app.Config.Video.MinTitleLength || len(*request.Title) > h.app.Config.Video.MaxTitleLength {
			return fmt.Errorf("title must be between %d and %d characters",
				h.app.Config.Video.MinTitleLength,
				h.app.Config.Video.MaxTitleLength)
		}
	}

	// Validate description if provided
	if request.Description != nil && len(*request.Description) > h.app.Config.Video.MaxDescLength {
		return fmt.Errorf("description cannot exceed %d characters", h.app.Config.Video.MaxDescLength)
	}

	return nil
}

// @Summary Delete video
// @Description Delete a video and its associated data
// @Tags video
// @Produce json
// @Security BearerAuth
// @Param id path string true "Video ID (UUID)"
// @Success 200 {object} APIResponse "Video deleted successfully"
// @Failure 400 {object} APIResponse "Invalid video ID format"
// @Failure 401 {object} APIResponse "Unauthorized"
// @Failure 404 {object} APIResponse "Video not found"
// @Failure 500 {object} APIResponse "Internal server error"
// @Router /video/{id} [delete]
func (h *VideoHandler) DeleteVideo(c *gin.Context) {
	requestID := c.GetString("request_id")
	videoID := c.Param("id")

	// Check authentication
	if !isAuthenticated(c) {
		h.app.ResponseHandler.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	// Parse UUID from string
	uuid, err := parseUUID(videoID)
	if err != nil {
		h.app.Logger.LogInfo("Invalid video ID format", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusBadRequest, "INVALID_ID", "Invalid video ID format", err)
		return
	}

	// Check if video exists
	video, err := h.app.Video.GetVideo(uuid)
	if err != nil {
		h.app.Logger.LogInfo("Failed to get video for deletion", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve video for deletion", err)
		return
	}

	if video == nil {
		h.app.Logger.LogInfo("Video not found for deletion", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusNotFound, "VIDEO_NOT_FOUND", "Video not found", nil)
		return
	}

	// Delete the video
	if err := h.app.Video.DeleteVideo(uuid); err != nil {
		h.app.Logger.LogInfo("Failed to delete video", map[string]interface{}{
			"request_id": requestID,
			"video_id":   videoID,
			"error":      err.Error(),
		})
		h.app.ResponseHandler.ErrorResponse(c, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete video", err)
		return
	}

	h.app.Logger.LogInfo("Video deleted successfully", map[string]interface{}{
		"request_id": requestID,
		"video_id":   videoID,
	})

	// Directly pass a success message to SuccessResponse without wrapping it in APIResponse
	h.app.ResponseHandler.SuccessResponse(c, nil, "Video deleted successfully")
}
