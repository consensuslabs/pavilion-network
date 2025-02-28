package unit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/consensuslabs/pavilion-network/backend/internal/video/tests/helpers"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetVideo_Success(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	// Create a test UUID
	videoID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/video/%s", videoID), nil)
	c.Params = []gin.Param{{Key: "id", Value: videoID.String()}}

	// Add authentication header
	helpers.AuthenticateRequest(c)

	// Setup mock dependencies
	mockVideoService, mockResponseHandler, mockLogger, app := helpers.SetupMockDependencies()

	// Create test video
	testVideo := &video.Video{
		ID:          videoID,
		FileID:      "test-file-id",
		Title:       "Test Video",
		Description: "This is a test video",
		StoragePath: "videos/test.mp4",
		IPFSCID:     "test-ipfs-cid",
		FileSize:    1024,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Upload: &video.VideoUpload{
			ID:        uuid.New(),
			VideoID:   videoID,
			StartTime: time.Now(),
			EndTime:   nil,
			Status:    video.UploadStatusCompleted,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Transcodes: []video.Transcode{
			{
				ID:        uuid.New(),
				VideoID:   videoID,
				Format:    "mp4",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Segments: []video.TranscodeSegment{
					{
						ID:          uuid.New(),
						StoragePath: "videos/test-720p.mp4",
						IPFSCID:     "test-ipfs-cid-720p",
						Duration:    120,
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					},
				},
			},
		},
	}

	// Set up mock expectations
	mockVideoService.On("GetVideo", videoID).Return(testVideo, nil)
	mockLogger.On("LogInfo", "Video details retrieved successfully", mock.Anything).Return()
	mockResponseHandler.On("SuccessResponse", mock.Anything, mock.Anything, "Video details retrieved successfully").Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.GetVideo(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 200, w.Code, "Should return HTTP 200 OK")
}

func TestGetVideo_InvalidID(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	invalidID := "invalid-uuid"
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/video/%s", invalidID), nil)
	c.Params = []gin.Param{{Key: "id", Value: invalidID}}

	// Add authentication header
	helpers.AuthenticateRequest(c)

	// Setup mock dependencies
	mockVideoService, mockResponseHandler, mockLogger, app := helpers.SetupMockDependencies()

	// Set up mock expectations for invalid ID case
	mockLogger.On("LogInfo", "Invalid video ID format", mock.Anything).Return()
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusBadRequest, "INVALID_ID", "Invalid video ID format", mock.Anything).Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.GetVideo(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 400, w.Code, "Should return HTTP 400 Bad Request")
}

func TestGetVideo_NotFound(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	// Use a valid UUID that doesn't exist in the database
	videoID := uuid.New()
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/video/%s", videoID), nil)
	c.Params = []gin.Param{{Key: "id", Value: videoID.String()}}

	// Add authentication header
	helpers.AuthenticateRequest(c)

	// Setup mock dependencies
	mockVideoService, mockResponseHandler, mockLogger, app := helpers.SetupMockDependencies()

	// Set up mock expectations for not found case
	notFoundErr := fmt.Errorf("video not found: %s", videoID)
	mockVideoService.On("GetVideo", videoID).Return(nil, notFoundErr)
	mockLogger.On("LogInfo", "Video not found or has been deleted", mock.Anything).Return()
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusNotFound, "VIDEO_NOT_FOUND", fmt.Sprintf("video not found: %s", videoID), nil).Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.GetVideo(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 404, w.Code, "Should return HTTP 404 Not Found")
}

func TestGetVideo_DatabaseError(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	videoID := uuid.New()
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/video/%s", videoID), nil)
	c.Params = []gin.Param{{Key: "id", Value: videoID.String()}}

	// Add authentication header
	helpers.AuthenticateRequest(c)

	// Setup mock dependencies
	mockVideoService, mockResponseHandler, mockLogger, app := helpers.SetupMockDependencies()

	// Set up mock expectations for database error case
	dbErr := fmt.Errorf("database error")
	mockVideoService.On("GetVideo", videoID).Return(nil, dbErr)
	mockLogger.On("LogInfo", "Failed to get video details", mock.Anything).Return()
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve video details", dbErr).Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.GetVideo(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 500, w.Code, "Should return HTTP 500 Internal Server Error")
}
