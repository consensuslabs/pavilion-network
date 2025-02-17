package s3

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/consensuslabs/pavilion-network/backend/internal/storage"
	"github.com/minio/minio-go/v7"
	miniocreds "github.com/minio/minio-go/v7/pkg/credentials"
)

// Service implements the S3Service interface
type Service struct {
	client *minio.Client
	bucket string
	logger storage.Logger
}

// NewService creates a new S3 service instance
func NewService(cfg *storage.S3Config, logger storage.Logger) (*Service, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  miniocreds.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %v", err)
	}

	return &Service{
		client: client,
		bucket: cfg.Bucket,
		logger: logger,
	}, nil
}

// UploadFile uploads a file to S3
func (s *Service) UploadFile(filePath, fileKey string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %v", err)
	}

	// Upload the file to S3
	ctx := context.Background()
	result, err := s.client.PutObject(ctx, s.bucket, fileKey, file, fileInfo.Size(), minio.PutObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %v", err)
	}

	return result.Location, nil
}

// UploadFileStream uploads a file stream to S3
func (s *Service) UploadFileStream(file io.Reader, fileKey string) (string, error) {
	// Upload the file to S3
	ctx := context.Background()
	result, err := s.client.PutObject(ctx, s.bucket, fileKey, file, -1, minio.PutObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %v", err)
	}

	return result.Location, nil
}

// Close closes any open S3 connections and resources
func (s *Service) Close() error {
	// Currently, the S3 service doesn't maintain any long-lived connections
	// This is a placeholder for future connection cleanup if needed
	return nil
}
