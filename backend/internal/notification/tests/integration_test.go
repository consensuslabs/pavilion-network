package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification"
	"github.com/consensuslabs/pavilion-network/backend/testhelper"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProducerConsumerIntegration tests the full producer-consumer cycle
// Note: This test will be skipped by default since it requires a running Pulsar instance
func TestProducerConsumerIntegration(t *testing.T) {
	// Skip the test if Pulsar is not running
	t.Skip("Skipping Pulsar integration test - requires running Pulsar instance")

	ctx := context.Background()
	logger := testhelper.NewTestLogger(true)

	// Create configuration
	config := notification.DefaultConfig()

	// Create a mock repository
	repository := NewMockRepository()

	// Create the notification service (producer)
	service, err := notification.NewService(ctx, config, logger, repository)
	require.NoError(t, err)
	defer service.Close()

	// Create a mock consumer to verify messages
	consumer, err := NewMockConsumer(config, config.VideoEventsTopic, logger)
	require.NoError(t, err)
	defer consumer.Close()

	// Sleep a bit to ensure consumer is ready
	time.Sleep(500 * time.Millisecond)

	// Create a unique video ID for this test
	videoID := uuid.New()
	userID := uuid.New()
	title := "Test Video for Integration Test"

	// Create and send video event
	videoEvent := &notification.VideoEvent{
		BaseEvent: notification.BaseEvent{
			ID:             uuid.New(),
			Type:           notification.VideoUploaded,
			CreatedAt:      time.Now(),
			EventKey:       videoID.String(),
			SequenceNumber: time.Now().UnixNano(),
		},
		VideoID:  videoID,
		UserID:   userID,
		Title:    title,
		Metadata: map[string]interface{}{"test": true},
	}

	// Publish the event
	err = service.PublishVideoEvent(ctx, videoEvent)
	require.NoError(t, err)

	// Wait for message to be processed (with timeout)
	timeout := time.After(5 * time.Second)
	var receivedMessages []pulsar.Message

	success := false
	for !success {
		select {
		case <-timeout:
			t.Fatalf("Timed out waiting for message to be processed")
			return
		default:
			receivedMessages = consumer.GetMessages()
			if len(receivedMessages) > 0 {
				success = true
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	// Verify we received the message
	assert.GreaterOrEqual(t, len(receivedMessages), 1)
	
	// Check the content of the message
	msg := receivedMessages[0]
	assert.Equal(t, videoID.String(), msg.Key())
	
	// Verify properties
	assert.Equal(t, string(notification.VideoUploaded), msg.Properties()["eventType"])
	assert.Equal(t, videoID.String(), msg.Properties()["videoId"])
	assert.Equal(t, userID.String(), msg.Properties()["userId"])
	
	// Verify payload
	var receivedEvent notification.VideoEvent
	err = json.Unmarshal(msg.Payload(), &receivedEvent)
	require.NoError(t, err)
	
	assert.Equal(t, videoEvent.ID, receivedEvent.ID)
	assert.Equal(t, videoEvent.Type, receivedEvent.Type)
	assert.Equal(t, videoEvent.VideoID, receivedEvent.VideoID)
	assert.Equal(t, videoEvent.UserID, receivedEvent.UserID)
	assert.Equal(t, videoEvent.Title, receivedEvent.Title)
	assert.Equal(t, videoEvent.EventKey, receivedEvent.EventKey)
}