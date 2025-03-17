package types

import (
	"context"

	"github.com/google/uuid"
)

// NotificationService defines the interface for notification operations
type NotificationService interface {
	// Publish methods for different event types
	PublishVideoEvent(ctx context.Context, event *VideoEvent) error
	PublishCommentEvent(ctx context.Context, event *CommentEvent) error
	PublishUserEvent(ctx context.Context, event *UserEvent) error

	// Get notifications for a user
	GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Notification, error)
	
	// Get count of unread notifications
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	
	// Mark notifications as read
	MarkAsRead(ctx context.Context, notificationID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
	
	// Close the service and releases resources
	Close() error
} 