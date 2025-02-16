package migrations

import (
	"time"

	"gorm.io/gorm"
)

type VideoUpload struct {
	ID                uint   `gorm:"primaryKey"`
	TempFileId        string `gorm:"index;not null"`
	Title             string `gorm:"not null"`
	Description       string
	FileSize          int64  `gorm:"not null"`
	IPFSBytesUploaded int64  `gorm:"column:ipfs_bytes_uploaded;default:0"`
	S3BytesUploaded   int64  `gorm:"column:s3_bytes_uploaded;default:0"`
	UploadStatus      string `gorm:"not null;type:string"`
	CurrentPhase      string `gorm:"not null"`
	IPFSCID           string `gorm:"column:ipfs_cid"`
	S3URL             string
	ErrorMessage      string
	IPFSStartTime     *time.Time
	IPFSEndTime       *time.Time
	S3StartTime       *time.Time
	S3EndTime         *time.Time
	CreatedAt         time.Time `gorm:"not null"`
	UpdatedAt         time.Time `gorm:"not null"`
}

type Migration struct {
	db *gorm.DB
}

func NewMigration(db *gorm.DB) *Migration {
	return &Migration{db: db}
}

func (m *Migration) Up() error {
	return m.db.AutoMigrate(&VideoUpload{})
}

func (m *Migration) Down() error {
	return m.db.Migrator().DropTable(&VideoUpload{})
}
