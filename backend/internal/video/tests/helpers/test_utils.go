package helpers

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/consensuslabs/pavilion-network/backend/internal/video/tests/mocks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateTestFile creates a temporary file with specified content for testing
func CreateTestFile(t *testing.T, content []byte) (*os.File, error) {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test-upload-*.mp4")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	})

	if _, err := tmpFile.Write(content); err != nil {
		return nil, err
	}

	if _, err := tmpFile.Seek(0, 0); err != nil {
		return nil, err
	}

	return tmpFile, nil
}

// CreateMultipartRequest creates a multipart HTTP request with the specified file and params
func CreateMultipartRequest(t *testing.T, filePath string, params map[string]string) (*http.Request, error) {
	t.Helper()
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	for key, val := range params {
		if err := writer.WriteField(key, val); err != nil {
			return nil, err
		}
	}

	// Add file
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "/upload", body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

// VideoConfigForTest returns a test video configuration
func VideoConfigForTest() *video.Config {
	config := &video.Config{}
	config.Video.MaxFileSize = 10 * 1024 * 1024 // 10MB
	config.Video.MinTitleLength = 3
	config.Video.MaxTitleLength = 100
	config.Video.MaxDescLength = 1000
	config.Video.AllowedFormats = []string{"mp4", "mov", "avi"}

	// Set FFmpeg config
	config.FFmpeg.Path = "/usr/bin/ffmpeg"
	config.FFmpeg.ProbePath = "/usr/bin/ffprobe"
	config.FFmpeg.VideoCodec = "h264"
	config.FFmpeg.AudioCodec = "aac"
	config.FFmpeg.Preset = "medium"
	config.FFmpeg.OutputPath = "/tmp/videos"
	config.FFmpeg.Resolutions = []string{"720p", "480p", "360p"}

	return config
}

// SetupTestVideos creates test video objects for testing
func SetupTestVideos(count int) []video.Video {
	videos := make([]video.Video, count)
	for i := 0; i < count; i++ {
		videos[i] = video.Video{
			ID:          uuid.New(),
			FileID:      uuid.New().String(),
			Title:       "Test Video " + string(rune(i+1)),
			Description: "Test Description " + string(rune(i+1)),
			StoragePath: "/test/path/" + uuid.New().String(),
			IPFSCID:     "testcid" + string(rune(i+1)),
			Checksum:    "checksum" + string(rune(i+1)),
			FileSize:    int64(1024 * (i + 1)),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}
	return videos
}

// SetupTestUploads creates test upload objects for testing
func SetupTestUploads(videos []video.Video) []video.VideoUpload {
	uploads := make([]video.VideoUpload, len(videos))
	for i, v := range videos {
		now := time.Now()
		uploads[i] = video.VideoUpload{
			ID:        uuid.New(),
			VideoID:   v.ID,
			Status:    video.UploadStatusCompleted,
			StartTime: now.Add(-time.Minute),
			EndTime:   &now,
			CreatedAt: now.Add(-time.Minute),
			UpdatedAt: now,
		}
	}
	return uploads
}

// SetupTestGinEngine returns a configured gin engine for testing
func SetupTestGinEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())
	return router
}

// SetupMockServices returns mock services for testing
func SetupMockServices() (*mocks.MockVideoService, *mocks.MockIPFSService, *mocks.MockStorageService, *mocks.MockFFmpegService, *mocks.MockTempFileManager, *mocks.MockResponseHandler, *mocks.MockLogger) {
	mockVideoService := new(mocks.MockVideoService)
	mockIPFSService := new(mocks.MockIPFSService)
	mockStorage := new(mocks.MockStorageService)
	mockFfmpeg := new(mocks.MockFFmpegService)
	mockTempManager := new(mocks.MockTempFileManager)
	mockResponseHandler := new(mocks.MockResponseHandler)
	mockLogger := new(mocks.MockLogger)

	return mockVideoService, mockIPFSService, mockStorage, mockFfmpeg, mockTempManager, mockResponseHandler, mockLogger
}

// SetupTestContext creates a Gin test context with a recorder for HTTP responses
func SetupTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "test-request-id")
	return c, w
}

// SetupMockDependencies creates and returns all mock dependencies for testing
func SetupMockDependencies() (*mocks.MockVideoService, *mocks.MockResponseHandler, *mocks.MockLogger, *video.App) {
	mockVideoService := new(mocks.MockVideoService)
	mockIPFSService := new(mocks.MockIPFSService)
	mockResponseHandler := new(mocks.MockResponseHandler)
	mockLogger := new(mocks.MockLogger)

	app := &video.App{
		Config:          &video.Config{},
		Video:           mockVideoService,
		IPFS:            mockIPFSService,
		Logger:          mockLogger,
		ResponseHandler: mockResponseHandler,
	}

	return mockVideoService, mockResponseHandler, mockLogger, app
}

// AuthenticateRequest adds authentication headers to a request
func AuthenticateRequest(c *gin.Context) {
	c.Request.Header.Set("Authorization", "Bearer test-token")
}
