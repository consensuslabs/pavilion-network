package video

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	videostorage "github.com/consensuslabs/pavilion-network/backend/internal/storage/video"
	"github.com/consensuslabs/pavilion-network/backend/internal/video/ffmpeg"
	"github.com/consensuslabs/pavilion-network/backend/internal/video/tempfile"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VideoServiceImpl implements the VideoService interface
type VideoServiceImpl struct {
	db          *gorm.DB
	ipfs        IPFSService
	storage     videostorage.Service
	ffmpeg      *ffmpeg.Service
	tempManager tempfile.TempFileManager
	logger      Logger
}

// NewVideoService creates a new video service instance
func NewVideoService(
	db *gorm.DB,
	ipfs IPFSService,
	storage videostorage.Service,
	ffmpeg *ffmpeg.Service,
	tempManager tempfile.TempFileManager,
	logger Logger,
) VideoService {
	return &VideoServiceImpl{
		db:          db,
		ipfs:        ipfs,
		storage:     storage,
		ffmpeg:      ffmpeg,
		tempManager: tempManager,
		logger:      logger,
	}
}

// InitializeUpload creates a new video upload record
func (s *VideoServiceImpl) InitializeUpload(title, description string, size int64) (*VideoUpload, error) {
	videoID := uuid.New()
	fileID := uuid.New().String()

	// Create the video record
	video := &Video{
		ID:          videoID,
		FileID:      fileID,
		Title:       title,
		Description: description,
		StoragePath: fmt.Sprintf("videos/%s/original.mp4", videoID),
		FileSize:    size,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create the upload record
	upload := &VideoUpload{
		VideoID:   videoID,
		StartTime: time.Now(),
		Status:    UploadStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Start a transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(video).Error; err != nil {
			return fmt.Errorf("failed to create video record: %w", err)
		}
		if err := tx.Create(upload).Error; err != nil {
			return fmt.Errorf("failed to create upload record: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	upload.Video = video
	return upload, nil
}

// ProcessUpload handles the video upload process
func (s *VideoServiceImpl) ProcessUpload(upload *VideoUpload, file multipart.File, header *multipart.FileHeader) error {
	ctx := context.Background()

	// Update status to uploading
	upload.Status = UploadStatusUploading
	if err := s.db.Model(upload).Update("status", UploadStatusUploading).Error; err != nil {
		return fmt.Errorf("failed to update upload status: %w", err)
	}

	// Create temporary directory for processing
	tempDir, err := s.tempManager.CreateTempDir()
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer s.tempManager.CleanupDir(tempDir)

	// Save original file
	originalPath := filepath.Join(tempDir, "original.mp4")
	tempFile, err := os.Create(originalPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	if _, err := io.Copy(tempFile, file); err != nil {
		return fmt.Errorf("failed to save temp file: %w", err)
	}

	// Get video metadata
	metadata, err := s.ffmpeg.GetMetadata(ctx, originalPath)
	if err != nil {
		s.logger.LogError("Failed to get video metadata", map[string]interface{}{
			"error": err.Error(),
			"path":  originalPath,
		})
		return fmt.Errorf("failed to get video metadata: %w", err)
	}

	// Upload original to S3
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	_, err = s.storage.UploadVideo(ctx, upload.VideoID, "original", file)
	if err != nil {
		upload.Status = UploadStatusFailed
		s.db.Model(upload).Updates(map[string]interface{}{
			"status":     UploadStatusFailed,
			"end_time":   time.Now(),
			"updated_at": time.Now(),
		})
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Upload original to IPFS
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	cid, err := s.ipfs.UploadFileStream(file)
	if err != nil {
		s.logger.LogError("Failed to upload to IPFS", map[string]interface{}{
			"error": err.Error(),
			"path":  originalPath,
		})
		// Continue processing even if IPFS upload fails
	}

	// Create transcode record for original format
	originalTranscode := &Transcode{
		VideoID:   upload.VideoID,
		Format:    "mp4",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create segment for original file
	originalSegment := &TranscodeSegment{
		StoragePath: upload.Video.StoragePath,
		IPFSCID:     cid,
		Duration:    int(metadata.Duration),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Process transcoding for different resolutions
	transcodeResults := make([]*Transcode, 0)

	for _, resolution := range []string{"720p", "480p", "360p"} {
		transcode := &Transcode{
			VideoID:   upload.VideoID,
			Format:    "mp4",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		outputPath := filepath.Join(tempDir, fmt.Sprintf("%s.mp4", resolution))
		if err := s.ffmpeg.Transcode(ctx, originalPath, outputPath, resolution); err != nil {
			s.logger.LogError("Failed to transcode video", map[string]interface{}{
				"error":      err.Error(),
				"resolution": resolution,
				"input":      originalPath,
				"output":     outputPath,
			})
			continue // Skip this resolution but continue with others
		}

		// Upload transcoded file to S3
		transcodedFile, err := os.Open(outputPath)
		if err != nil {
			s.logger.LogError("Failed to open transcoded file", map[string]interface{}{
				"error": err.Error(),
				"path":  outputPath,
			})
			continue
		}

		_, err = s.storage.UploadVideo(ctx, upload.VideoID, resolution, transcodedFile)
		transcodedFile.Close()
		if err != nil {
			s.logger.LogError("Failed to upload transcoded file to S3", map[string]interface{}{
				"error":      err.Error(),
				"resolution": resolution,
			})
			continue
		}

		// Upload transcoded file to IPFS
		transcodedFile, err = os.Open(outputPath)
		if err != nil {
			s.logger.LogError("Failed to open transcoded file for IPFS", map[string]interface{}{
				"error": err.Error(),
				"path":  outputPath,
			})
			continue
		}

		transcodedCID, err := s.ipfs.UploadFileStream(transcodedFile)
		transcodedFile.Close()
		if err != nil {
			s.logger.LogError("Failed to upload transcoded file to IPFS", map[string]interface{}{
				"error":      err.Error(),
				"resolution": resolution,
			})
			// Continue without IPFS CID
		}

		// Get transcoded file metadata
		transcodedMetadata, err := s.ffmpeg.GetMetadata(ctx, outputPath)
		if err != nil {
			s.logger.LogError("Failed to get transcoded video metadata", map[string]interface{}{
				"error":      err.Error(),
				"resolution": resolution,
				"path":       outputPath,
			})
			continue
		}

		segment := &TranscodeSegment{
			StoragePath: fmt.Sprintf("videos/%s/%s.mp4", upload.VideoID, resolution),
			IPFSCID:     transcodedCID,
			Duration:    int(transcodedMetadata.Duration),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		transcode.Segments = []TranscodeSegment{*segment}
		transcodeResults = append(transcodeResults, transcode)
	}

	// Start a transaction to update all records
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Update video with IPFS CID
		if err := tx.Model(upload.Video).Updates(map[string]interface{}{
			"ipfs_cid":   cid,
			"updated_at": time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("failed to update video record: %w", err)
		}

		// Create original transcode and segment
		if err := tx.Create(originalTranscode).Error; err != nil {
			return fmt.Errorf("failed to create original transcode record: %w", err)
		}

		originalSegment.TranscodeID = originalTranscode.ID
		if err := tx.Create(originalSegment).Error; err != nil {
			return fmt.Errorf("failed to create original segment record: %w", err)
		}

		// Create transcodes and segments for each resolution
		for _, transcode := range transcodeResults {
			if err := tx.Create(transcode).Error; err != nil {
				return fmt.Errorf("failed to create transcode record: %w", err)
			}

			segment := transcode.Segments[0]
			segment.TranscodeID = transcode.ID
			if err := tx.Create(&segment).Error; err != nil {
				return fmt.Errorf("failed to create segment record: %w", err)
			}
		}

		// Update upload status to completed
		return tx.Model(upload).Updates(map[string]interface{}{
			"status":     UploadStatusCompleted,
			"end_time":   time.Now(),
			"updated_at": time.Now(),
		}).Error
	})

	if err != nil {
		return fmt.Errorf("failed to update records: %w", err)
	}

	return nil
}

// GetVideo retrieves a video by ID
func (s *VideoServiceImpl) GetVideo(videoID uuid.UUID) (*Video, error) {
	var video Video
	if err := s.db.Preload("Upload").Preload("Transcodes").Preload("Transcodes.Segments").First(&video, videoID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get video: %w", err)
	}
	return &video, nil
}

// ListVideos retrieves a list of videos with pagination
func (s *VideoServiceImpl) ListVideos(page, limit int) ([]Video, error) {
	var videos []Video
	offset := (page - 1) * limit

	if err := s.db.Preload("Upload").Preload("Transcodes").Preload("Transcodes.Segments").
		Offset(offset).Limit(limit).Find(&videos).Error; err != nil {
		return nil, fmt.Errorf("failed to list videos: %w", err)
	}

	return videos, nil
}

// DeleteVideo deletes a video by ID
func (s *VideoServiceImpl) DeleteVideo(videoID uuid.UUID) error {
	ctx := context.Background()

	// Get video details first
	video, err := s.GetVideo(videoID)
	if err != nil {
		return fmt.Errorf("failed to get video details: %w", err)
	}
	if video == nil {
		return fmt.Errorf("video not found: %s", videoID)
	}

	// Delete files from S3
	if err := s.storage.DeleteVideo(ctx, videoID); err != nil {
		s.logger.LogError("Failed to delete video files from S3", map[string]interface{}{
			"error":   err.Error(),
			"videoID": videoID,
		})
		// Continue with database deletion even if S3 deletion fails
	}

	// Delete from database (will cascade to related records)
	if err := s.db.Delete(&Video{}, videoID).Error; err != nil {
		return fmt.Errorf("failed to delete video: %w", err)
	}

	return nil
}

// UpdateVideo updates a video's metadata
func (s *VideoServiceImpl) UpdateVideo(videoID uuid.UUID, title, description string) error {
	updates := map[string]interface{}{
		"title":       title,
		"description": description,
		"updated_at":  time.Now(),
	}

	if err := s.db.Model(&Video{}).Where("id = ?", videoID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update video: %w", err)
	}
	return nil
}
