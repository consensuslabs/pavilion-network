# Testing Guide

## Overview

This document outlines the testing setup, configuration, and procedures for the Pavilion Network backend. We use Go's built-in testing framework along with a custom test helper package to facilitate integration tests with a dedicated test database.

## Test Environment Setup

### Configuration Files

The test environment uses two main configuration files:

1. **config_test.yaml**
   - Located in: `backend/config_test.yaml`
   - Contains test-specific configuration including:
     - Test database connection details
     - Server configuration
     - Storage settings
     - Logging configuration

2. **.env.test**
   - Located in: `backend/.env.test`
   - Contains environment-specific variables:
     ```env
     ENV=test
     AUTO_MIGRATE=true
     FORCE_MIGRATION=true
     ```

### Test Database

The test environment uses a dedicated CockroachDB database (`pavilion_test`) to ensure isolation from development and production data. This database is automatically configured with:
- Separate schema
- Clean state for each test run
- Automatic migrations

## Test Helper Package

The `testhelper` package (`backend/testhelper/`) provides utilities for setting up test environments:

### Key Components

1. **SetupTestDB**
   ```go
   func SetupTestDB(t *testing.T) *gorm.DB
   ```
   - Initializes test database connection
   - Loads test configuration
   - Runs necessary migrations
   - Verifies database connection
   - Returns configured GORM DB instance

2. **LoadTestConfig**
   ```go
   func LoadTestConfig() (*Config, error)
   ```
   - Loads test configuration from `config_test.yaml`
   - Merges with `.env.test` variables
   - Supports override via `TEST_CONFIG_FILE` environment variable

## Test Organization

We organize tests into a structured directory hierarchy to improve maintainability and clarity:

### Video API Test Structure

The video API tests follow a modular organization pattern:

