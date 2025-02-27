package video

import "time"

// App represents the application context needed by video handlers
type App struct {
	Config          *Config
	Logger          Logger
	Video           VideoService
	IPFS            IPFSService
	ResponseHandler ResponseHandler
}

// Config represents the configuration for video handling
type Config struct {
	Video struct {
		MaxFileSize    int64    `yaml:"max_file_size"`    // Maximum allowed file size in bytes
		MinTitleLength int      `yaml:"min_title_length"` // Minimum length for video title
		MaxTitleLength int      `yaml:"max_title_length"` // Maximum length for video title
		MaxDescLength  int      `yaml:"max_desc_length"`  // Maximum length for video description
		AllowedFormats []string `yaml:"allowed_formats"`  // List of allowed video formats
	}
	FFmpeg FfmpegConfig `yaml:"ffmpeg"` // FFmpeg configuration
}

// FfmpegConfig represents FFmpeg configuration settings
type FfmpegConfig struct {
	Path        string   `yaml:"path"`        // Path to FFmpeg binary
	ProbePath   string   `yaml:"probe_path"`  // Path to FFprobe binary
	VideoCodec  string   `yaml:"video_codec"` // Video codec to use (e.g., h264)
	AudioCodec  string   `yaml:"audio_codec"` // Audio codec to use (e.g., aac)
	Preset      string   `yaml:"preset"`      // Encoding preset
	OutputPath  string   `yaml:"output_path"` // Path for transcoded outputs
	Resolutions []string `yaml:"resolutions"` // List of output resolutions
}

// UploadStatus represents the status of a video upload
type UploadStatus string

const (
	UploadStatusPending   UploadStatus = "pending"
	UploadStatusUploading UploadStatus = "uploading"
	UploadStatusCompleted UploadStatus = "completed"
	UploadStatusFailed    UploadStatus = "failed"
)

// IsValid checks if the status is a valid upload status
func (s UploadStatus) IsValid() bool {
	switch s {
	case UploadStatusPending, UploadStatusUploading, UploadStatusCompleted, UploadStatusFailed:
		return true
	}
	return false
}

// APIResponse represents a standardized API response
type APIResponse struct {
	Message string      `json:"message,omitempty"`
	Status  string      `json:"status"`
	Data    interface{} `json:"data,omitempty"`
}

// UploadResponse represents the response for a video upload
type UploadResponse struct {
	ID          string          `json:"id"`
	FileID      string          `json:"file_id"`
	StoragePath string          `json:"storage_path"`
	IPFSCID     string          `json:"ipfs_cid"`
	Status      string          `json:"status"`
	Transcodes  []TranscodeInfo `json:"transcodes,omitempty"`
}

// VideoListResponse represents the response for listing videos
type VideoListResponse struct {
	Videos []VideoDetailsResponse `json:"videos"`
	Total  int64                  `json:"total"`
	Page   int                    `json:"page"`
	Limit  int                    `json:"limit"`
}

// VideoInfo represents the basic video information
type VideoInfo struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// VideoDetailsResponse represents the detailed video information
type VideoDetailsResponse struct {
	ID          string          `json:"id"`
	FileID      string          `json:"file_id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	StoragePath string          `json:"storage_path"`
	IPFSCID     string          `json:"ipfs_cid"`
	Status      string          `json:"status"`
	FileSize    int64           `json:"file_size"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Transcodes  []TranscodeInfo `json:"transcodes,omitempty"`
}

// TranscodeInfo represents transcode information in responses
type TranscodeInfo struct {
	ID         string                 `json:"id"`
	Format     string                 `json:"format"`
	Resolution string                 `json:"resolution"`
	Segments   []TranscodeSegmentInfo `json:"segments,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// TranscodeSegmentInfo represents transcode segment information in responses
type TranscodeSegmentInfo struct {
	ID          string `json:"id"`
	StoragePath string `json:"storage_path"`
	IPFSCID     string `json:"ipfs_cid"`
	Duration    int    `json:"duration"`
}

// VideoUpdateRequest represents the request for updating video metadata
type VideoUpdateRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}
