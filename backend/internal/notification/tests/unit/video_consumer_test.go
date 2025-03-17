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

// TestVideoConsumer_ProcessVideoMessage tests the processVideoMessage method directly
func TestVideoConsumer_ProcessVideoMessage(t *testing.T) {
	// Create mock logger and repository
	_, mockRepo := mocks.SetupConsumerTest()
	
	// Create test event
	videoID := uuid.New()
	userID := uuid.New()
	title := "Test Video"
	
	videoEvent := types.VideoEvent{
		BaseEvent: types.BaseEvent{
			ID:        uuid.New(),
			Type:      types.VideoUploaded,
			CreatedAt: time.Now(),
		},
		VideoID: videoID,
		UserID:  userID,
		Title:   title,
	}
	
	// Serialize event
	data, err := json.Marshal(videoEvent)
	require.NoError(t, err)
	
	// Create mock message
	msg := mocks.NewMockMessage(data, videoID.String())
	msg.SetProperty("event_type", string(types.VideoUploaded))
	
	// Process message
	err = helpers.ProcessVideoMessage(msg, mockRepo)
	require.NoError(t, err)
	
	// Verify notification was created
	notifications := mockRepo.GetNotifications()
	require.Len(t, notifications, 1)
	
	notif := notifications[0]
	assert.Equal(t, userID, notif.UserID)
	assert.Equal(t, string(types.VideoUploaded), notif.Type)
	assert.Contains(t, notif.Content, title)
}

// TestVideoConsumer_ProcessInvalidMessage tests handling of invalid messages
func TestVideoConsumer_ProcessInvalidMessage(t *testing.T) {
	// Create mock logger and repository
	_, mockRepo := mocks.SetupConsumerTest()
	
	// Create invalid payload (not valid JSON)
	invalidPayload := []byte("this is not valid JSON")
	
	// Create mock message
	msg := mocks.NewMockMessage(invalidPayload, "invalid-key")
	msg.SetProperty("event_type", string(types.VideoUploaded))
	
	// Process message - should return an error
	err := helpers.ProcessVideoMessage(msg, mockRepo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal video event")
	
	// Verify no notification was created
	notifications := mockRepo.GetNotifications()
	assert.Empty(t, notifications)
} 