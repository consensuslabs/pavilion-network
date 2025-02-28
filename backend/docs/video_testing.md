# Video API Testing Guide

## Overview

This document provides a comprehensive guide to the testing infrastructure for the Pavilion Network's video API. It covers the test organization, different types of tests, and how to run them effectively.

## Test Organization

The video API tests follow a modular organization pattern to improve maintainability and clarity:

```
internal/video/tests/
├── mocks/              # Mock implementations for testing
│   ├── mock_service.go     # Video service mocks
│   ├── mock_storage.go     # Storage service mocks
│   ├── mock_logger.go      # Logger mocks
│   └── mock_ffmpeg.go      # FFmpeg service mocks
├── helpers/            # Test helper functions
│   └── test_utils.go       # Common test utilities
├── unit/               # Unit tests for specific functionality
│   ├── upload_test.go      # Tests for upload endpoints
│   ├── get_video_test.go   # Tests for getting videos
│   ├── list_videos_test.go # Tests for listing videos
│   ├── video_status_test.go # Tests for video status
│   └── update_delete_test.go # Tests for update/delete operations
├── integration/        # Integration tests
│   └── api_test.go         # API endpoint tests
└── e2e/                # End-to-end tests
    └── video_e2e_test.go   # Complete video lifecycle tests
```

This structure provides several benefits:
- **Better organization**: Tests are grouped by functionality
- **Improved maintainability**: Smaller, focused test files are easier to maintain
- **Enhanced readability**: Tests are easier to find and understand
- **Reduced merge conflicts**: Multiple developers can work on different test files without conflicts
- **Easier extension**: New tests can be added to the appropriate files

## Types of Tests

### Unit Tests

Unit tests focus on testing individual components in isolation. They use mock implementations to simulate dependencies.

#### Key Characteristics:
- Test a single function or method
- Use mock dependencies to isolate the component being tested
- Fast execution
- No external dependencies (database, network, etc.)

#### Important Note on Mock Expectations

When setting up mock expectations, it's crucial to match the exact method signature and parameters that will be called. For example, if a handler calls `SuccessResponse(context, response, message)`, the mock expectation must match these parameters exactly:

```go
// Correct mock setup
mockResponseHandler.On("SuccessResponse", mock.Anything, mock.Anything, "Video details retrieved successfully").Return()

// Incorrect mock setup (missing or incorrect message parameter)
mockResponseHandler.On("SuccessResponse", mock.Anything, mock.Anything, "").Return() // Will fail
```

Common mock expectation issues to watch for:

1. **Missing or incorrect success messages**: Each handler uses a specific success message in the `SuccessResponse` call. Make sure to check the handler implementation and use the exact same message in your mock expectation.

2. **Log message mismatches**: Ensure that the log messages in your mock expectations match exactly what the handler is logging. For example, if the handler logs "Video soft deleted successfully" but your mock expects "Video deleted successfully", the test will fail.

3. **Error handling in not-found scenarios**: When testing not-found scenarios, make sure to return an appropriate error from the service mock rather than returning `nil, nil`. This ensures the handler's error handling logic is properly tested.

Example of a correct not-found test setup:
```go
// Create a not found error
notFoundErr := fmt.Errorf("video not found: %s", videoID)

// Set up mock expectations for not found case
mockVideoService.On("GetVideo", videoID).Return(nil, notFoundErr)
mockLogger.On("LogInfo", "Video not found or has been deleted", mock.Anything).Return()
mockResponseHandler.On("ErrorResponse", mock.Anything, http.StatusNotFound, "VIDEO_NOT_FOUND", fmt.Sprintf("video not found: %s", videoID), nil).Return()
```

Mismatched mock expectations will cause test failures with detailed error messages showing the expected vs. actual method calls. These error messages are very helpful for debugging - they show exactly what parameters were passed to the method and what was expected.

#### Example Unit Test:

