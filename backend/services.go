package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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

// AuthService handles authentication-related business logic
type AuthService struct {
	db *gorm.DB
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
	return &VideoService{
		db:     db,
		ipfs:   ipfs,
		s3:     s3,
		config: config,
	}
}

// NewAuthService creates a new auth service instance
func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{
		db: db,
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
		FileId:      "", // Will be set after IPFS upload
		Title:       title,
		Description: description,
		Status:      VideoStatusUploading,
		StatusMsg:   "Uploading to IPFS...",
		FileSize:    fileHeader.Size,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save initial video record to get an ID
	if err := s.db.Create(video).Error; err != nil {
		return nil, fmt.Errorf("failed to create initial video record: %v", err)
	}

	// Create progress reader for IPFS upload (50% of total progress)
	ipfsProgress := func(bytesRead, totalBytes int64) {
		progress := float64(bytesRead) / float64(totalBytes) * 50.0 // 0-50%
		video.StatusMsg = fmt.Sprintf("Uploading to IPFS... %.1f%%", progress)
		s.updateVideoStatus(video, VideoStatusUploading, video.StatusMsg)
	}
	ipfsReader := NewProgressReader(file, fileHeader.Size, ipfsProgress)

	// 1. Upload to IPFS first to get the CID
	ipfsCID, err := s.ipfs.UploadFileStream(ipfsReader)
	if err != nil {
		video.Status = VideoStatusFailed
		video.StatusMsg = fmt.Sprintf("Failed to upload to IPFS: %v", err)
		s.updateVideoStatus(video, video.Status, video.StatusMsg)
		return nil, fmt.Errorf("failed to upload to IPFS: %v", err)
	}

	// Update video with IPFS information
	video.FileId = ipfsCID
	video.IPFSCID = ipfsCID
	video.Status = VideoStatusUploading
	video.StatusMsg = "Uploading to S3..."
	s.updateVideoStatus(video, video.Status, video.StatusMsg)

	// Seek the file back to the beginning after IPFS upload
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		video.Status = VideoStatusFailed
		video.StatusMsg = fmt.Sprintf("Failed to process file: %v", err)
		s.updateVideoStatus(video, video.Status, video.StatusMsg)
		return nil, fmt.Errorf("failed to seek file to beginning: %v", err)
	}

	// Create progress reader for S3 upload (50-100% of total progress)
	s3Progress := func(bytesRead, totalBytes int64) {
		progress := 50.0 + (float64(bytesRead) / float64(totalBytes) * 50.0) // 50-100%
		video.StatusMsg = fmt.Sprintf("Uploading to S3... %.1f%%", progress)
		s.updateVideoStatus(video, VideoStatusUploading, video.StatusMsg)
	}
	s3Reader := NewProgressReader(file, fileHeader.Size, s3Progress)

	// 2. Upload to S3 using the IPFS CID as the filename
	fileExt := filepath.Ext(fileHeader.Filename)
	fileKey := ipfsCID + fileExt
	s3URL, err := s.s3.UploadFileStream(s3Reader, fileKey)
	if err != nil {
		video.Status = VideoStatusFailed
		video.StatusMsg = fmt.Sprintf("Failed to upload to S3: %v", err)
		s.updateVideoStatus(video, video.Status, video.StatusMsg)
		return nil, fmt.Errorf("failed to upload to S3: %v", err)
	}

	// Update final status and information
	video.FilePath = s3URL
	video.Checksum = checksum
	video.Status = VideoStatusCompleted
	video.StatusMsg = "Upload completed successfully"
	video.UpdatedAt = time.Now()
	s.updateVideoStatus(video, video.Status, video.StatusMsg)

	return video, nil
}

// isAllowedFileType checks if the given file extension is in the allowed formats list
func (s *VideoService) isAllowedFileType(ext string) bool {
	ext = strings.TrimPrefix(ext, ".")
	for _, format := range s.config.Video.AllowedFormats {
		if strings.EqualFold(format, ext) {
			return true
		}
	}
	return false
}

func (s *VideoService) validateVideoUpload(file *multipart.FileHeader, title, description string) error {
	if file == nil {
		return fmt.Errorf("no file provided")
	}

	if file.Size > s.config.Video.MaxSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", s.config.Video.MaxSize)
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !s.isAllowedFileType(ext) {
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	if title == "" {
		return fmt.Errorf("title is required")
	}

	if len(title) < s.config.Video.MinTitleLength {
		return fmt.Errorf("title must be at least %d characters", s.config.Video.MinTitleLength)
	}

	if len(title) > s.config.Video.MaxTitleLength {
		return fmt.Errorf("title must not exceed %d characters", s.config.Video.MaxTitleLength)
	}

	if len(description) > s.config.Video.MaxDescLength {
		return fmt.Errorf("description must not exceed %d characters", s.config.Video.MaxDescLength)
	}

	return nil
}

// updateVideoStatus updates the video status and message
func (s *VideoService) updateVideoStatus(video *Video, status, message string) {
	video.Status = status
	video.StatusMsg = message
	video.UpdatedAt = time.Now()
	s.db.Save(video)
}

// cleanupFailedUpload handles cleanup of failed uploads
func (s *VideoService) cleanupFailedUpload(video *Video) {
	// Update status
	s.updateVideoStatus(video, VideoStatusFailed, "Upload failed")

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

// Login handles user authentication
func (s *AuthService) Login(email string) (*User, error) {
	user := &User{
		Name:      "Test User",
		Email:     email,
		CreatedAt: time.Now(),
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}
