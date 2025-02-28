package unit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/consensuslabs/pavilion-network/backend/internal/video/tests/helpers"
)

// TestUpdateVideo_Success tests the successful update of a video
func TestUpdateVideo_Success(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	// Create a test UUID
	videoID := uuid.New()

	// Create update request body
	title := "Updated Title"
	description := "Updated Description"
	updateData := map[string]*string{
		"title":       &title,
		"description": &description,
	}
	jsonData, _ := json.Marshal(updateData)

	// Create request
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/videos/%s", videoID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{{Key: "id", Value: videoID.String()}}

	// Add authentication header
	helpers.AuthenticateRequest(c)

	// Setup mock dependencies
	mockVideoService, mockResponseHandler, mockLogger, app := helpers.SetupMockDependencies()

	// Set up config with proper validation values
	app.Config = helpers.VideoConfigForTest()

	// Set up mock expectations
	mockVideoService.On("GetVideo", videoID).Return(&video.Video{
		ID:          videoID,
		Title:       "Original Title",
		Description: "Original Description",
	}, nil)
	mockVideoService.On("UpdateVideo", videoID, title, description).Return(nil)
	mockLogger.On("LogInfo", "Video updated successfully", mock.Anything).Return()
	mockResponseHandler.On("SuccessResponse", mock.Anything, mock.Anything, "Video updated successfully").Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.UpdateVideo(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 200, w.Code, "Should return HTTP 200 OK")
}

// TestUpdateVideo_Unauthorized tests updating a video when the request is not authenticated
func TestUpdateVideo_Unauthorized(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	// Create a test UUID
	videoID := uuid.New()

	// Create update request body
	updateData := map[string]string{
		"title":       "Updated Title",
		"description": "Updated Description",
	}
	jsonData, _ := json.Marshal(updateData)

	// Create request
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/videos/%s", videoID), bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{{Key: "id", Value: videoID.String()}}

	// Do not add authentication header
	// helpers.AuthenticateRequest(c)

	// Setup mock dependencies
	_, mockResponseHandler, _, app := helpers.SetupMockDependencies()

	// Set up mock expectations for unauthorized case
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", mock.Anything).Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.UpdateVideo(c)

	// Verify expectations
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 401, w.Code, "Should return HTTP 401 Unauthorized")
}

// TestUpdateVideo_InvalidRequest tests updating a video with invalid request data
func TestUpdateVideo_InvalidRequest(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	// Create a test UUID
	videoID := uuid.New()

	// Create invalid JSON data
	invalidJSON := []byte("{invalid json")

	// Create request
	c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/videos/%s", videoID), bytes.NewBuffer(invalidJSON))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{{Key: "id", Value: videoID.String()}}

	// Add authentication header
	helpers.AuthenticateRequest(c)

	// Setup mock dependencies
	_, mockResponseHandler, mockLogger, app := helpers.SetupMockDependencies()

	// Set up mock expectations
	mockLogger.On("LogInfo", "Invalid update request format", mock.Anything).Return()
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request format", mock.Anything).Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.UpdateVideo(c)

	// Verify expectations
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 400, w.Code, "Should return HTTP 400 Bad Request")
}

// TestDeleteVideo_Success tests the successful deletion of a video
func TestDeleteVideo_Success(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	// Create a test UUID
	videoID := uuid.New()

	// Create request
	c.Request = httptest.NewRequest("DELETE", fmt.Sprintf("/videos/%s", videoID), nil)
	c.Params = []gin.Param{{Key: "id", Value: videoID.String()}}

	// Add authentication header
	helpers.AuthenticateRequest(c)

	// Setup mock dependencies
	mockVideoService, mockResponseHandler, mockLogger, app := helpers.SetupMockDependencies()

	// Set up mock expectations
	mockVideoService.On("GetVideo", videoID).Return(&video.Video{
		ID:          videoID,
		Title:       "Test Video",
		Description: "Test Description",
	}, nil)
	mockVideoService.On("DeleteVideo", videoID).Return(nil)
	mockLogger.On("LogInfo", "Video soft deleted successfully", mock.Anything).Return()
	mockResponseHandler.On("SuccessResponse", mock.Anything, mock.Anything, "Video deleted successfully").Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.DeleteVideo(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 200, w.Code, "Should return HTTP 200 OK")
}

// TestDeleteVideo_Unauthorized tests deleting a video when the request is not authenticated
func TestDeleteVideo_Unauthorized(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	// Create a test UUID
	videoID := uuid.New()

	// Create request
	c.Request = httptest.NewRequest("DELETE", fmt.Sprintf("/videos/%s", videoID), nil)
	c.Params = []gin.Param{{Key: "id", Value: videoID.String()}}

	// Do not add authentication header
	// helpers.AuthenticateRequest(c)

	// Setup mock dependencies
	_, mockResponseHandler, _, app := helpers.SetupMockDependencies()

	// Set up mock expectations for unauthorized case
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", mock.Anything).Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.DeleteVideo(c)

	// Verify expectations
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 401, w.Code, "Should return HTTP 401 Unauthorized")
}

// TestDeleteVideo_NotFound tests deleting a video that doesn't exist
func TestDeleteVideo_NotFound(t *testing.T) {
	// Setup test context
	c, w := helpers.SetupTestContext()

	// Create a test UUID
	videoID := uuid.New()

	// Create request
	c.Request = httptest.NewRequest("DELETE", fmt.Sprintf("/videos/%s", videoID), nil)
	c.Params = []gin.Param{{Key: "id", Value: videoID.String()}}

	// Add authentication header
	helpers.AuthenticateRequest(c)

	// Setup mock dependencies
	mockVideoService, mockResponseHandler, mockLogger, app := helpers.SetupMockDependencies()

	// Set up mock expectations
	mockVideoService.On("GetVideo", videoID).Return(nil, nil)
	mockLogger.On("LogInfo", "Video not found for deletion", mock.Anything).Return()
	mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusNotFound, "VIDEO_NOT_FOUND", "Video not found", mock.Anything).Return()

	// Create handler and call it
	handler := video.NewVideoHandler(app)
	handler.DeleteVideo(c)

	// Verify expectations
	mockVideoService.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
	mockResponseHandler.AssertExpectations(t)

	// Additional assertions
	assert.Equal(t, 404, w.Code, "Should return HTTP 404 Not Found")
}
