// Package notification provides functionality for handling notifications
package notification

import (
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
)

// TimeFunc is a function that returns the current time
type TimeFunc func() time.Time

// Type aliases for notification types
// These aliases allow other packages to use notification types without
// directly importing the types package

// EventType represents the type of notification event
type EventType = types.EventType

// Event type constants
const (
	VideoUploaded  = types.VideoUploaded
	VideoProcessed = types.VideoProcessed
	VideoUpdated   = types.VideoUpdated
	VideoDeleted   = types.VideoDeleted
	VideoLiked     = types.VideoLiked
	VideoUnliked   = types.VideoUnliked

	CommentCreated  = types.CommentCreated
	CommentReplied  = types.CommentReplied
	CommentReaction = types.CommentReaction
	CommentMention  = types.CommentMention

	UserFollowed   = types.UserFollowed
	UserUnfollowed = types.UserUnfollowed
	UserMentioned  = types.UserMentioned
	AuthEvent      = types.AuthEvent
)

// BaseEvent contains common fields for all event types
type BaseEvent = types.BaseEvent

// VideoEvent represents a video-related notification event
type VideoEvent = types.VideoEvent

// CommentEvent represents a comment-related notification event
type CommentEvent = types.CommentEvent

// UserEvent represents a user-related notification event
type UserEvent = types.UserEvent

// Notification represents a notification stored in the database
type Notification = types.Notification

// NotificationError represents an error in the notification system
type NotificationError = types.NotificationError

// Error variables
var (
	// ErrNotificationNotFound is returned when a notification is not found
	ErrNotificationNotFound = types.ErrNotificationNotFound
	
	// ErrInvalidNotification is returned when a notification is invalid
	ErrInvalidNotification = types.ErrInvalidNotification
	
	// ErrInvalidEventType is returned when an event type is invalid
	ErrInvalidEventType = types.ErrInvalidEventType
	
	// ErrServiceDisabled is returned when the notification service is disabled
	ErrServiceDisabled = types.ErrServiceDisabled
	
	// ErrConnectionFailed is returned when a connection to the message broker fails
	ErrConnectionFailed = types.ErrConnectionFailed
	
	// NewError creates a new NotificationError
	NewError = types.NewError
)

// Helper function aliases
var (
	// ToJSON converts a notification to JSON bytes
	ToJSON = types.ToJSON
	
	// FromJSON creates a notification from JSON bytes
	FromJSON = types.FromJSON
	
	// IsRead returns true if the notification has been read
	IsRead = types.IsRead
) 