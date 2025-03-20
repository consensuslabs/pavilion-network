package scylladb

import (
	"fmt"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/gocql/gocql"
)

// MigrationSchemaManager handles the creation of ScyllaDB migration tracking tables
type MigrationSchemaManager struct {
	session  *gocql.Session
	keyspace string
	logger   logger.Logger
}

// NewMigrationSchemaManager creates a new schema manager for migration tables
func NewMigrationSchemaManager(session *gocql.Session, keyspace string, logger logger.Logger) *MigrationSchemaManager {
	return &MigrationSchemaManager{
		session:  session,
		keyspace: keyspace,
		logger:   logger,
	}
}

// CreateTables creates the necessary tables for tracking migrations if they don't exist
func (m *MigrationSchemaManager) CreateTables() error {
	m.logger.LogInfo("Creating ScyllaDB migration tracking tables", nil)
	
	// Create schema_migrations table
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.schema_migrations (
			id uuid,
			name text,
			hash text,
			applied_at timestamp,
			batch_no int,
			PRIMARY KEY (name)
		)`,
		m.keyspace,
	)
	
	if err := m.session.Query(query).Exec(); err != nil {
		m.logger.LogError(err, "Failed to create schema_migrations table")
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}
	
	m.logger.LogInfo("ScyllaDB migration tracking tables created successfully", nil)
	return nil
} 