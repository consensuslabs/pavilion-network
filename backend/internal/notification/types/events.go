package types

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of notification event
type EventType string

const (
	// Video related events
	VideoUploaded  EventType = "VIDEO_UPLOADED"
	VideoProcessed EventType = "VIDEO_PROCESSED"
	VideoUpdated   EventType = "VIDEO_UPDATED"
	VideoDeleted   EventType = "VIDEO_DELETED"
	VideoLiked     EventType = "VIDEO_LIKED"
	VideoUnliked   EventType = "VIDEO_UNLIKED"

	// Comment related events
	CommentCreated  EventType = "COMMENT_CREATED"
	CommentReplied  EventType = "COMMENT_REPLIED"
	CommentReaction EventType = "COMMENT_REACTION"
	CommentMention  EventType = "COMMENT_MENTION"

	// User related events
	UserFollowed   EventType = "USER_FOLLOWED"
	UserUnfollowed EventType = "USER_UNFOLLOWED"
	UserMentioned  EventType = "USER_MENTIONED"
	AuthEvent      EventType = "AUTH_EVENT"
)

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