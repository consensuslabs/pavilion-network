package storage

import (
	"context"
	"io"

	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/google/uuid"
)

// VideoIPFSAdapter adapts our storage IPFS service to video package's interface
type VideoIPFSAdapter struct {
	service IPFSService
}

// NewVideoIPFSAdapter creates a new adapter for video IPFS operations
func NewVideoIPFSAdapter(service IPFSService) video.IPFSService {
	return &VideoIPFSAdapter{service: service}
}

// UploadFileStream uploads a file stream to IPFS
func (a *VideoIPFSAdapter) UploadFileStream(reader io.Reader) (string, error) {
	// For IPFS uploads, we don't need the video ID or resolution
	// We'll just use placeholder values since we're not pinning in MVP
	return a.service.UploadVideo(context.Background(), uuid.Nil, "original", reader)
}

// DownloadFile downloads a file from IPFS
func (a *VideoIPFSAdapter) DownloadFile(cid string) (string, error) {
	return a.service.DownloadFile(cid)
}
