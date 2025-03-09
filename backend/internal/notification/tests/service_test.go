package tests

import (
	"context"
	"testing"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/notification"
	"github.com/consensuslabs/pavilion-network/backend/testhelper"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServiceDisabled checks that the service can be created in disabled mode
func TestServiceDisabled(t *testing.T) {
	// Create a disabled config
	config := notification.DefaultConfig()
	config.Enabled = false

	// Get a test logger
	logger := testhelper.NewTestLogger(true)

	// Create the service with nil repository (disabled mode doesn't need a repository)
	service, err := notification.NewService(context.Background(), config, logger, nil)
	require.NoError(t, err)
	require.NotNil(t, service)

	// Verify it doesn't fail when used
	err = service.PublishVideoEvent(context.Background(), &notification.VideoEvent{
		BaseEvent: notification.BaseEvent{
			ID:   uuid.New(),
			Type: notification.VideoUploaded,
		},
		VideoID: uuid.New(),
		UserID:  uuid.New(),
	})
	assert.NoError(t, err)

	// Closing should not fail
	err = service.Close()
	assert.NoError(t, err)
}

// TestDefaultConfig verifies the default configuration is valid
func TestDefaultConfig(t *testing.T) {
	config := notification.DefaultConfig()
	assert.NotNil(t, config)
	assert.NotEmpty(t, config.PulsarURL)
	assert.NotEmpty(t, config.VideoEventsTopic)
	assert.NotEmpty(t, config.CommentEventsTopic)
	assert.NotEmpty(t, config.UserEventsTopic)
	assert.True(t, config.Enabled)
}

// TestServiceWithDefaultValues tries to create a service with defaults
// Note: This test will be skipped by default since it requires a running Pulsar instance
func TestServiceWithDefaultValues(t *testing.T) {
	// Skip the test if Pulsar is not running
	t.Skip("Skipping Pulsar test - requires running Pulsar instance")

	// Create a config with defaults
	config := notification.DefaultConfig()

	// Get a test logger
	logger := testhelper.NewTestLogger(true)

	// Create the service with a mock repository
	mockRepo := &MockRepository{}
	service, err := notification.NewService(context.Background(), config, logger, mockRepo)
	require.NoError(t, err)
	require.NotNil(t, service)
	defer service.Close()

	// Basic event creation test
	videoEvent := &notification.VideoEvent{
		BaseEvent: notification.BaseEvent{
			ID:             uuid.New(),
			Type:           notification.VideoUploaded,
			CreatedAt:      time.Now(),
			EventKey:       "test-video-1",
			SequenceNumber: time.Now().UnixNano(),
		},
		VideoID: uuid.New(),
		UserID:  uuid.New(),
		Title:   "Test Video",
	}

	// Try to publish an event
	err = service.PublishVideoEvent(context.Background(), videoEvent)
	assert.NoError(t, err)

	// Basic read capability tests - these are stubs for now
	notifs, err := service.GetUserNotifications(context.Background(), uuid.New(), 10, 0)
	assert.NoError(t, err)
	assert.Empty(t, notifs)

	count, err := service.GetUnreadCount(context.Background(), uuid.New())
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}