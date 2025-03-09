package notification

import (
	"fmt"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/gocql/gocql"
)

// SchemaManager handles the ScyllaDB schema for notifications
type SchemaManager struct {
	session  *gocql.Session
	keyspace string
	logger   logger.Logger
}

// NewSchemaManager creates a new schema manager for notifications
func NewSchemaManager(session *gocql.Session, keyspace string, logger logger.Logger) *SchemaManager {
	return &SchemaManager{
		session:  session,
		keyspace: keyspace,
		logger:   logger,
	}
}

// CreateTables creates the necessary tables for notifications if they don't exist
func (m *SchemaManager) CreateTables() error {
	// Create notifications table
	if err := m.createNotificationsTable(); err != nil {
		return err
	}

	// Create indexes
	if err := m.createIndexes(); err != nil {
		return err
	}

	m.logger.LogInfo("Notification tables created successfully", nil)
	return nil
}

// createNotificationsTable creates the notifications table
func (m *SchemaManager) createNotificationsTable() error {
	// Create the notifications table
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.notifications (
			id uuid,
			user_id uuid,
			type text,
			content text,
			metadata blob,
			read_at timeuuid,
			created_at timestamp,
			PRIMARY KEY ((user_id), created_at, id)
		) WITH CLUSTERING ORDER BY (created_at DESC, id ASC)`,
		m.keyspace,
	)

	if err := m.session.Query(query).Exec(); err != nil {
		m.logger.LogError(err, "Failed to create notifications table")
		return fmt.Errorf("failed to create notifications table: %w", err)
	}

	return nil
}

// createIndexes creates the necessary indexes for the notifications table
func (m *SchemaManager) createIndexes() error {
	// Create index on notification ID for quick lookups
	idIndexQuery := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS notifications_id_idx ON %s.notifications (id)`,
		m.keyspace,
	)

	if err := m.session.Query(idIndexQuery).Exec(); err != nil {
		m.logger.LogError(err, "Failed to create notification ID index")
		return fmt.Errorf("failed to create notification ID index: %w", err)
	}

	// Create index on notification type for filtering
	typeIndexQuery := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS notifications_type_idx ON %s.notifications (type)`,
		m.keyspace,
	)

	if err := m.session.Query(typeIndexQuery).Exec(); err != nil {
		m.logger.LogError(err, "Failed to create notification type index")
		return fmt.Errorf("failed to create notification type index: %w", err)
	}

	// Create index on read_at for filtering unread notifications
	readAtIndexQuery := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS notifications_read_at_idx ON %s.notifications (read_at)`,
		m.keyspace,
	)

	if err := m.session.Query(readAtIndexQuery).Exec(); err != nil {
		m.logger.LogError(err, "Failed to create notification read_at index")
		return fmt.Errorf("failed to create notification read_at index: %w", err)
	}

	return nil
}

// DropTables drops all notification-related tables
func (m *SchemaManager) DropTables() error {
	query := fmt.Sprintf(`DROP TABLE IF EXISTS %s.notifications`, m.keyspace)
	if err := m.session.Query(query).Exec(); err != nil {
		m.logger.LogError(err, "Failed to drop notifications table")
		return fmt.Errorf("failed to drop notifications table: %w", err)
	}

	m.logger.LogInfo("Notification tables dropped successfully", nil)
	return nil
}