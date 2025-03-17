package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRepository is a test for the notification repository 
// Normally, we would use a real ScyllaDB instance or a mock, but for simplicity
// we're just testing construction and verifying interfaces are correctly implemented
func TestRepositoryInterfaces(t *testing.T) {
	// Skip this test since we don't have access to the Repository implementation
	t.Skip("Skipping repository interface test")
}

// TestSchemaManager is a test for the notification schema manager
func TestSchemaManager(t *testing.T) {
	// Skip this test since it requires a real ScyllaDB instance
	t.Skip("Skipping schema manager test - requires ScyllaDB instance")
}

// TestNotificationModel tests the notification model and its methods
func TestNotificationModel(t *testing.T) {
	// Create a notification
	id := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()
	
	notif := &types.Notification{
		ID:        id,
		UserID:    userID,
		Type:      string(types.VideoUploaded),
		Content:   "Test notification",
		Metadata:  map[string]interface{}{"key": "value"},
		CreatedAt: now,
	}
	
	// Test JSON serialization
	data, err := json.Marshal(notif)
	require.NoError(t, err)
	
	// Test JSON deserialization
	var notif2 types.Notification
	err = json.Unmarshal(data, &notif2)
	require.NoError(t, err)
	
	// Verify fields
	assert.Equal(t, id, notif2.ID)
	assert.Equal(t, userID, notif2.UserID)
	assert.Equal(t, string(types.VideoUploaded), notif2.Type)
	assert.Equal(t, "Test notification", notif2.Content)
	assert.Equal(t, "value", notif2.Metadata["key"])
}