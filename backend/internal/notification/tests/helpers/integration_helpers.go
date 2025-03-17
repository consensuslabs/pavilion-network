package helpers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/consumer"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/producer"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/tests/mocks"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// SetupTopics ensures the required Pulsar topics exist
func SetupTopics(client pulsar.Client, config *types.ServiceConfig) error {
	// For Pulsar, topics are created by the Docker init script
	// Here we just verify that the topics exist by creating readers
	
	// List of topics to verify
	topics := []string{
		config.Topics.VideoEvents,
		config.Topics.CommentEvents,
		config.Topics.UserEvents,
		config.Topics.DeadLetter,
		config.Topics.RetryQueue,
	}
	
	// Verify each topic exists
	for _, topic := range topics {
		if topic == "" {
			continue // Skip empty topics
		}
		
		// Create a reader for this topic (readers don't create topics, so this is safe)
		reader, err := client.CreateReader(pulsar.ReaderOptions{
			Topic:          topic,
			StartMessageID: pulsar.EarliestMessageID(),
		})
		
		if err != nil {
			return fmt.Errorf("topic %s does not exist or cannot be accessed: %w", topic, err)
		}
		
		// Close the reader immediately
		reader.Close()
	}
	
	return nil
}

// CleanupTestData removes test notifications from ScyllaDB
func CleanupTestData(session *gocql.Session, userID uuid.UUID) error {
	// Delete test notifications for the specified user
	query := "DELETE FROM notifications WHERE user_id = ?"
	
	// Convert UUID to binary for ScyllaDB like in the real implementation
	userIDBytes, err := userID.MarshalBinary()
	if err != nil {
		return fmt.Errorf("error marshaling user ID: %w", err)
	}
	
	return session.Query(query, userIDBytes).Exec()
}

// CreateNotificationRepository creates a real ScyllaDB repository
func CreateNotificationRepository(session *gocql.Session) (notification.NotificationRepository, error) {
	logger := mocks.SetupTestLogger()
	
	// Try to load test configuration for keyspace
	keyspace := "pavilion_test" // Default from updated config_test.yaml
	
	// Create the repository
	return notification.NewRepository(session, logger, keyspace, &types.ServiceConfig{}), nil
}

// InitializeScyllaDBSchema initializes the ScyllaDB schema if needed
func InitializeScyllaDBSchema(session *gocql.Session) error {
	// Create the notifications table if it doesn't exist with the same structure as the real implementation
	createTable := `
	CREATE TABLE IF NOT EXISTS notifications (
		id uuid,
		user_id uuid,
		type text,
		content text,
		metadata map<text, text>,
		read_at timeuuid,
		created_at timestamp,
		PRIMARY KEY ((user_id), created_at, id)
	) WITH CLUSTERING ORDER BY (created_at DESC, id ASC)
	`
	
	// Execute the create table statement
	err := session.Query(createTable).Exec()
	if err != nil {
		// If error is about keyspace not existing, we need to create it
		if strings.Contains(err.Error(), "Keyspace") && strings.Contains(err.Error(), "does not exist") {
			// Default keyspace for tests
			keyspaceName := "pavilion_test" // Default from updated config_test.yaml
			
			// Create a new cluster without keyspace specification
			cluster := gocql.NewCluster("localhost")
			cluster.Keyspace = ""
			cluster.Consistency = gocql.One
			cluster.Timeout = 30 * time.Second
			
			// Create system session
			systemSession, err := cluster.CreateSession()
			if err != nil {
				return fmt.Errorf("failed to create system session: %w", err)
			}
			defer systemSession.Close()
			
			// Create the keyspace with SimpleStrategy and replication factor 1 for tests
			createKeyspace := fmt.Sprintf(`
			CREATE KEYSPACE IF NOT EXISTS %s 
			WITH REPLICATION = { 
				'class' : 'SimpleStrategy', 
				'replication_factor' : 1 
			}`, keyspaceName)
			
			if err := systemSession.Query(createKeyspace).Exec(); err != nil {
				return fmt.Errorf("failed to create keyspace: %w", err)
			}
			
			// Now create a new session with the keyspace
			cluster.Keyspace = keyspaceName
			newSession, err := cluster.CreateSession()
			if err != nil {
				return fmt.Errorf("failed to create session with new keyspace: %w", err)
			}
			defer newSession.Close()
			
			// Try to create the table with the new session
			return newSession.Query(createTable).Exec()
		}
		return err
	}
	
	// Create indexes similar to the real implementation
	indexQueries := []string{
		`CREATE INDEX IF NOT EXISTS notifications_id_idx ON notifications (id)`,
		`CREATE INDEX IF NOT EXISTS notifications_type_idx ON notifications (type)`,
		`CREATE INDEX IF NOT EXISTS notifications_read_at_idx ON notifications (read_at)`,
	}
	
	for _, query := range indexQueries {
		if err := session.Query(query).Exec(); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}
	
	return nil
}