```go
func TestGetVideo_Success(t *testing.T) {
    // Setup test context
    c, w := helpers.SetupTestContext()
    
    // Create a test UUID
    videoID := uuid.New()
    c.Params = []gin.Param{{Key: "id", Value: videoID.String()}}
    
    // Add authentication header
    helpers.AuthenticateRequest(c)
    
    // Setup mock dependencies
    mockVideoService, mockResponseHandler, mockLogger, app := helpers.SetupMockDependencies()
    
    // Create test video
    testVideo := &video.Video{
        ID:          videoID,
        Title:       "Test Video",
        Description: "Test Description",
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
    assert.Equal(t, 200, w.Code)
}
```

#### Available Unit Tests:

1. **Upload Tests** (`upload_test.go`):
   - Tests for video upload validation
   - Tests for handling upload requests
   - Tests for error cases (invalid file, missing parameters)

2. **Get Video Tests** (`get_video_test.go`):
   - Tests for retrieving video details
   - Tests for handling invalid video IDs
   - Tests for not found scenarios
   - Tests for database errors

3. **List Videos Tests** (`list_videos_test.go`):
   - Tests for listing videos with pagination
   - Tests for handling invalid pagination parameters
   - Tests for database errors

4. **Video Status Tests** (`video_status_test.go`):
   - Tests for retrieving video upload status
   - Tests for handling invalid video IDs
   - Tests for not found scenarios

5. **Update/Delete Tests** (`update_delete_test.go`):
   - Tests for updating video metadata
   - Tests for deleting videos
   - Tests for handling invalid requests
   - Tests for not found scenarios

### Integration Tests

Integration tests verify how multiple components work together. For the video API, our integration tests:

1. **Use mock services**: Unlike true end-to-end tests, our integration tests use mock services to avoid external dependencies
2. **Test API endpoints**: They test the HTTP endpoints and routing
3. **Verify request/response flow**: They ensure the entire request/response cycle works correctly

#### Key Characteristics:
- Test multiple components working together
- Use mock services for external dependencies
- Test HTTP endpoints and routing
- Verify the entire request/response cycle

#### Example Integration Test:

```go
func TestVideoEndpoints_Integration(t *testing.T) {
    // Setup test environment
    gin.SetMode(gin.TestMode)
    
    // Create mock services
    mockVideoService, mockIPFSService, _, _, _, mockResponseHandler, mockLogger := helpers.SetupMockServices()
    
    // Setup test videos
    testVideos := helpers.SetupTestVideos(3)
    
    // Set expectations for the mock services
    mockVideoService.On("ListVideos", 1, 10).Return(testVideos, nil)
    mockLogger.On("LogInfo", mock.Anything, mock.Anything).Return()
    
    // Create app instance with mocks
    app := &video.App{
        Config:          config,
        Logger:          mockLogger,
        Video:           mockVideoService,
        ResponseHandler: mockResponseHandler,
    }
    
    // Setup router with endpoints
    router := gin.New()
    apiGroup := router.Group("/api/v1")
    videoRoutes := apiGroup.Group("/videos")
    videoRoutes.GET("", video.NewVideoHandler(app).ListVideos)
    
    // Make request
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/videos", nil)
    req.Header.Set("Authorization", "Bearer test-token")
    router.ServeHTTP(w, req)
    
    // Check response
    assert.Equal(t, http.StatusOK, w.Code)
}
```

### End-to-End (E2E) Tests

End-to-end tests verify the complete functionality of the system using real dependencies. Unlike unit and integration tests, end-to-end tests:

1. **Use real services**: They use actual service implementations instead of mocks
2. **Connect to the test database**: They interact with a real database (the test database)
3. **Test complete workflows**: They test entire user workflows from start to finish

#### Key Characteristics:
- Test the entire system from end to end
- Use real dependencies (database, storage, etc.)
- Test complete user workflows
- Slower execution but more comprehensive coverage

