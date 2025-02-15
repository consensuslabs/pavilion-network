package video

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
)

// validateVideoUpload validates the video upload request against configuration settings
func (h *VideoHandler) validateVideoUpload(file *multipart.FileHeader, title, description string) error {
	// Check file size
	if file.Size > h.app.Config.Video.MaxSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d MB", h.app.Config.Video.MaxSize/1024/1024)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validExt := false
	for _, format := range h.app.Config.Video.AllowedFormats {
		if ext == format {
			validExt = true
			break
		}
	}
	if !validExt {
		return fmt.Errorf("unsupported file type: %s. Allowed types: %v", ext, h.app.Config.Video.AllowedFormats)
	}

	// Validate title
	title = strings.TrimSpace(title)
	if len(title) < h.app.Config.Video.MinTitleLength {
		return fmt.Errorf("title must be at least %d characters", h.app.Config.Video.MinTitleLength)
	}
	if len(title) > h.app.Config.Video.MaxTitleLength {
		return fmt.Errorf("title must not exceed %d characters", h.app.Config.Video.MaxTitleLength)
	}

	// Validate description
	if len(description) > h.app.Config.Video.MaxDescLength {
		return fmt.Errorf("description must not exceed %d characters", h.app.Config.Video.MaxDescLength)
	}

	return nil
}
