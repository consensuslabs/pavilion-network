package video

import (
	"io"

	"github.com/gin-gonic/gin"
)

// VideoService interface for video operations
type VideoService interface {
	InitializeUpload(title, description string, fileSize int64) (*VideoUpload, error)
	ProcessUpload(upload *VideoUpload, file interface{}, header interface{}) error
	GetVideoList() ([]VideoUpload, error)
	GetVideoStatus(fileID string) (*VideoUpload, error)
	ProcessTranscode(cid string) (*TranscodeResult, error)
}

// IPFSService interface for IPFS operations
type IPFSService interface {
	GetGatewayURL(cid string) string
	UploadFileStream(reader io.Reader) (string, error)
	UploadFile(filePath string) (string, error)
	DownloadFile(cid string) (string, error)
}

// S3Service interface for S3 operations
type S3Service interface {
	UploadFileStream(reader io.Reader, key string) (string, error)
	UploadFile(filePath string, key string) (string, error)
}

// ResponseHandler interface for HTTP responses
type ResponseHandler interface {
	SuccessResponse(c *gin.Context, data interface{}, message string)
	ErrorResponse(c *gin.Context, status int, code, message string, err error)
}

// Logger interface for logging operations
type Logger interface {
	LogInfo(msg string, fields map[string]interface{})
	LogError(err error, msg string) error
}
