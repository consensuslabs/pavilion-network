package video

import (
	"io"
	"mime/multipart"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// VideoService defines the interface for video operations
type VideoService interface {
	InitializeUpload(title, description string, size int64) (*VideoUpload, error)
	ProcessUpload(upload *VideoUpload, file multipart.File, header *multipart.FileHeader) error
	GetVideo(videoID uuid.UUID) (*Video, error)
	ListVideos(page, limit int) ([]Video, error)
	// DeleteVideo performs a soft delete of a video by setting its DeletedAt field
	DeleteVideo(videoID uuid.UUID) error
	UpdateVideo(videoID uuid.UUID, title, description string) error
}

// IPFSService defines the interface for IPFS operations
type IPFSService interface {
	UploadFileStream(file io.Reader) (string, error)
	DownloadFile(cid string) (string, error)
}

// ResponseHandler defines the interface for HTTP response handling
type ResponseHandler interface {
	SuccessResponse(c *gin.Context, data interface{}, message string)
	ErrorResponse(c *gin.Context, statusCode int, code string, message string, err error)
}

// Logger defines the interface for logging operations
type Logger interface {
	LogInfo(message string, fields map[string]interface{})
	LogError(message string, fields map[string]interface{})
}
