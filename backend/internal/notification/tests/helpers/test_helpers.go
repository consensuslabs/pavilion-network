package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/tests/mocks"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/util"
	"github.com/google/uuid"
)

// ProcessVideoMessage is a helper function for processing video messages in tests
func ProcessVideoMessage(msg pulsar.Message, repo *mocks.MockRepository) error {
	// Parse the video event
	var event types.VideoEvent
	if err := json.Unmarshal(msg.Payload(), &event); err != nil {
		return fmt.Errorf("failed to unmarshal video event: %w", err)
	}
	
	// Create a notification based on the event
	notification := &types.Notification{
		ID:        uuid.New(),
		UserID:    event.UserID,
		Type:      string(event.Type),
		CreatedAt: time.Now(),
		Content:   fmt.Sprintf("Your video '%s' has been uploaded successfully", event.Title),
		Metadata: map[string]interface{}{
			"videoId": event.VideoID.String(),
			"eventId": event.ID.String(),
		},
	}
	
	// Save the notification
	return repo.SaveNotification(context.Background(), notification)
}

// ProcessCommentMessage is a helper function for processing comment messages in tests
func ProcessCommentMessage(msg pulsar.Message, repo *mocks.MockRepository) error {
	// Parse the comment event
	var event types.CommentEvent
	if err := json.Unmarshal(msg.Payload(), &event); err != nil {
		return fmt.Errorf("failed to unmarshal comment event: %w", err)
	}
	
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
	default:
		notification.Content = fmt.Sprintf("Comment event: %s", event.Type)
	}
	
	// Save the notification
	return repo.SaveNotification(context.Background(), notification)
}

// ProcessUserMessage is a helper function for processing user messages in tests
func ProcessUserMessage(msg pulsar.Message, repo *mocks.MockRepository) error {
	// Parse the user event
	var event types.UserEvent
	if err := json.Unmarshal(msg.Payload(), &event); err != nil {
		return fmt.Errorf("failed to unmarshal user event: %w", err)
	}
	
	// Create a notification based on the event type
	notification := &types.Notification{
		ID:        uuid.New(),
		UserID:    event.TargetUserID, // Target user receives the notification
		Type:      string(event.Type),
		CreatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"userId":       event.UserID.String(),
			"targetUserId": event.TargetUserID.String(),
			"eventId":      event.ID.String(),
		},
	}
	
	// Set content based on event type
	switch event.Type {
	case types.UserFollowed:
		notification.Content = "Someone started following you"
		if event.Metadata != nil {
			if username, ok := event.Metadata["username"].(string); ok {
				notification.Content = fmt.Sprintf("%s started following you", username)
			}
		}
	case types.UserUnfollowed:
		notification.Content = "Someone unfollowed you"
		if event.Metadata != nil {
			if username, ok := event.Metadata["username"].(string); ok {
				notification.Content = fmt.Sprintf("%s unfollowed you", username)
			}
		}
	case types.UserMentioned:
		notification.Content = "Someone mentioned you in a comment"
		if event.Content != "" {
			notification.Content = fmt.Sprintf("You were mentioned in a comment: %s", util.TruncateContent(event.Content, 50))
		}
	default:
		notification.Content = fmt.Sprintf("User event: %s", event.Type)
	}
	
	// Save the notification
	return repo.SaveNotification(context.Background(), notification)
}

// CreateTestNotification creates a notification for testing
func CreateTestNotification(userID uuid.UUID) *types.Notification {
	return &types.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      "TEST_NOTIFICATION",
		Content:   "Test notification content",
		Metadata:  map[string]interface{}{"test": true},
		CreatedAt: time.Now(),
	}
}

// CreateTestVideoEvent creates a video event for testing
func CreateTestVideoEvent(userID, videoID uuid.UUID, title string) *types.VideoEvent {
	return &types.VideoEvent{
		BaseEvent: types.BaseEvent{
			ID:             uuid.New(),
			Type:           "VIDEO_UPLOADED",
			CreatedAt:      time.Now(),
			EventKey:       videoID.String(),
			SequenceNumber: time.Now().UnixNano(),
		},
		VideoID:  videoID,
		UserID:   userID,
		Title:    title,
		Metadata: map[string]interface{}{"test": true},
	}
}

// CreateTestCommentEvent creates a comment event for testing
func CreateTestCommentEvent(userID, videoID, commentID uuid.UUID, content string) *types.CommentEvent {
	return &types.CommentEvent{
		BaseEvent: types.BaseEvent{
			ID:             uuid.New(),
			Type:           "COMMENT_CREATED",
			CreatedAt:      time.Now(),
			EventKey:       commentID.String(),
			SequenceNumber: time.Now().UnixNano(),
		},
		VideoID:   videoID,
		UserID:    userID,
		CommentID: commentID,
		Content:   content,
		Metadata:  map[string]interface{}{"test": true},
	}
}

// CreateTestUserEvent creates a user event for testing
func CreateTestUserEvent(userID, targetUserID uuid.UUID) *types.UserEvent {
	return &types.UserEvent{
		BaseEvent: types.BaseEvent{
			ID:             uuid.New(),
			Type:           "USER_FOLLOWED",
			CreatedAt:      time.Now(),
			EventKey:       targetUserID.String(),
			SequenceNumber: time.Now().UnixNano(),
		},
		UserID:       userID,
		TargetUserID: targetUserID,
		Metadata:     map[string]interface{}{"username": "testuser"},
	}
}