package main

import (
	"io"
	"os"
	"path/filepath"
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

	// Ensure upload directory exists
	if err := os.MkdirAll(s.config.Storage.UploadDir, 0755); err != nil {
		return nil, err
	}

	// Create the destination file
	destFile, err := os.Create(destination)
	if err != nil {
		return nil, err
	}
	defer destFile.Close()

	// Copy the uploaded file to the destination
	if _, err := io.Copy(destFile, file); err != nil {
		return nil, err
	}

	// Upload to IPFS
	cid, err := s.ipfs.UploadFile(destination)
	if err != nil {
		return nil, err
	}

	// Create video record
	video := &Video{
		FileId:      fileId,
		Title:       title,
		Description: description,
		FilePath:    destination,
		IPFSCID:     cid,
		CreatedAt:   time.Now(),
	}

	if err := s.db.Create(video).Error; err != nil {
		return nil, err
	}

	return video, nil
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
