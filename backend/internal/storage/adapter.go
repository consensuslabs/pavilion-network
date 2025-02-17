package storage

import (
	"io"

	"github.com/consensuslabs/pavilion-network/backend/internal/video"
)

// VideoIPFSAdapter adapts our storage IPFS service to video package's interface
type VideoIPFSAdapter struct {
	service IPFSService
}

// NewVideoIPFSAdapter creates a new adapter for video IPFS operations
func NewVideoIPFSAdapter(service IPFSService) video.IPFSService {
	return &VideoIPFSAdapter{service: service}
}

// GetGatewayURL returns the IPFS gateway URL for a given CID
func (a *VideoIPFSAdapter) GetGatewayURL(cid string) string {
	return a.service.GetGatewayURL(cid)
}

// UploadFileStream uploads a file stream to IPFS
func (a *VideoIPFSAdapter) UploadFileStream(reader io.Reader) (string, error) {
	// We don't need the key parameter for IPFS uploads
	return a.service.UploadFileStream(reader, "")
}

// UploadFile uploads a file to IPFS
func (a *VideoIPFSAdapter) UploadFile(filePath string) (string, error) {
	// We don't need the key parameter for IPFS uploads
	return a.service.UploadFile(filePath, "")
}

// DownloadFile downloads a file from IPFS
func (a *VideoIPFSAdapter) DownloadFile(cid string) (string, error) {
	return a.service.DownloadFile(cid)
}
