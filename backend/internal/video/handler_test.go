package video

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// VideoConfigTest represents the video configuration for testing
type VideoConfigTest struct {
	MaxFileSize    int64    `yaml:"max_file_size"`
	MinTitleLength int      `yaml:"min_title_length"`
	MaxTitleLength int      `yaml:"max_title_length"`
	MaxDescLength  int      `yaml:"max_desc_length"`
	AllowedFormats []string `yaml:"allowed_formats"`
}

// MockVideoService is a mock implementation of VideoService
type MockVideoService struct {
	mock.Mock
}

func (m *MockVideoService) InitializeUpload(title, description string, size int64) (*VideoUpload, error) {
	args := m.Called(title, description, size)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*VideoUpload), args.Error(1)
}

func (m *MockVideoService) ProcessUpload(upload *VideoUpload, file multipart.File, header *multipart.FileHeader) error {
	args := m.Called(upload, file, header)
	return args.Error(0)
}

func (m *MockVideoService) GetVideo(videoID int64) (*Video, error) {
	args := m.Called(videoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Video), args.Error(1)
}

func (m *MockVideoService) ListVideos(page, limit int) ([]Video, error) {
	args := m.Called(page, limit)
	return args.Get(0).([]Video), args.Error(1)
}

func (m *MockVideoService) DeleteVideo(videoID int64) error {
	args := m.Called(videoID)
	return args.Error(0)
}

func (m *MockVideoService) UpdateVideo(videoID int64, title, description string) error {
	args := m.Called(videoID, title, description)
	return args.Error(0)
}

// MockIPFSService is a mock implementation of IPFSService
type MockIPFSService struct {
	mock.Mock
}

func (m *MockIPFSService) UploadFileStream(reader io.Reader) (string, error) {
	args := m.Called(reader)
	return args.String(0), args.Error(1)
}

func (m *MockIPFSService) DownloadFile(cid string) (string, error) {
	args := m.Called(cid)
	return args.String(0), args.Error(1)
}

// MockResponseHandler is a mock implementation of ResponseHandler
type MockResponseHandler struct {
	mock.Mock
}

func (m *MockResponseHandler) SuccessResponse(c *gin.Context, data interface{}, message string) {
	m.Called(c, data, message)
}

func (m *MockResponseHandler) ErrorResponse(c *gin.Context, status int, code, message string, err error) {
	m.Called(c, status, code, message, err)
}

// MockLogger is a mock implementation of Logger
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) LogInfo(message string, fields map[string]interface{}) {
	m.Called(message, fields)
}

func (m *MockLogger) LogError(message string, fields map[string]interface{}) {
	m.Called(message, fields)
}

// Helper function to create a test file
func createTestFile(t *testing.T, content []byte) (*os.File, error) {
	tmpFile, err := os.CreateTemp("", "test-video-*.mp4")
	if err != nil {
		return nil, err
	}
	if _, err := tmpFile.Write(content); err != nil {
		return nil, err
	}
	if err := tmpFile.Close(); err != nil {
		return nil, err
	}
	return tmpFile, nil
}

