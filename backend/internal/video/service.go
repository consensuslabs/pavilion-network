package video

import (
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"
)

// Service implements the VideoService interface
type Service struct {
	db     *gorm.DB
	ipfs   IPFSService
	s3     S3Service
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

// NewService creates a new video service instance
func NewService(db *gorm.DB, ipfs IPFSService, s3 S3Service, config *Config) *Service {
	return &Service{
		db:     db,
		ipfs:   ipfs,
		s3:     s3,
		config: config,
	}
}

// ProcessVideo processes a video upload
func (s *Service) ProcessVideo(file interface{}, header interface{}, title, description string) (*Video, error) {
	videoFile, ok := file.(multipart.File)
	if !ok {
		return nil, fmt.Errorf("invalid file type")
	}

	fileHeader, ok := header.(*multipart.FileHeader)
	if !ok {
		return nil, fmt.Errorf("invalid file header type")
	}

	// Calculate the SHA256 checksum of the file
	hasher := sha256.New()
	if _, err := io.Copy(hasher, videoFile); err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %v", err)
	}

	// Seek the file back to the beginning
	if _, err := videoFile.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek file to beginning: %v", err)
	}

	// Set initial status as uploading
	video := &Video{
		FileId:       "", // Will be set after IPFS upload
		Title:        title,
		Description:  description,
		UploadStatus: "uploading",
		FileSize:     fileHeader.Size,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save initial video record to get an ID
	if err := s.db.Create(video).Error; err != nil {
		return nil, fmt.Errorf("failed to create initial video record: %v", err)
	}

	// Create progress reader for IPFS upload
	ipfsProgress := func(bytesRead, totalBytes int64) {
		s.updateVideoStatus(video, "uploading")
	}
	ipfsReader := NewProgressReader(videoFile, fileHeader.Size, ipfsProgress)

	// 1. Upload to IPFS first to get the CID
	ipfsCID, err := s.ipfs.UploadFileStream(ipfsReader)
	if err != nil {
		s.updateVideoStatus(video, "failed")
		return nil, fmt.Errorf("failed to upload to IPFS: %v", err)
	}

	// Update video with IPFS information
	video.FileId = ipfsCID
	video.IPFSCID = ipfsCID
	s.updateVideoStatus(video, "uploading")

	// Seek the file back to the beginning after IPFS upload
	if _, err := videoFile.Seek(0, io.SeekStart); err != nil {
		s.updateVideoStatus(video, "failed")
		return nil, fmt.Errorf("failed to seek file to beginning: %v", err)
	}

	// Create progress reader for S3 upload
	s3Progress := func(bytesRead, totalBytes int64) {
		s.updateVideoStatus(video, "uploading")
	}
	s3Reader := NewProgressReader(videoFile, fileHeader.Size, s3Progress)

	// 2. Upload to S3 using the IPFS CID as the filename
	fileExt := filepath.Ext(fileHeader.Filename)
	fileKey := ipfsCID + fileExt
	s3URL, err := s.s3.UploadFileStream(s3Reader, fileKey)
	if err != nil {
		s.updateVideoStatus(video, "failed")
		return nil, fmt.Errorf("failed to upload to S3: %v", err)
	}

	// Update final status and information
	video.FilePath = s3URL
	s.updateVideoStatus(video, "completed")

	return video, nil
}

// GetVideoList returns a list of videos
func (s *Service) GetVideoList() ([]Video, error) {
	var videos []Video
	err := s.db.Order("created_at desc").Find(&videos).Error
	return videos, err
}

// GetVideoStatus returns the status of a video
func (s *Service) GetVideoStatus(fileID string) (*Video, error) {
	var video Video
	if err := s.db.Where("file_id = ?", fileID).First(&video).Error; err != nil {
		return nil, err
	}
	return &video, nil
}

// updateVideoStatus updates the video status
func (s *Service) updateVideoStatus(video *Video, status string) {
	video.UploadStatus = status
	video.UpdatedAt = time.Now()
	s.db.Save(video)
}

// cleanupFailedUpload handles cleanup of failed uploads
func (s *Service) cleanupFailedUpload(video *Video) {
	// Update status
	s.updateVideoStatus(video, "failed")

	// Remove the file if it exists
	if video.FilePath != "" {
		os.Remove(video.FilePath)
	}
}
