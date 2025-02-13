package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// VideoService handles video-related business logic
type VideoService struct {
	db     *gorm.DB
	ipfs   *IPFSService
	s3     *S3Service
	config *Config
}

// ProgressReader wraps an io.Reader to track read progress
type ProgressReader struct {
	reader     io.Reader
	total      int64
	read       int64
	onProgress func(bytesRead, totalBytes int64)
}

func NewProgressReader(reader io.Reader, total int64, onProgress func(bytesRead, totalBytes int64)) *ProgressReader {
	return &ProgressReader{
		reader:     reader,
		total:      total,
		onProgress: onProgress,
	}
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)
	if pr.onProgress != nil {
		pr.onProgress(pr.read, pr.total)
	}
	return n, err
}

// NewVideoService creates a new video service instance
func NewVideoService(db *gorm.DB, ipfs *IPFSService, s3 *S3Service, config *Config) *VideoService {
	if config == nil {
		log.Fatal("Config cannot be nil when creating VideoService")
	}
	if config.Video.MaxSize == 0 {
		log.Printf("Warning: Video max size is set to 0, using default of 100MB")
		config.Video.MaxSize = 100 * 1024 * 1024 // 100MB default
	}
	return &VideoService{
		db:     db,
		ipfs:   ipfs,
		s3:     s3,
		config: config,
	}
}

// ProcessVideo handles the video upload process
func (s *VideoService) ProcessVideo(file multipart.File, fileHeader *multipart.FileHeader, title, description string) (*Video, error) {
	// Calculate the SHA256 checksum of the file
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %v", err)
	}
	checksum := hex.EncodeToString(hasher.Sum(nil))

	// Seek the file back to the beginning
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek file to beginning: %v", err)
	}

	// Set initial status as uploading
	video := &Video{
		FileId:       "", // Will be set after IPFS upload
		Title:        title,
		Description:  description,
		UploadStatus: UploadStatusUploading,
		FileSize:     fileHeader.Size,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save initial video record to get an ID
	if err := s.db.Create(video).Error; err != nil {
		return nil, fmt.Errorf("failed to create initial video record: %v", err)
	}

	// Create progress reader for IPFS upload (50% of total progress)
	ipfsProgress := func(bytesRead, totalBytes int64) {
		s.updateVideoStatus(video, UploadStatusUploading)
	}
	ipfsReader := NewProgressReader(file, fileHeader.Size, ipfsProgress)

	// 1. Upload to IPFS first to get the CID
	ipfsCID, err := s.ipfs.UploadFileStream(ipfsReader)
	if err != nil {
		s.updateVideoStatus(video, UploadStatusFailed)
		return nil, fmt.Errorf("failed to upload to IPFS: %v", err)
	}

	// Update video with IPFS information
	video.FileId = ipfsCID
	video.IPFSCID = ipfsCID
	s.updateVideoStatus(video, UploadStatusUploading)

	// Seek the file back to the beginning after IPFS upload
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		s.updateVideoStatus(video, UploadStatusFailed)
		return nil, fmt.Errorf("failed to seek file to beginning: %v", err)
	}

	// Create progress reader for S3 upload (50-100% of total progress)
	s3Progress := func(bytesRead, totalBytes int64) {
		s.updateVideoStatus(video, UploadStatusUploading)
	}
	s3Reader := NewProgressReader(file, fileHeader.Size, s3Progress)

	// 2. Upload to S3 using the IPFS CID as the filename
	fileExt := filepath.Ext(fileHeader.Filename)
	fileKey := ipfsCID + fileExt
	s3URL, err := s.s3.UploadFileStream(s3Reader, fileKey)
	if err != nil {
		s.updateVideoStatus(video, UploadStatusFailed)
		return nil, fmt.Errorf("failed to upload to S3: %v", err)
	}

	// Update final status and information
	video.FilePath = s3URL
	video.Checksum = checksum
	s.updateVideoStatus(video, UploadStatusCompleted)

	return video, nil
}

// isAllowedFileType checks if the given file extension is in the allowed formats list
func (s *VideoService) isAllowedFileType(ext string) bool {
	// Remove the dot from the extension if present and convert to lowercase
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))

	// Log the extension being checked
	log.Printf("Checking file extension: %s against allowed formats: %v", ext, s.config.Video.AllowedFormats)

	for _, format := range s.config.Video.AllowedFormats {
		format = strings.ToLower(format)
		if format == ext {
			log.Printf("Found matching format: %s", format)
			return true
		}
	}

	log.Printf("No matching format found for extension: %s", ext)
	return false
}

func (s *VideoService) validateVideoUpload(file *multipart.FileHeader, title, description string) error {
	if s.config == nil {
		return fmt.Errorf("video service configuration is not properly initialized")
	}

	if file == nil {
		return fmt.Errorf("no file provided")
	}

	log.Printf("Validating video upload - Filename: %s, Size: %d, Title length: %d, Description length: %d",
		file.Filename, file.Size, len(title), len(description))

	// Check file size
	if file.Size > s.config.Video.MaxSize {
		return fmt.Errorf("file size (%d bytes) exceeds maximum allowed size of %d bytes", file.Size, s.config.Video.MaxSize)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	log.Printf("Extracted file extension: %s from filename: %s", ext, file.Filename)

	if !s.isAllowedFileType(ext) {
		log.Printf("File type validation failed for extension: %s", ext)
		return fmt.Errorf("unsupported file type: %s. Allowed types: %v", ext, s.config.Video.AllowedFormats)
	}

	// Validate title
	if title == "" {
		return fmt.Errorf("title is required")
	}

	if len(title) < s.config.Video.MinTitleLength {
		return fmt.Errorf("title must be at least %d characters", s.config.Video.MinTitleLength)
	}

	if len(title) > s.config.Video.MaxTitleLength {
		return fmt.Errorf("title must not exceed %d characters", s.config.Video.MaxTitleLength)
	}

	// Validate description length
	if len(description) > s.config.Video.MaxDescLength {
		return fmt.Errorf("description must not exceed %d characters", s.config.Video.MaxDescLength)
	}

	log.Printf("Video upload validation successful for file: %s", file.Filename)
	return nil
}

// updateVideoStatus updates the video status
func (s *VideoService) updateVideoStatus(video *Video, status UploadStatus) {
	video.UploadStatus = status
	video.UpdatedAt = time.Now()
	s.db.Save(video)
}

// cleanupFailedUpload handles cleanup of failed uploads
func (s *VideoService) cleanupFailedUpload(video *Video) {
	// Update status
	s.updateVideoStatus(video, UploadStatusFailed)

	// Remove the file if it exists
	if video.FilePath != "" {
		os.Remove(video.FilePath)
	}
}

// GetVideoStatus retrieves the current status of a video
func (s *VideoService) GetVideoStatus(fileId string) (*Video, error) {
	var video Video
	if err := s.db.Where("file_id = ?", fileId).First(&video).Error; err != nil {
		return nil, err
	}
	return &video, nil
}

// GetVideoList retrieves a list of videos with their transcodes
func (s *VideoService) GetVideoList() ([]Video, error) {
	var videos []Video
	err := s.db.Preload("Transcodes").Order("created_at desc").Find(&videos).Error
	return videos, err
}
