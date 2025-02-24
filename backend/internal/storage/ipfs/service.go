package ipfs

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/consensuslabs/pavilion-network/backend/internal/storage"
	"github.com/google/uuid"
	shell "github.com/ipfs/go-ipfs-api"
)

// Service implements the IPFSService interface
type Service struct {
	shell      *shell.Shell
	gatewayURL string
	logger     storage.Logger
}

// NewService creates a new IPFS service instance
func NewService(cfg *storage.IPFSConfig, logger storage.Logger) *Service {
	return &Service{
		shell:      shell.NewShell(cfg.APIAddress),
		gatewayURL: cfg.Gateway,
		logger:     logger,
	}
}

// UploadVideo uploads a video file to IPFS and returns its CID
func (s *Service) UploadVideo(_ context.Context, _ uuid.UUID, _ string, reader io.Reader) (string, error) {
	cid, err := s.shell.Add(reader)
	if err != nil {
		return "", fmt.Errorf("failed to upload to IPFS: %v", err)
	}
	return cid, nil
}

// GetVideoURL returns the IPFS gateway URL for a given CID
func (s *Service) GetVideoURL(_ context.Context, cid string) (string, error) {
	return s.gatewayURL + cid, nil
}

// DeleteVideo is a no-op for IPFS as we don't pin files in MVP
func (s *Service) DeleteVideo(_ context.Context, _ uuid.UUID) error {
	// No-op for MVP as we don't pin files
	return nil
}

// DownloadFile downloads a file from IPFS using its CID
func (s *Service) DownloadFile(cid string) (string, error) {
	r, err := s.shell.Cat(cid)
	if err != nil {
		return "", err
	}
	defer r.Close()

	tempFile := "temp_" + uuid.New().String() + ".mp4"
	outFile, err := os.Create(tempFile)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, r)
	if err != nil {
		return "", err
	}

	return tempFile, nil
}

// GetGatewayURL returns the IPFS gateway URL for a given CID
func (s *Service) GetGatewayURL(cid string) string {
	return s.gatewayURL + cid
}

// Close closes any open IPFS connections and resources
func (s *Service) Close() error {
	// Currently, the IPFS service doesn't maintain any long-lived connections
	// This is a placeholder for future connection cleanup if needed
	return nil
}
