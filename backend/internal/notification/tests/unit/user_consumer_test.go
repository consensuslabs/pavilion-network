package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/notification/tests/helpers"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/tests/mocks"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserConsumer_ProcessUserFollowedMessage tests the processUserMessage method directly
func TestUserConsumer_ProcessUserFollowedMessage(t *testing.T) {
	// Create mock logger and repository
	_, mockRepo := mocks.SetupConsumerTest()
	
	// Create test event
	userID := uuid.New()
	targetUserID := uuid.New()
	
	userEvent := types.UserEvent{
		BaseEvent: types.BaseEvent{
			ID:        uuid.New(),
			Type:      types.UserFollowed,
			CreatedAt: time.Now(),
		},
		UserID:       userID,
		TargetUserID: targetUserID,
	}
	
	// Serialize event
	data, err := json.Marshal(userEvent)
	require.NoError(t, err)
	
	// Create mock message
	msg := mocks.NewMockMessage(data, userID.String())
	msg.SetProperty("event_type", string(types.UserFollowed))
	
	// Process message
	err = helpers.ProcessUserMessage(msg, mockRepo)
	require.NoError(t, err)
	
	// Verify notification was created
	notifications := mockRepo.GetNotifications()
	require.Len(t, notifications, 1)
	
	notif := notifications[0]
	assert.Equal(t, targetUserID, notif.UserID) // Target user receives the notification
	assert.Equal(t, string(types.UserFollowed), notif.Type)
	assert.Contains(t, notif.Content, "Someone started following you")
	assert.Equal(t, userID.String(), notif.Metadata["userId"])
	assert.Equal(t, targetUserID.String(), notif.Metadata["targetUserId"])
}

// TestUserConsumer_ProcessUserMentionedMessage tests the processUserMessage method with a mention event
func TestUserConsumer_ProcessUserMentionedMessage(t *testing.T) {
	// Create mock logger and repository
	_, mockRepo := mocks.SetupConsumerTest()
	
	// Create test event
	userID := uuid.New()
	targetUserID := uuid.New()
	content := "Hey @user, check this out!"
	
	userEvent := types.UserEvent{
		BaseEvent: types.BaseEvent{
			ID:        uuid.New(),
			Type:      types.UserMentioned,
			CreatedAt: time.Now(),
		},
		UserID:       userID,
		TargetUserID: targetUserID,
		Content:      content,
	}
	
	// Serialize event
	data, err := json.Marshal(userEvent)
	require.NoError(t, err)
	
	// Create mock message
	msg := mocks.NewMockMessage(data, userID.String())
	msg.SetProperty("event_type", string(types.UserMentioned))
	
	// Process message
	err = helpers.ProcessUserMessage(msg, mockRepo)
	require.NoError(t, err)
	
	// Verify notification was created
	notifications := mockRepo.GetNotifications()
	require.Len(t, notifications, 1)
	
	notif := notifications[0]
	assert.Equal(t, targetUserID, notif.UserID) // Target user receives the notification
	assert.Equal(t, string(types.UserMentioned), notif.Type)
	assert.Contains(t, notif.Content, "You were mentioned in a comment")
	assert.Contains(t, notif.Content, content)
	assert.Equal(t, userID.String(), notif.Metadata["userId"])
	assert.Equal(t, targetUserID.String(), notif.Metadata["targetUserId"])
} 