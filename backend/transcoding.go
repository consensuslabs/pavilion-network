package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TranscodeService handles video transcoding operations
type TranscodeService struct {
	db     *gorm.DB
	ipfs   *IPFSService
	s3     *S3Service
	config *Config
	logger *Logger
}

// NewTranscodeService creates a new transcoding service instance
func NewTranscodeService(db *gorm.DB, ipfs *IPFSService, s3 *S3Service, config *Config, logger *Logger) *TranscodeService {
	return &TranscodeService{
		db:     db,
		ipfs:   ipfs,
		s3:     s3,
		config: config,
		logger: logger,
	}
}

// ProcessTranscode handles the complete transcoding process for a video
func (s *TranscodeService) ProcessTranscode(cid string) (*TranscodeResult, error) {
	// Download video from IPFS once
	localFilePath, err := s.ipfs.DownloadFile(cid)
	if err != nil {
		return nil, fmt.Errorf("failed to download video from IPFS: %v", err)
	}

	// Ensure cleanup of local file when we're done
	defer func() {
		if err := os.Remove(localFilePath); err != nil {
			s.logger.LogWarn("Failed to cleanup local file", map[string]interface{}{
				"path":  localFilePath,
				"error": err.Error(),
			})
		}
	}()

	result := &TranscodeResult{
		Transcodes:        make([]Transcode, 0),
		TranscodeSegments: make([]TranscodeSegment, 0),
	}

	// Process all formats and resolutions
	resolutions := []string{"720", "480", "360"}
	storageTypes := []string{"ipfs", "s3"}

	// Process HLS format
	for _, resolution := range resolutions {
		for _, storageType := range storageTypes {
			hlsResult, err := s.processHLSFormat(localFilePath, resolution, storageType)
			if err != nil {
				return nil, fmt.Errorf("failed to process HLS for %sp (%s): %v", resolution, storageType, err)
			}
			result.Transcodes = append(result.Transcodes, hlsResult.Transcode)
			result.TranscodeSegments = append(result.TranscodeSegments, hlsResult.Segments...)
		}
	}

	// Process MP4 format
	for _, resolution := range resolutions {
		for _, storageType := range storageTypes {
			mp4Result, err := s.processMP4Format(localFilePath, resolution, storageType)
			if err != nil {
				return nil, fmt.Errorf("failed to process MP4 for %sp (%s): %v", resolution, storageType, err)
			}
			result.Transcodes = append(result.Transcodes, mp4Result)
		}
	}

	return result, nil
}

type HLSResult struct {
	Transcode Transcode
	Segments  []TranscodeSegment
}

func (s *TranscodeService) processHLSFormat(inputPath, resolution, storageType string) (*HLSResult, error) {
	// Create temporary directory for HLS output
	outputDir := fmt.Sprintf("temp_hls_%s_%s_%s",
		filepath.Base(inputPath),
		resolution,
		uuid.New().String())

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}
	defer os.RemoveAll(outputDir)

	// Set up HLS transcoding parameters
	manifestPath := filepath.Join(outputDir, "playlist.m3u8")
	segmentPattern := filepath.Join(outputDir, "segment_%03d.ts")

	// Perform HLS transcoding
	cmdArgs := []string{
		"-i", inputPath,
		"-vf", fmt.Sprintf("scale=-2:%s", resolution),
		"-c:v", s.config.Ffmpeg.VideoCodec,
		"-c:a", s.config.Ffmpeg.AudioCodec,
		"-preset", s.config.Ffmpeg.Preset,
		"-hls_time", fmt.Sprintf("%d", s.config.Ffmpeg.HLSTime),
		"-hls_playlist_type", s.config.Ffmpeg.HLSPlaylistType,
		"-hls_segment_filename", segmentPattern,
		"-f", "hls",
		manifestPath,
	}
	s.logger.LogInfo("Executing ffmpeg command", map[string]interface{}{
		"ffmpegPath": s.config.Ffmpeg.Path,
		"args":       cmdArgs,
	})
	cmd := exec.Command(s.config.Ffmpeg.Path, cmdArgs...)

	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg transcoding failed: %v, output: %s", err, string(output))
	}

	// Process segments and manifest
	segments, manifestURL, manifestCID, err := s.processHLSFiles(outputDir, manifestPath, resolution, storageType)
	if err != nil {
		return nil, err
	}

	// Create transcode record
	transcode := Transcode{
		FilePath:    manifestURL,
		FileCID:     manifestCID,
		Format:      "hls",
		Resolution:  resolution,
		StorageType: storageType,
		Type:        "manifest",
		CreatedAt:   time.Now(),
	}

	if err := s.db.Create(&transcode).Error; err != nil {
		return nil, fmt.Errorf("failed to create transcode record: %v", err)
	}

	// Create segment records
	var segmentRecords []TranscodeSegment
	for _, seg := range segments {
		seg.TranscodeID = transcode.ID
		if err := s.db.Create(&seg).Error; err != nil {
			return nil, fmt.Errorf("failed to create segment record: %v", err)
		}
		segmentRecords = append(segmentRecords, seg)
	}

	return &HLSResult{
		Transcode: transcode,
		Segments:  segmentRecords,
	}, nil
}

