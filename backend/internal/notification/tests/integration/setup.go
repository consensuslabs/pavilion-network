package integration

import (
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/tests/helpers"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
	"github.com/consensuslabs/pavilion-network/backend/testhelper"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// SetupIntegrationTest prepares the environment for an integration test
func SetupIntegrationTest(t *testing.T) (*testhelper.Config, pulsar.Client, *gocql.Session, func()) {
	t.Helper()
	
	// Skip if short tests are requested
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Load configuration
	config, err := testhelper.SetupNotificationTestConfig()
	if err != nil {
		t.Logf("Error loading test configuration: %v", err)
		t.Skip("Skipping test because test configuration could not be loaded")
	}
	
	// Check if Pulsar is available
	if !testhelper.CheckPulsarAvailable(config) {
		t.Logf("Pulsar not available at URL: %s", config.Pulsar.URL)
		t.Skip("Skipping test because Pulsar is not available")
	}
	
	// Check if ScyllaDB is available
	if !testhelper.CheckScyllaDBAvailable(config) {
		t.Logf("ScyllaDB not available at hosts: %v", config.ScyllaDB.Hosts)
		t.Skip("Skipping test because ScyllaDB is not available")
	}
	
	// Create Pulsar client
	pulsarClient, err := testhelper.CreatePulsarClient(config)
	if err != nil {
		t.Logf("Error creating Pulsar client: %v", err)
		t.Skip("Skipping test because Pulsar client could not be created")
	}
	
	// Create ScyllaDB session
	scyllaSession, err := testhelper.CreateScyllaDBSession(config)
	if err != nil {
		pulsarClient.Close()
		t.Logf("Error creating ScyllaDB session: %v", err)
		t.Skip("Skipping test because ScyllaDB session could not be created")
	}
	
	// Create test service config with existing topics
	serviceConfig := CreateTestServiceConfig()
	
	// Ensure topics exist
	if err := helpers.SetupTopics(pulsarClient, serviceConfig); err != nil {
		scyllaSession.Close()
		pulsarClient.Close()
		t.Logf("Error setting up topics: %v", err)
		t.Skip("Skipping test because topics could not be set up")
	}
	
	// Return cleanup function
	cleanup := func() {
		scyllaSession.Close()
		pulsarClient.Close()
	}
	
	return config, pulsarClient, scyllaSession, cleanup
}

// GetUniqueTopicName generates a unique topic name for a test
func GetUniqueTopicName(baseName string) string {
	return "test-" + baseName + "-" + uuid.New().String()
}

// CreateTestServiceConfig creates a test service config with existing topics
func CreateTestServiceConfig() *types.ServiceConfig {
	config := &types.ServiceConfig{}
	
	// Use the exact topic names that are created by the Docker init script
	config.Topics.VideoEvents = "persistent://pavilion/notifications/video-events"
	config.Topics.CommentEvents = "persistent://pavilion/notifications/comment-events"
	config.Topics.UserEvents = "persistent://pavilion/notifications/user-events"
	config.Topics.DeadLetter = "persistent://pavilion/notifications/dead-letter"
	config.Topics.RetryQueue = "persistent://pavilion/notifications/retry-queue"
	
	return config
}

// WaitForNotifications waits for notifications to be processed
func WaitForNotifications(t *testing.T, duration time.Duration) {
	t.Helper()
	time.Sleep(duration)
} 