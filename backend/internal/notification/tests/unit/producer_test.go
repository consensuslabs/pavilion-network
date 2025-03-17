package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/notification/producer"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/tests/mocks"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// setupProducerTest creates and configures common mocks for producer tests
func setupProducerTest() (*mocks.MockPulsarClient, *mocks.MockPulsarProducer, *mocks.MockLogger, *mocks.MockRepository, *mocks.MockMessageID, *mocks.MockTableView, *mocks.MockTransaction, *mocks.MockTxnID, *types.ServiceConfig) {
	mockClient := new(mocks.MockPulsarClient)
	mockProducer := new(mocks.MockPulsarProducer)
	mockLogger := mocks.SetupTestLogger()
	mockRepo := mocks.SetupTestRepository()
	mockMsgID := new(mocks.MockMessageID)
	mockTableView := new(mocks.MockTableView)
	mockTransaction := new(mocks.MockTransaction)
	mockTxnID := new(mocks.MockTxnID)
	
	// Configure mocks
	mockClient.On("CreateProducer", mock.Anything).Return(mockProducer, nil)
	mockClient.On("CreateTableView", mock.Anything).Return(mockTableView, nil)
	mockClient.On("NewTransaction", mock.Anything).Return(mockTransaction, nil)
	mockProducer.On("Send", mock.Anything, mock.Anything).Return(mockMsgID, nil)
	mockProducer.On("Close").Return()
	mockTransaction.On("GetTxnID").Return(mockTxnID)
	
	// Create config with test topics
	config := &types.ServiceConfig{}
	
	return mockClient, mockProducer, mockLogger, mockRepo, mockMsgID, mockTableView, mockTransaction, mockTxnID, config
}

// TestVideoProducer_Publish tests the VideoProducer's Publish method
func TestVideoProducer_Publish(t *testing.T) {
	// Setup test fixtures
	mockClient, mockProducer, mockLogger, mockRepo, _, _, _, _, config := setupProducerTest()
	config.Topics.VideoEvents = "test-video-events"
	
	// Create producer
	videoProducer, err := producer.NewVideoProducer(mockClient, mockLogger, mockRepo, config)
	require.NoError(t, err)
	defer videoProducer.Close()
	
	// Create test event
	videoID := uuid.New()
	userID := uuid.New()
	event := types.VideoEvent{
		BaseEvent: types.BaseEvent{
			Type: types.VideoUploaded,
		},
		VideoID: videoID,
		UserID:  userID,
		Title:   "Test Video",
	}
	
	// Test publish
	err = videoProducer.Publish(context.Background(), event)
	require.NoError(t, err)
	
	// Verify message was sent with correct properties
	mockProducer.AssertCalled(t, "Send", mock.Anything, mock.Anything)
	sentMessages := mockProducer.GetSentMessages()
	require.Len(t, sentMessages, 1)
	
	msg := sentMessages[0]
	assert.Equal(t, videoID.String(), msg.Key)
	assert.Equal(t, string(types.VideoUploaded), msg.Properties["event_type"])
	assert.Equal(t, userID.String(), msg.Properties["user_id"])
	assert.Equal(t, videoID.String(), msg.Properties["video_id"])
	
	// Verify payload
	var decodedEvent types.VideoEvent
	err = json.Unmarshal(msg.Payload, &decodedEvent)
	require.NoError(t, err)
	assert.Equal(t, event.Type, decodedEvent.Type)
	assert.Equal(t, event.VideoID, decodedEvent.VideoID)
	assert.Equal(t, event.UserID, decodedEvent.UserID)
	assert.Equal(t, event.Title, decodedEvent.Title)
}

