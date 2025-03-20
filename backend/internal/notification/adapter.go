package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/google/uuid"
)

// VideoNotificationAdapter wraps a NotificationService to be used from the video package
type VideoNotificationAdapter struct {
	service NotificationService
}

// NewVideoNotificationAdapter creates a new adapter for video notifications
func NewVideoNotificationAdapter(service NotificationService) *VideoNotificationAdapter {
	return &VideoNotificationAdapter{
		service: service,
	}
}

// PublishVideoEvent adapts the video.VideoEvent to notification.VideoEvent
func (a *VideoNotificationAdapter) PublishVideoEvent(ctx interface{}, event interface{}) error {
	// Convert context to proper type
	ctxVal, ok := ctx.(context.Context)
	if !ok {
		return fmt.Errorf("invalid context type")
	}

	// Convert event to proper type
	videoEvent, ok := event.(*video.VideoEvent)
	if !ok {
		return fmt.Errorf("invalid event type, expected *video.VideoEvent")
	}

	// Create a notification.VideoEvent from video.VideoEvent
	notifEvent := &VideoEvent{
		BaseEvent: BaseEvent{
			ID:   videoEvent.ID,
			Type: EventType(videoEvent.Type),
		},
		VideoID:  videoEvent.VideoID,
		UserID:   videoEvent.UserID,
		Title:    videoEvent.Title,
		Metadata: videoEvent.Metadata,
	}

	// Publish the notification event
	return a.service.PublishVideoEvent(ctxVal, notifEvent)
}

// CommentNotificationAdapter wraps a NotificationService to be used from the comment package
type CommentNotificationAdapter struct {
	service NotificationService
}

// NewCommentNotificationAdapter creates a new adapter for comment notifications
func NewCommentNotificationAdapter(service NotificationService) *CommentNotificationAdapter {
	return &CommentNotificationAdapter{
		service: service,
	}
}

// PublishCommentCreatedEvent publishes a comment creation event
func (a *CommentNotificationAdapter) PublishCommentCreatedEvent(ctx context.Context, userID, videoID, commentID uuid.UUID, content string) error {
	event := &CommentEvent{
		BaseEvent: BaseEvent{
			ID:        uuid.New(),
			Type:      CommentCreated,
			CreatedAt: time.Now(),
		},
		CommentID: commentID,
		UserID:    userID,
		VideoID:   videoID,
		Content:   content,
	}

	return a.service.PublishCommentEvent(ctx, event)
}

// PublishCommentReplyEvent publishes a comment reply event
func (a *CommentNotificationAdapter) PublishCommentReplyEvent(ctx context.Context, userID, videoID, commentID, parentID uuid.UUID, content string) error {
	event := &CommentEvent{
		BaseEvent: BaseEvent{
			ID:        uuid.New(),
			Type:      CommentReplied,
			CreatedAt: time.Now(),
		},
		CommentID: commentID,
		UserID:    userID,
		VideoID:   videoID,
		ParentID:  parentID,
		Content:   content,
	}

	return a.service.PublishCommentEvent(ctx, event)
}