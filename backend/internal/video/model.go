package video

import (
	"time"
)

// Video represents a video entity
type Video struct {
	ID           uint      `json:"id"`
	FileId       string    `json:"fileId"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	FilePath     string    `json:"filePath"`
	IPFSCID      string    `gorm:"column:ipfs_cid" json:"ipfsCid"`
	UploadStatus string    `json:"status"`
	FileSize     int64     `json:"fileSize"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// GetUploadStatusMessage returns a human-readable status message
func (v *Video) GetUploadStatusMessage() string {
	switch v.UploadStatus {
	case "pending":
		return "Upload pending"
	case "uploading":
		return "Upload in progress"
	case "completed":
		return "Upload completed successfully"
	case "failed":
		return "Upload failed"
	default:
		return "Unknown status"
	}
}
