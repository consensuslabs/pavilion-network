package scylladb

import (
	"fmt"

	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/gocql/gocql"
)

// SchemaManager handles ScyllaDB schema creation and migrations
type SchemaManager struct {
	session *gocql.Session
	config  Config
	logger  video.Logger
}

// NewSchemaManager creates a new schema manager
func NewSchemaManager(session *gocql.Session, config Config, logger video.Logger) *SchemaManager {
	return &SchemaManager{
		session: session,
		config:  config,
		logger:  logger,
	}
}

// CreateKeyspaceIfNotExists creates the keyspace if it doesn't exist
func (m *SchemaManager) CreateKeyspaceIfNotExists() error {
	query := fmt.Sprintf(`
		CREATE KEYSPACE IF NOT EXISTS %s
		WITH REPLICATION = {
			'class': '%s',
			'replication_factor': %d
		}
	`, m.config.Keyspace, m.config.Replication.Class, m.config.Replication.ReplicationFactor)

	return m.session.Query(query).Exec()
}

// InitializeSchema initializes all tables needed for the comment system
func (m *SchemaManager) InitializeSchema() error {
	// Create comments table
	if err := m.createCommentsTable(); err != nil {
		return err
	}

	// Create comment_by_video index table
	if err := m.createCommentByVideoTable(); err != nil {
		return err
	}

	// Create replies index table
	if err := m.createRepliesTable(); err != nil {
		return err
	}

	// Create reactions table
	if err := m.createReactionsTable(); err != nil {
		return err
	}

	return nil
}

// Helper functions for schema creation
func (m *SchemaManager) createCommentsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS comments (
		id uuid PRIMARY KEY,
		video_id uuid,
		user_id uuid,
		content text,
		created_at timestamp,
		updated_at timestamp,
		deleted_at timestamp,
		parent_id uuid,
		likes int,
		dislikes int,
		status text
	)
	`
	return m.session.Query(query).Exec()
}

func (m *SchemaManager) createCommentByVideoTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS comments_by_video (
		video_id uuid,
		comment_id uuid,
		created_at timestamp,
		PRIMARY KEY (video_id, created_at, comment_id)
	) WITH CLUSTERING ORDER BY (created_at DESC, comment_id ASC)
	`
	return m.session.Query(query).Exec()
}

func (m *SchemaManager) createRepliesTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS replies (
		parent_id uuid,
		comment_id uuid,
		created_at timestamp,
		PRIMARY KEY (parent_id, created_at, comment_id)
	) WITH CLUSTERING ORDER BY (created_at ASC, comment_id ASC)
	`
	return m.session.Query(query).Exec()
}

func (m *SchemaManager) createReactionsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS reactions (
		comment_id uuid,
		user_id uuid,
		type text,
		created_at timestamp,
		updated_at timestamp,
		PRIMARY KEY (comment_id, user_id)
	)
	`
	return m.session.Query(query).Exec()
}
