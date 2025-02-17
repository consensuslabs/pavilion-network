package ipfs

import (
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

// UploadFile uploads a file to IPFS and returns its CID
func (s *Service) UploadFile(filePath, _ string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	cid, err := s.shell.Add(file)
	if err != nil {
		return "", fmt.Errorf("failed to upload to IPFS: %v", err)
	}
	return cid, nil
}

// UploadFileStream uploads a file stream to IPFS and returns its CID
func (s *Service) UploadFileStream(file io.Reader, _ string) (string, error) {
	cid, err := s.shell.Add(file)
	if err != nil {
		return "", fmt.Errorf("failed to upload to IPFS: %v", err)
	}
	return cid, nil
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
