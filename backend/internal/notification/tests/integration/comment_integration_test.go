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

// TestCommentEventIntegration tests the complete flow of comment events through the notification system
func TestCommentEventIntegration(t *testing.T) {
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
	videoOwnerID := uuid.New()
	commentAuthorID := uuid.New()
	videoID := uuid.New()
	commentID := uuid.New()
	content := "This is a test comment for integration testing"
	
	// Clean up any existing test data
	err = helpers.CleanupTestData(scyllaSession, videoOwnerID)
	require.NoError(t, err, "Failed to clean up test data for video owner")
	err = helpers.CleanupTestData(scyllaSession, commentAuthorID)
	require.NoError(t, err, "Failed to clean up test data for comment author")
	
	// Set up producer
	commentProducer, err := helpers.SetupCommentProducer(pulsarClient, repo, serviceConfig)
	require.NoError(t, err, "Failed to set up comment producer")
	defer commentProducer.Close()
	
	// Set up consumer
	commentConsumer := helpers.SetupCommentConsumer(pulsarClient, repo, serviceConfig)
	defer commentConsumer.Stop()
	
	// Ensure topics exist
	err = helpers.SetupTopics(pulsarClient, serviceConfig)
	require.NoError(t, err, "Failed to set up topics")
	
	// Start consumer in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go func() {
		err := commentConsumer.Start(ctx)
		if err != nil && err != context.Canceled {
			t.Errorf("Consumer error: %v", err)
		}
	}()
	
	// Allow consumer to start
	time.Sleep(1 * time.Second)
	
	// Test 1: Comment Created Event
	t.Run("CommentCreatedEvent", func(t *testing.T) {
		// Publish a comment created event
		err = commentProducer.PublishCommentCreatedEvent(context.Background(), commentAuthorID, videoID, commentID, content)
		require.NoError(t, err, "Failed to publish comment created event")
		
		// Wait for event to be processed
		helpers.WaitForProcessing(5 * time.Second)
		
		// Check if notification was created
		exists, notification, err := helpers.CheckNotificationExists(
			context.Background(),
			repo,
			commentAuthorID,
			string(types.CommentCreated),
		)
		require.NoError(t, err, "Failed to check if notification exists")
		
		// Verify notification
		assert.True(t, exists, "Notification was not created")
		if exists && notification != nil {
			assert.Equal(t, commentAuthorID, notification.UserID, "Wrong user ID in notification")
			assert.Equal(t, string(types.CommentCreated), notification.Type, "Wrong notification type")
			assert.Contains(t, notification.Content, "comment", "Notification content doesn't mention comment")
			
			// Check metadata
			commentIDFromMetadata, hasCommentID := notification.Metadata["commentId"]
			assert.True(t, hasCommentID, "Missing commentId in notification metadata")
			assert.Equal(t, commentID.String(), commentIDFromMetadata, "Wrong commentId in notification metadata")
			
			videoIDFromMetadata, hasVideoID := notification.Metadata["videoId"]
			assert.True(t, hasVideoID, "Missing videoId in notification metadata")
			assert.Equal(t, videoID.String(), videoIDFromMetadata, "Wrong videoId in notification metadata")
		}
	})
	
	// Test 2: Comment Reply Event
	t.Run("CommentReplyEvent", func(t *testing.T) {
		// Create parent comment and comment reply IDs
		parentCommentID := uuid.New()
		replyCommentID := uuid.New()
		parentCommentAuthorID := uuid.New()
		replyContent := "This is a test reply to a comment"
		
		// Clean up any existing test data for the parent comment author
		err = helpers.CleanupTestData(scyllaSession, parentCommentAuthorID)
		require.NoError(t, err, "Failed to clean up test data for parent comment author")
		
		// Publish a comment reply event
		err = commentProducer.PublishCommentReplyEvent(context.Background(), 
			commentAuthorID, videoID, replyCommentID, parentCommentID, replyContent)
		require.NoError(t, err, "Failed to publish comment reply event")
		
		// Wait for event to be processed
		helpers.WaitForProcessing(5 * time.Second)
		
		// Check if notification was created
		exists, notification, err := helpers.CheckNotificationExists(
			context.Background(),
			repo,
			commentAuthorID,
			string(types.CommentReplied),
		)
		require.NoError(t, err, "Failed to check if notification exists")
		
		// Verify notification
		assert.True(t, exists, "Notification was not created")
		if exists && notification != nil {
			assert.Equal(t, commentAuthorID, notification.UserID, "Wrong user ID in notification")
			assert.Equal(t, string(types.CommentReplied), notification.Type, "Wrong notification type")
			assert.Contains(t, notification.Content, "reply", "Notification content doesn't mention reply")
			
			// Check metadata
			commentIDFromMetadata, hasCommentID := notification.Metadata["commentId"]
			assert.True(t, hasCommentID, "Missing commentId in notification metadata")
			assert.Equal(t, replyCommentID.String(), commentIDFromMetadata, "Wrong commentId in notification metadata")
			
			parentIDFromMetadata, hasParentID := notification.Metadata["parentId"]
			assert.True(t, hasParentID, "Missing parentId in notification metadata")
			assert.Equal(t, parentCommentID.String(), parentIDFromMetadata, "Wrong parentId in notification metadata")
			
			videoIDFromMetadata, hasVideoID := notification.Metadata["videoId"]
			assert.True(t, hasVideoID, "Missing videoId in notification metadata")
			assert.Equal(t, videoID.String(), videoIDFromMetadata, "Wrong videoId in notification metadata")
		}
	})
} 