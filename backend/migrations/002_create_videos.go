package migrations

import (
	"time"

	"gorm.io/gorm"
)

type Video struct {
	ID           int64     `gorm:"primaryKey;autoIncrement:false;type:bigint;default:unique_rowid()" json:"id"`
	FileId       string    `gorm:"unique" json:"fileId"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	FilePath     string    `json:"filePath"`
	IPFSCID      string    `gorm:"column:ipfs_cid" json:"ipfsCid"`
	Checksum     string    `json:"checksum"`
	UploadStatus string    `gorm:"type:string;default:'pending'" json:"uploadStatus"`
	FileSize     int64     `json:"fileSize"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type VideoMigration struct {
	db *gorm.DB
}

func NewVideoMigration(db *gorm.DB) *VideoMigration {
	return &VideoMigration{db: db}
}

func (m *VideoMigration) Up() error {
	// Create the videos table
	if err := m.db.AutoMigrate(&Video{}); err != nil {
		return err
	}

	// Create the transcodes table
	if err := m.db.AutoMigrate(&Transcode{}); err != nil {
		return err
	}

	// Create the transcode_segments table
	if err := m.db.AutoMigrate(&TranscodeSegment{}); err != nil {
		return err
	}

	return nil
}

func (m *VideoMigration) Down() error {
	// Drop tables in reverse order to handle foreign key constraints
	if err := m.db.Migrator().DropTable(&TranscodeSegment{}); err != nil {
		return err
	}
	if err := m.db.Migrator().DropTable(&Transcode{}); err != nil {
		return err
	}
	return m.db.Migrator().DropTable(&Video{})
}

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
