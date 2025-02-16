package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/consensuslabs/pavilion-network/backend/internal/config"
	"github.com/minio/minio-go/v7"
	miniocreds "github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Service handles S3 operations
type S3Service struct {
	client *minio.Client
	bucket string
}

// NewS3Service creates a new S3 service instance
func NewS3Service(cfg *config.Config) (*S3Service, error) {
	client, err := minio.New(cfg.Storage.S3.Endpoint, &minio.Options{
		Creds:  miniocreds.NewStaticV4(cfg.Storage.S3.AccessKeyID, cfg.Storage.S3.SecretAccessKey, ""),
		Secure: cfg.Storage.S3.UseSSL,
		Region: cfg.Storage.S3.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %v", err)
	}

	return &S3Service{
		client: client,
		bucket: cfg.Storage.S3.Bucket,
	}, nil
}

// UploadFile uploads a file to S3
func (s *S3Service) UploadFile(filePath, fileKey string) (string, error) {
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
func (s *S3Service) UploadFileStream(file io.Reader, fileKey string) (string, error) {
	// Upload the file to S3
	ctx := context.Background()
	result, err := s.client.PutObject(ctx, s.bucket, fileKey, file, -1, minio.PutObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %v", err)
	}

	return result.Location, nil
}
