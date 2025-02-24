package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
)

// Service handles FFmpeg operations
type Service struct {
	config *Config
	logger logger.Logger
}

// Config represents FFmpeg configuration
type Config struct {
	Path        string   // Path to FFmpeg binary
	ProbePath   string   // Path to FFprobe binary
	VideoCodec  string   // Video codec to use (e.g., h264)
	AudioCodec  string   // Audio codec to use (e.g., aac)
	Preset      string   // Encoding preset (e.g., medium)
	OutputPath  string   // Path for transcoded outputs
	Resolutions []string // List of output resolutions
}

// VideoMetadata represents video file metadata
type VideoMetadata struct {
	Duration   float64 // Duration in seconds
	Width      int     // Width in pixels
	Height     int     // Height in pixels
	Format     string  // Container format
	VideoCodec string  // Video codec
	AudioCodec string  // Audio codec
	Bitrate    int64   // Bitrate in bits per second
}

// NewService creates a new FFmpeg service
func NewService(config *Config, logger logger.Logger) *Service {
	return &Service{
		config: config,
		logger: logger,
	}
}

// GetMetadata extracts metadata from a video file
func (s *Service) GetMetadata(ctx context.Context, filePath string) (*VideoMetadata, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Run ffprobe command
	cmd := exec.CommandContext(ctx, s.config.ProbePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		s.logger.LogError(err, fmt.Sprintf("Failed to get video metadata: path=%s", filePath))
		return nil, fmt.Errorf("failed to get video metadata: %w", err)
	}

	// Parse the JSON output and extract metadata
	// For MVP, we'll use a simplified approach with string parsing
	metadata := &VideoMetadata{}

	// Extract duration
	if durationStr := s.extractValue(string(output), "duration"); durationStr != "" {
		if duration, err := strconv.ParseFloat(durationStr, 64); err == nil {
			metadata.Duration = duration
		}
	}

	// Extract resolution
	if width := s.extractValue(string(output), "width"); width != "" {
		if w, err := strconv.Atoi(width); err == nil {
			metadata.Width = w
		}
	}
	if height := s.extractValue(string(output), "height"); height != "" {
		if h, err := strconv.Atoi(height); err == nil {
			metadata.Height = h
		}
	}

	// Extract format
	metadata.Format = s.extractValue(string(output), "format_name")
	metadata.VideoCodec = s.extractValue(string(output), "codec_name")
	metadata.AudioCodec = s.extractValue(string(output), "codec_name")

	// Extract bitrate
	if bitrate := s.extractValue(string(output), "bit_rate"); bitrate != "" {
		if br, err := strconv.ParseInt(bitrate, 10, 64); err == nil {
			metadata.Bitrate = br
		}
	}

	return metadata, nil
}

// Transcode transcodes a video file to the specified resolution
func (s *Service) Transcode(ctx context.Context, inputPath, outputPath, resolution string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		s.logger.LogError(err, fmt.Sprintf("Failed to create output directory: path=%s", outputPath))
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build FFmpeg command
	cmd := exec.CommandContext(ctx, s.config.Path,
		"-i", inputPath,
		"-c:v", s.config.VideoCodec,
		"-c:a", s.config.AudioCodec,
		"-s", resolution,
		"-preset", s.config.Preset,
		"-y", // Overwrite output file if it exists
		outputPath,
	)

	// Capture stderr for logging
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.logger.LogError(err, "Failed to create stderr pipe")
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		s.logger.LogError(err, fmt.Sprintf("Failed to start transcoding: input=%s, output=%s", inputPath, outputPath))
		return fmt.Errorf("failed to start transcoding: %w", err)
	}

	// Read stderr in a goroutine
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				s.logger.LogInfo("FFmpeg output", map[string]interface{}{
					"output": string(buf[:n]),
				})
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait for the command to complete
	if err := cmd.Wait(); err != nil {
		s.logger.LogError(err, fmt.Sprintf("Transcoding failed: input=%s, output=%s", inputPath, outputPath))
		return fmt.Errorf("transcoding failed: %w", err)
	}

	s.logger.LogInfo("Transcoding completed successfully", map[string]interface{}{
		"input":      inputPath,
		"output":     outputPath,
		"resolution": resolution,
	})

	return nil
}

// extractValue is a helper function to extract values from ffprobe output
func (s *Service) extractValue(output, key string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, fmt.Sprintf("\"%s\":", key)) {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				value := strings.TrimSpace(parts[1])
				value = strings.Trim(value, "\",")
				return value
			}
		}
	}
	return ""
}