// Helper function to create a multipart request
func createMultipartRequest(t *testing.T, filePath string, params map[string]string) (*http.Request, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("video", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	if _, err := part.Write(fileContent); err != nil {
		return nil, err
	}

	for key, value := range params {
		if err := writer.WriteField(key, value); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req := httptest.NewRequest("POST", "/video/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer test-token")
	return req, nil
}

func TestHandleUpload_Success(t *testing.T) {
	// Create test dependencies
	mockVideoService := new(MockVideoService)
	mockIPFSService := new(MockIPFSService)
	mockResponseHandler := new(MockResponseHandler)
	mockLogger := new(MockLogger)

	app := &App{
		Config: &Config{
			Video: VideoConfigTest{
				MaxFileSize:    500 * 1024 * 1024, // 500MB
				MinTitleLength: 3,
				MaxTitleLength: 100,
				MaxDescLength:  1000,
				AllowedFormats: []string{".mp4", ".mov"},
			},
		},
		Video:           mockVideoService,
		IPFS:            mockIPFSService,
		Logger:          mockLogger,
		ResponseHandler: mockResponseHandler,
	}

	handler := NewVideoHandler(app)

	// Create a test file
	testContent := []byte("test video content")
	tmpFile, err := createTestFile(t, testContent)
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Create test request
	params := map[string]string{
		"title":       "Test Video",
		"description": "Test Description",
	}
	req, err := createMultipartRequest(t, tmpFile.Name(), params)
	assert.NoError(t, err)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Set up mock expectations
	expectedUpload := &VideoUpload{
		VideoID:      time.Now().UnixNano(),
		Title:        "Test Video",
		Description:  "Test Description",
		UploadStatus: UploadStatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mockVideoService.On("InitializeUpload", "Test Video", "Test Description", int64(len(testContent))).Return(expectedUpload, nil)
	mockVideoService.On("ProcessUpload", expectedUpload, mock.Anything, mock.Anything).Return(nil)
	mockVideoService.On("GetVideo", expectedUpload.VideoID).Return(&Video{
		ID:           expectedUpload.VideoID,
		Title:        expectedUpload.Title,
		Description:  expectedUpload.Description,
		UploadStatus: expectedUpload.UploadStatus,
		Transcodes:   []Transcode{},
	}, nil)
	mockLogger.On("LogInfo", "Video upload completed successfully", mock.MatchedBy(func(fields map[string]interface{}) bool {
		return fields["video_id"] == expectedUpload.VideoID &&
			fields["file_path"] == expectedUpload.FilePath &&
			fields["request_id"] != nil
	})).Return()
	mockResponseHandler.On("SuccessResponse", mock.Anything, mock.Anything, mock.Anything).Return()

	// Call the handler
	handler.HandleUpload(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

func TestHandleUpload_InvalidFile(t *testing.T) {
	// Create test dependencies
	mockVideoService := new(MockVideoService)
	mockIPFSService := new(MockIPFSService)
	mockResponseHandler := new(MockResponseHandler)
	mockLogger := new(MockLogger)

	app := &App{
		Config: &Config{
			Video: VideoConfigTest{
				MaxFileSize:    500 * 1024 * 1024,
				MinTitleLength: 3,
				MaxTitleLength: 100,
				MaxDescLength:  1000,
				AllowedFormats: []string{".mp4", ".mov"},
			},
		},
		Video:           mockVideoService,
		IPFS:            mockIPFSService,
		Logger:          mockLogger,
		ResponseHandler: mockResponseHandler,
	}

	handler := NewVideoHandler(app)

	// Create test request without file
	req := httptest.NewRequest("POST", "/video/upload", nil)
	req.Header.Set("Content-Type", "multipart/form-data")
	req.Header.Set("Authorization", "Bearer test-token")

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Set up mock expectations
	mockLogger.On("LogInfo", "No video file received", mock.Anything).Return()
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusBadRequest, "ERR_NO_FILE", "No video file received", mock.Anything).Return()

	// Call the handler
	handler.HandleUpload(c)

	// Verify expectations
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)
}

func TestHandleUpload_SizeExceeded(t *testing.T) {
	// Create test dependencies
	mockVideoService := new(MockVideoService)
	mockIPFSService := new(MockIPFSService)
	mockResponseHandler := new(MockResponseHandler)
	mockLogger := new(MockLogger)

	maxSize := int64(10) // Small max size for testing
	app := &App{
		Config: &Config{
			Video: VideoConfigTest{
				MaxFileSize:    maxSize,
				MinTitleLength: 3,
				MaxTitleLength: 100,
				MaxDescLength:  1000,
				AllowedFormats: []string{".mp4", ".mov"},
			},
		},
		Video:           mockVideoService,
		IPFS:            mockIPFSService,
		Logger:          mockLogger,
		ResponseHandler: mockResponseHandler,
	}

	handler := NewVideoHandler(app)

	// Create a test file larger than maxSize
	testContent := bytes.Repeat([]byte("a"), int(maxSize+1))
	tmpFile, err := createTestFile(t, testContent)
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Create test request
	params := map[string]string{
		"title":       "Test Video",
		"description": "Test Description",
	}
	req, err := createMultipartRequest(t, tmpFile.Name(), params)
	assert.NoError(t, err)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Set up mock expectations
	mockLogger.On("LogInfo", "Video upload validation failed", mock.Anything).Return()
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusBadRequest, "ERR_VALIDATION", mock.Anything, mock.Anything).Return()

	// Call the handler
	handler.HandleUpload(c)

	// Verify expectations
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)
}

func TestHandleUpload_Unauthorized(t *testing.T) {
	// Create test dependencies
	mockVideoService := new(MockVideoService)
	mockIPFSService := new(MockIPFSService)
	mockResponseHandler := new(MockResponseHandler)
	mockLogger := new(MockLogger)

	app := &App{
		Config: &Config{
			Video: VideoConfigTest{
				MaxFileSize:    500 * 1024 * 1024,
				MinTitleLength: 3,
				MaxTitleLength: 100,
				MaxDescLength:  1000,
				AllowedFormats: []string{".mp4", ".mov"},
			},
		},
		Video:           mockVideoService,
		IPFS:            mockIPFSService,
		Logger:          mockLogger,
		ResponseHandler: mockResponseHandler,
	}

	handler := NewVideoHandler(app)

	// Create test request without authentication
	req := httptest.NewRequest("POST", "/video/upload", nil)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Set up mock expectations
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", mock.Anything).Return()

	// Call the handler
	handler.HandleUpload(c)

	// Verify expectations
	mockResponseHandler.AssertExpectations(t)
}

func TestHandleUpload_TranscodingFailed(t *testing.T) {
	// Create test dependencies
	mockVideoService := new(MockVideoService)
	mockIPFSService := new(MockIPFSService)
	mockResponseHandler := new(MockResponseHandler)
	mockLogger := new(MockLogger)

	app := &App{
		Config: &Config{
			Video: VideoConfigTest{
				MaxFileSize:    500 * 1024 * 1024,
				MinTitleLength: 3,
				MaxTitleLength: 100,
				MaxDescLength:  1000,
				AllowedFormats: []string{".mp4", ".mov"},
			},
		},
		Video:           mockVideoService,
		IPFS:            mockIPFSService,
		Logger:          mockLogger,
		ResponseHandler: mockResponseHandler,
	}

	handler := NewVideoHandler(app)

	// Create a test file
	testContent := []byte("test video content")
	tmpFile, err := createTestFile(t, testContent)
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Create test request
	params := map[string]string{
		"title":       "Test Video",
		"description": "Test Description",
	}
	req, err := createMultipartRequest(t, tmpFile.Name(), params)
	assert.NoError(t, err)

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Set up mock expectations
	expectedUpload := &VideoUpload{
		VideoID:      time.Now().UnixNano(),
		Title:        "Test Video",
		Description:  "Test Description",
		UploadStatus: UploadStatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	transcodeError := fmt.Errorf("transcoding failed")
	mockVideoService.On("InitializeUpload", "Test Video", "Test Description", int64(len(testContent))).Return(expectedUpload, nil)
	mockVideoService.On("ProcessUpload", expectedUpload, mock.Anything, mock.Anything).Return(transcodeError)
	mockLogger.On("LogInfo", mock.Anything, mock.Anything).Return()
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusInternalServerError, "TRANSCODE_FAILED", "Video transcoding failed", transcodeError).Return()

	// Call the handler
	handler.HandleUpload(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)
}
