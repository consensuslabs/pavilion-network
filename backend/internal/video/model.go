package video

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Video represents a video entity in the database
type Video struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	FileID      string         `gorm:"unique;not null" json:"file_id"`
	Title       string         `gorm:"not null" json:"title"`
	Description string         `json:"description"`
	StoragePath string         `gorm:"not null" json:"storage_path"`
	IPFSCID     string         `gorm:"column:ipfs_cid" json:"ipfs_cid"`
	Checksum    string         `gorm:"size:64" json:"checksum"`
	FileSize    int64          `gorm:"not null" json:"file_size"`
	CreatedAt   time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Upload      *VideoUpload   `gorm:"foreignKey:VideoID" json:"upload,omitempty"`
	Transcodes  []Transcode    `gorm:"foreignKey:VideoID" json:"transcodes,omitempty"`
}

// VideoUpload represents the upload process tracking
type VideoUpload struct {
	ID        uuid.UUID    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	VideoID   uuid.UUID    `gorm:"type:uuid;not null;unique" json:"video_id"`
	StartTime time.Time    `gorm:"not null" json:"start_time"`
	EndTime   *time.Time   `json:"end_time,omitempty"`
	Status    UploadStatus `gorm:"type:upload_status;not null" json:"status"`
	CreatedAt time.Time    `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time    `gorm:"not null;default:now()" json:"updated_at"`
	Video     *Video       `gorm:"foreignKey:VideoID" json:"-"`
}

// Transcode represents a transcoded version of a video
type Transcode struct {
	ID        uuid.UUID          `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	VideoID   uuid.UUID          `gorm:"type:uuid;not null" json:"video_id"`
	Format    string             `gorm:"type:text;not null;check:format IN ('mp4', 'hls')" json:"format"`
	CreatedAt time.Time          `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time          `gorm:"not null;default:now()" json:"updated_at"`
	Video     *Video             `gorm:"foreignKey:VideoID" json:"-"`
	Segments  []TranscodeSegment `gorm:"foreignKey:TranscodeID" json:"segments,omitempty"`
}

// TranscodeSegment represents a segment of a transcoded video
type TranscodeSegment struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TranscodeID uuid.UUID  `gorm:"type:uuid;not null" json:"transcode_id"`
	StoragePath string     `gorm:"not null" json:"storage_path"`
	IPFSCID     string     `gorm:"column:ipfs_cid" json:"ipfs_cid"`
	Duration    int        `json:"duration"`
	CreatedAt   time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null;default:now()" json:"updated_at"`
	Transcode   *Transcode `gorm:"foreignKey:TranscodeID" json:"-"`
}

// BeforeCreate hook for Video
func (v *Video) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

// ToVideoInfo converts Video to VideoInfo response type
func (v *Video) ToVideoInfo() VideoInfo {
	var status string
	if v.Upload != nil {
		status = string(v.Upload.Status)
	}
	return VideoInfo{
		ID:          v.ID.String(),
		Title:       v.Title,
		Description: v.Description,
		Status:      status,
		CreatedAt:   v.CreatedAt,
	}
}

// ToVideoDetailsResponse converts Video to VideoDetailsResponse
func (v *Video) ToVideoDetailsResponse() VideoDetailsResponse {
	transcodes := make([]TranscodeInfo, 0, len(v.Transcodes))
	for _, t := range v.Transcodes {
		segments := make([]TranscodeSegmentInfo, 0, len(t.Segments))
		for _, s := range t.Segments {
			segments = append(segments, TranscodeSegmentInfo{
				ID:          s.ID.String(),
				StoragePath: s.StoragePath,
				IPFSCID:     s.IPFSCID,
				Duration:    s.Duration,
			})
		}
		transcodes = append(transcodes, TranscodeInfo{
			ID:        t.ID.String(),
			Format:    t.Format,
			Segments:  segments,
			CreatedAt: t.CreatedAt,
		})
	}

	var status string
	if v.Upload != nil {
		status = string(v.Upload.Status)
	}

	return VideoDetailsResponse{
		ID:          v.ID.String(),
		FileID:      v.FileID,
		Title:       v.Title,
		Description: v.Description,
		StoragePath: v.StoragePath,
		IPFSCID:     v.IPFSCID,
		Status:      status,
		FileSize:    v.FileSize,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
		Transcodes:  transcodes,
	}
}

// ToAPIResponse converts Video to a generic APIResponse
func (v *Video) ToAPIResponse(message string) APIResponse {
	return APIResponse{
		Message: message,
		Status:  "success",
		Data:    v.ToVideoInfo(),
	}
}
