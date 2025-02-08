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
	s3     *S3Service
	config *Config
}

// AuthService handles authentication-related business logic
type AuthService struct {
	db *gorm.DB
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
func (s *VideoService) ProcessVideo(file multipart.File, fileHeader *multipart.FileHeader) (*Video, error) {
	// Generate a unique file ID
	fileId := uuid.New().String()

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

	// 1. Upload to IPFS
	ipfsCID, err := s.ipfs.UploadFileStream(file)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to IPFS: %v", err)
	}

	// Seek the file back to the beginning after IPFS upload
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek file to beginning: %v", err)
	}

	// 2. Upload to S3
	fileKey := fileId + filepath.Ext(fileHeader.Filename)
	s3URL, err := s.s3.UploadFileStream(file, fileKey)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %v", err)
	}

	// Create video record in the database
	video := &Video{
		FileId:    fileId,
		Title:     fileHeader.Filename,
		FilePath:  s3URL,
		IPFSCID:   ipfsCID,
		Checksum:  checksum,
		Status:    VideoStatusPending,
		FileSize:  fileHeader.Size,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.Create(video).Error; err != nil {
		return nil, fmt.Errorf("failed to create video record: %v", err)
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