#### Example E2E Test:

The `TestVideoLifecycle` test in `video_e2e_test.go` tests the complete lifecycle of a video:

1. **Upload_Video**: Tests uploading a video file
2. **Get_Video**: Tests retrieving the uploaded video
3. **List_Videos**: Tests listing videos including the uploaded one
4. **Update_Video**: Tests updating the video metadata
5. **Delete_Video**: Tests soft-deleting the video and verifying it's no longer accessible

This test uses real implementations of all services, including:
- Database access
- File storage
- Video transcoding
- API endpoints

## Mock Implementations

The `mocks/` directory contains mock implementations of all interfaces used by the video package. These mocks use the `github.com/stretchr/testify/mock` package to provide test expectations and return values.

### Available Mocks

- `MockVideoService`: Implements the `VideoService` interface
- `MockIPFSService`: Implements the `IPFSService` interface
- `MockStorageService`: Implements the storage service interface
- `MockFFmpegService`: Implements the `ffmpeg.Service` interface
- `MockTempFileManager`: Implements the `tempfile.TempFileManager` interface
- `MockResponseHandler`: Implements the `ResponseHandler` interface
- `MockLogger`: Implements the `Logger` interface
- `TestLoggerAdapter`: Adapts the `testhelper.TestLogger` to our `Logger` interface

## Test Helpers

The `helpers/` directory contains utility functions to simplify writing tests:

- `CreateTestFile`: Creates a temporary file with specific content
- `CreateMultipartRequest`: Creates a multipart HTTP request for file uploads
- `VideoConfigForTest`: Returns a standard video configuration for tests
- `SetupTestVideos`: Creates test video objects
- `SetupTestUploads`: Creates test upload objects
- `SetupTestGinEngine`: Sets up a Gin engine for testing
- `SetupMockServices`: Creates mock services for testing

## Running Tests

To run the tests, use the following commands:

### Unit Tests

```bash
# Run all unit tests
ENV=test go test ./internal/video/tests/unit/...

# Run a specific unit test file
ENV=test go test ./internal/video/tests/unit/get_video_test.go

# Run a specific test function
ENV=test go test ./internal/video/tests/unit/... -run TestGetVideo_Success
```

### Integration Tests

```bash
# Run all integration tests
ENV=test go test ./internal/video/tests/integration/...
```

### End-to-End Tests

```bash
# Run all end-to-end tests
ENV=test E2E_TEST=true go test ./internal/video/tests/e2e/...

# Run a specific end-to-end test
ENV=test E2E_TEST=true go test ./internal/video/tests/e2e/... -run TestVideoLifecycle
```

Note: End-to-end tests require the `E2E_TEST` environment variable to be set to `true`. Otherwise, they will be skipped.

## Common Test Failures and How to Fix Them

When working with the video API tests, you might encounter several common types of failures. Here's how to diagnose and fix them:

### 1. Mock Expectation Failures

**Symptom**: Test fails with an error like:
```
mock: Unexpected Method Call
-----------------------------
SuccessResponse(*gin.Context,video.VideoDetailsResponse,string)
                0: &gin.Context{...}
                1: video.VideoDetailsResponse{...}
                2: "Video updated successfully"

The closest call I have is: 
SuccessResponse(string,string,string)
                0: "mock.Anything"
                1: "mock.Anything"
                2: ""
```

**Cause**: The mock expectation doesn't match the actual method call, often due to a missing or incorrect message parameter.

**Fix**: 
1. Check the handler implementation to see what message it's using in the `SuccessResponse` call
2. Update the mock expectation to match exactly:
```go
// Before (incorrect)
mockResponseHandler.On("SuccessResponse", mock.Anything, mock.Anything, "").Return()

// After (correct)
mockResponseHandler.On("SuccessResponse", mock.Anything, mock.Anything, "Video updated successfully").Return()
```

### 2. Nil Pointer Dereference