```
internal/video/tests/
├── mocks/              # Mock implementations for testing
│   ├── mock_service.go     # Video service mocks
│   ├── mock_storage.go     # Storage service mocks
│   ├── mock_logger.go      # Logger mocks
│   └── mock_response_handler.go # Response handler mocks
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

## Running Tests

### Basic Test Execution

To run all tests:
```bash
ENV=test go test ./... -v
```

To run tests for a specific package:
```bash
ENV=test go test ./internal/auth -v
```

To run only unit tests for the video API:
```bash
ENV=test go test ./internal/video/tests/unit/... -v
```

To run only integration tests for the video API:
```bash
ENV=test go test ./internal/video/tests/integration/... -v
```

To run only end-to-end tests for the video API:
```bash
E2E_TEST=true ENV=test go test ./internal/video/tests/e2e/... -v
```

### Environment Variables

Key environment variables for testing:
- `ENV=test`: Ensures test configuration is loaded
- `AUTO_MIGRATE=true`: Enables automatic schema migrations
- `FORCE_MIGRATION=true`: Forces migration execution
- `TEST_DB`: (Optional) Override test database name
- `TEST_CONFIG_FILE`: (Optional) Specify custom config file location
- `E2E_TEST=true`: (Optional) Enable end-to-end tests that require external services

### Test Database Setup

Before running tests, ensure:
1. CockroachDB is running
2. Test database exists:
   ```sql
   CREATE DATABASE pavilion_test;
   ```

## Writing Tests

### Unit Tests

Unit tests focus on testing individual components in isolation. They use mock implementations to simulate dependencies.

Example of a unit test for the video API:

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
    mockResponseHandler.On("SuccessResponse", mock.Anything, mock.Anything, "").Return()
    
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

### Integration Tests

Integration tests verify how multiple components work together. For the video API, our integration tests:

1. **Use mock services**: Unlike true end-to-end tests, our integration tests use mock services to avoid external dependencies
2. **Test API endpoints**: They test the HTTP endpoints and routing
3. **Verify request/response flow**: They ensure the entire request/response cycle works correctly

Example of an integration test:

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

### End-to-End Tests

End-to-end tests verify the complete functionality of the system using real dependencies. Unlike unit and integration tests, end-to-end tests:

1. **Use real services**: They use actual service implementations instead of mocks
2. **Connect to the test database**: They interact with a real database (the test database)
3. **Test complete workflows**: They test entire user workflows from start to finish
4. **Create and clean up real data**: They create actual records in the database

For the video API, our end-to-end tests cover the complete lifecycle of a video:

1. **User authentication**: Creating a test user and obtaining an access token
2. **Video upload**: Uploading a test video file
3. **Video retrieval**: Getting the video details
4. **Video listing**: Listing all videos
5. **Video update**: Updating video metadata
6. **Video deletion**: Deleting the video

Example of an end-to-end test:

```go
func TestVideoLifecycle(t *testing.T) {
    // Skip if not running in E2E mode
    if os.Getenv("E2E_TEST") != "true" {
        t.Skip("Skipping E2E test: E2E_TEST environment variable not set to true")
    }

    // Setup test environment with real services
    router, videoService, authService := setupTestEnvironment(t)

    // Create test user and get access token
    accessToken := createTestUser(t, authService)
    
    // Extract user ID from token
    claims, err := authService.ValidateToken(accessToken)
    require.NoError(t, err)
    userID, err := uuid.Parse(claims.Subject)
    require.NoError(t, err)

    // Test: Upload a video
    t.Run("Upload Video", func(t *testing.T) {
        // Create a test video file
        videoFilePath := createTestVideoFile(t)
        
        // Create multipart form with the video file
        body := &bytes.Buffer{}
        writer := multipart.NewWriter(body)
        
        // Add the file to the form
        file, err := os.Open(videoFilePath)
        require.NoError(t, err)
        defer file.Close()
        
        part, err := writer.CreateFormFile("video", filepath.Base(videoFilePath))
        require.NoError(t, err)
        io.Copy(part, file)
        
        // Add metadata fields
        writer.WriteField("title", "Test Upload Video")
        writer.WriteField("description", "This is a test upload video")
        writer.Close()
        
        // Create and send request
        req, _ := http.NewRequest("POST", "/api/v1/videos/upload", body)
        req.Header.Set("Content-Type", writer.FormDataContentType())
        req.Header.Set("Authorization", "Bearer "+accessToken)
        
        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)
        
        // Verify response
        assert.Equal(t, http.StatusOK, w.Code)
    })

    // Additional tests for get, list, update, and delete operations
    // ...
}
```

#### Setting Up End-to-End Tests

To set up end-to-end tests:

1. **Create a test environment**:
   ```go
   func setupTestEnvironment(t *testing.T) (*gin.Engine, *video.Service, *auth.Service) {
       // Set Gin to test mode
       gin.SetMode(gin.TestMode)
   
       // Setup test DB
       db := testhelper.SetupTestDB(t)
   
       // Load test configuration
       cfg, err := testhelper.LoadTestConfig()
       require.NoError(t, err)
   
       // Create real services (not mocks)
       log := logger.NewLogger(cfg.Logger)
       storageService := storage.NewService(cfg.Storage, log)
       videoService := video.NewService(db, storageService, cfg.Video, log)
       
       // Setup router with real handlers
       router := gin.New()
       // ... configure routes ...
       
       return router, videoService, authService
   }
   ```

2. **Create test data**:
   ```go
   func createTestUser(t *testing.T, authService *auth.Service) string {
       // Register and login a test user
       // Return the access token
   }
   
   func createTestVideo(t *testing.T, videoService *video.Service, userID uuid.UUID) *video.Video {
       // Create a test video in the database
       // Return the video object
   }
   ```

3. **Clean up after tests**:
   ```go
   // In each test function
   t.Cleanup(func() {
       // Delete test data from the database
       db.Exec("DELETE FROM videos WHERE user_id = ?", userID)
       db.Exec("DELETE FROM users WHERE id = ?", userID)
   })
   ```

### Creating Test Video Files

For end-to-end tests involving video uploads, we need actual video files. There are two approaches:

1. **Using FFmpeg to generate test videos**:
   ```go
   func createTestVideoFile(t *testing.T) string {
       // Check if ffmpeg is available
       _, err := exec.LookPath("ffmpeg")
       if err != nil {
           t.Skip("Skipping test: ffmpeg not available")
       }
   
       // Create a temporary directory for test files
       testDir, err := os.MkdirTemp("", "video-test")
       require.NoError(t, err)
   
       // Create a path for the output video
       outputPath := filepath.Join(testDir, "test-video.mp4")
   
       // Generate a test video file using ffmpeg
       cmd := exec.Command(
           "ffmpeg",
           "-f", "lavfi",           // Use libavfilter virtual input
           "-i", "testsrc=duration=5:size=640x480:rate=30", // Generate a test pattern
           "-c:v", "libx264",       // Use H.264 codec
           "-pix_fmt", "yuv420p",   // Use standard pixel format
           "-movflags", "+faststart", // Optimize for web playback
           outputPath,
       )
   
       // Run the command
       output, err := cmd.CombinedOutput()
       if err != nil {
           t.Logf("ffmpeg output: %s", string(output))
           t.Skip("Skipping test: failed to create test video file")
       }
   
       // Register cleanup function
       t.Cleanup(func() {
           os.RemoveAll(testDir)
       })
   
       return outputPath
   }
   ```

2. **Using a simple file with the correct extension** (fallback):
   ```go
   func createSimpleTestVideoFile(t *testing.T) string {
       // Create a temporary file with .mp4 extension
       tempFile, err := os.CreateTemp("", "test-video-*.mp4")
       require.NoError(t, err)
       
       // Write some dummy data to the file
       _, err = tempFile.WriteString("This is test video content. Not a real MP4 file but has the correct extension.")
       require.NoError(t, err)
       tempFile.Close()
       
       // Register cleanup function
       t.Cleanup(func() {
           os.Remove(tempFile.Name())
       })
       
       return tempFile.Name()
   }
   ```

### External Dependencies for End-to-End Tests

For complete end-to-end testing of the video functionality, the following external services must be available:

1. **CockroachDB**: The test database must be running and accessible
   - Connection details are specified in `config_test.yaml`
   - The database should be created before running tests

2. **Storage Services**:
   - **Local Storage**: For tests using local file storage
   - **S3-compatible Storage**: For tests involving S3 uploads
     - Can use MinIO as a local S3-compatible service
     - Configure in `config_test.yaml` with appropriate credentials
   - **IPFS Node**: For tests involving IPFS uploads
     - Can use a local IPFS node or a test gateway
     - Configure in `config_test.yaml`

3. **FFmpeg**: For generating and processing test video files
   - Required for creating valid test videos
   - Tests will be skipped if FFmpeg is not available

### Running End-to-End Tests

To run end-to-end tests that require external services:

```bash
# Start required services (example using Docker)
docker-compose -f docker-compose.test.yml up -d

