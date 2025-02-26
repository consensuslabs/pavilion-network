package unit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/consensuslabs/pavilion-network/backend/internal/video/tests/helpers"
)

// TestListVideos_Success tests the successful listing of videos
func TestListVideos_Success(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	// Create test request with query parameters
	c.Request = httptest.NewRequest("GET", "/videos?page=1&limit=10", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token")
	c.Set("authenticated", true)
	c.Set("userId", uuid.New())
	c.Set("request_id", "test-request-id")

	// Add query parameters
	c.Request.URL.RawQuery = "page=1&limit=10"

	// Setup mock dependencies
	mockVideoService, mockResponseHandler, mockLogger, app := helpers.SetupMockDependencies()

	// Create test videos
	testVideos := helpers.SetupTestVideos(3)

	// Set up mock expectations
	mockVideoService.On("ListVideos", 1, 10).Return(testVideos, nil)
	mockLogger.On("LogInfo", "Videos retrieved successfully", mock.Anything).Return()
	mockResponseHandler.On("SuccessResponse", mock.Anything, mock.Anything, "").Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.ListVideos(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 200, w.Code, "Should return HTTP 200 OK")
}

// TestListVideos_InvalidParameters tests listing videos with invalid parameters
func TestListVideos_InvalidParameters(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	// Create test request with invalid query parameters
	c.Request = httptest.NewRequest("GET", "/videos?page=invalid&limit=invalid", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token")
	c.Set("authenticated", true)
	c.Set("userId", uuid.New())
	c.Set("request_id", "test-request-id")

	// Add invalid query parameters
	c.Request.URL.RawQuery = "page=invalid&limit=invalid"

	// Setup mock dependencies
	mockVideoService, mockResponseHandler, mockLogger, app := helpers.SetupMockDependencies()

	// Set up mock expectations
	mockLogger.On("LogInfo", "Invalid limit parameter", mock.Anything).Return()
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusBadRequest, "INVALID_PARAMETER", "Invalid limit parameter, must be a positive integer", mock.Anything).Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.ListVideos(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 400, w.Code, "Should return HTTP 400 Bad Request")
}

// TestListVideos_DatabaseError tests listing videos when a database error occurs
func TestListVideos_DatabaseError(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	// Create test request with query parameters
	c.Request = httptest.NewRequest("GET", "/videos?page=1&limit=10", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token")
	c.Set("authenticated", true)
	c.Set("userId", uuid.New())
	c.Set("request_id", "test-request-id")

	// Add query parameters
	c.Request.URL.RawQuery = "page=1&limit=10"

	// Setup mock dependencies
	mockVideoService, mockResponseHandler, mockLogger, app := helpers.SetupMockDependencies()

	// Set up mock expectations for database error
	dbErr := fmt.Errorf("database error")
	mockVideoService.On("ListVideos", 1, 10).Return(nil, dbErr)
	mockLogger.On("LogInfo", "Failed to list videos", mock.Anything).Return()
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve videos", dbErr).Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.ListVideos(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 500, w.Code, "Should return HTTP 500 Internal Server Error")
}
