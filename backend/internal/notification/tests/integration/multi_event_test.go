package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/notification/producer"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/tests/helpers"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiEventProcessing tests that multiple events can be processed concurrently
func TestMultiEventProcessing(t *testing.T) {
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
	
	// Create test data - a single user that generates various events
	userID := uuid.New()
	
	// Clean up any existing test data
	err = helpers.CleanupTestData(scyllaSession, userID)
	require.NoError(t, err, "Failed to clean up test data")
	
	// Set up all producers
	videoProducer, err := helpers.SetupVideoProducer(pulsarClient, repo, serviceConfig)
	require.NoError(t, err, "Failed to set up video producer")
	defer videoProducer.Close()
	
	commentProducer, err := helpers.SetupCommentProducer(pulsarClient, repo, serviceConfig)
	require.NoError(t, err, "Failed to set up comment producer")
	defer commentProducer.Close()
	
	userProducer, err := helpers.SetupUserProducer(pulsarClient, repo, serviceConfig)
	require.NoError(t, err, "Failed to set up user producer")
	defer userProducer.Close()
	
	// Set up all consumers
	videoConsumer := helpers.SetupVideoConsumer(pulsarClient, repo, serviceConfig)
	defer videoConsumer.Stop()
	
	commentConsumer := helpers.SetupCommentConsumer(pulsarClient, repo, serviceConfig)
	defer commentConsumer.Stop()
	
	userConsumer := helpers.SetupUserConsumer(pulsarClient, repo, serviceConfig)
	defer userConsumer.Stop()
	
	// Ensure topics exist
	err = helpers.SetupTopics(pulsarClient, serviceConfig)
	require.NoError(t, err, "Failed to set up topics")
	
	// Start all consumers in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	var wg sync.WaitGroup
	wg.Add(3) // Three consumers
	
	go func() {
		defer wg.Done()
		err := videoConsumer.Start(ctx)
		if err != nil && err != context.Canceled {
			t.Errorf("Video consumer error: %v", err)
		}
	}()
	
	go func() {
		defer wg.Done()
		err := commentConsumer.Start(ctx)
		if err != nil && err != context.Canceled {
			t.Errorf("Comment consumer error: %v", err)
		}
	}()
	
	go func() {
		defer wg.Done()
		err := userConsumer.Start(ctx)
		if err != nil && err != context.Canceled {
			t.Errorf("User consumer error: %v", err)
		}
	}()
	
	// Allow consumers to start
	time.Sleep(1 * time.Second)
	
	// Define the events to publish
	type eventInfo struct {
		producer producer.Producer
		publish  func() error
		eventType string
	}
	
	// Generate a batch of events
	videoID := uuid.New()
	commentID := uuid.New()
	targetUserID := uuid.New()
	
	events := []eventInfo{
		{
			producer: videoProducer,
			publish: func() error {
				return videoProducer.PublishVideoUploadEvent(
					context.Background(), userID, videoID, "Multi-Event Test Video")
			},
			eventType: string(types.VideoUploaded),
		},
		{
			producer: commentProducer,
			publish: func() error {
				return commentProducer.PublishCommentCreatedEvent(
					context.Background(), userID, videoID, commentID, "Multi-Event Test Comment")
			},
			eventType: string(types.CommentCreated),
		},
		{
			producer: userProducer,
			publish: func() error {
				return userProducer.PublishFollowEvent(
					context.Background(), userID, targetUserID)
			},
			eventType: string(types.UserFollowed),
		},
	}
	
	// Publish all events concurrently
	t.Log("Publishing multiple events concurrently")
	var publishWg sync.WaitGroup
	publishWg.Add(len(events))
	
	for _, e := range events {
		go func(event eventInfo) {
			defer publishWg.Done()
			err := event.publish()
			if err != nil {
				t.Errorf("Failed to publish event: %v", err)
			}
		}(e)
	}
	
	publishWg.Wait()
	t.Log("All events published")
	
	// Wait for all events to be processed
	helpers.WaitForProcessing(10 * time.Second)
	
	// Check if notifications were created for the user
	notifications, err := repo.GetNotificationsByUserID(context.Background(), userID, 10, 0)
	require.NoError(t, err, "Failed to get notifications")
	
	// Track which event types we've seen
	seenEvents := make(map[string]bool)
	for _, notification := range notifications {
		seenEvents[notification.Type] = true
	}
	
	// Verify that we received notifications for all event types
	for _, event := range events {
		if event.eventType == string(types.UserFollowed) {
			// User follow events go to the target user, not the actor
			exists, _, err := helpers.CheckNotificationExists(
				context.Background(),
				repo,
				targetUserID,
				event.eventType,
			)
			require.NoError(t, err, "Failed to check if notification exists")
			assert.True(t, exists, "No notification for event type: %s", event.eventType)
		} else {
			assert.True(t, seenEvents[event.eventType], "No notification for event type: %s", event.eventType)
		}
	}
	
	// Verify that all events were processed concurrently (within a short time window)
	// For simplicity, we're just checking that we got the expected number of notifications
	assert.GreaterOrEqual(t, len(notifications), 2, "Expected at least 2 notifications for the user")
} 