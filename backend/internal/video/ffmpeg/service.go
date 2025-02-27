package ffmpeg

import (
	"context"
	"fmt"
	"io"
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
	// Log detailed input values at the start
	s.logger.LogInfo("Beginning transcoding process", map[string]interface{}{
		"input_path": inputPath,
		"output_path": outputPath,
		"resolution": resolution,
		"ffmpeg_path": s.config.Path,
		"ffprobe_path": s.config.ProbePath,
		"video_codec": s.config.VideoCodec,
		"audio_codec": s.config.AudioCodec,
		"preset": s.config.Preset,
		"output_dir": filepath.Dir(outputPath),
	})

	// Verify that input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		errMsg := fmt.Sprintf("Input file does not exist: %s", inputPath)
		s.logger.LogError(err, errMsg)
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		errMsg := fmt.Sprintf("Failed to create output directory: path=%s", outputDir)
		s.logger.LogError(err, errMsg)
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	s.logger.LogInfo("Output directory created or verified", map[string]interface{}{
		"output_dir": outputDir,
	})

	// Get input video metadata to determine original dimensions
	s.logger.LogInfo("Getting video metadata", map[string]interface{}{
		"input_path": inputPath,
	})
	
	metadata, err := s.GetMetadata(ctx, inputPath)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get video metadata: path=%s", inputPath)
		s.logger.LogError(err, errMsg)
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	s.logger.LogInfo("Video metadata extracted", map[string]interface{}{
		"duration": metadata.Duration,
		"width": metadata.Width,
		"height": metadata.Height,
		"format": metadata.Format,
		"video_codec": metadata.VideoCodec,
		"audio_codec": metadata.AudioCodec,
		"bitrate": metadata.Bitrate,
	})

	// Convert resolution string to actual dimensions
	var width, height int
	switch resolution {
	case "720p":
		width, height = 1280, 720
	case "480p":
		width, height = 854, 480
	case "360p":
		width, height = 640, 360
	case "original":
		// Use original dimensions
		width, height = metadata.Width, metadata.Height
	default:
		errMsg := fmt.Sprintf("Unsupported resolution: %s", resolution)
		s.logger.LogError(nil, errMsg)
		return fmt.Errorf(errMsg)
	}

	// Skip upscaling if the target resolution is higher than the original
	if width > metadata.Width || height > metadata.Height {
		s.logger.LogInfo("Skipping upscaling", map[string]interface{}{
			"original_width":  metadata.Width,
			"original_height": metadata.Height,
			"target_width":    width,
			"target_height":   height,
			"resolution":      resolution,
			"action": "adjusting dimensions to avoid upscaling",
		})
		// Use original dimensions but maintain aspect ratio
		if metadata.Width > metadata.Height {
			// Landscape orientation
			height = (metadata.Height * width) / metadata.Width
			// Ensure height is even
			if height%2 != 0 {
				height += 1 // Add 1 instead of subtracting to avoid making it too small
			}
		} else {
			// Portrait or square orientation
			width = (metadata.Width * height) / metadata.Height
			// Ensure width is even
			if width%2 != 0 {
				width += 1 // Add 1 instead of subtracting to avoid making it too small
			}
		}
	}

	// Ensure dimensions are even (required by most codecs)
	// Double-check to make sure both dimensions are even, regardless of previous calculations
	if width%2 != 0 {
		width += 1 // Add 1 instead of subtracting to avoid making it too small
	}
	if height%2 != 0 {
		height += 1 // Add 1 instead of subtracting to avoid making it too small
	}

	// Format the resolution as "widthxheight"
	resolutionArg := fmt.Sprintf("%dx%d", width, height)

	s.logger.LogInfo("Starting transcoding", map[string]interface{}{
		"input":           inputPath,
		"output":          outputPath,
		"resolution_name": resolution,
		"dimensions":      resolutionArg,
		"original_width":  metadata.Width,
		"original_height": metadata.Height,
		"video_codec":     s.config.VideoCodec,
		"audio_codec":     s.config.AudioCodec,
		"preset":          s.config.Preset,
	})

	// Build FFmpeg command with proper resolution format
	cmd := exec.CommandContext(ctx, s.config.Path,
		"-i", inputPath,
		"-c:v", s.config.VideoCodec,
		"-c:a", s.config.AudioCodec,
		"-s", resolutionArg, // Use the formatted resolution
		"-preset", s.config.Preset,
		"-y", // Overwrite output file if it exists
		outputPath,
	)

	// Log the exact command being executed
	s.logger.LogInfo("Executing FFmpeg command", map[string]interface{}{
		"command": cmd.String(),
		"arguments": cmd.Args,
	})

	// Capture stderr for logging
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.logger.LogError(err, "Failed to create stderr pipe")
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		errMsg := fmt.Sprintf("Failed to start transcoding: input=%s, output=%s", inputPath, outputPath)
		s.logger.LogError(err, errMsg)
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	s.logger.LogInfo("FFmpeg process started", map[string]interface{}{
		"pid": cmd.Process.Pid,
	})

	// Read stderr in a goroutine
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				s.logger.LogInfo("FFmpeg output", map[string]interface{}{
					"output": string(buf[:n]),
					"pid": cmd.Process.Pid,
				})
			}
			if err != nil {
				if err != io.EOF {
					s.logger.LogError(err, "Error reading FFmpeg stderr")
				}
				break
			}
		}
	}()

	// Wait for the command to complete
	if err := cmd.Wait(); err != nil {
		errMsg := fmt.Sprintf("Transcoding failed: input=%s, output=%s, dimensions=%s",
			inputPath, outputPath, resolutionArg)
		s.logger.LogError(err, errMsg)
		
		// Check if output file exists despite error
		if _, statErr := os.Stat(outputPath); statErr == nil {
			s.logger.LogInfo("Note: Output file exists despite transcoding error", map[string]interface{}{
				"output_path": outputPath,
				"file_size": getFileSize(outputPath),
			})
		}
		
		return fmt.Errorf("TRANSCODE_FAILED: %s: %w", errMsg, err)
	}

	// Verify output file exists and has content
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		errMsg := fmt.Sprintf("Transcoded file not found: %s", outputPath)
		s.logger.LogError(err, errMsg)
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	if fileInfo.Size() == 0 {
		errMsg := fmt.Sprintf("Transcoded file is empty: %s", outputPath)
		s.logger.LogError(nil, errMsg)
		return fmt.Errorf(errMsg)
	}

	s.logger.LogInfo("Transcoding completed successfully", map[string]interface{}{
		"input":           inputPath,
		"output":          outputPath,
		"resolution_name": resolution,
		"dimensions":      resolutionArg,
		"file_size":       fileInfo.Size(),
		"output_exists":   true,
	})

	return nil
}

// getFileSize is a helper to safely get file size
func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return -1
	}
	return info.Size()
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
