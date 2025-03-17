package types

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Notification represents a notification stored in the database
// @Description A notification entity with metadata and status information
type Notification struct {
	// Unique identifier for the notification
	ID        uuid.UUID          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// User ID who should receive this notification
	UserID    uuid.UUID          `json:"userId" example:"550e8400-e29b-41d4-a716-446655440001"`
	// Type of notification (VIDEO_UPLOADED, COMMENT_CREATED, etc.)
	Type      string             `json:"type" example:"VIDEO_UPLOADED"`
	// Human-readable notification content
	Content   string             `json:"content" example:"Your video 'My awesome video' has been uploaded successfully"`
	// Additional metadata about the notification
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	// When the notification was marked as read (null if unread)
	ReadAt    *time.Time         `json:"readAt,omitempty"`
	// When the notification was created
	CreatedAt time.Time          `json:"createdAt" example:"2025-03-05T21:26:06Z"`
}

// TruncateContent truncates a string to the specified length and adds ellipsis if needed
func (n *Notification) TruncateContent(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}
	return content[:maxLength-3] + "..."
}

// NotificationRepository defines the interface for notification storage operations
type NotificationRepository interface {
	// CRUD operations
	SaveNotification(ctx context.Context, notification *Notification) error
	GetNotificationsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Notification, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	MarkAsRead(ctx context.Context, notificationID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
	
	// Resource management
	Close() error
	Ping(ctx context.Context) error
}

// ToJSON converts a notification to JSON bytes
func ToJSON(n *Notification) ([]byte, error) {
	return json.Marshal(n)
}

// FromJSON creates a notification from JSON bytes
func FromJSON(data []byte) (*Notification, error) {
	var n Notification
	if err := json.Unmarshal(data, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

// IsRead returns true if the notification has been read
func IsRead(n *Notification) bool {
	return n.ReadAt != nil
} 