package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/consensuslabs/pavilion-network/backend/internal/notification/tests/helpers"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/tests/mocks"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
)

// TestCommentConsumer_ProcessCommentMessage tests the processCommentMessage method directly
func TestCommentConsumer_ProcessCommentMessage(t *testing.T) {
	// Create mock logger and repository
	_, mockRepo := mocks.SetupConsumerTest()
	
	// Create test event
	commentID := uuid.New()
	userID := uuid.New()
	videoID := uuid.New()
	content := "This is a test comment"
	
	commentEvent := types.CommentEvent{
		BaseEvent: types.BaseEvent{
			ID:        uuid.New(),
			Type:      types.CommentCreated,
			CreatedAt: time.Now(),
		},
		CommentID: commentID,
		UserID:    userID,
		VideoID:   videoID,
		Content:   content,
	}
	
	// Serialize event
	data, err := json.Marshal(commentEvent)
	require.NoError(t, err)
	
	// Create mock message
	msg := mocks.NewMockMessage(data, commentID.String())
	msg.SetProperty("event_type", string(types.CommentCreated))
	
	// Process message
	err = helpers.ProcessCommentMessage(msg, mockRepo)
	require.NoError(t, err)
	
	// Verify notification was created
	notifications := mockRepo.GetNotifications()
	require.Len(t, notifications, 1)
	
	notif := notifications[0]
	assert.Equal(t, userID, notif.UserID)
	assert.Equal(t, string(types.CommentCreated), notif.Type)
	assert.Contains(t, notif.Content, "New comment on your video")
	assert.Contains(t, notif.Content, content)
	assert.Equal(t, commentID.String(), notif.Metadata["commentId"])
	assert.Equal(t, videoID.String(), notif.Metadata["videoId"])
}

// TestCommentConsumer_ProcessCommentReplyMessage tests the processCommentMessage method with a reply event
func TestCommentConsumer_ProcessCommentReplyMessage(t *testing.T) {
	// Create mock logger and repository
	_, mockRepo := mocks.SetupConsumerTest()
	
	// Create test event
	commentID := uuid.New()
	parentID := uuid.New()
	userID := uuid.New()
	videoID := uuid.New()
	content := "This is a test reply"
	
	commentEvent := types.CommentEvent{
		BaseEvent: types.BaseEvent{
			ID:        uuid.New(),
			Type:      types.CommentReplied,
			CreatedAt: time.Now(),
		},
		CommentID: commentID,
		ParentID:  parentID,
		UserID:    userID,
		VideoID:   videoID,
		Content:   content,
	}
	
	// Serialize event
	data, err := json.Marshal(commentEvent)
	require.NoError(t, err)
	
	// Create mock message
	msg := mocks.NewMockMessage(data, commentID.String())
	msg.SetProperty("event_type", string(types.CommentReplied))
	
	// Process message
	err = helpers.ProcessCommentMessage(msg, mockRepo)
	require.NoError(t, err)
	
	// Verify notification was created
	notifications := mockRepo.GetNotifications()
	require.Len(t, notifications, 1)
	
	notif := notifications[0]
	assert.Equal(t, userID, notif.UserID)
	assert.Equal(t, string(types.CommentReplied), notif.Type)
	assert.Contains(t, notif.Content, "New reply to your comment")
	assert.Contains(t, notif.Content, content)
	assert.Equal(t, commentID.String(), notif.Metadata["commentId"])
	assert.Equal(t, videoID.String(), notif.Metadata["videoId"])
	assert.Equal(t, parentID.String(), notif.Metadata["parentId"])
} 