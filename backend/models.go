package main

import (
	"time"
)

// User model definition.
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	Email     string    `gorm:"unique" json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

// Video model definition.
// fileId stores the unique identifier generated during upload.
// Transcodes represents the one-to-many relation to the Transcode table.
type Video struct {
	ID          uint        `gorm:"primaryKey" json:"id"`
	FileId      string      `json:"fileId"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	FilePath    string      `json:"filePath"`
	IPFSCID     string      `gorm:"column:ipfs_cid" json:"ipfsCid"`
	Checksum    string      `json:"checksum"`
	Status      string      `json:"status"` // pending, processing, completed, failed
	StatusMsg   string      `json:"statusMsg"`
	FileSize    int64       `json:"fileSize"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	Transcodes  []Transcode `json:"transcodes"`
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
	ID         uint      `gorm:"primaryKey" json:"id"`
	VideoID    uint      `json:"videoId"`
	FilePath   string    `json:"filePath"`
	FileCID    string    `gorm:"column:file_cid" json:"fileCid"`
	Type       string    `json:"type"`       // "hlsManifest", "hlsSegment", or "h264"
	Resolution string    `json:"resolution"` // e.g., "720", "480", "360", "240"
	CreatedAt  time.Time `json:"createdAt"`
}

// TranscodeSegment model for transcoded HLS segments.
type TranscodeSegment struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TranscodeID uint      `json:"transcodeId"`
	FilePath    string    `json:"filePath"`
	FileCID     string    `gorm:"column:file_cid" json:"fileCid"`
	Sequence    int       `json:"sequence"`
	CreatedAt   time.Time `json:"createdAt"`
}

// TranscodeTarget defines parameters for each transcoded output.
type TranscodeTarget struct {
	Label      string // e.g., "720pMp4", "480pMp4", "360pMp4", "240pMp4"
	Resolution string // target height (e.g., "720", "480", "360", "240")
	OutputExt  string // "m3u8" for HLS outputs, "mp4" for progressive MP4
}