func (s *TranscodeService) processMP4Format(inputPath, resolution, storageType string) (Transcode, error) {
	// Create temporary output file
	outputPath := fmt.Sprintf("temp_mp4_%s_%s_%s.mp4",
		filepath.Base(inputPath),
		resolution,
		uuid.New().String())
	defer os.Remove(outputPath)

	// Perform MP4 transcoding
	cmd := exec.Command(
		s.config.Ffmpeg.Path,
		"-i", inputPath,
		"-vf", fmt.Sprintf("scale=-2:%s", resolution),
		"-c:v", s.config.Ffmpeg.VideoCodec,
		"-c:a", s.config.Ffmpeg.AudioCodec,
		"-preset", s.config.Ffmpeg.Preset,
		outputPath,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return Transcode{}, fmt.Errorf("ffmpeg transcoding failed: %v, output: %s", err, string(output))
	}

	var filePath, fileCID string
	var err error

	if storageType == "ipfs" {
		fileCID, err = s.ipfs.UploadFile(outputPath)
		if err != nil {
			return Transcode{}, fmt.Errorf("failed to upload MP4 to IPFS: %v", err)
		}
		filePath = s.ipfs.GetGatewayURL(fileCID)
	} else {
		s3Key := fmt.Sprintf("mp4/%s_%sp.mp4", uuid.New().String(), resolution)
		filePath, err = s.s3.UploadFile(outputPath, s3Key)
		if err != nil {
			return Transcode{}, fmt.Errorf("failed to upload MP4 to S3: %v", err)
		}
	}

	transcode := Transcode{
		FilePath:    filePath,
		FileCID:     fileCID,
		Format:      "mp4",
		Resolution:  resolution,
		StorageType: storageType,
		Type:        "video",
		CreatedAt:   time.Now(),
	}

	if err := s.db.Create(&transcode).Error; err != nil {
		return Transcode{}, fmt.Errorf("failed to create transcode record: %v", err)
	}

	return transcode, nil
}

func (s *TranscodeService) processHLSFiles(outputDir, manifestPath, resolution, storageType string) ([]TranscodeSegment, string, string, error) {
	// Find all .ts files in the output directory
	segmentFiles, err := filepath.Glob(filepath.Join(outputDir, "*.ts"))
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to list segment files: %v", err)
	}

	segments := make([]TranscodeSegment, 0, len(segmentFiles))
	segmentURLs := make(map[string]string)

	// Process each segment
	for i, segFile := range segmentFiles {
		var segmentURL, segmentCID string

		if storageType == "ipfs" {
			segmentCID, err = s.ipfs.UploadFile(segFile)
			if err != nil {
				return nil, "", "", fmt.Errorf("failed to upload segment to IPFS: %v", err)
			}
			segmentURL = s.ipfs.GetGatewayURL(segmentCID)
		} else {
			s3Key := fmt.Sprintf("hls/segments/%s/%d.ts", uuid.New().String(), i)
			segmentURL, err = s.s3.UploadFile(segFile, s3Key)
			if err != nil {
				return nil, "", "", fmt.Errorf("failed to upload segment to S3: %v", err)
			}
		}

		// Get segment duration using ffprobe
		duration, err := s.getSegmentDuration(segFile)
		if err != nil {
			s.logger.LogWarn("Failed to get segment duration", map[string]interface{}{
				"error": err.Error(),
				"file":  segFile,
			})
			duration = float64(s.config.Ffmpeg.HLSTime) // Use configured duration as fallback
		}

		segments = append(segments, TranscodeSegment{
			FilePath:    segmentURL,
			FileCID:     segmentCID,
			StorageType: storageType,
			Sequence:    i + 1,
			Duration:    duration,
			CreatedAt:   time.Now(),
		})

		segmentURLs[filepath.Base(segFile)] = segmentURL
	}

	// Update manifest with correct URLs
	updatedManifest, err := s.updateManifestContent(manifestPath, segmentURLs)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to update manifest: %v", err)
	}

	// Write updated manifest to temporary file
	tempManifest := manifestPath + ".temp"
	if err := os.WriteFile(tempManifest, []byte(updatedManifest), 0644); err != nil {
		return nil, "", "", fmt.Errorf("failed to write updated manifest: %v", err)
	}
	defer os.Remove(tempManifest)

	var manifestURL, manifestCID string

	if storageType == "ipfs" {
		manifestCID, err = s.ipfs.UploadFile(tempManifest)
		if err != nil {
			return nil, "", "", fmt.Errorf("failed to upload manifest to IPFS: %v", err)
		}
		manifestURL = s.ipfs.GetGatewayURL(manifestCID)
	} else {
		s3Key := fmt.Sprintf("hls/manifests/%s_%sp.m3u8", uuid.New().String(), resolution)
		manifestURL, err = s.s3.UploadFile(tempManifest, s3Key)
		if err != nil {
			return nil, "", "", fmt.Errorf("failed to upload manifest to S3: %v", err)
		}
	}

	return segments, manifestURL, manifestCID, nil
}

func (s *TranscodeService) getSegmentDuration(segmentPath string) (float64, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		segmentPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var duration float64
	if _, err := fmt.Sscanf(string(output), "%f", &duration); err != nil {
		return 0, err
	}

	return duration, nil
}

func (s *TranscodeService) updateManifestContent(manifestPath string, segmentURLs map[string]string) (string, error) {
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("failed to read manifest: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	var updatedLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		if strings.HasPrefix(trimmedLine, "#") {
			updatedLines = append(updatedLines, line)
			continue
		}

		if url, exists := segmentURLs[trimmedLine]; exists {
			updatedLines = append(updatedLines, url)
		} else {
			updatedLines = append(updatedLines, line)
		}
	}

	return strings.Join(updatedLines, "\n"), nil
}

type TranscodeResult struct {
	Transcodes        []Transcode
	TranscodeSegments []TranscodeSegment
}
