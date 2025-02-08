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

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VideoService handles video-related business logic
type VideoService struct {
	db     *gorm.DB
	ipfs   *IPFSService
	config *Config
}

// AuthService handles authentication-related business logic
type AuthService struct {
	db *gorm.DB
}

// NewVideoService creates a new video service instance
func NewVideoService(db *gorm.DB, ipfs *IPFSService, config *Config) *VideoService {
	return &VideoService{
		db:     db,
		ipfs:   ipfs,
		config: config,
	}
}

// NewAuthService creates a new auth service instance
func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{
		db: db,
	}
}

// UploadVideo handles the video upload process
func (s *VideoService) UploadVideo(file io.Reader, filename, title, description string) (*Video, error) {
	// Generate a unique file ID
	fileId := uuid.New().String()
	ext := filepath.Ext(filename)
	uniqueName := fileId + ext
	destination := filepath.Join(s.config.Storage.UploadDir, uniqueName)

	// Create video record with initial status
	video := &Video{
		FileId:      fileId,
		Title:       title,
		Description: description,
		FilePath:    destination,
		Status:      VideoStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.db.Create(video).Error; err != nil {
		return nil, fmt.Errorf("failed to create video record: %w", err)
	}

	// Ensure upload directory exists
	if err := os.MkdirAll(s.config.Storage.UploadDir, 0755); err != nil {
		s.updateVideoStatus(video, VideoStatusFailed, "Failed to create upload directory")
		return nil, err
	}

	// Create the destination file
	destFile, err := os.Create(destination)
	if err != nil {
		s.updateVideoStatus(video, VideoStatusFailed, "Failed to create destination file")
		return nil, err
	}
	defer destFile.Close()

	// Create a hash writer for checksum calculation
	hash := sha256.New()
	writer := io.MultiWriter(destFile, hash)

	// Update status to uploading
	s.updateVideoStatus(video, VideoStatusUploading, "Uploading file")

	// Copy the uploaded file to the destination while calculating checksum
	written, err := io.Copy(writer, file)
	if err != nil {
		s.cleanupFailedUpload(video)
		return nil, err
	}

	// Update video record with file info
	checksum := hex.EncodeToString(hash.Sum(nil))
	video.Checksum = checksum
	video.FileSize = written
	video.Status = VideoStatusProcessing
	video.StatusMsg = "Processing upload"
	video.UpdatedAt = time.Now()

	if err := s.db.Save(video).Error; err != nil {
		s.cleanupFailedUpload(video)
		return nil, err
	}

	// Upload to IPFS
	cid, err := s.ipfs.UploadFile(destination)
	if err != nil {
		s.cleanupFailedUpload(video)
		return nil, err
	}

	// Update video record with IPFS info
	video.IPFSCID = cid
	video.Status = VideoStatusCompleted
	video.StatusMsg = "Upload completed"
	video.UpdatedAt = time.Now()

	if err := s.db.Save(video).Error; err != nil {
		return nil, err
	}

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
		return nil
	}

	if file.Size > s.config.Video.MaxSize {
		return nil
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !s.isAllowedFileType(ext) {
		return nil
	}

	if title == "" {
		return nil
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