// SetupVideoProducer creates and configures a video producer
func SetupVideoProducer(
	client pulsar.Client,
	repo notification.NotificationRepository,
	config *types.ServiceConfig,
) (*producer.VideoProducer, error) {
	logger := mocks.SetupTestLogger()
	return producer.NewVideoProducer(client, logger, repo, config)
}

// SetupCommentProducer creates and configures a comment producer
func SetupCommentProducer(
	client pulsar.Client,
	repo notification.NotificationRepository,
	config *types.ServiceConfig,
) (*producer.CommentProducer, error) {
	logger := mocks.SetupTestLogger()
	return producer.NewCommentProducer(client, logger, repo, config)
}

// SetupUserProducer creates and configures a user producer
func SetupUserProducer(
	client pulsar.Client,
	repo notification.NotificationRepository,
	config *types.ServiceConfig,
) (*producer.UserProducer, error) {
	logger := mocks.SetupTestLogger()
	return producer.NewUserProducer(client, logger, repo, config)
}

// SetupVideoConsumer creates and configures a video consumer
func SetupVideoConsumer(
	client pulsar.Client,
	repo notification.NotificationRepository,
	config *types.ServiceConfig,
) *consumer.VideoConsumer {
	logger := mocks.SetupTestLogger()
	return consumer.NewVideoConsumer(client, logger, repo, config)
}

// SetupCommentConsumer creates and configures a comment consumer
func SetupCommentConsumer(
	client pulsar.Client,
	repo notification.NotificationRepository,
	config *types.ServiceConfig,
) *consumer.CommentConsumer {
	logger := mocks.SetupTestLogger()
	return consumer.NewCommentConsumer(client, logger, repo, config)
}

// SetupUserConsumer creates and configures a user consumer
func SetupUserConsumer(
	client pulsar.Client,
	repo notification.NotificationRepository,
	config *types.ServiceConfig,
) *consumer.UserConsumer {
	logger := mocks.SetupTestLogger()
	return consumer.NewUserConsumer(client, logger, repo, config)
}

// WaitForProcessing waits for event processing to complete
func WaitForProcessing(duration time.Duration) {
	// In real environments, message processing may take longer
	// Default duration should be sufficient for test environments
	// Add a minimum duration to ensure enough time for processing
	minDuration := 20 * time.Second
	if duration < minDuration {
		duration = minDuration
	}
	time.Sleep(duration)
}

// CheckNotificationExists checks if a notification exists in the repository
func CheckNotificationExists(
	ctx context.Context,
	repo notification.NotificationRepository,
	userID uuid.UUID,
	notificationType string,
) (bool, *types.Notification, error) {
	fmt.Printf("Checking for notifications with UserID: %s, Type: %s\n", userID, notificationType)
	
	notifications, err := repo.GetNotificationsByUserID(ctx, userID, 10, 0)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get notifications: %w", err)
	}
	
	fmt.Printf("Found %d notifications for user\n", len(notifications))
	
	for i, notification := range notifications {
		fmt.Printf("Notification %d: ID=%s, Type=%s, Content=%s\n", 
			i, notification.ID, notification.Type, notification.Content)
		
		if notification.Type == notificationType {
			fmt.Printf("Matched notification found: %s\n", notification.ID)
			return true, notification, nil
		}
	}
	
	fmt.Printf("No matching notification found for type: %s\n", notificationType)
	return false, nil, nil
} 