// TestVideoProducer_PublishVideoUploadEvent tests the PublishVideoUploadEvent method
func TestVideoProducer_PublishVideoUploadEvent(t *testing.T) {
	// Setup test fixtures
	mockClient, mockProducer, mockLogger, mockRepo, _, _, _, _, config := setupProducerTest()
	config.Topics.VideoEvents = "test-video-events"
	
	// Create producer
	videoProducer, err := producer.NewVideoProducer(mockClient, mockLogger, mockRepo, config)
	require.NoError(t, err)
	defer videoProducer.Close()
	
	// Create test data
	videoID := uuid.New()
	userID := uuid.New()
	title := "Test Upload Video"
	
	// Test publish video upload event
	err = videoProducer.PublishVideoUploadEvent(context.Background(), userID, videoID, title)
	require.NoError(t, err)
	
	// Verify message was sent with correct properties
	mockProducer.AssertCalled(t, "Send", mock.Anything, mock.Anything)
	sentMessages := mockProducer.GetSentMessages()
	require.Len(t, sentMessages, 1)
	
	msg := sentMessages[0]
	assert.Equal(t, videoID.String(), msg.Key)
	assert.Equal(t, string(types.VideoUploaded), msg.Properties["event_type"])
	
	// Verify payload
	var decodedEvent types.VideoEvent
	err = json.Unmarshal(msg.Payload, &decodedEvent)
	require.NoError(t, err)
	assert.Equal(t, types.VideoUploaded, decodedEvent.Type)
	assert.Equal(t, videoID, decodedEvent.VideoID)
	assert.Equal(t, userID, decodedEvent.UserID)
	assert.Equal(t, title, decodedEvent.Title)
}

