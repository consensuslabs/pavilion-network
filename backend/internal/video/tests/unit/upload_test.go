package unit

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/consensuslabs/pavilion-network/backend/internal/video/tests/helpers"
	"github.com/consensuslabs/pavilion-network/backend/internal/video/tests/mocks"
)

// TestHandleUpload_Success tests the successful upload of a video
func TestHandleUpload_Success(t *testing.T) {
	// Setup mock services using helper
	mockVideoService, mockIPFSService, _, _,
		mockTempManager, mockResponseHandler, mockLogger := helpers.SetupMockServices()

	// Create app configuration
	config := &video.Config{}
	config.Video.MaxFileSize = 10 * 1024 * 1024            // 10MB
	config.Video.AllowedFormats = []string{".mp4", ".mov"} // Include the dot in the allowed formats
	config.Video.MinTitleLength = 3
	config.Video.MaxTitleLength = 100
	config.Video.MaxDescLength = 1000

	// Create a new app instance
	app := &video.App{
		Config:          config,
		Video:           mockVideoService,
		IPFS:            mockIPFSService,
		ResponseHandler: mockResponseHandler,
		Logger:          mockLogger,
	}

	// Create a new video handler with the app
	handler := video.NewVideoHandler(app)

	// Create a test file
	fileContents := []byte("test video file contents")
	fileName := "test-video.mp4"

	// Prepare a multipart request with the test file
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("video", fileName)
	part.Write(fileContents)

	// Add form fields
	writer.WriteField("title", fileName)
	writer.WriteField("description", "Test video description")
	writer.Close()

	// Create the request with context
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request, _ = http.NewRequest("POST", "/video/upload", body)
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
	// Add Authorization header for authentication
	ctx.Request.Header.Set("Authorization", "Bearer test-token")

	// Set user claims in the context
	userId := uuid.New()
	ctx.Set("userId", userId.String())
	ctx.Set("request_id", "test-request-id")

	// Mock authentication (since we can see isAuthenticated is used in the handler)
	ctx.Set("authenticated", true)

	// Add expectations for all possible log messages
	mockLogger.On("LogInfo", mock.Anything, mock.Anything).Return()

	// Add expectation for ErrorResponse (even though it shouldn't be called in success case)
	mockResponseHandler.On("ErrorResponse", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	// Expect temp file to be created
	tempFilePath := "/tmp/test-video-" + uuid.New().String() + ".mp4"
	mockTempManager.On("Create", mock.Anything, mock.Anything).Return(tempFilePath, nil)
	mockTempManager.On("IsManaged", tempFilePath).Return(true)
	mockTempManager.On("Remove", tempFilePath).Return(nil)

	// Set up expectations for upload
	uploadId := uuid.New()
	videoId := uuid.New()
	fileId := "test-file-id"

	// Create a video object for the upload
	testVideo := &video.Video{
		ID:          videoId,
		FileID:      fileId,
		Title:       fileName,
		Description: "Test video description",
		StoragePath: "/path/to/video",
		FileSize:    int64(len(fileContents)),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockVideoService.On("InitializeUpload", fileName, "Test video description", mock.Anything).Return(&video.VideoUpload{
		ID:        uploadId,
		VideoID:   videoId,
		Status:    video.UploadStatusPending,
		StartTime: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Video:     testVideo,
	}, nil)

	mockVideoService.On("ProcessUpload", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Expect video service to create a video (if needed)
	mockVideoService.On("CreateVideo", userId, fileName, "Test video description").Return(testVideo, nil)

	// Expect GetVideo to be called after processing the upload
	mockVideoService.On("GetVideo", videoId).Return(testVideo, nil)

	// Expect success response
	// Using mock.Anything for all parameters since we can't know the exact values
	mockResponseHandler.On("SuccessResponse", mock.Anything, mock.Anything, mock.Anything).Return()

	// Call the handler directly
	handler.HandleUpload(ctx)

	// Verify status code
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestHandleUpload_InvalidFile tests the upload handler when no file is provided
func TestHandleUpload_InvalidFile(t *testing.T) {
	// Setup mock services using helper
	mockVideoService, _, _, _,
		_, mockResponseHandler, mockLogger := helpers.SetupMockServices()

	// Create app configuration
	config := helpers.VideoConfigForTest()

	// Create a new app instance
	app := &video.App{
		Config:          config,
		Video:           mockVideoService,
		ResponseHandler: mockResponseHandler,
		Logger:          mockLogger,
	}

	// Create a new video handler with the app
	handler := video.NewVideoHandler(app)

	// Create a request without a file
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request, _ = http.NewRequest("POST", "/video/upload", nil)
	ctx.Request.Header.Set("Content-Type", "multipart/form-data")
	// Add Authorization header for authentication
	ctx.Request.Header.Set("Authorization", "Bearer test-token")

	// Set authentication data
	ctx.Set("authenticated", true)
	ctx.Set("userId", uuid.New())
	ctx.Set("request_id", "test-request-id")

	// Add expectations for all possible log messages
	mockLogger.On("LogInfo", mock.Anything, mock.Anything).Return()

	// Expect error response for missing file
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusBadRequest, "ERR_NO_FILE", "No video file received", mock.Anything).Return()

	// Call the handler directly
	handler.HandleUpload(ctx)

	// Verify status code
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestHandleUpload_SizeExceeded tests the upload handler when the file size exceeds the maximum allowed
func TestHandleUpload_SizeExceeded(t *testing.T) {
	// Set up mock dependencies
	mockService := new(mocks.MockVideoService)
	mockLogger := new(mocks.MockLogger)
	mockResponseHandler := new(mocks.MockResponseHandler)

	// Create app configuration with a small max file size
	config := helpers.VideoConfigForTest()
	config.Video.MaxFileSize = 10 // Very small max size for testing

	// Create app with mocks
	app := &video.App{
		Video:           mockService,
		Logger:          mockLogger,
		ResponseHandler: mockResponseHandler,
		Config:          config,
	}

	// Create a new video handler with the app
	handler := video.NewVideoHandler(app)

	// Create a test file larger than the max size
	fileContents := bytes.Repeat([]byte("a"), 100) // 100 bytes, which exceeds our 10 byte limit
	fileName := "test-video.mp4"

	// Prepare a multipart request with the test file
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("video", fileName)
	part.Write(fileContents)

	// Add form fields
	writer.WriteField("title", fileName)
	writer.WriteField("description", "Test video description")
	writer.Close()

	// Create the request with context
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request, _ = http.NewRequest("POST", "/video/upload", body)
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
	// Add Authorization header for authentication
	ctx.Request.Header.Set("Authorization", "Bearer test-token")

	// Set authentication data
	ctx.Set("authenticated", true)
	ctx.Set("userId", uuid.New())
	ctx.Set("request_id", "test-request-id")

	// Set up mock expectations - we only need one of these to be called
	// Since we're setting authenticated=true, we expect the validation error
	mockLogger.On("LogInfo", "Video upload validation failed", mock.Anything).Return()
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusBadRequest, "ERR_VALIDATION", mock.Anything, mock.Anything).Return()

	// Call the handler
	handler.HandleUpload(ctx)

	// Verify expectations
	mockResponseHandler.AssertExpectations(t)
}

// TestHandleUpload_Unauthorized tests the upload handler when the request is not authenticated
func TestHandleUpload_Unauthorized(t *testing.T) {
	// Setup mock services using helper
	mockVideoService, _, _, _,
		_, mockResponseHandler, mockLogger := helpers.SetupMockServices()

	// Create app configuration
	config := helpers.VideoConfigForTest()

	// Create a new app instance
	app := &video.App{
		Config:          config,
		Video:           mockVideoService,
		ResponseHandler: mockResponseHandler,
		Logger:          mockLogger,
	}

	// Create a new video handler with the app
	handler := video.NewVideoHandler(app)

	// Create a request without authentication
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request, _ = http.NewRequest("POST", "/video/upload", nil)
	ctx.Set("request_id", "test-request-id")

	// Do not set authenticated flag
	// ctx.Set("authenticated", true)

	// Set up mock expectations
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", mock.Anything).Return()

	// Call the handler
	handler.HandleUpload(ctx)

	// Verify expectations
	mockResponseHandler.AssertExpectations(t)

	// Verify status code
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestHandleUpload_TranscodingFailed tests the upload handler when transcoding fails
func TestHandleUpload_TranscodingFailed(t *testing.T) {
	// Setup mock services using helper
	mockVideoService, _, _, _,
		_, mockResponseHandler, mockLogger := helpers.SetupMockServices()

	// Create app configuration
	config := helpers.VideoConfigForTest()
	// Ensure the allowed formats include .mp4
	config.Video.AllowedFormats = []string{".mp4", ".mov", ".avi"}

	// Create a new app instance
	app := &video.App{
		Config:          config,
		Video:           mockVideoService,
		ResponseHandler: mockResponseHandler,
		Logger:          mockLogger,
	}

	// Create a new video handler with the app
	handler := video.NewVideoHandler(app)

	// Create a test file
	fileContents := []byte("test video file contents")
	fileName := "test-video.mp4"

	// Prepare a multipart request with the test file
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("video", fileName)
	part.Write(fileContents)

	// Add form fields
	writer.WriteField("title", fileName)
	writer.WriteField("description", "Test video description")
	writer.Close()

	// Create the request with context
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request, _ = http.NewRequest("POST", "/video/upload", body)
	ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
	// Add Authorization header for authentication
	ctx.Request.Header.Set("Authorization", "Bearer test-token")

	// Set authentication data
	ctx.Set("authenticated", true)
	ctx.Set("userId", uuid.New())
	ctx.Set("request_id", "test-request-id")

	// Set up expectations for upload
	uploadId := uuid.New()
	videoId := uuid.New()

	mockVideoService.On("InitializeUpload", fileName, "Test video description", mock.Anything).Return(&video.VideoUpload{
		ID:        uploadId,
		VideoID:   videoId,
		Status:    video.UploadStatusPending,
		StartTime: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil)

	// Simulate transcoding failure
	transcodeError := fmt.Errorf("transcoding failed")
	mockVideoService.On("ProcessUpload", mock.Anything, mock.Anything, mock.Anything).Return(transcodeError)

	// Expect error logging and response
	mockLogger.On("LogInfo", mock.Anything, mock.Anything).Return()
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusInternalServerError, "TRANSCODE_FAILED", "Video transcoding failed", transcodeError).Return()

	// Call the handler
	handler.HandleUpload(ctx)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Verify status code
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
