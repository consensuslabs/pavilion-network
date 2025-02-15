package video

import (
	"io"

	"github.com/gin-gonic/gin"
)

// Logger interface for logging operations
type Logger interface {
	LogInfo(msg string, fields map[string]interface{})
	LogError(err error, msg string) error
}

// VideoService interface for video operations
type VideoService interface {
	ProcessVideo(file interface{}, header interface{}, title, description string) (*Video, error)
	GetVideoList() ([]Video, error)
	GetVideoStatus(fileID string) (*Video, error)
}

// IPFSService interface for IPFS operations
type IPFSService interface {
	GetGatewayURL(cid string) string
	UploadFileStream(reader io.Reader) (string, error)
}

// S3Service interface for S3 operations
type S3Service interface {
	UploadFileStream(reader io.Reader, key string) (string, error)
}

// ResponseHandler interface for HTTP responses
type ResponseHandler interface {
	SuccessResponse(c *gin.Context, data interface{}, message string)
	ErrorResponse(c *gin.Context, status int, code, message string, err error)
}
