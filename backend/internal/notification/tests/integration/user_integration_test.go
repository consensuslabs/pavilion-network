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

// TestUserEventIntegration tests the complete flow of user events through the notification system
func TestUserEventIntegration(t *testing.T) {
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
	followerID := uuid.New()
	targetUserID := uuid.New()
	
	// Clean up any existing test data
	err = helpers.CleanupTestData(scyllaSession, followerID)
	require.NoError(t, err, "Failed to clean up test data for follower")
	err = helpers.CleanupTestData(scyllaSession, targetUserID)
	require.NoError(t, err, "Failed to clean up test data for target user")
	
	// Set up producer
	userProducer, err := helpers.SetupUserProducer(pulsarClient, repo, serviceConfig)
	require.NoError(t, err, "Failed to set up user producer")
	defer userProducer.Close()
	
	// Set up consumer
	userConsumer := helpers.SetupUserConsumer(pulsarClient, repo, serviceConfig)
	defer userConsumer.Stop()
	
	// Ensure topics exist
	err = helpers.SetupTopics(pulsarClient, serviceConfig)
	require.NoError(t, err, "Failed to set up topics")
	
	// Start consumer in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go func() {
		err := userConsumer.Start(ctx)
		if err != nil && err != context.Canceled {
			t.Errorf("Consumer error: %v", err)
		}
	}()
	
	// Allow consumer to start
	time.Sleep(1 * time.Second)
	
	// Test 1: User Follow Event
	t.Run("UserFollowEvent", func(t *testing.T) {
		// Publish a user follow event
		err = userProducer.PublishFollowEvent(context.Background(), followerID, targetUserID)
		require.NoError(t, err, "Failed to publish user follow event")
		
		// Wait for event to be processed
		helpers.WaitForProcessing(5 * time.Second)
		
		// Check if notification was created for the target user
		exists, notification, err := helpers.CheckNotificationExists(
			context.Background(),
			repo,
			targetUserID,
			string(types.UserFollowed),
		)
		require.NoError(t, err, "Failed to check if notification exists")
		
		// Verify notification
		assert.True(t, exists, "Notification was not created")
		if exists && notification != nil {
			assert.Equal(t, targetUserID, notification.UserID, "Wrong user ID in notification")
			assert.Equal(t, string(types.UserFollowed), notification.Type, "Wrong notification type")
			assert.Contains(t, notification.Content, "following", "Notification content doesn't mention following")
			
			// Check metadata
			followerIDFromMetadata, hasFollowerID := notification.Metadata["userId"]
			assert.True(t, hasFollowerID, "Missing userId in notification metadata")
			assert.Equal(t, followerID.String(), followerIDFromMetadata, "Wrong userId in notification metadata")
			
			targetUserIDFromMetadata, hasTargetUserID := notification.Metadata["targetUserId"]
			assert.True(t, hasTargetUserID, "Missing targetUserId in notification metadata")
			assert.Equal(t, targetUserID.String(), targetUserIDFromMetadata, "Wrong targetUserId in notification metadata")
		}
	})
	
	// Test 2: User Unfollow Event (if applicable)
	t.Run("UserUnfollowEvent", func(t *testing.T) {
		// Clean up previous notifications for the target user
		err = helpers.CleanupTestData(scyllaSession, targetUserID)
		require.NoError(t, err, "Failed to clean up test data for target user")
		
		// Publish a user unfollow event
		err = userProducer.PublishUnfollowEvent(context.Background(), followerID, targetUserID)
		require.NoError(t, err, "Failed to publish user unfollow event")
		
		// Wait for event to be processed
		helpers.WaitForProcessing(5 * time.Second)
		
		// Check if notification was created for the target user
		exists, notification, err := helpers.CheckNotificationExists(
			context.Background(),
			repo,
			targetUserID,
			string(types.UserUnfollowed),
		)
		require.NoError(t, err, "Failed to check if notification exists")
		
		// Verify notification
		assert.True(t, exists, "Notification was not created")
		if exists && notification != nil {
			assert.Equal(t, targetUserID, notification.UserID, "Wrong user ID in notification")
			assert.Equal(t, string(types.UserUnfollowed), notification.Type, "Wrong notification type")
			assert.Contains(t, notification.Content, "unfollowed", "Notification content doesn't mention unfollowing")
			
			// Check metadata
			followerIDFromMetadata, hasFollowerID := notification.Metadata["userId"]
			assert.True(t, hasFollowerID, "Missing userId in notification metadata")
			assert.Equal(t, followerID.String(), followerIDFromMetadata, "Wrong userId in notification metadata")
			
			targetUserIDFromMetadata, hasTargetUserID := notification.Metadata["targetUserId"]
			assert.True(t, hasTargetUserID, "Missing targetUserId in notification metadata")
			assert.Equal(t, targetUserID.String(), targetUserIDFromMetadata, "Wrong targetUserId in notification metadata")
		}
	})
} 