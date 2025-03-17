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

// CommentConsumer consumes comment-related events and creates notifications
type CommentConsumer struct {
	*BaseConsumer
}

// NewCommentConsumer creates a new comment consumer
func NewCommentConsumer(
	client pulsar.Client,
	logger logger.Logger,
	repository types.NotificationRepository,
	config *types.ServiceConfig,
) *CommentConsumer {
	baseConsumer := NewBaseConsumer(
		client,
		config.Topics.CommentEvents,
		"comment-notification-consumer",
		logger,
		repository,
		config.Topics.DeadLetter,
		config.Topics.RetryQueue,
	)
	
	consumer := &CommentConsumer{
		BaseConsumer: baseConsumer,
	}
	
	// Set the process message function
	baseConsumer.processMessage = consumer.processCommentMessage
	
	return consumer
}

// processCommentMessage processes a comment event message
func (c *CommentConsumer) processCommentMessage(msg pulsar.Message) error {
	// Parse the comment event
	var event types.CommentEvent
	if err := json.Unmarshal(msg.Payload(), &event); err != nil {
		return fmt.Errorf("failed to unmarshal comment event: %w", err)
	}

	c.logger.LogInfo("Processing comment event", map[string]interface{}{
		"eventId":   event.ID.String(),
		"eventType": string(event.Type),
		"commentId": event.CommentID.String(),
		"userId":    event.UserID.String(),
		"videoId":   event.VideoID.String(),
	})

	// Create a notification based on the event type
	notification := &types.Notification{
		ID:        uuid.New(),
		UserID:    event.UserID,
		Type:      string(event.Type),
		CreatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"commentId": event.CommentID.String(),
			"videoId":   event.VideoID.String(),
			"eventId":   event.ID.String(),
		},
	}

	// Add parent ID to metadata if it exists
	if event.ParentID != uuid.Nil {
		notification.Metadata["parentId"] = event.ParentID.String()
	}

	// Set content based on event type
	switch event.Type {
	case types.CommentCreated:
		notification.Content = fmt.Sprintf("New comment on your video: %s", util.TruncateContent(event.Content, 50))
	case types.CommentReplied:
		notification.Content = fmt.Sprintf("New reply to your comment: %s", util.TruncateContent(event.Content, 50))
	case types.CommentReaction:
		notification.Content = "Someone reacted to your comment"
		if event.Metadata != nil {
			if reaction, ok := event.Metadata["reaction"].(string); ok {
				notification.Content = fmt.Sprintf("Someone reacted with %s to your comment", reaction)
			}
		}
	default:
		notification.Content = fmt.Sprintf("Comment event: %s", event.Type)
	}

	// Save the notification
	if err := c.repository.SaveNotification(context.Background(), notification); err != nil {
		return fmt.Errorf("failed to save notification: %w", err)
	}

	c.logger.LogInfo("Comment notification created", map[string]interface{}{
		"notificationId": notification.ID.String(),
		"userId":         notification.UserID.String(),
		"type":           string(notification.Type),
	})

	return nil
} 