// TestVideoProducer_InvalidEventType tests handling of invalid event types
func TestVideoProducer_InvalidEventType(t *testing.T) {
	// Setup test fixtures
	mockClient, _, mockLogger, mockRepo, _, _, _, _, config := setupProducerTest()
	config.Topics.VideoEvents = "test-video-events"
	
	// Create producer
	videoProducer, err := producer.NewVideoProducer(mockClient, mockLogger, mockRepo, config)
	require.NoError(t, err)
	defer videoProducer.Close()
	
	// Create invalid event (wrong type)
	invalidEvent := struct {
		Name string
	}{
		Name: "Invalid Event",
	}
	
	// Test publish with invalid event
	err = videoProducer.Publish(context.Background(), invalidEvent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid event type")
}

// TestCommentProducer_Publish tests the CommentProducer's Publish method
func TestCommentProducer_Publish(t *testing.T) {
	// Setup test fixtures
	mockClient, mockProducer, mockLogger, mockRepo, _, _, _, _, config := setupProducerTest()
	config.Topics.CommentEvents = "test-comment-events"
	
	// Create producer
	commentProducer, err := producer.NewCommentProducer(mockClient, mockLogger, mockRepo, config)
	require.NoError(t, err)
	defer commentProducer.Close()
	
	// Create test event
	commentID := uuid.New()
	userID := uuid.New()
	videoID := uuid.New()
	event := types.CommentEvent{
		BaseEvent: types.BaseEvent{
			Type: types.CommentCreated,
		},
		CommentID: commentID,
		UserID:    userID,
		VideoID:   videoID,
		Content:   "Test Comment",
	}
	
	// Test publish
	err = commentProducer.Publish(context.Background(), event)
	require.NoError(t, err)
	
	// Verify message was sent with correct properties
	mockProducer.AssertCalled(t, "Send", mock.Anything, mock.Anything)
	sentMessages := mockProducer.GetSentMessages()
	require.Len(t, sentMessages, 1)
	
	msg := sentMessages[0]
	assert.Equal(t, commentID.String(), msg.Key)
	assert.Equal(t, string(types.CommentCreated), msg.Properties["event_type"])
	assert.Equal(t, userID.String(), msg.Properties["user_id"])
	assert.Equal(t, commentID.String(), msg.Properties["comment_id"])
	
	// Verify payload
	var decodedEvent types.CommentEvent
	err = json.Unmarshal(msg.Payload, &decodedEvent)
	require.NoError(t, err)
	assert.Equal(t, event.Type, decodedEvent.Type)
	assert.Equal(t, event.CommentID, decodedEvent.CommentID)
	assert.Equal(t, event.UserID, decodedEvent.UserID)
	assert.Equal(t, event.VideoID, decodedEvent.VideoID)
	assert.Equal(t, event.Content, decodedEvent.Content)
}

// TestCommentProducer_PublishCommentCreatedEvent tests the PublishCommentCreatedEvent method
func TestCommentProducer_PublishCommentCreatedEvent(t *testing.T) {
	// Setup test fixtures
	mockClient, mockProducer, mockLogger, mockRepo, _, _, _, _, config := setupProducerTest()
	config.Topics.CommentEvents = "test-comment-events"
	
	// Create producer
	commentProducer, err := producer.NewCommentProducer(mockClient, mockLogger, mockRepo, config)
	require.NoError(t, err)
	defer commentProducer.Close()
	
	// Create test data
	commentID := uuid.New()
	userID := uuid.New()
	videoID := uuid.New()
	content := "Test Comment Content"
	
	// Test publish comment created event
	err = commentProducer.PublishCommentCreatedEvent(context.Background(), userID, videoID, commentID, content)
	require.NoError(t, err)
	
	// Verify message was sent with correct properties
	mockProducer.AssertCalled(t, "Send", mock.Anything, mock.Anything)
	sentMessages := mockProducer.GetSentMessages()
	require.Len(t, sentMessages, 1)
	
	msg := sentMessages[0]
	assert.Equal(t, commentID.String(), msg.Key)
	assert.Equal(t, string(types.CommentCreated), msg.Properties["event_type"])
	
	// Verify payload
	var decodedEvent types.CommentEvent
	err = json.Unmarshal(msg.Payload, &decodedEvent)
	require.NoError(t, err)
	assert.Equal(t, types.CommentCreated, decodedEvent.Type)
	assert.Equal(t, commentID, decodedEvent.CommentID)
	assert.Equal(t, userID, decodedEvent.UserID)
	assert.Equal(t, videoID, decodedEvent.VideoID)
	assert.Equal(t, content, decodedEvent.Content)
}

// TestCommentProducer_PublishCommentReplyEvent tests the PublishCommentReplyEvent method
func TestCommentProducer_PublishCommentReplyEvent(t *testing.T) {
	// Setup test fixtures
	mockClient, mockProducer, mockLogger, mockRepo, _, _, _, _, config := setupProducerTest()
	config.Topics.CommentEvents = "test-comment-events"
	
	// Create producer
	commentProducer, err := producer.NewCommentProducer(mockClient, mockLogger, mockRepo, config)
	require.NoError(t, err)
	defer commentProducer.Close()
	
	// Create test data
	commentID := uuid.New()
	parentID := uuid.New()
	userID := uuid.New()
	videoID := uuid.New()
	content := "Test Reply Content"
	
	// Test publish comment reply event
	err = commentProducer.PublishCommentReplyEvent(context.Background(), userID, videoID, commentID, parentID, content)
	require.NoError(t, err)
	
	// Verify message was sent with correct properties
	mockProducer.AssertCalled(t, "Send", mock.Anything, mock.Anything)
	sentMessages := mockProducer.GetSentMessages()
	require.Len(t, sentMessages, 1)
	
	msg := sentMessages[0]
	assert.Equal(t, commentID.String(), msg.Key)
	assert.Equal(t, string(types.CommentReplied), msg.Properties["event_type"])
	
	// Verify payload
	var decodedEvent types.CommentEvent
	err = json.Unmarshal(msg.Payload, &decodedEvent)
	require.NoError(t, err)
	assert.Equal(t, types.CommentReplied, decodedEvent.Type)
	assert.Equal(t, commentID, decodedEvent.CommentID)
	assert.Equal(t, userID, decodedEvent.UserID)
	assert.Equal(t, videoID, decodedEvent.VideoID)
	assert.Equal(t, parentID, decodedEvent.ParentID)
	assert.Equal(t, content, decodedEvent.Content)
}

// TestUserProducer_Publish tests the Publish method of UserProducer
func TestUserProducer_Publish(t *testing.T) {
	// Setup test fixtures
	mockClient, mockProducer, mockLogger, mockRepo, _, _, _, _, config := setupProducerTest()
	config.Topics.UserEvents = "user-events"
	
	userProducer, err := producer.NewUserProducer(mockClient, mockLogger, mockRepo, config)
	require.NoError(t, err)

	// Create test event
	userID := uuid.New()
	targetUserID := uuid.New()
	userEvent := types.UserEvent{
		BaseEvent: types.BaseEvent{
			Type:      types.UserFollowed,
			CreatedAt: time.Now(),
		},
		UserID:       userID,
		TargetUserID: targetUserID,
	}

	// Publish event
	err = userProducer.Publish(context.Background(), userEvent)
	require.NoError(t, err)

	// Verify message was sent
	sentMessages := mockProducer.GetSentMessages()
	require.Len(t, sentMessages, 1)
	sentMsg := sentMessages[0]

	// Verify message properties
	assert.Equal(t, userID.String(), sentMsg.Key)
	assert.Equal(t, string(types.UserFollowed), sentMsg.Properties["event_type"])
	assert.Equal(t, userID.String(), sentMsg.Properties["user_id"])
	assert.Equal(t, targetUserID.String(), sentMsg.Properties["target_user_id"])

	// Verify payload
	var sentEvent types.UserEvent
	err = json.Unmarshal(sentMsg.Payload, &sentEvent)
	require.NoError(t, err)
	assert.Equal(t, types.UserFollowed, sentEvent.Type)
	assert.Equal(t, userID, sentEvent.UserID)
	assert.Equal(t, targetUserID, sentEvent.TargetUserID)
}

// TestUserProducer_PublishFollowEvent tests the PublishFollowEvent method
func TestUserProducer_PublishFollowEvent(t *testing.T) {
	// Setup test fixtures
	mockClient, mockProducer, mockLogger, mockRepo, _, _, _, _, config := setupProducerTest()
	config.Topics.UserEvents = "user-events"
	
	userProducer, err := producer.NewUserProducer(mockClient, mockLogger, mockRepo, config)
	require.NoError(t, err)

	// Create test data
	userID := uuid.New()
	targetUserID := uuid.New()

	// Publish follow event
	err = userProducer.PublishFollowEvent(context.Background(), userID, targetUserID)
	require.NoError(t, err)

	// Verify message was sent
	sentMessages := mockProducer.GetSentMessages()
	require.Len(t, sentMessages, 1)
	sentMsg := sentMessages[0]

	// Verify message properties
	assert.Equal(t, userID.String(), sentMsg.Key)
	assert.Equal(t, string(types.UserFollowed), sentMsg.Properties["event_type"])

	// Verify payload
	var sentEvent types.UserEvent
	err = json.Unmarshal(sentMsg.Payload, &sentEvent)
	require.NoError(t, err)
	assert.Equal(t, types.UserFollowed, sentEvent.Type)
	assert.Equal(t, userID, sentEvent.UserID)
	assert.Equal(t, targetUserID, sentEvent.TargetUserID)
}

// TestUserProducer_PublishUnfollowEvent tests the PublishUnfollowEvent method
func TestUserProducer_PublishUnfollowEvent(t *testing.T) {
	// Setup test fixtures
	mockClient, mockProducer, mockLogger, mockRepo, _, _, _, _, config := setupProducerTest()
	config.Topics.UserEvents = "user-events"
	
	userProducer, err := producer.NewUserProducer(mockClient, mockLogger, mockRepo, config)
	require.NoError(t, err)

	// Create test data
	userID := uuid.New()
	targetUserID := uuid.New()

	// Publish unfollow event
	err = userProducer.PublishUnfollowEvent(context.Background(), userID, targetUserID)
	require.NoError(t, err)

	// Verify message was sent
	sentMessages := mockProducer.GetSentMessages()
	require.Len(t, sentMessages, 1)
	sentMsg := sentMessages[0]

	// Verify message properties
	assert.Equal(t, userID.String(), sentMsg.Key)
	assert.Equal(t, string(types.UserUnfollowed), sentMsg.Properties["event_type"])

	// Verify payload
	var sentEvent types.UserEvent
	err = json.Unmarshal(sentMsg.Payload, &sentEvent)
	require.NoError(t, err)
	assert.Equal(t, types.UserUnfollowed, sentEvent.Type)
	assert.Equal(t, userID, sentEvent.UserID)
	assert.Equal(t, targetUserID, sentEvent.TargetUserID)
} 