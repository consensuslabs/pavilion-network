package video

import (
	"time"

	"gorm.io/gorm"
)

// Video represents a completed video
type Video struct {
	ID           int64        `gorm:"primaryKey;autoIncrement:false;type:bigint;default:unique_rowid()" json:"id"`
	FileId       string       `gorm:"unique" json:"fileId"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	FilePath     string       `json:"filePath"`
	IPFSCID      string       `gorm:"column:ipfs_cid" json:"ipfsCid"`
	Checksum     string       `json:"checksum"`
	UploadStatus UploadStatus `gorm:"type:string;default:'pending'" json:"uploadStatus"`
	FileSize     int64        `json:"fileSize"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
	Transcodes   []Transcode  `gorm:"foreignKey:VideoID" json:"transcodes"`
}

// VideoUpload represents a video upload in progress
type VideoUpload struct {
	ID                uint         `json:"id" gorm:"primaryKey"`
	TempFileId        string       `json:"tempFileId" gorm:"index;not null"`
	Title             string       `json:"title" gorm:"not null"`
	Description       string       `json:"description"`
	FileSize          int64        `json:"fileSize" gorm:"not null"`
	IPFSBytesUploaded int64        `json:"ipfsBytesUploaded" gorm:"column:ipfs_bytes_uploaded;default:0"`
	S3BytesUploaded   int64        `json:"s3BytesUploaded" gorm:"column:s3_bytes_uploaded;default:0"`
	UploadStatus      UploadStatus `json:"status" gorm:"not null;type:string"`
	CurrentPhase      string       `json:"phase" gorm:"not null"`
	IPFSCID           string       `json:"ipfsCid" gorm:"column:ipfs_cid"`
	S3URL             string       `json:"s3Url"`
	ErrorMessage      string       `json:"error,omitempty"`
	IPFSStartTime     *time.Time   `json:"ipfsStartTime"`
	IPFSEndTime       *time.Time   `json:"ipfsEndTime"`
	S3StartTime       *time.Time   `json:"s3StartTime"`
	S3EndTime         *time.Time   `json:"s3EndTime"`
	CreatedAt         time.Time    `json:"createdAt" gorm:"not null"`
	UpdatedAt         time.Time    `json:"updatedAt" gorm:"not null"`
}

// Transcode represents a transcoded version of a video
type Transcode struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false;type:bigint;default:unique_rowid()" json:"id"`
	VideoID     int64     `json:"videoId"`
	FilePath    string    `json:"filePath"`
	FileCID     string    `gorm:"column:file_cid" json:"fileCid"`
	Format      string    `json:"format"`      // "hls" or "mp4"
	Resolution  string    `json:"resolution"`  // e.g., "720", "480", "360"
	StorageType string    `json:"storageType"` // "ipfs" or "s3"
	Type        string    `json:"type"`        // "manifest" or "video"
	CreatedAt   time.Time `json:"createdAt"`
}

// TranscodeSegment represents a segment of a transcoded HLS video
type TranscodeSegment struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false;type:bigint;default:unique_rowid()" json:"id"`
	TranscodeID int64     `json:"transcodeId"`
	FilePath    string    `json:"filePath"`
	FileCID     string    `gorm:"column:file_cid" json:"fileCid"`
	StorageType string    `json:"storageType"` // "ipfs" or "s3"
	Sequence    int       `json:"sequence"`
	Duration    float64   `json:"duration"`
	CreatedAt   time.Time `json:"createdAt"`
}

// BeforeCreate hook to validate UploadStatus before saving
func (v *Video) BeforeCreate(tx *gorm.DB) error {
	if v.UploadStatus == "" {
		v.UploadStatus = StatusPending
	}
	return nil
}

// GetUploadProgress calculates the total upload progress as a percentage
func (v *VideoUpload) GetUploadProgress() float64 {
	if v.UploadStatus == StatusCompleted {
		return 100.0
	}

	if v.FileSize == 0 {
		return 0.0
	}

	var progress float64
	if v.CurrentPhase == "IPFS" {
		// IPFS progress represents first 50%
		progress = float64(v.IPFSBytesUploaded) / float64(v.FileSize) * 50.0
	} else if v.CurrentPhase == "S3" {
		// S3 progress represents remaining 50%
		progress = 50.0 + (float64(v.S3BytesUploaded) / float64(v.FileSize) * 50.0)
	}

	return progress
}

// GetIPFSDuration returns the duration of IPFS upload
func (v *VideoUpload) GetIPFSDuration() time.Duration {
	if v.IPFSStartTime != nil && v.IPFSEndTime != nil {
		return v.IPFSEndTime.Sub(*v.IPFSStartTime)
	}
	return 0
}

// GetS3Duration returns the duration of S3 upload
func (v *VideoUpload) GetS3Duration() time.Duration {
	if v.S3StartTime != nil && v.S3EndTime != nil {
		return v.S3EndTime.Sub(*v.S3StartTime)
	}
	return 0
}