**Symptom**: Test fails with an error like:
```
panic: runtime error: invalid memory address or nil pointer dereference
```

**Cause**: Often occurs in not-found scenarios where the handler tries to access a property of a nil object.

**Fix**:
1. Make sure your test properly simulates the error condition by returning an appropriate error:
```go
// Before (incorrect - can cause nil pointer dereference)
mockVideoService.On("GetVideo", videoID).Return(nil, nil)

// After (correct)
notFoundErr := fmt.Errorf("video not found: %s", videoID)
mockVideoService.On("GetVideo", videoID).Return(nil, notFoundErr)
```

2. Ensure the handler has proper nil checks before accessing properties of potentially nil objects.

### 3. Log Message Mismatches

**Symptom**: Test fails with an error about unexpected log message:
```
mock: Unexpected Method Call
-----------------------------
LogInfo(string,map[string]interface {})
                0: "Video soft deleted successfully"
                1: map[string]interface {}{"request_id":"test-request-id", "video_id":"..."}

The closest call I have is: 
LogInfo(string,string)
                0: "Video deleted successfully"
                1: "mock.Anything"
```

**Fix**: Update the mock expectation to match the exact log message used in the handler:
```go
// Before (incorrect)
mockLogger.On("LogInfo", "Video deleted successfully", mock.Anything).Return()

// After (correct)
mockLogger.On("LogInfo", "Video soft deleted successfully", mock.Anything).Return()
```

### 4. End-to-End Tests Being Skipped

**Symptom**: End-to-end tests are skipped with a message like:
```
Skipping E2E test: E2E_TEST environment variable not set to true
```

**Fix**: Set the `E2E_TEST` environment variable to `true` when running the tests:
```bash
ENV=test E2E_TEST=true go test ./internal/video/tests/e2e/...
```

## Soft Delete Testing

The video API implements soft delete functionality, where videos are marked as deleted but not actually removed from the database. This requires special testing considerations:

### Testing Soft Delete Functionality

The `Delete_Video` test in the `TestVideoLifecycle` suite verifies that:

1. A video can be soft-deleted via the API
2. The soft-deleted video cannot be retrieved using the `GetVideo` endpoint
3. The soft-deleted video is excluded from the results of the `ListVideos` endpoint

The implementation ensures that:
- The `GetVideo` method returns an appropriate error for soft-deleted videos, specifically distinguishing between "not found" and "soft-deleted" videos
- The `ListVideos` method automatically excludes soft-deleted videos by using the condition `WHERE deleted_at IS NULL`
- All handlers properly handle the error cases for soft-deleted videos, returning a 404 status code with a specific error code:
  - `VIDEO_NOT_FOUND`: When the video doesn't exist
  - `VIDEO_DELETED`: When the video exists but has been soft-deleted

This approach provides better user experience by clearly communicating whether a video doesn't exist or has been deleted.

### Implementation Details

The soft delete functionality is implemented at multiple levels:

1. **Database Level**: GORM's soft delete feature automatically adds a `deleted_at` timestamp when a record is deleted
2. **Service Level**: The `GetVideo` method checks if a video exists but is soft-deleted and returns a specific error message
3. **Handler Level**: All handlers (`GetVideo`, `GetVideoStatus`, `UpdateVideo`) check for specific error messages to determine if a video is not found or soft-deleted

To test this functionality:
```bash
ENV=test E2E_TEST=true go test ./internal/video/tests/e2e -run TestVideoLifecycle/Delete_Video -v
```

## Conclusion

The video API testing infrastructure provides comprehensive coverage of all functionality through a combination of unit, integration, and end-to-end tests. This multi-layered approach ensures that:

1. Individual components work correctly in isolation
2. Components work together as expected
3. The entire system functions correctly from end to end

By following the guidelines in this document, you can effectively run existing tests and add new tests to maintain and improve the quality of the video API. 