package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
	"github.com/google/uuid"
)

// VideoConsumer consumes video-related events and creates notifications
type VideoConsumer struct {
	*BaseConsumer
}

// NewVideoConsumer creates a new video consumer
func NewVideoConsumer(
	client pulsar.Client,
	logger logger.Logger,
	repository types.NotificationRepository,
	config *types.ServiceConfig,
) *VideoConsumer {
	baseConsumer := NewBaseConsumer(
		client,
		config.Topics.VideoEvents,
		"video-notification-consumer",
		logger,
		repository,
		config.Topics.DeadLetter,
		config.Topics.RetryQueue,
	)
	
	consumer := &VideoConsumer{
		BaseConsumer: baseConsumer,
	}
	
	// Set the process message function
	baseConsumer.processMessage = consumer.processVideoMessage
	
	return consumer
}

// processVideoMessage processes a video event message
func (c *VideoConsumer) processVideoMessage(msg pulsar.Message) error {
	// Parse the video event
	var event types.VideoEvent
	if err := json.Unmarshal(msg.Payload(), &event); err != nil {
		return fmt.Errorf("failed to unmarshal video event: %w", err)
	}

	c.logger.LogInfo("Processing video event", map[string]interface{}{
		"eventId":   event.ID.String(),
		"eventType": string(event.Type),
		"videoId":   event.VideoID.String(),
		"userId":    event.UserID.String(),
	})

	// Create a notification based on the event type
	notification := &types.Notification{
		ID:        uuid.New(),
		UserID:    event.UserID,
		Type:      string(event.Type),
		CreatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"videoId": event.VideoID.String(),
			"eventId": event.ID.String(),
		},
	}

	// Set content based on event type
	switch event.Type {
	case types.VideoUploaded:
		notification.Content = fmt.Sprintf("Your video '%s' has been uploaded successfully", event.Title)
	case types.VideoProcessed:
		notification.Content = fmt.Sprintf("Your video '%s' has been processed and is now available", event.Title)
	case types.VideoUpdated:
		notification.Content = fmt.Sprintf("Your video '%s' has been updated", event.Title)
	case types.VideoDeleted:
		notification.Content = fmt.Sprintf("Your video '%s' has been deleted", event.Title)
	default:
		notification.Content = fmt.Sprintf("Video event: %s", event.Type)
	}

	// Save the notification
	if err := c.repository.SaveNotification(context.Background(), notification); err != nil {
		return fmt.Errorf("failed to save notification: %w", err)
	}

	c.logger.LogInfo("Video notification created", map[string]interface{}{
		"notificationId": notification.ID.String(),
		"userId":         notification.UserID.String(),
		"type":           string(notification.Type),
	})

	return nil
}

// ProcessMessage processes a message directly (for testing)
func (c *VideoConsumer) ProcessMessage(msg pulsar.Message) error {
	return c.processVideoMessage(msg)
} 