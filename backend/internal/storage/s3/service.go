package s3

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	videostorage "github.com/consensuslabs/pavilion-network/backend/internal/storage/video"
	"github.com/google/uuid"
)

type S3Service struct {
	client *s3.Client
	config *videostorage.Config
	logger logger.Logger
}

func NewService(cfg *videostorage.Config, logger logger.Logger) (*S3Service, error) {
	// Debug log for S3 configuration
	logger.LogInfo("S3 Service Configuration", map[string]interface{}{
		"endpoint":        cfg.Endpoint,
		"region":          cfg.Region,
		"bucket":          cfg.Bucket,
		"useSSL":          cfg.UseSSL,
		"accessKeyID":     cfg.AccessKeyID,
		"secretAccessKey": cfg.SecretAccessKey != "", // Don't log the actual secret
		"accessKeyLength": len(cfg.AccessKeyID),
		"secretKeyLength": len(cfg.SecretAccessKey),
	})

	// Create AWS credentials
	creds := credentials.NewStaticCredentialsProvider(
		cfg.AccessKeyID,
		cfg.SecretAccessKey,
		"",
	)

	// Load AWS configuration
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(creds),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &S3Service{
		client: client,
		config: cfg,
		logger: logger,
	}, nil
}

// UploadVideo uploads a video file to S3 with the standardized path structure
func (s *S3Service) UploadVideo(ctx context.Context, videoID uuid.UUID, resolution string, reader io.Reader) (string, error) {
	// Validate resolution
	if !videostorage.ValidateResolution(resolution) {
		return "", fmt.Errorf("invalid resolution: %s", resolution)
	}

	// Get the root directory, default to "videos" if not specified
	rootDir := "videos"
	if s.config.RootDirectory != "" {
		rootDir = s.config.RootDirectory
	}

	// Construct the standardized path: {root_dir}/{video_id}/[original|720p|480p|360p].mp4
	key := fmt.Sprintf("%s/%s/%s.mp4", rootDir, videoID, resolution)

	// Upload the file
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
		Body:   reader,
	})
	if err != nil {
		s.logger.LogError(err, fmt.Sprintf("Failed to upload video to S3: video_id=%s, resolution=%s, bucket=%s, key=%s",
			videoID, resolution, s.config.Bucket, key))
		return "", fmt.Errorf("failed to upload video to S3: %w", err)
	}

	s.logger.LogInfo("Successfully uploaded video to S3", map[string]interface{}{
		"video_id":   videoID,
		"resolution": resolution,
		"bucket":     s.config.Bucket,
		"key":        key,
	})

	return key, nil
}

// GetVideoURL returns the URL for a video in S3
func (s *S3Service) GetVideoURL(ctx context.Context, key string) (string, error) {
	// Get the root directory, default to "videos" if not specified
	rootDir := "videos"
	if s.config.RootDirectory != "" {
		rootDir = s.config.RootDirectory
	}

	// Validate the key format - should start with the root directory
	if !strings.HasPrefix(key, rootDir+"/") {
		return "", fmt.Errorf("invalid video key format: %s", key)
	}

	// Create presigned URL
	presignClient := s3.NewPresignClient(s.client)
	presignedURL, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		s.logger.LogError(err, fmt.Sprintf("Failed to generate presigned URL: bucket=%s, key=%s",
			s.config.Bucket, key))
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL.URL, nil
}

// DeleteVideo deletes a video and its transcoded versions from S3
func (s *S3Service) DeleteVideo(ctx context.Context, videoID uuid.UUID) error {
	// Get the root directory, default to "videos" if not specified
	rootDir := "videos"
	if s.config.RootDirectory != "" {
		rootDir = s.config.RootDirectory
	}

	// List all objects with the video ID prefix
	prefix := fmt.Sprintf("%s/%s/", rootDir, videoID)

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.config.Bucket),
		Prefix: aws.String(prefix),
	})

	var deleteErr error
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			s.logger.LogError(err, fmt.Sprintf("Failed to list objects for deletion: video_id=%s, prefix=%s",
				videoID, prefix))
			return fmt.Errorf("failed to list objects for deletion: %w", err)
		}

		// Delete objects in this page
		for _, obj := range page.Contents {
			_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(s.config.Bucket),
				Key:    obj.Key,
			})
			if err != nil {
				s.logger.LogError(err, fmt.Sprintf("Failed to delete object: video_id=%s, key=%s",
					videoID, *obj.Key))
				deleteErr = err
			}
		}
	}

	if deleteErr != nil {
		return fmt.Errorf("failed to delete some video files: %w", deleteErr)
	}

	s.logger.LogInfo("Successfully deleted video files from S3", map[string]interface{}{
		"video_id": videoID,
		"prefix":   prefix,
	})

	return nil
}

// Close implements the storage.Service interface
func (s *S3Service) Close() error {
	// No need to close the S3 client
	return nil
}
