package main

import (
	"fmt"
	"io"
	"os"

	"github.com/consensuslabs/pavilion-network/backend/internal/config"
	"github.com/google/uuid"
	shell "github.com/ipfs/go-ipfs-api"
)

// IPFSService handles IPFS operations
type IPFSService struct {
	shell      *shell.Shell
	gatewayURL string
}

// NewIPFSService creates a new IPFS service instance
func NewIPFSService(cfg *config.Config) *IPFSService {
	return &IPFSService{
		shell:      shell.NewShell(cfg.Storage.IPFS.APIAddress),
		gatewayURL: cfg.Storage.IPFS.Gateway,
	}
}

// UploadFile uploads a file to IPFS and returns its CID
func (s *IPFSService) UploadFile(filePath string) (string, error) {
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

// UploadFileStream uploads a file stream to IPFS and returns its CID.
func (s *IPFSService) UploadFileStream(file io.Reader) (string, error) {
	cid, err := s.shell.Add(file)
	if err != nil {
		return "", fmt.Errorf("failed to upload to IPFS: %v", err)
	}
	return cid, nil
}

// DownloadFile downloads a file from IPFS using its CID
func (s *IPFSService) DownloadFile(cid string) (string, error) {
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
func (s *IPFSService) GetGatewayURL(cid string) string {
	return s.gatewayURL + cid
}

// Close closes any open IPFS connections and resources
func (s *IPFSService) Close() error {
	// Currently, the IPFS service doesn't maintain any long-lived connections
	// This is a placeholder for future connection cleanup if needed
	return nil
}
