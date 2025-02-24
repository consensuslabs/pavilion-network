package storage

import (
	"context"
	"io"

	videostorage "github.com/consensuslabs/pavilion-network/backend/internal/storage/video"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// IPFSService defines IPFS-specific operations
type IPFSService interface {
	videostorage.Service
	GetGatewayURL(cid string) string
	DownloadFile(cid string) (string, error)
}

// S3Service defines S3-specific operations
type S3Service interface {
	videostorage.Service
	// UploadVideo uploads a video file to S3 with the standardized path structure
	UploadVideo(ctx context.Context, videoID uuid.UUID, resolution string, reader io.Reader) (string, error)
	// GetVideoURL returns the URL for a video in S3
	GetVideoURL(ctx context.Context, key string) (string, error)
	// DeleteVideo deletes a video and its transcoded versions from S3
	DeleteVideo(ctx context.Context, videoID uuid.UUID) error
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
