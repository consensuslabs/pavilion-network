package main

import (
	"io"
	"os"

	"github.com/google/uuid"
	shell "github.com/ipfs/go-ipfs-api"
)

// IPFSService handles all IPFS-related operations
type IPFSService struct {
	shell *shell.Shell
	host  string
}

// NewIPFSService creates a new IPFS service instance
func NewIPFSService(config *Config) *IPFSService {
	return &IPFSService{
		shell: shell.NewShell(config.IPFS.Host),
		host:  config.IPFS.GatewayURL,
	}
}

// UploadFile uploads a file to IPFS and returns its CID
func (s *IPFSService) UploadFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	cid, err := s.shell.Add(file)
	if err != nil {
		return "", err
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
	return s.host + cid
}
