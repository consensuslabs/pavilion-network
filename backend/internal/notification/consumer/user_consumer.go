package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/util"
	"github.com/google/uuid"
)

// UserConsumer consumes user-related events and creates notifications
type UserConsumer struct {
	*BaseConsumer
}

// NewUserConsumer creates a new user consumer
func NewUserConsumer(
	client pulsar.Client,
	logger logger.Logger,
	repository types.NotificationRepository,
	config *types.ServiceConfig,
) *UserConsumer {
	baseConsumer := NewBaseConsumer(
		client,
		config.Topics.UserEvents,
		"user-notification-consumer",
		logger,
		repository,
		config.Topics.DeadLetter,
		config.Topics.RetryQueue,
	)
	
	consumer := &UserConsumer{
		BaseConsumer: baseConsumer,
	}
	
	// Set the process message function
	baseConsumer.processMessage = consumer.processUserMessage
	
	return consumer
}

// processUserMessage processes a user event message
func (c *UserConsumer) processUserMessage(msg pulsar.Message) error {
	// Parse the user event
	var event types.UserEvent
	if err := json.Unmarshal(msg.Payload(), &event); err != nil {
		return fmt.Errorf("failed to unmarshal user event: %w", err)
	}

	c.logger.LogInfo("Processing user event", map[string]interface{}{
		"eventId":      event.ID.String(),
		"eventType":    string(event.Type),
		"userId":       event.UserID.String(),
		"targetUserId": event.TargetUserID.String(),
	})

	// Create a notification based on the event type
	notification := &types.Notification{
		ID:        uuid.New(),
		UserID:    event.TargetUserID, // Target user receives the notification
		Type:      string(event.Type),
		CreatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"userId":       event.UserID.String(),
			"targetUserId": event.TargetUserID.String(),
			"eventId":      event.ID.String(),
		},
	}

	// Set content based on event type
	switch event.Type {
	case types.UserFollowed:
		notification.Content = "Someone started following you"
		if event.Metadata != nil {
			if username, ok := event.Metadata["username"].(string); ok {
				notification.Content = fmt.Sprintf("%s started following you", username)
			}
		}
	case types.UserUnfollowed:
		notification.Content = "Someone unfollowed you"
		if event.Metadata != nil {
			if username, ok := event.Metadata["username"].(string); ok {
				notification.Content = fmt.Sprintf("%s unfollowed you", username)
			}
		}
	case types.UserMentioned:
		notification.Content = "Someone mentioned you in a comment"
		if event.Content != "" {
			notification.Content = fmt.Sprintf("You were mentioned in a comment: %s", util.TruncateContent(event.Content, 50))
		}
	case types.AuthEvent:
		notification.Content = "New login detected on your account"
		if event.Metadata != nil {
			if device, ok := event.Metadata["device"].(string); ok {
				notification.Content = fmt.Sprintf("New login detected on your account from %s", device)
			}
		}
	default:
		notification.Content = fmt.Sprintf("User event: %s", event.Type)
	}

	// Save the notification
	if err := c.repository.SaveNotification(context.Background(), notification); err != nil {
		return fmt.Errorf("failed to save notification: %w", err)
	}

	c.logger.LogInfo("User notification created", map[string]interface{}{
		"notificationId": notification.ID.String(),
		"userId":         notification.UserID.String(),
		"type":           string(notification.Type),
	})

	return nil
}
