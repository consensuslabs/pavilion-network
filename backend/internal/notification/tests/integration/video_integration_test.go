package integration

import (
	"context"
	"testing"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/notification/tests/helpers"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVideoEventIntegration tests the complete flow of video events through the notification system
func TestVideoEventIntegration(t *testing.T) {
	// Set up integration test environment
	_, pulsarClient, scyllaSession, cleanup := SetupIntegrationTest(t)
	defer cleanup()
	
	// Initialize ScyllaDB schema if needed
	err := helpers.InitializeScyllaDBSchema(scyllaSession)
	require.NoError(t, err, "Failed to initialize ScyllaDB schema")
	
	// Create a test service config with unique topics
	serviceConfig := CreateTestServiceConfig()
	
	// Set up notification repository
	repo, err := helpers.CreateNotificationRepository(scyllaSession)
	require.NoError(t, err, "Failed to create notification repository")
	
	// Create test data
	userID := uuid.New()
	videoID := uuid.New()
	title := "Test Integration Video Upload"
	
	// Print the test data for debugging
	t.Logf("Test with UserID: %s, VideoID: %s, Title: %s", userID, videoID, title)
	
	// Clean up any existing test data for this user
	err = helpers.CleanupTestData(scyllaSession, userID)
	require.NoError(t, err, "Failed to clean up test data")
	
	// Set up producer
	videoProducer, err := helpers.SetupVideoProducer(pulsarClient, repo, serviceConfig)
	require.NoError(t, err, "Failed to set up video producer")
	defer videoProducer.Close()
	
	// Set up consumer
	videoConsumer := helpers.SetupVideoConsumer(pulsarClient, repo, serviceConfig)
	defer videoConsumer.Stop()
	
	// Ensure topics exist
	err = helpers.SetupTopics(pulsarClient, serviceConfig)
	require.NoError(t, err, "Failed to set up topics")
	
	// Start consumer in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Create a channel to signal when consumer is ready
	consumerReady := make(chan struct{})
	var consumerErr error
	
	go func() {
		t.Logf("Starting video consumer...")
		consumerReady <- struct{}{}
		consumerErr = videoConsumer.Start(ctx)
		if consumerErr != nil && consumerErr != context.Canceled {
			t.Logf("Consumer error: %v", consumerErr)
		}
	}()
	
	// Wait for consumer to signal it's ready to receive messages
	<-consumerReady
	
	// Allow consumer to start
	t.Logf("Waiting for consumer to initialize...")
	time.Sleep(2 * time.Second)
	
	// Publish a video upload event
	t.Logf("Publishing video upload event...")
	err = videoProducer.PublishVideoUploadEvent(context.Background(), userID, videoID, title)
	require.NoError(t, err, "Failed to publish video upload event")
	
	// Wait for event to be processed
	t.Logf("Waiting for event to be processed...")
	helpers.WaitForProcessing(5 * time.Second)
	
	// Check if notification was created
	t.Logf("Checking if notification was created...")
	exists, notification, err := helpers.CheckNotificationExists(
		context.Background(),
		repo,
		userID,
		string(types.VideoUploaded),
	)
	require.NoError(t, err, "Failed to check if notification exists")
	
	// Get all notifications for debugging
	allNotifications, err := repo.GetNotificationsByUserID(context.Background(), userID, 100, 0)
	require.NoError(t, err, "Failed to get all notifications")
	t.Logf("Found %d notifications for user", len(allNotifications))
	for i, n := range allNotifications {
		t.Logf("Notification %d: ID=%s, Type=%s, Content=%s", i, n.ID, n.Type, n.Content)
	}
	
	// Verify notification
	assert.True(t, exists, "Notification was not created")
	if exists && notification != nil {
		assert.Equal(t, userID, notification.UserID, "Wrong user ID in notification")
		assert.Equal(t, string(types.VideoUploaded), notification.Type, "Wrong notification type")
		assert.Contains(t, notification.Content, title, "Notification content doesn't contain video title")
		
		// Check metadata
		videoIDFromMetadata, hasVideoID := notification.Metadata["videoId"]
		assert.True(t, hasVideoID, "Missing videoId in notification metadata")
		assert.Equal(t, videoID.String(), videoIDFromMetadata, "Wrong videoId in notification metadata")
	}
} 