# Notification System

This package provides a notification system for the Pavilion Network platform.

## Package Structure

The notification package is organized into several sub-packages:

- `types`: Contains domain models, interfaces, and error types
- `consumer`: Handles consuming events from Pulsar and creating notifications
- `producer`: Handles publishing events to Pulsar
- `util`: Contains utility functions for message handling and content formatting
- `tests`: Contains unit and integration tests

## Key Components

### Interfaces

- `NotificationService`: Core service interface for notification operations
- `NotificationRepository`: Interface for notification storage operations
- `Consumer`: Interface for event consumers
- `Producer`: Interface for event producers

### Domain Models

- `Notification`: Represents a notification entity with metadata and status information
  - Methods:
    - `TruncateContent(content string, maxLength int) string`: Truncates content to the specified length and adds ellipsis if needed
    - `IsRead() bool`: Checks if the notification has been read

- `Event Types`: Various event types (VideoEvent, CommentEvent, UserEvent)

### Utility Functions

The `util` package provides several utility functions:

- Message handling:
  - `GenerateEventID(id uuid.UUID) uuid.UUID`: Generates a new event ID if not provided
  - `GenerateEventTime(t time.Time) time.Time`: Generates a new event time if not provided
  - `GenerateSequenceNumber(seq int64) int64`: Generates a new sequence number if not provided
  - `CreateProducerMessage(event interface{}, key string, properties map[string]string, eventTime time.Time) (*pulsar.ProducerMessage, error)`: Creates a producer message from an event
  - `TruncateContent(content string, maxLength int) string`: Truncates content to the specified length and adds ellipsis if needed

- Content formatting:
  - `FormatVideoContent(title string, action string) string`: Formats content for video notifications
  - `FormatCommentContent(content string, action string, maxLength int) string`: Formats content for comment notifications
  - `FormatUserContent(username string, action string) string`: Formats content for user notifications

## Event Types

### Video Events
- `VideoUploaded`: When a new video is uploaded
- `VideoProcessed`: When video processing is complete
- `VideoUpdated`: When video metadata is updated
- `VideoDeleted`: When a video is deleted
- `VideoLiked`: When a video receives a like
- `VideoUnliked`: When a like is removed from a video

### Comment Events
- `CommentCreated`: When a comment is posted
- `CommentReplied`: When a reply is added to a comment
- `CommentReaction`: When a reaction is added to a comment
- `CommentMention`: When a user is mentioned in a comment

### User Events
- `UserFollowed`: When a user is followed
- `UserUnfollowed`: When a user is unfollowed
- `UserMentioned`: When a user is mentioned
- `AuthEvent`: Authentication-related events

## Configuration

The notification system uses a structured configuration approach with the `ServiceConfig` type:

```go
type ServiceConfig struct {
    // General settings
    Enabled bool
    
    // Pulsar connection settings
    BrokerURL     string
    OperationTimeout time.Duration
    ConnectionTimeout time.Duration
    
    // Security settings
    TLSEnabled bool
    TLSCertPath string
    TLSKeyPath string
    
    // Topic configuration
    Topics struct {
        VideoEvents   string
        CommentEvents string
        UserEvents    string
        DeadLetter    string
        RetryQueue    string
    }
    
    // Retention settings
    RetentionDays int
    
    // Deduplication settings
    DeduplicationEnabled bool
    DeduplicationWindow time.Duration
    
    // Resilience settings
    MaxRetries     int
    RetryDelay     time.Duration
    BackoffFactor  float64
}
```

## Usage

### Publishing Events

```go
// Create a video event
videoEvent := &notification.VideoEvent{
    ID:        uuid.New(),
    Type:      notification.VideoUploaded,
    VideoID:   videoID,
    UserID:    userID,
    CreatedAt: time.Now(),
}

// Publish the event
err := notificationService.PublishVideoEvent(ctx, videoEvent)
```

### Retrieving Notifications

```go
// Get notifications for a user
notifications, err := notificationService.GetUserNotifications(ctx, userID, 10, 0)

// Mark a notification as read
err := notificationService.MarkAsRead(ctx, notificationID)

// Mark all notifications as read
err := notificationService.MarkAllAsRead(ctx, userID)
```

## Error Handling

The notification system uses standardized error types for consistent error handling:

```go
var (
    ErrNotificationNotFound = errors.New("notification not found")
    ErrInvalidNotification  = errors.New("invalid notification")
    ErrInvalidEventType     = errors.New("invalid event type")
    ErrServiceDisabled      = errors.New("notification service is disabled")
    ErrConnectionFailed     = errors.New("failed to connect to message broker")
)
```

You can create custom errors using the `NotificationError` type:

```go
err := notification.NewError(
    notification.ErrConnectionFailed,
    "PublishVideoEvent",
    "Failed to connect to Pulsar broker"
)
``` 