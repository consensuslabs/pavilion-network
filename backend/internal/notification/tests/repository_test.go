package tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/notification"
	"github.com/consensuslabs/pavilion-network/backend/testhelper"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRepository is a test for the notification repository 
// Normally, we would use a real ScyllaDB instance or a mock, but for simplicity
// we're just testing construction and verifying interfaces are correctly implemented
func TestRepositoryInterfaces(t *testing.T) {
	// This is just a compile-time check to ensure the repository implements the interface
	var _ notification.NotificationRepository = (*notification.Repository)(nil)
}

// TestSchemaManager is a test for the notification schema manager
func TestSchemaManager(t *testing.T) {
	// Skip this test since it requires a real ScyllaDB instance
	t.Skip("Skipping schema manager test - requires ScyllaDB instance")

	// This is just a compile-time check to ensure the schema manager has the correct methods
	logger := testhelper.NewTestLogger(true)
	schemaManager := notification.NewSchemaManager(nil, "testKeyspace", logger)
	
	// Check that methods exist (but don't call them as we have no actual session)
	assert.NotNil(t, schemaManager.CreateTables)
	assert.NotNil(t, schemaManager.DropTables)
}

// TestNotificationModel tests the notification model and its methods
func TestNotificationModel(t *testing.T) {
	// Create a notification
	id := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()
	
	notif := &notification.Notification{
		ID:        id,
		UserID:    userID,
		Type:      notification.VideoUploaded,
		Content:   "Test notification",
		Metadata:  map[string]interface{}{"key": "value"},
		CreatedAt: now,
	}
	
	// Test ID
	assert.Equal(t, id, notif.ID)
	
	// Test marshalling
	data, err := notif.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	
	// Test unmarshalling
	var unmarshalled notification.Notification
	err = json.Unmarshal(data, &unmarshalled)
	require.NoError(t, err)
	assert.Equal(t, notif.ID, unmarshalled.ID)
	assert.Equal(t, notif.UserID, unmarshalled.UserID)
	assert.Equal(t, notif.Type, unmarshalled.Type)
	assert.Equal(t, notif.Content, unmarshalled.Content)
	assert.Equal(t, now.Unix(), unmarshalled.CreatedAt.Unix())
	
	// Test read status
	assert.False(t, notif.IsRead())
	
	// Now mark as read
	readTime := time.Now()
	notif.ReadAt = &readTime
	assert.True(t, notif.IsRead())
}