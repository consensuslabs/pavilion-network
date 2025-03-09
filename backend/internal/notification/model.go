package notification

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TimeFunc is a function that returns the current time
type TimeFunc func() time.Time

// BaseEvent contains common fields for all event types
type BaseEvent struct {
	ID             uuid.UUID       `json:"id"`
	Type           EventType       `json:"type"`
	CreatedAt      time.Time       `json:"createdAt"`
	EventKey       string          `json:"eventKey"`
	SequenceNumber int64           `json:"sequenceNumber"`
}

// VideoEvent represents a video-related notification event
type VideoEvent struct {
	BaseEvent
	VideoID   uuid.UUID          `json:"videoId"`
	UserID    uuid.UUID          `json:"userId"`
	Title     string             `json:"title,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CommentEvent represents a comment-related notification event
type CommentEvent struct {
	BaseEvent
	CommentID uuid.UUID          `json:"commentId"`
	UserID    uuid.UUID          `json:"userId"`
	VideoID   uuid.UUID          `json:"videoId,omitempty"`
	ParentID  uuid.UUID          `json:"parentId,omitempty"`
	Content   string             `json:"content,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// UserEvent represents a user-related notification event
type UserEvent struct {
	BaseEvent
	UserID       uuid.UUID          `json:"userId"`
	TargetUserID uuid.UUID          `json:"targetUserId"`
	Content      string             `json:"content,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Notification represents a notification stored in the database
// @Description A notification entity with metadata and status information
type Notification struct {
	// Unique identifier for the notification
	ID        uuid.UUID          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// User ID who should receive this notification
	UserID    uuid.UUID          `json:"userId" example:"550e8400-e29b-41d4-a716-446655440001"`
	// Type of notification (VIDEO_UPLOADED, COMMENT_CREATED, etc.)
	Type      EventType          `json:"type" example:"VIDEO_UPLOADED"`
	// Human-readable notification content
	Content   string             `json:"content" example:"Your video 'My awesome video' has been uploaded successfully"`
	// Additional metadata about the notification
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	// When the notification was marked as read (null if unread)
	ReadAt    *time.Time         `json:"readAt,omitempty"`
	// When the notification was created
	CreatedAt time.Time          `json:"createdAt" example:"2025-03-05T21:26:06Z"`
}

// ToJSON converts a notification to JSON bytes
func (n *Notification) ToJSON() ([]byte, error) {
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
func (n *Notification) IsRead() bool {
	return n.ReadAt != nil
}