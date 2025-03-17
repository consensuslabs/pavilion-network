package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/notification/util"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTruncateContent tests the TruncateContent utility function
func TestTruncateContent(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		maxLength   int
		expected    string
		description string
	}{
		{
			name:        "Short content",
			content:     "This is a short message",
			maxLength:   50,
			expected:    "This is a short message",
			description: "Content shorter than max length should remain unchanged",
		},
		{
			name:        "Exact length content",
			content:     "This is exactly 30 characters long",
			maxLength:   30,
			expected:    "This is exactly 30 characte...",
			description: "Content exactly at max length should be truncated with ellipsis",
		},
		{
			name:        "Long content",
			content:     "This is a very long message that needs to be truncated because it exceeds the maximum length",
			maxLength:   20,
			expected:    "This is a very lo...",
			description: "Content longer than max length should be truncated with ellipsis",
		},
		{
			name:        "Empty content",
			content:     "",
			maxLength:   10,
			expected:    "",
			description: "Empty content should remain empty",
		},
		{
			name:        "Very short max length",
			content:     "Hello world",
			maxLength:   5,
			expected:    "He...",
			description: "Very short max length should still include ellipsis",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := util.TruncateContent(tc.content, tc.maxLength)
			assert.Equal(t, tc.expected, result, tc.description)
		})
	}
}

// TestGenerateEventID tests the GenerateEventID utility function
func TestGenerateEventID(t *testing.T) {
	t.Run("Nil UUID", func(t *testing.T) {
		id := util.GenerateEventID(uuid.Nil)
		assert.NotEqual(t, uuid.Nil, id, "Should generate a new UUID when given a nil UUID")
	})

	t.Run("Existing UUID", func(t *testing.T) {
		existingID := uuid.New()
		id := util.GenerateEventID(existingID)
		assert.Equal(t, existingID, id, "Should return the existing UUID when given a non-nil UUID")
	})
}

// TestGenerateEventTime tests the GenerateEventTime utility function
func TestGenerateEventTime(t *testing.T) {
	t.Run("Zero time", func(t *testing.T) {
		zeroTime := time.Time{}
		generatedTime := util.GenerateEventTime(zeroTime)
		assert.NotEqual(t, zeroTime, generatedTime, "Should generate a new time when given a zero time")
		assert.True(t, time.Now().After(generatedTime), "Generated time should be in the past")
		assert.True(t, time.Now().Sub(generatedTime) < time.Second, "Generated time should be recent")
	})

	t.Run("Existing time", func(t *testing.T) {
		existingTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		generatedTime := util.GenerateEventTime(existingTime)
		assert.Equal(t, existingTime, generatedTime, "Should return the existing time when given a non-zero time")
	})
}

// TestGenerateSequenceNumber tests the GenerateSequenceNumber utility function
func TestGenerateSequenceNumber(t *testing.T) {
	t.Run("Zero sequence", func(t *testing.T) {
		seq := util.GenerateSequenceNumber(0)
		assert.NotEqual(t, int64(0), seq, "Should generate a new sequence number when given zero")
		assert.True(t, seq > 0, "Generated sequence number should be positive")
	})

	t.Run("Existing sequence", func(t *testing.T) {
		existingSeq := int64(12345)
		seq := util.GenerateSequenceNumber(existingSeq)
		assert.Equal(t, existingSeq, seq, "Should return the existing sequence number when given non-zero")
	})
}

// TestCreateProducerMessage tests the CreateProducerMessage utility function
func TestCreateProducerMessage(t *testing.T) {
	// Create test event
	type TestEvent struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	}
	
	event := TestEvent{
		ID:   uuid.New(),
		Name: "Test Event",
	}
	
	// Create test properties
	key := "test-key"
	properties := map[string]string{
		"prop1": "value1",
		"prop2": "value2",
	}
	eventTime := time.Now()
	
	// Create producer message
	msg, err := util.CreateProducerMessage(event, key, properties, eventTime)
	require.NoError(t, err, "CreateProducerMessage should not return an error")
	
	// Verify message properties
	assert.Equal(t, key, msg.Key, "Message key should match the provided key")
	assert.Equal(t, properties, msg.Properties, "Message properties should match the provided properties")
	assert.Equal(t, eventTime, msg.EventTime, "Message event time should match the provided event time")
	
	// Verify payload
	var decodedEvent TestEvent
	err = json.Unmarshal(msg.Payload, &decodedEvent)
	require.NoError(t, err, "Should be able to unmarshal the message payload")
	assert.Equal(t, event.ID, decodedEvent.ID, "Decoded event ID should match the original event ID")
	assert.Equal(t, event.Name, decodedEvent.Name, "Decoded event name should match the original event name")
}

// TestFormatContentFunctions tests the content formatting utility functions
func TestFormatContentFunctions(t *testing.T) {
	t.Run("FormatVideoContent", func(t *testing.T) {
		title := "Test Video"
		action := "uploaded"
		expected := "uploaded: Test Video"
		
		result := util.FormatVideoContent(title, action)
		assert.Equal(t, expected, result, "FormatVideoContent should format the content correctly")
	})
	
	t.Run("FormatCommentContent", func(t *testing.T) {
		content := "This is a test comment that is quite long and should be truncated"
		action := "commented"
		maxLength := 20
		expected := "commented: This is a test co..."
		
		result := util.FormatCommentContent(content, action, maxLength)
		assert.Equal(t, expected, result, "FormatCommentContent should format and truncate the content correctly")
	})
	
	t.Run("FormatUserContent", func(t *testing.T) {
		username := "testuser"
		action := "followed you"
		expected := "testuser followed you"
		
		result := util.FormatUserContent(username, action)
		assert.Equal(t, expected, result, "FormatUserContent should format the content correctly")
	})
} 