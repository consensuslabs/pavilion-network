package main

import (
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// S3Service handles interactions with AWS S3
type S3Service struct {
	uploader *s3manager.Uploader
	config   *Config
}

// NewS3Service creates a new S3 service instance
func NewS3Service(config *Config) (*S3Service, error) {
	// Debug log S3 configuration
	fmt.Printf("S3 Configuration:\n")
	fmt.Printf("Region: %s\n", config.Storage.S3.Region)
	fmt.Printf("Bucket: %s\n", config.Storage.S3.Bucket)
	fmt.Printf("AccessKeyId: %s\n", config.Storage.S3.AccessKeyId)
	fmt.Printf("SecretAccessKey length: %d\n", len(config.Storage.S3.SecretAccessKey))
	fmt.Printf("VideoPost Directory: %s\n", config.Storage.S3.Directories.VideoPost)

	// Get AWS credentials from environment variables
	creds := credentials.NewStaticCredentials(
		config.Storage.S3.AccessKeyId,
		config.Storage.S3.SecretAccessKey,
		"", // No session token needed for basic credentials
	)

	// Verify credentials are not empty
	if _, err := creds.Get(); err != nil {
		return nil, fmt.Errorf("invalid AWS credentials: %v", err)
	}

	// Create a new AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(config.Storage.S3.Region),
		Credentials: creds,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %v", err)
	}

	// Create an S3 uploader
	uploader := s3manager.NewUploader(sess)

	return &S3Service{
		uploader: uploader,
		config:   config,
	}, nil
}

// UploadFile uploads a file to S3
func (s *S3Service) UploadFile(filePath, fileKey string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Upload the file to S3
	result, err := s.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.config.Storage.S3.Bucket),
		Key:    aws.String(fileKey),
		Body:   file,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %v", err)
	}

	return result.Location, nil
}

// UploadFileStream uploads a file stream to S3.
func (s *S3Service) UploadFileStream(file io.Reader, fileKey string) (string, error) {
	// Upload the file to S3
	result, err := s.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.config.Storage.S3.Bucket),
		Key:    aws.String(s.config.Storage.S3.Directories.VideoPost + fileKey),
		Body:   file,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %v", err)
	}

	return result.Location, nil
}
