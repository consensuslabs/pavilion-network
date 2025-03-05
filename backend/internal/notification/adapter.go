package notification

import (
	"context"
	"fmt"

	"github.com/consensuslabs/pavilion-network/backend/internal/video"
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