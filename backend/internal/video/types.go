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
	Storage struct {
		UploadDir string
	}
	Video struct {
		MaxSize        int64
		MinTitleLength int
		MaxTitleLength int
		MaxDescLength  int
		AllowedFormats []string
	}
	TempDir string
	Ffmpeg  FfmpegConfig
}

// FfmpegConfig represents FFmpeg configuration settings
type FfmpegConfig struct {
	Path            string `yaml:"path"`
	ProbePath       string `yaml:"probePath"`
	VideoCodec      string `yaml:"videoCodec"`
	AudioCodec      string `yaml:"audioCodec"`
	Preset          string `yaml:"preset"`
	HLSTime         int    `yaml:"hlsTime"`
	HLSPlaylistType string `yaml:"hlsPlaylistType"`
}

// UploadStatus represents the current status of a video upload
type UploadStatus string

const (
	StatusPending       UploadStatus = "pending"
	StatusIPFSUploading UploadStatus = "ipfs_uploading"
	StatusIPFSCompleted UploadStatus = "ipfs_completed"
	StatusIPFSFailed    UploadStatus = "ipfs_failed"
	StatusS3Uploading   UploadStatus = "s3_uploading"
	StatusS3Failed      UploadStatus = "s3_failed"
	StatusCompleted     UploadStatus = "completed"
	StatusFailed        UploadStatus = "failed"
)

// IsValid checks if the status is a valid upload status
func (s UploadStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusIPFSUploading, StatusIPFSCompleted, StatusIPFSFailed,
		StatusS3Uploading, StatusS3Failed, StatusCompleted, StatusFailed:
		return true
	}
	return false
}

// String returns the string representation of the status
func (s UploadStatus) String() string {
	return string(s)
}

// GetMessage returns a human-readable message for the status
func (s UploadStatus) GetMessage() string {
	switch s {
	case StatusPending:
		return "Upload pending"
	case StatusIPFSUploading:
		return "Uploading to IPFS"
	case StatusIPFSCompleted:
		return "IPFS upload completed"
	case StatusIPFSFailed:
		return "IPFS upload failed"
	case StatusS3Uploading:
		return "Uploading to S3"
	case StatusS3Failed:
		return "S3 upload failed"
	case StatusCompleted:
		return "Upload completed successfully"
	case StatusFailed:
		return "Upload failed"
	default:
		return "Unknown status"
	}
}

// UploadResponse represents the response for video upload
type UploadResponse struct {
	FileID      string `json:"fileId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// StatusResponse represents the detailed status response
type StatusResponse struct {
	FileID            string     `json:"fileId"`
	Title             string     `json:"title"`
	Status            string     `json:"status"`
	CurrentPhase      string     `json:"currentPhase"`
	TotalSize         int64      `json:"totalSize"`
	TotalProgress     float64    `json:"totalProgress"`
	IPFSProgress      *Progress  `json:"ipfsProgress,omitempty"`
	S3Progress        *Progress  `json:"s3Progress,omitempty"`
	ErrorMessage      string     `json:"errorMessage,omitempty"`
	CompletedAt       *time.Time `json:"completedAt,omitempty"`
	EstimatedDuration string     `json:"estimatedDuration,omitempty"`
}

// Progress represents upload progress for a specific phase
type Progress struct {
	BytesUploaded int64      `json:"bytesUploaded"`
	Percentage    float64    `json:"percentage"`
	StartTime     *time.Time `json:"startTime,omitempty"`
	EndTime       *time.Time `json:"endTime,omitempty"`
	Duration      string     `json:"duration,omitempty"`
}

// HLSResult represents the result of HLS transcoding
type HLSResult struct {
	Transcode Transcode
	Segments  []TranscodeSegment
}

// TranscodeResult represents the complete result of transcoding process
type TranscodeResult struct {
	Transcodes        []Transcode
	TranscodeSegments []TranscodeSegment
}
