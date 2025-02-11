package main

import (
	"time"

	"gorm.io/gorm"
)

// User model definition.
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	Email     string    `gorm:"unique" json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

// UploadStatus represents the video upload status
type UploadStatus string

// Upload status enum values
const (
	UploadStatusPending   UploadStatus = "pending"
	UploadStatusUploading UploadStatus = "uploading"
	UploadStatusCompleted UploadStatus = "completed"
	UploadStatusFailed    UploadStatus = "failed"
)

// Video model definition.
// fileId stores the unique identifier generated during upload.
// Transcodes represents the one-to-many relation to the Transcode table.
type Video struct {
	ID           uint         `gorm:"primaryKey" json:"id"`
	FileId       string       `json:"fileId"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	FilePath     string       `json:"filePath"`
	IPFSCID      string       `gorm:"column:ipfs_cid" json:"ipfsCid"`
	Checksum     string       `json:"checksum"`
	UploadStatus UploadStatus `gorm:"type:upload_status;default:'pending'" json:"uploadStatus"`
	FileSize     int64        `json:"fileSize"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
	Transcodes   []Transcode  `gorm:"foreignKey:VideoID" json:"transcodes"`
}

// BeforeCreate hook to validate UploadStatus before saving
func (v *Video) BeforeCreate(tx *gorm.DB) error {
	if v.UploadStatus == "" {
		v.UploadStatus = UploadStatusPending
	}
	return nil
}

// IsValidUploadStatus checks if a status is valid
func IsValidUploadStatus(status UploadStatus) bool {
	switch status {
	case UploadStatusPending, UploadStatusUploading, UploadStatusCompleted, UploadStatusFailed:
		return true
	}
	return false
}

// GetUploadStatusMessage returns a user-friendly message for the upload status
func (v *Video) GetUploadStatusMessage() string {
	switch v.UploadStatus {
	case UploadStatusPending:
		return "Upload pending"
	case UploadStatusUploading:
		return "Upload in progress"
	case UploadStatusCompleted:
		return "Upload completed successfully"
	case UploadStatusFailed:
		return "Upload failed"
	default:
		return "Unknown status"
	}
}

// VideoStatus represents possible video states
const (
	VideoStatusPending    = "pending"
	VideoStatusUploading  = "uploading"
	VideoStatusProcessing = "processing"
	VideoStatusCompleted  = "completed"
	VideoStatusFailed     = "failed"
)

// Transcode model definition.
type Transcode struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	VideoID     uint      `json:"videoId"`
	FilePath    string    `json:"filePath"`
	FileCID     string    `gorm:"column:file_cid" json:"fileCid"`
	Format      string    `json:"format"`      // "hls" or "mp4"
	Resolution  string    `json:"resolution"`  // e.g., "720", "480", "360"
	StorageType string    `json:"storageType"` // "ipfs" or "s3"
	Type        string    `json:"type"`        // "manifest" or "video"
	CreatedAt   time.Time `json:"createdAt"`
}

// TranscodeSegment model for transcoded HLS segments.
type TranscodeSegment struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TranscodeID uint      `json:"transcodeId"`
	FilePath    string    `json:"filePath"`
	FileCID     string    `gorm:"column:file_cid" json:"fileCid"`
	StorageType string    `json:"storageType"` // "ipfs" or "s3"
	Sequence    int       `json:"sequence"`
	Duration    float64   `json:"duration"`
	CreatedAt   time.Time `json:"createdAt"`
}

// TranscodeTarget defines parameters for each transcoded output.
type TranscodeTarget struct {
	Label       string // e.g., "720p"
	Resolution  string // target height (e.g., "720", "480", "360")
	Format      string // "hls" or "mp4"
	StorageType string // "ipfs" or "s3"
}
