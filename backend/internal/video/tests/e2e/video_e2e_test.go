package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/auth"
	httpPkg "github.com/consensuslabs/pavilion-network/backend/internal/http"
	"github.com/consensuslabs/pavilion-network/backend/internal/storage"
	"github.com/consensuslabs/pavilion-network/backend/internal/storage/ipfs"
	"github.com/consensuslabs/pavilion-network/backend/internal/storage/s3"
	videostorage "github.com/consensuslabs/pavilion-network/backend/internal/storage/video"
	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/consensuslabs/pavilion-network/backend/internal/video/ffmpeg"
	"github.com/consensuslabs/pavilion-network/backend/internal/video/tempfile"
	"github.com/consensuslabs/pavilion-network/backend/testhelper"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Response represents the API response structure
type Response struct {
	Success bool           `json:"success"`
	Data    interface{}    `json:"data,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
}

// ErrorResponse represents the error structure in responses
type ErrorResponse struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Field   string `json:"field,omitempty"`
}

// createTestVideoFile creates a valid test video file for testing
// This function creates a real MP4 file that can be processed by transcoding services
func createTestVideoFile(t *testing.T) string {
	// Check if ffmpeg is available
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("Skipping test: ffmpeg not available")
	}

	// Create a temporary directory for test files
	testDir, err := os.MkdirTemp("", "video-test")
	require.NoError(t, err, "Failed to create temp directory")

	// Create a path for the output video
	outputPath := filepath.Join(testDir, "test-video.mp4")

	// Generate a test video file using ffmpeg
	// This creates a 5-second test video with a color pattern
	cmd := exec.Command(
		"ffmpeg",
		"-f", "lavfi", // Use libavfilter virtual input
		"-i", "testsrc=duration=5:size=640x480:rate=30", // Generate a test pattern
		"-c:v", "libx264", // Use H.264 codec
		"-pix_fmt", "yuv420p", // Use standard pixel format
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

// getTestVideoPath returns the path to the sample test video file
func getTestVideoPath(t *testing.T) string {
	// Path to the sample video in testdata directory
	samplePath := "../../../testdata/videos/sample.mp4"

	// Get the absolute path to make sure it's found regardless of working directory
	absPath, err := filepath.Abs(samplePath)
	if err != nil {
		t.Logf("Failed to get absolute path for sample video: %v", err)
		// Fall back to creating a temporary video
		return createTestVideoFile(t)
	}

	// Check if the file exists
	_, err = os.Stat(absPath)
	if err != nil {
		t.Logf("Sample video not found at %s: %v", absPath, err)
		// Fall back to creating a temporary video
		return createTestVideoFile(t)
	}

	return absPath
}

// createSimpleTestVideoFile creates a simple file with .mp4 extension
// This is a fallback when ffmpeg is not available
func createSimpleTestVideoFile(t *testing.T) string {
	// Create a temporary file with .mp4 extension
	tempFile, err := os.CreateTemp("", "test-video-*.mp4")
	require.NoError(t, err, "Failed to create temp file")

	// Write some dummy data to the file
	// Note: This is not a valid MP4 file, but it has the correct extension
	// This is useful for tests that only check file extension
	_, err = tempFile.WriteString("This is test video content. Not a real MP4 file but has the correct extension.")
	require.NoError(t, err, "Failed to write to temp file")
	tempFile.Close()

	// Register cleanup function
	t.Cleanup(func() {
		os.Remove(tempFile.Name())
	})

	return tempFile.Name()
}

// setupTestEnvironment creates a test environment with real services
// - Database
// - Auth service
// - IPFS service
// - S3 service
// - FFmpeg service
// - Temp file manager
// - IPFS node (if using IPFS)
func setupTestEnvironment(t *testing.T) (*gin.Engine, video.VideoService, *auth.Service) {
	// Skip test if not in E2E mode
	if os.Getenv("E2E_TEST") != "true" {
		t.Skip("Skipping E2E test: E2E_TEST environment variable not set to true")
	}

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create logger using testhelper
	testLogger := testhelper.NewTestLogger(true)

	// Load test configuration
	testConfig, err := testhelper.LoadTestConfig()
	require.NoError(t, err, "Failed to load test configuration")

	// Setup test DB using the testhelper
	db := testhelper.SetupTestDB(t)

	// Create auth service for authentication
	jwtService := auth.NewJWTService(&auth.Config{
		JWT: struct {
			Secret          string
			AccessTokenTTL  time.Duration
			RefreshTokenTTL time.Duration
		}{
			Secret:          "test-secret-key",
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: time.Hour * 24 * 7,
		},
	})
	refreshTokenRepo := auth.NewRefreshTokenRepository(db, testLogger)
	authConfig := &auth.Config{
		JWT: struct {
			Secret          string
			AccessTokenTTL  time.Duration
			RefreshTokenTTL time.Duration
		}{
			Secret:          "test-secret-key",
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: time.Hour * 24 * 7,
		},
	}
	authService := auth.NewService(db, jwtService, refreshTokenRepo, authConfig, testLogger)

	// Create IPFS service
	ipfsConfig := &storage.IPFSConfig{
		APIAddress: testConfig.Storage.IPFS.APIAddress,
		Gateway:    testConfig.Storage.IPFS.Gateway,
	}
	ipfsService := ipfs.NewService(ipfsConfig, testLogger)
	ipfsAdapter := storage.NewVideoIPFSAdapter(ipfsService)

	// Create S3 service
	s3Config := &videostorage.Config{
		Endpoint:        testConfig.Storage.S3.Endpoint,
		AccessKeyID:     testConfig.Storage.S3.AccessKeyID,
		SecretAccessKey: testConfig.Storage.S3.SecretAccessKey,
		UseSSL:          testConfig.Storage.S3.UseSSL,
		Region:          testConfig.Storage.S3.Region,
		Bucket:          testConfig.Storage.S3.Bucket,
		RootDirectory:   testConfig.Storage.S3.RootDirectory,
	}
	s3Service, err := s3.NewService(s3Config, testLogger)
	require.NoError(t, err, "Failed to create storage service")

	// Create FFmpeg service
	ffmpegConfig := &ffmpeg.Config{
		Path:        testConfig.FFmpeg.Path,
		ProbePath:   testConfig.FFmpeg.ProbePath,
		VideoCodec:  testConfig.FFmpeg.VideoCodec,
		AudioCodec:  testConfig.FFmpeg.AudioCodec,
		Preset:      testConfig.FFmpeg.Preset,
		OutputPath:  testConfig.FFmpeg.OutputPath,
		Resolutions: testConfig.FFmpeg.Resolutions,
	}
	ffmpegService := ffmpeg.NewService(ffmpegConfig, testLogger)

	// Create temp file manager
	tempConfig := &tempfile.Config{
		BaseDir:     testConfig.Storage.TempDir,
		Permissions: 0755,
	}
	tempManager, err := tempfile.NewManager(tempConfig, testLogger)
	require.NoError(t, err, "Failed to create temp file manager")

	// Create video service with real dependencies
	videoService := video.NewVideoService(
		db,
		ipfsAdapter,
		s3Service,
		ffmpegService,
		tempManager,
		video.NewLoggerAdapter(testLogger),
	)

	// Setup router
	router := gin.New()
	router.Use(gin.Recovery())

	// Create response handler
	responseHandler := httpPkg.NewResponseHandler(testLogger)

	// Create auth middleware
	authMiddleware := func() gin.HandlerFunc {
		return auth.AuthMiddleware(authService, responseHandler)
	}

	// Setup routes
	apiGroup := router.Group("/api/v1")
	videoRoutes := apiGroup.Group("/videos")
	videoRoutes.Use(authMiddleware())

	// Create video app
	videoConfig := &video.Config{
		Video: struct {
			MaxFileSize    int64    `yaml:"max_file_size"`
			MinTitleLength int      `yaml:"min_title_length"`
			MaxTitleLength int      `yaml:"max_title_length"`
			MaxDescLength  int      `yaml:"max_desc_length"`
			AllowedFormats []string `yaml:"allowed_formats"`
		}{
			MaxFileSize:    testConfig.Video.MaxSize,
			MinTitleLength: testConfig.Video.MinTitleLength,
			MaxTitleLength: testConfig.Video.MaxTitleLength,
			MaxDescLength:  testConfig.Video.MaxDescLength,
			AllowedFormats: testConfig.Video.AllowedFormats,
		},
		FFmpeg: video.FfmpegConfig{
			Path:        testConfig.FFmpeg.Path,
			ProbePath:   testConfig.FFmpeg.ProbePath,
			VideoCodec:  testConfig.FFmpeg.VideoCodec,
			AudioCodec:  testConfig.FFmpeg.AudioCodec,
			Preset:      testConfig.FFmpeg.Preset,
			OutputPath:  testConfig.FFmpeg.OutputPath,
			Resolutions: testConfig.FFmpeg.Resolutions,
		},
	}

	videoApp := &video.App{
		Config:          videoConfig,
		Logger:          video.NewLoggerAdapter(testLogger),
		IPFS:            ipfsAdapter,
		ResponseHandler: responseHandler,
		Video:           videoService,
	}

	// Register video routes
	videoHandler := video.NewVideoHandler(videoApp)
	videoRoutes.GET("", videoHandler.ListVideos)
	videoRoutes.GET("/:id", videoHandler.GetVideo)
	videoRoutes.GET("/:id/status", videoHandler.GetVideoStatus)
	videoRoutes.POST("/upload", videoHandler.HandleUpload)
	videoRoutes.PUT("/:id", videoHandler.UpdateVideo)
	videoRoutes.DELETE("/:id", videoHandler.DeleteVideo)

	return router, videoService, authService
}

// createTestUser creates a test user and returns the access token
func createTestUser(t *testing.T, authService *auth.Service) string {
	username := fmt.Sprintf("testuser-%s", uuid.New().String())
	email := fmt.Sprintf("%s@example.com", username)

	// Register user
	regReq := auth.RegisterRequest{
		Username: username,
		Email:    email,
		Password: "Pass123!",
		Name:     "Test User",
	}

	user, err := authService.Register(regReq)
	require.NoError(t, err, "Failed to register test user")
	require.NotNil(t, user, "User should not be nil")

	// Mark email as verified (required for login)
	err = authService.MarkEmailVerified(user.ID)
	require.NoError(t, err, "Failed to mark email as verified")

	// Login user
	loginResp, err := authService.Login(email, "Pass123!")
	require.NoError(t, err, "Failed to login test user")
	require.NotEmpty(t, loginResp.AccessToken, "Access token should not be empty")

	return loginResp.AccessToken
}

// createTestVideo creates a test video in the database
func createTestVideo(t *testing.T, videoService video.VideoService, userID uuid.UUID) *video.Video {
	// Create a new video
	newVideo := &video.Video{
		ID:          uuid.New(),
		FileID:      uuid.New().String(),
		Title:       "Test Video",
		Description: "This is a test video for e2e testing",
		StoragePath: "/test/path/" + uuid.New().String(),
		IPFSCID:     "testcid",
		Checksum:    "checksum",
		FileSize:    1024,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create a video upload record
	upload := &video.VideoUpload{
		ID:        uuid.New(),
		VideoID:   newVideo.ID,
		Status:    video.UploadStatusCompleted,
		StartTime: time.Now().Add(-time.Minute),
		EndTime:   func() *time.Time { now := time.Now(); return &now }(),
		CreatedAt: time.Now().Add(-time.Minute),
		UpdatedAt: time.Now(),
	}

	// Create mock transcode records
	transcode := &video.Transcode{
		ID:        uuid.New(),
		VideoID:   newVideo.ID,
		Format:    "mp4",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create mock transcode segment
	segment := &video.TranscodeSegment{
		ID:          uuid.New(),
		TranscodeID: transcode.ID,
		StoragePath: fmt.Sprintf("videos/%s/720p.mp4", newVideo.ID),
		IPFSCID:     "test-segment-cid",
		Duration:    5, // 5 seconds
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save the video to the database
	db := testhelper.SetupTestDB(t)
	err := db.Create(newVideo).Error
	require.NoError(t, err, "Failed to create test video")

	// Save the upload record
	err = db.Create(upload).Error
	require.NoError(t, err, "Failed to create test video upload")

	// Save the transcode record
	err = db.Create(transcode).Error
	require.NoError(t, err, "Failed to create test transcode")

	// Save the transcode segment
	err = db.Create(segment).Error
	require.NoError(t, err, "Failed to create test transcode segment")

	return newVideo
}

// TestVideoLifecycle tests the complete lifecycle of a video
func TestVideoLifecycle(t *testing.T) {
	// Skip if not running in E2E mode
	if os.Getenv("E2E_TEST") != "true" {
		t.Skip("Skipping E2E test: E2E_TEST environment variable not set to true")
	}

	// Setup test environment
	router, videoService, authService := setupTestEnvironment(t)

	// Create test user and get access token
	accessToken := createTestUser(t, authService)

	// Extract user ID from token
	claims, err := authService.ValidateToken(accessToken)
	require.NoError(t, err, "Failed to validate token")
	userID, err := uuid.Parse(claims.Subject)
	require.NoError(t, err, "Failed to parse user ID")

	// Test: Upload a video
	t.Run("Upload Video", func(t *testing.T) {
		// Get the test video file path
		videoFilePath := getTestVideoPath(t)
		if t.Skipped() {
			// Fallback to simple test file if no video is available
			videoFilePath = createSimpleTestVideoFile(t)
		}

		// Create multipart form
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add file to form
		file, err := os.Open(videoFilePath)
		require.NoError(t, err, "Failed to open test video file")
		defer file.Close()

		part, err := writer.CreateFormFile("video", filepath.Base(videoFilePath))
		require.NoError(t, err, "Failed to create form file")

		// Copy file content to form
		_, err = io.Copy(part, file)
		require.NoError(t, err, "Failed to copy file content")

		// Add title and description fields
		err = writer.WriteField("title", "Test Upload Video")
		require.NoError(t, err, "Failed to write title field")

		err = writer.WriteField("description", "This is a test upload video")
		require.NoError(t, err, "Failed to write description field")

		// Close the writer
		err = writer.Close()
		require.NoError(t, err, "Failed to close writer")

		// Create request
		req, err := http.NewRequest("POST", "/api/v1/videos/upload", body)
		require.NoError(t, err, "Failed to create request")

		// Set content type and authorization headers
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+accessToken)

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code, "Response status should be OK")

		// Parse response
		var response Response
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Failed to unmarshal response")

		// Verify response
		assert.True(t, response.Success, "Response should be successful")
		assert.Nil(t, response.Error, "Error should be nil")

		// Store the video ID for later tests if available
		if videoData, ok := response.Data.(map[string]interface{}); ok {
			if videoID, ok := videoData["id"].(string); ok {
				t.Logf("Uploaded video ID: %s", videoID)
			}
		}
	})

	// Create a test video in the database
	testVideo := createTestVideo(t, videoService, userID)

	// Test: Get video by ID
	t.Run("Get Video", func(t *testing.T) {
		// Create request
		req, err := http.NewRequest("GET", fmt.Sprintf("/api/v1/videos/%s", testVideo.ID), nil)
		require.NoError(t, err, "Failed to create request")

		// Set authorization header
		req.Header.Set("Authorization", "Bearer "+accessToken)

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code, "Response status should be OK")

		// Parse response
		var response Response
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Failed to unmarshal response")

		// Verify response
		assert.True(t, response.Success, "Response should be successful")
		assert.Nil(t, response.Error, "Error should be nil")

		// Verify video data
		videoData := response.Data.(map[string]interface{})
		assert.Equal(t, testVideo.ID.String(), videoData["id"], "Video ID should match")
		assert.Equal(t, testVideo.Title, videoData["title"], "Video title should match")
	})

	// Test: List videos
	t.Run("List Videos", func(t *testing.T) {
		// Create request
		req, err := http.NewRequest("GET", "/api/v1/videos", nil)
		require.NoError(t, err, "Failed to create request")

		// Set authorization header
		req.Header.Set("Authorization", "Bearer "+accessToken)

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code, "Response status should be OK")

		// Parse response
		var response Response
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Failed to unmarshal response")

		// Verify response
		assert.True(t, response.Success, "Response should be successful")
		assert.Nil(t, response.Error, "Error should be nil")

		// Verify videos list
		videosData := response.Data.(map[string]interface{})
		videos := videosData["videos"].([]interface{})
		assert.GreaterOrEqual(t, len(videos), 1, "Should have at least one video")
	})

	// Test: Update video
	t.Run("Update Video", func(t *testing.T) {
		// Create update payload
		updatePayload := map[string]interface{}{
			"title":       "Updated Test Video",
			"description": "This is an updated test video",
		}

		payloadBytes, err := json.Marshal(updatePayload)
		require.NoError(t, err, "Failed to marshal update payload")

		// Create request
		req, err := http.NewRequest("PUT", fmt.Sprintf("/api/v1/videos/%s", testVideo.ID), bytes.NewBuffer(payloadBytes))
		require.NoError(t, err, "Failed to create request")

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code, "Response status should be OK")

		// Parse response
		var response Response
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Failed to unmarshal response")

		// Verify response
		assert.True(t, response.Success, "Response should be successful")
		assert.Nil(t, response.Error, "Error should be nil")

		// Verify video was updated in database
		updatedVideo, err := videoService.GetVideo(testVideo.ID)
		require.NoError(t, err, "Failed to get updated video")
		assert.Equal(t, "Updated Test Video", updatedVideo.Title, "Video title should be updated")
		assert.Equal(t, "This is an updated test video", updatedVideo.Description, "Video description should be updated")
	})

	// Test: Delete video
	t.Run("Delete Video", func(t *testing.T) {
		// Create request
		req, err := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/videos/%s", testVideo.ID), nil)
		require.NoError(t, err, "Failed to create request")

		// Set authorization header
		req.Header.Set("Authorization", "Bearer "+accessToken)

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code, "Response status should be OK")

		// Parse response
		var response Response
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Failed to unmarshal response")

		// Verify response
		assert.True(t, response.Success, "Response should be successful")
		assert.Nil(t, response.Error, "Error should be nil")

		// Verify video was deleted from database
		_, err = videoService.GetVideo(testVideo.ID)
		assert.Error(t, err, "Video should be deleted")
	})
}

// TestVideoErrorCases tests various error cases for video operations
func TestVideoErrorCases(t *testing.T) {
	// Skip if not running in E2E mode
	if os.Getenv("E2E_TEST") != "true" {
		t.Skip("Skipping E2E test: E2E_TEST environment variable not set to true")
	}

	// Setup test environment
	router, _, authService := setupTestEnvironment(t)

	// Create test user and get access token
	accessToken := createTestUser(t, authService)

	// Test: Get non-existent video
	t.Run("Get Non-existent Video", func(t *testing.T) {
		// Create request with random UUID
		nonExistentID := uuid.New()
		req, err := http.NewRequest("GET", fmt.Sprintf("/api/v1/videos/%s", nonExistentID), nil)
		require.NoError(t, err, "Failed to create request")

		// Set authorization header
		req.Header.Set("Authorization", "Bearer "+accessToken)

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusNotFound, w.Code, "Response status should be Not Found")

		// Parse response
		var response Response
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Failed to unmarshal response")

		// Verify response
		assert.False(t, response.Success, "Response should not be successful")
		assert.NotNil(t, response.Error, "Error should not be nil")
		assert.Equal(t, "VIDEO_NOT_FOUND", response.Error.Code, "Error code should be VIDEO_NOT_FOUND")
	})

	// Test: Upload with invalid file type
	t.Run("Upload Invalid File Type", func(t *testing.T) {
		// Create a test text file
		tempFile, err := os.CreateTemp("", "test-file-*.txt")
		require.NoError(t, err, "Failed to create temp file")
		defer os.Remove(tempFile.Name())

		// Write some dummy data to the file
		_, err = tempFile.WriteString("this is not a video file")
		require.NoError(t, err, "Failed to write to temp file")
		tempFile.Close()

		// Create multipart form
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add file to form
		part, err := writer.CreateFormFile("video", filepath.Base(tempFile.Name()))
		require.NoError(t, err, "Failed to create form file")

		// Open the file again for reading
		file, err := os.Open(tempFile.Name())
		require.NoError(t, err, "Failed to open temp file")
		defer file.Close()

		// Copy file content to form
		_, err = io.Copy(part, file)
		require.NoError(t, err, "Failed to copy file content")

		// Add title and description fields
		err = writer.WriteField("title", "Invalid File Type Test")
		require.NoError(t, err, "Failed to write title field")

		err = writer.WriteField("description", "This should fail due to invalid file type")
		require.NoError(t, err, "Failed to write description field")

		// Close the writer
		err = writer.Close()
		require.NoError(t, err, "Failed to close writer")

		// Create request
		req, err := http.NewRequest("POST", "/api/v1/videos/upload", body)
		require.NoError(t, err, "Failed to create request")

		// Set content type and authorization headers
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+accessToken)

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusBadRequest, w.Code, "Response status should be Bad Request")

		// Parse response
		var response Response
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Failed to unmarshal response")

		// Verify response
		assert.False(t, response.Success, "Response should not be successful")
		assert.NotNil(t, response.Error, "Error should not be nil")
		assert.Equal(t, "ERR_VALIDATION", response.Error.Code, "Error code should be ERR_VALIDATION")
	})

	// Test: Unauthorized access
	t.Run("Unauthorized Access", func(t *testing.T) {
		// Create request without token
		req, err := http.NewRequest("GET", "/api/v1/videos", nil)
		require.NoError(t, err, "Failed to create request")

		// Perform request
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusUnauthorized, w.Code, "Response status should be Unauthorized")

		// Parse response
		var response Response
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Failed to unmarshal response")

		// Verify response
		assert.False(t, response.Success, "Response should not be successful")
		assert.NotNil(t, response.Error, "Error should not be nil")
		assert.Equal(t, "UNAUTHORIZED", response.Error.Code, "Error code should be UNAUTHORIZED")
	})
}