# Run the end-to-end tests
E2E_TEST=true ENV=test go test ./internal/video/tests/e2e/... -v

# Clean up services
docker-compose -f docker-compose.test.yml down
```

The `E2E_TEST=true` environment variable is used to indicate that external services are available and end-to-end tests should be run. Tests will be skipped if this variable is not set.

### True End-to-End Tests

For true end-to-end tests that interact with a real database:

1. Use the `testhelper.SetupTestDB` function to initialize a test database
2. Create real service instances instead of mocks
3. Use the test database for data persistence
4. Clean up test data after each test

### Best Practices

1. **Database State**
   - Always use `SetupTestDB` for database tests
   - Clean up test data after each test
   - Don't assume database state

2. **Configuration**
   - Use test configuration files
   - Don't modify production/development configs
   - Use environment variables for overrides

3. **Test Structure**
   - Follow Go's testing conventions
   - Use table-driven tests where appropriate
   - Include both positive and negative test cases

4. **Mocking**
   - Use the `mocks` package for consistent mock implementations
   - Set clear expectations for mock behavior
   - Verify all expectations are met

5. **Test Helpers**
   - Use helper functions from the `helpers` package
   - Create reusable test utilities
   - Keep test setup code DRY (Don't Repeat Yourself)

6. **End-to-End Testing**
   - Test complete user workflows
   - Use real services and database connections
   - Create and clean up test data
   - Test both success and error scenarios
   - Skip tests when external dependencies are not available
   - Use environment variables to control test execution

7. **Test Video Files**
   - Use FFmpeg to generate valid test videos when possible
   - Have fallback mechanisms when FFmpeg is not available
   - Clean up temporary files after tests

## Continuous Integration

Our CI pipeline:
1. Sets up test database
2. Loads test configuration
3. Runs all tests
4. Reports test coverage

For end-to-end tests in CI:
1. Start required external services (database, S3, IPFS)
2. Set the `E2E_TEST=true` environment variable
3. Run the end-to-end tests
4. Shut down external services

## Troubleshooting

Common issues and solutions:

1. **Database Connection Issues**
   - Verify CockroachDB is running
   - Check database exists
   - Confirm connection settings in `config_test.yaml`

2. **Configuration Loading Failures**
   - Verify `.env.test` exists and is accessible
   - Check `config_test.yaml` syntax
   - Confirm environment variables

3. **Migration Failures**
   - Check `AUTO_MIGRATE` and `FORCE_MIGRATION` settings
   - Verify database permissions
   - Review migration logs

4. **Mock Expectation Failures**
   - Ensure all expected method calls are set up
   - Check parameter matching
   - Verify return values match expected types

5. **End-to-End Test Failures**
   - Check if the test database is properly set up
   - Verify that all required services are initialized correctly
   - Ensure test data is properly created and cleaned up
   - Check for environment-specific issues
   - Verify external services are running (S3, IPFS)
   - Check if FFmpeg is installed for video generation

6. **Video Processing Issues**
   - Verify FFmpeg is installed and in the PATH
   - Check if the generated test videos are valid
   - Ensure storage services are properly configured
   - Check for sufficient disk space for temporary files

## Future Improvements

Planned enhancements to the testing framework:
1. Add test data factories
2. Implement test database cleanup helpers
3. Add performance testing utilities
4. Enhance test coverage reporting
5. Add more integration tests for complex scenarios
6. Implement property-based testing for edge cases
7. Expand end-to-end test coverage for all major features
8. Create a Docker Compose file specifically for test dependencies
9. Add pre-generated test video files to avoid FFmpeg dependency
10. Implement parallel test execution for faster test runs 