package videostorage

import (
	"context"
	"io"

	"github.com/google/uuid"
)

// Service defines the interface for video storage operations
type Service interface {
	// UploadVideo uploads a video file with the standardized path structure
	UploadVideo(ctx context.Context, videoID uuid.UUID, resolution string, reader io.Reader) (string, error)
	// GetVideoURL returns the URL for a video
	GetVideoURL(ctx context.Context, key string) (string, error)
	// DeleteVideo deletes a video and its transcoded versions
	DeleteVideo(ctx context.Context, videoID uuid.UUID) error
	// Close closes any open connections
	Close() error
}

// Config represents the configuration for video storage
type Config struct {
	// S3-specific configuration
	Endpoint        string `mapstructure:"endpoint" yaml:"endpoint"`
	AccessKeyID     string `mapstructure:"accessKeyId" yaml:"access_key_id"`
	SecretAccessKey string `mapstructure:"secretAccessKey" yaml:"secret_access_key"`
	UseSSL          bool   `mapstructure:"useSSL" yaml:"use_ssl"`
	Region          string `mapstructure:"region" yaml:"region"`
	Bucket          string `mapstructure:"bucket" yaml:"bucket"`
}

// ValidateResolution checks if the resolution is valid
func ValidateResolution(resolution string) bool {
	validResolutions := map[string]bool{
		"original": true,
		"720p":     true,
		"480p":     true,
		"360p":     true,
	}
	return validResolutions[resolution]
}
