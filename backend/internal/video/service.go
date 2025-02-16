package video

import (
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service implements the VideoService interface
type Service struct {
	db     *gorm.DB
	ipfs   IPFSService
	s3     S3Service
	config *Config
	logger Logger
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
	// Verify database connection
	var count int64
	if err := db.Table("video_uploads").Count(&count).Error; err != nil {
		fmt.Printf("Error verifying database connection: %v\n", err)
	} else {
		fmt.Printf("Database connection verified. Found %d existing uploads.\n", count)
	}

	return &Service{
		db:     db,
		ipfs:   ipfs,
		s3:     s3,
		config: config,
	}
}

// InitializeUpload creates the initial upload record
func (s *Service) InitializeUpload(title, description string, fileSize int64) (*VideoUpload, error) {
	// Generate a temporary file ID
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%s-%d-%d", title, description, fileSize, time.Now().UnixNano())))
	tempFileId := fmt.Sprintf("%x", hasher.Sum(nil))
	fmt.Printf("Generated tempFileId: %s\n", tempFileId)

	// Create initial upload record
	now := time.Now()
	upload := &VideoUpload{
		TempFileId:    tempFileId,
		Title:         title,
		Description:   description,
		FileSize:      fileSize,
		UploadStatus:  StatusPending,
		CurrentPhase:  "IPFS",
		IPFSStartTime: &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Save initial upload record
	if err := s.db.Create(upload).Error; err != nil {
		fmt.Printf("Error creating upload record: %v\n", err)
		return nil, fmt.Errorf("failed to create upload record: %v", err)
	}
	fmt.Printf("Created initial upload record with ID: %d\n", upload.ID)

	return upload, nil
}

// ProcessUpload handles the actual upload process
func (s *Service) ProcessUpload(upload *VideoUpload, file interface{}, header interface{}) error {
	videoFile, ok := file.(multipart.File)
	if !ok {
		return fmt.Errorf("invalid file type")
	}

	fileHeader, ok := header.(*multipart.FileHeader)
	if !ok {
		return fmt.Errorf("invalid file header type")
	}

	// Create progress reader for IPFS upload
	ipfsProgress := func(bytesRead, totalBytes int64) {
		upload.IPFSBytesUploaded = bytesRead
		upload.UploadStatus = StatusIPFSUploading
		if err := s.db.Save(upload).Error; err != nil {
			fmt.Printf("Error saving IPFS progress: %v\n", err)
		}
		fmt.Printf("IPFS Progress: %d/%d bytes\n", bytesRead, totalBytes)
	}
	ipfsReader := NewProgressReader(videoFile, fileHeader.Size, ipfsProgress)

	// Upload to IPFS
	ipfsCID, err := s.ipfs.UploadFileStream(ipfsReader)
	if err != nil {
		upload.UploadStatus = StatusIPFSFailed
		upload.ErrorMessage = fmt.Sprintf("IPFS upload failed: %v", err)
		upload.UpdatedAt = time.Now()
		s.db.Save(upload)
		return fmt.Errorf("failed to upload to IPFS: %v", err)
	}

	// Update IPFS completion status
	now := time.Now()
	upload.IPFSEndTime = &now
	upload.IPFSCID = ipfsCID
	upload.UploadStatus = StatusIPFSCompleted
	upload.UpdatedAt = now
	s.db.Save(upload)

	// Seek the file back to the beginning for S3 upload
	if _, err := videoFile.Seek(0, io.SeekStart); err != nil {
		upload.UploadStatus = StatusFailed
		upload.ErrorMessage = fmt.Sprintf("Failed to prepare for S3 upload: %v", err)
		upload.UpdatedAt = time.Now()
		s.db.Save(upload)
		return fmt.Errorf("failed to seek file to beginning: %v", err)
	}

	// Update status for S3 phase
	now = time.Now()
	upload.CurrentPhase = "S3"
	upload.S3StartTime = &now
	upload.UploadStatus = StatusS3Uploading
	upload.UpdatedAt = now
	s.db.Save(upload)

	// Create progress reader for S3 upload
	s3Progress := func(bytesRead, totalBytes int64) {
		upload.S3BytesUploaded = bytesRead
		upload.UpdatedAt = time.Now()
		s.db.Save(upload)
		fmt.Printf("S3 Progress: %d/%d bytes\n", bytesRead, totalBytes)
	}
	s3Reader := NewProgressReader(videoFile, fileHeader.Size, s3Progress)

	// Upload to S3
	fileExt := filepath.Ext(fileHeader.Filename)
	fileKey := ipfsCID + fileExt
	s3URL, err := s.s3.UploadFileStream(s3Reader, fileKey)
	if err != nil {
		upload.UploadStatus = StatusS3Failed
		upload.ErrorMessage = fmt.Sprintf("S3 upload failed: %v", err)
		upload.UpdatedAt = time.Now()
		s.db.Save(upload)
		return fmt.Errorf("failed to upload to S3: %v", err)
	}

	// Update final status
	now = time.Now()
	upload.S3EndTime = &now
	upload.S3URL = s3URL
	upload.UploadStatus = StatusCompleted
	upload.UpdatedAt = now
	s.db.Save(upload)

	// Create record in the videos table
	video := &Video{
		FileId:       upload.TempFileId,
		Title:        upload.Title,
		Description:  upload.Description,
		IPFSCID:      upload.IPFSCID,
		FilePath:     s3URL,
		UploadStatus: "completed",
		FileSize:     upload.FileSize,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.db.Create(video).Error; err != nil {
		fmt.Printf("Error creating video record: %v\n", err)
		return fmt.Errorf("failed to create video record: %v", err)
	}
	fmt.Printf("Created video record with ID: %d\n", video.ID)

	return nil
}

// GetVideoStatus returns the status of a video upload
func (s *Service) GetVideoStatus(fileID string) (*VideoUpload, error) {
	var upload VideoUpload
	if err := s.db.Where("temp_file_id = ?", fileID).First(&upload).Error; err != nil {
		return nil, err
	}
	return &upload, nil
}

// GetVideoList returns a list of completed video uploads
func (s *Service) GetVideoList() ([]VideoUpload, error) {
	var uploads []VideoUpload
	err := s.db.Where("upload_status = ?", StatusCompleted).Order("created_at desc").Find(&uploads).Error
	return uploads, err
}

// cleanupFailedUpload handles cleanup of failed uploads
func (s *Service) cleanupFailedUpload(upload *VideoUpload) {
	now := time.Now()
	upload.UploadStatus = StatusFailed
	upload.UpdatedAt = now

	if upload.CurrentPhase == "IPFS" {
		upload.IPFSEndTime = &now
	} else if upload.CurrentPhase == "S3" {
		upload.S3EndTime = &now
	}

	s.db.Save(upload)
}

// ProcessTranscode handles the complete transcoding process for a video
func (s *Service) ProcessTranscode(cid string) (*TranscodeResult, error) {
	// Download video from IPFS once
	localFilePath, err := s.ipfs.DownloadFile(cid)
	if err != nil {
		return nil, fmt.Errorf("failed to download video from IPFS: %v", err)
	}

	// Ensure cleanup of local file when we're done
	defer func() {
		if err := os.Remove(localFilePath); err != nil {
			s.logger.LogInfo("Failed to cleanup local file", map[string]interface{}{
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

func (s *Service) processHLSFormat(inputPath, resolution, storageType string) (*HLSResult, error) {
	// Create temporary directory for HLS output
	outputDir := fmt.Sprintf("temp_hls_%s_%s_%s",
		filepath.Base(inputPath),
		resolution,
		uuid.New().String())

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}
	defer os.RemoveAll(outputDir)

	// Prepare ffmpeg command for HLS
	outputPath := filepath.Join(outputDir, "playlist.m3u8")
	cmd := exec.Command(
		s.config.Ffmpeg.Path,
		"-i", inputPath,
		"-vf", fmt.Sprintf("scale=-2:%s", resolution),
		"-c:v", s.config.Ffmpeg.VideoCodec,
		"-c:a", s.config.Ffmpeg.AudioCodec,
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", s.config.Ffmpeg.HLSTime),
		"-hls_playlist_type", s.config.Ffmpeg.HLSPlaylistType,
		"-hls_segment_filename", filepath.Join(outputDir, "segment_%03d.ts"),
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg command failed: %v", err)
	}

	// Create transcode record for playlist
	playlistTranscode := Transcode{
		Format:      "hls",
		Resolution:  resolution,
		StorageType: storageType,
		Type:        "manifest",
		CreatedAt:   time.Now(),
	}

	// Upload and save playlist file
	if storageType == "ipfs" {
		cid, err := s.ipfs.UploadFile(outputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to upload playlist to IPFS: %v", err)
		}
		playlistTranscode.FileCID = cid
	} else {
		s3Key := fmt.Sprintf("transcodes/hls/%s/playlist.m3u8", resolution)
		s3URL, err := s.s3.UploadFile(outputPath, s3Key)
		if err != nil {
			return nil, fmt.Errorf("failed to upload playlist to S3: %v", err)
		}
		playlistTranscode.FilePath = s3URL
	}

	// Process segments
	var segments []TranscodeSegment
	segmentFiles, err := filepath.Glob(filepath.Join(outputDir, "segment_*.ts"))
	if err != nil {
		return nil, fmt.Errorf("failed to list segment files: %v", err)
	}

	for i, segmentPath := range segmentFiles {
		segment := TranscodeSegment{
			Sequence:    i + 1,
			StorageType: storageType,
			CreatedAt:   time.Now(),
		}

		if storageType == "ipfs" {
			cid, err := s.ipfs.UploadFile(segmentPath)
			if err != nil {
				return nil, fmt.Errorf("failed to upload segment to IPFS: %v", err)
			}
			segment.FileCID = cid
		} else {
			s3Key := fmt.Sprintf("transcodes/hls/%s/segment_%03d.ts", resolution, i+1)
			s3URL, err := s.s3.UploadFile(segmentPath, s3Key)
			if err != nil {
				return nil, fmt.Errorf("failed to upload segment to S3: %v", err)
			}
			segment.FilePath = s3URL
		}

		segments = append(segments, segment)
	}

	return &HLSResult{
		Transcode: playlistTranscode,
		Segments:  segments,
	}, nil
}

func (s *Service) processMP4Format(inputPath, resolution, storageType string) (Transcode, error) {
	// Create output filename
	outputPath := fmt.Sprintf("temp_mp4_%s_%s_%s.mp4",
		filepath.Base(inputPath),
		resolution,
		uuid.New().String())

	// Prepare ffmpeg command for MP4
	cmd := exec.Command(
		s.config.Ffmpeg.Path,
		"-i", inputPath,
		"-vf", fmt.Sprintf("scale=-2:%s", resolution),
		"-c:v", s.config.Ffmpeg.VideoCodec,
		"-c:a", s.config.Ffmpeg.AudioCodec,
		"-preset", s.config.Ffmpeg.Preset,
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return Transcode{}, fmt.Errorf("ffmpeg command failed: %v", err)
	}
	defer os.Remove(outputPath)

	transcode := Transcode{
		Format:      "mp4",
		Resolution:  resolution,
		StorageType: storageType,
		Type:        "video",
		CreatedAt:   time.Now(),
	}

	if storageType == "ipfs" {
		cid, err := s.ipfs.UploadFile(outputPath)
		if err != nil {
			return Transcode{}, fmt.Errorf("failed to upload MP4 to IPFS: %v", err)
		}
		transcode.FileCID = cid
	} else {
		s3Key := fmt.Sprintf("transcodes/mp4/%s/video.mp4", resolution)
		s3URL, err := s.s3.UploadFile(outputPath, s3Key)
		if err != nil {
			return Transcode{}, fmt.Errorf("failed to upload MP4 to S3: %v", err)
		}
		transcode.FilePath = s3URL
	}

	return transcode, nil
}
