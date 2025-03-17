package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/producer"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/tests/mocks"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProducerConsumerIntegration tests the full producer-consumer cycle with real Pulsar
func TestProducerConsumerIntegration(t *testing.T) {
	// Skip the test since it requires a running Pulsar instance
	t.Skip("Skipping Pulsar integration test - requires running Pulsar instance")

	// Check if we're in CI environment - if so, skip the test
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create Pulsar client
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:               "pulsar://localhost:6650",
		OperationTimeout:  30 * time.Second,
		ConnectionTimeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create Pulsar client: %v", err)
	}
	defer client.Close()

	// Create a mock logger
	mockLogger := mocks.SetupTestLogger()

	// Create a mock repository to store notifications
	mockRepo := mocks.SetupTestRepository()

	// Create service config
	config := &types.ServiceConfig{}
	config.Topics.VideoEvents = "persistent://pavilion/notifications/video-events-test"
	config.Topics.CommentEvents = "persistent://pavilion/notifications/comment-events-test"
	config.Topics.UserEvents = "persistent://pavilion/notifications/user-events-test"
	config.Topics.DeadLetter = "persistent://pavilion/notifications/dead-letter-test"
	config.Topics.RetryQueue = "persistent://pavilion/notifications/retry-queue-test"

	// Create the test topics if they don't exist
	createTestTopics(t, client, config)

	// Create producer and consumer
	videoProducer, err := producer.NewVideoProducer(client, mockLogger, mockRepo, config)
	require.NoError(t, err)
	defer videoProducer.Close()

	// Create test data
	userID := uuid.New()
	videoID := uuid.New()
	title := "Test Integration Video"

	// Create a channel to receive notifications
	notificationCh := make(chan *types.Notification, 1)
	
	// Create a consumer to listen for notifications
	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:            config.Topics.VideoEvents,
		SubscriptionName: "test-integration-sub",
		Type:             pulsar.Exclusive,
	})
	require.NoError(t, err)
	defer consumer.Close()

	// Start a goroutine to consume messages
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	var wg sync.WaitGroup
	wg.Add(1)
	
	go func() {
		defer wg.Done()
		
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := consumer.Receive(ctx)
				if err != nil {
					if err == context.DeadlineExceeded {
						return
					}
					t.Logf("Error receiving message: %v", err)
					continue
				}
				
				// Process the message
				notification, err := processVideoMessage(msg, mockRepo)
				if err != nil {
					t.Logf("Error processing message: %v", err)
					consumer.Nack(msg)
					continue
				}
				
				// Send the notification to the channel
				notificationCh <- notification
				
				// Acknowledge the message
				consumer.Ack(msg)
			}
		}
	}()

	// Publish a video event
	err = videoProducer.PublishVideoUploadEvent(context.Background(), userID, videoID, title)
	require.NoError(t, err)
	
	// Wait for the notification to be received or timeout
	select {
	case notification := <-notificationCh:
		// Verify the notification
		assert.Equal(t, userID, notification.UserID)
		assert.Equal(t, string(types.VideoUploaded), notification.Type)
		assert.Contains(t, notification.Content, title)
		assert.Equal(t, videoID.String(), notification.Metadata["videoId"])
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for notification")
	}
	
	// Cancel the consumer goroutine and wait for it to finish
	cancel()
	wg.Wait()
}

// ParseVideoEvent parses a video event from a byte array
func ParseVideoEvent(data []byte) (*types.VideoEvent, error) {
	var event types.VideoEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal video event: %w", err)
	}
	return &event, nil
}

// Helper function to process a video message and return the notification
func processVideoMessage(msg pulsar.Message, repo *mocks.MockRepository) (*types.Notification, error) {
	// Parse the video event from the message
	event, err := ParseVideoEvent(msg.Payload())
	if err != nil {
		return nil, err
	}
	
	// Create a notification based on the event
	notification := &types.Notification{
		ID:        uuid.New(),
		UserID:    event.UserID,
		Type:      string(event.Type),
		CreatedAt: time.Now(),
		Content:   "Your video '" + event.Title + "' has been uploaded successfully",
		Metadata: map[string]interface{}{
			"videoId": event.VideoID.String(),
			"eventId": event.ID.String(),
		},
	}
	
	// Save the notification
	err = repo.SaveNotification(context.Background(), notification)
	if err != nil {
		return nil, err
	}
	
	return notification, nil
}

// Helper function to create test topics
func createTestTopics(t *testing.T, client pulsar.Client, config *types.ServiceConfig) {
	// Skip topic creation since we're using auto-created topics
	t.Log("Skipping explicit topic creation - using auto-created topics")
	
	// In a real environment, you might use the Pulsar admin API to create topics
	// with specific configurations, but for tests, auto-created topics are sufficient
}