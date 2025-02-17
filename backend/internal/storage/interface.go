package storage

import (
	"io"

	"github.com/gin-gonic/gin"
)

// StorageService defines the interface for storage operations
type StorageService interface {
	UploadFile(filePath, key string) (string, error)
	UploadFileStream(reader io.Reader, key string) (string, error)
	Close() error
}

// IPFSService defines IPFS-specific operations
type IPFSService interface {
	StorageService
	GetGatewayURL(cid string) string
	DownloadFile(cid string) (string, error)
}

// S3Service defines S3-specific operations
type S3Service interface {
	StorageService
}

// Logger interface for logging operations
type Logger interface {
	LogInfo(msg string, fields map[string]interface{})
	LogError(err error, msg string) error
}

// ResponseHandler interface for HTTP responses
type ResponseHandler interface {
	SuccessResponse(c *gin.Context, data interface{}, message string)
	ErrorResponse(c *gin.Context, status int, code, message string, err error)
}
