package notification

import (
	"context"

	"github.com/google/uuid"
)

// EventType represents the type of notification event
// @Description Type of notification event (e.g. VIDEO_UPLOADED, COMMENT_CREATED)
type EventType string

const (
	// Video related events
	VideoUploaded  EventType = "VIDEO_UPLOADED"
	VideoProcessed EventType = "VIDEO_PROCESSED"
	VideoUpdated   EventType = "VIDEO_UPDATED"
	VideoDeleted   EventType = "VIDEO_DELETED"

	// Comment related events
	CommentCreated  EventType = "COMMENT_CREATED"
	CommentReplied  EventType = "COMMENT_REPLIED"
	CommentReaction EventType = "COMMENT_REACTION"

	// User related events
	UserFollowed  EventType = "USER_FOLLOWED"
	UserMentioned EventType = "USER_MENTIONED"
	AuthEvent     EventType = "AUTH_EVENT"
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
	
	// Close the service and release resources
	Close() error
}

// NotificationRepository defines the interface for notification storage operations
type NotificationRepository interface {
	// CRUD operations
	SaveNotification(ctx context.Context, notification *Notification) error
	GetNotificationsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Notification, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	MarkAsRead(ctx context.Context, notificationID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
}