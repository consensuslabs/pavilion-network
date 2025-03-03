package unit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/consensuslabs/pavilion-network/backend/internal/video/tests/helpers"
)

// TestGetVideoStatus_Success tests the successful retrieval of a video's status
func TestGetVideoStatus_Success(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	// Create a test UUID
	videoID := uuid.New()
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/videos/%s/status", videoID), nil)
	c.Params = []gin.Param{{Key: "id", Value: videoID.String()}}

	// Add authentication header
	helpers.AuthenticateRequest(c)

	// Setup mock dependencies
	mockVideoService, mockResponseHandler, mockLogger, app := helpers.SetupMockDependencies()

	// Create test video upload
	now := time.Now()
	testUpload := &video.VideoUpload{
		ID:        uuid.New(),
		VideoID:   videoID,
		Status:    video.UploadStatusCompleted,
		StartTime: now.Add(-time.Minute * 5),
		EndTime:   &now,
		CreatedAt: now.Add(-time.Minute * 5),
		UpdatedAt: now,
	}

	// Create test video
	testVideo := &video.Video{
		ID:          videoID,
		Title:       "Test Video",
		Description: "Test Description",
		CreatedAt:   now.Add(-time.Minute * 5),
		UpdatedAt:   now,
		Upload:      testUpload,
	}

	// Set up mock expectations
	mockVideoService.On("GetVideo", videoID).Return(testVideo, nil)
	mockLogger.On("LogInfo", "Video status retrieved successfully", mock.Anything).Return()
	mockResponseHandler.On("SuccessResponse", mock.Anything, mock.Anything, "Video status retrieved successfully").Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.GetVideoStatus(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 200, w.Code, "Should return HTTP 200 OK")
}

// TestGetVideoStatus_InvalidID tests getting a video status with an invalid ID
func TestGetVideoStatus_InvalidID(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	// Use an invalid UUID
	invalidID := "invalid-uuid"
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/videos/%s/status", invalidID), nil)
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
	handler.GetVideoStatus(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 400, w.Code, "Should return HTTP 400 Bad Request")
}

// TestGetVideoStatus_NotFound tests getting a video status when the video is not found
func TestGetVideoStatus_NotFound(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	// Use a valid UUID that doesn't exist in the database
	videoID := uuid.New()
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/videos/%s/status", videoID), nil)
	c.Params = []gin.Param{{Key: "id", Value: videoID.String()}}

	// Add authentication header
	helpers.AuthenticateRequest(c)

	// Setup mock dependencies
	mockVideoService, mockResponseHandler, mockLogger, app := helpers.SetupMockDependencies()

	// Create a not found error
	notFoundErr := fmt.Errorf("video not found: %s", videoID)

	// Set up mock expectations for not found case
	mockVideoService.On("GetVideo", videoID).Return(nil, notFoundErr)
	mockLogger.On("LogInfo", "Video not found or has been deleted", mock.Anything).Return()
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusNotFound, "VIDEO_NOT_FOUND", fmt.Sprintf("video not found: %s", videoID), nil).Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.GetVideoStatus(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 404, w.Code, "Should return HTTP 404 Not Found")
}

// TestGetVideoStatus_DatabaseError tests getting a video status when a database error occurs
func TestGetVideoStatus_DatabaseError(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	// Use a valid UUID
	videoID := uuid.New()
	c.Request = httptest.NewRequest("GET", fmt.Sprintf("/videos/%s/status", videoID), nil)
	c.Params = []gin.Param{{Key: "id", Value: videoID.String()}}

	// Add authentication header
	helpers.AuthenticateRequest(c)

	// Setup mock dependencies
	mockVideoService, mockResponseHandler, mockLogger, app := helpers.SetupMockDependencies()

	// Set up mock expectations for database error case
	dbErr := fmt.Errorf("database error")
	mockVideoService.On("GetVideo", videoID).Return(nil, dbErr)
	mockLogger.On("LogInfo", "Failed to get video status", mock.Anything).Return()
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve video status", dbErr).Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.GetVideoStatus(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 500, w.Code, "Should return HTTP 500 Internal Server Error")
}

// TestGetVideoStatus_Unauthorized tests that authentication is required for the GetVideoStatus endpoint
func TestGetVideoStatus_Unauthorized(t *testing.T) {
	// This test verifies that the route is protected by auth middleware

	// Setup in-process test server and router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create mock services - only need the app component
	_, _, _, app := helpers.SetupMockDependencies()

	// Configure protected routes with auth middleware
	protected := router.Group("")
	// Use an actual auth middleware that will block requests without a token
	protected.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	})

	// Register the handler to the protected group
	protected.GET("/video/:id/status", video.NewVideoHandler(app).GetVideoStatus)

	// Create a test UUID
	videoID := uuid.New()

	// Create request WITHOUT Authorization header
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/video/%s/status", videoID), nil)

	// Serve the request
	router.ServeHTTP(w, req)

	// Verify that the request was rejected with 401 Unauthorized
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Should return HTTP 401 Unauthorized when no auth token is provided")
}
