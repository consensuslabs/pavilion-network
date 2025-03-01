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
	m.logger.LogInfo("Beginning schema initialization", map[string]interface{}{
		"keyspace": m.config.Keyspace,
	})

	// Create comments table
	m.logger.LogInfo("Creating comments table", nil)
	if err := m.createCommentsTable(); err != nil {
		m.logger.LogError("Failed to create comments table", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}
	m.logger.LogInfo("Comments table created successfully", nil)

	// Create comment_by_video index table
	m.logger.LogInfo("Creating comment_by_video table", nil)
	if err := m.createCommentByVideoTable(); err != nil {
		m.logger.LogError("Failed to create comment_by_video table", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}
	m.logger.LogInfo("Comment_by_video table created successfully", nil)

	// Create replies index table
	m.logger.LogInfo("Creating replies table", nil)
	if err := m.createRepliesTable(); err != nil {
		m.logger.LogError("Failed to create replies table", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}
	m.logger.LogInfo("Replies table created successfully", nil)

	// Create reactions table
	m.logger.LogInfo("Creating reactions table", nil)
	if err := m.createReactionsTable(); err != nil {
		m.logger.LogError("Failed to create reactions table", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}
	m.logger.LogInfo("Reactions table created successfully", nil)

	m.logger.LogInfo("Schema initialization completed successfully", map[string]interface{}{
		"keyspace": m.config.Keyspace,
	})

	return nil
}

// Helper functions for schema creation
func (m *SchemaManager) createCommentsTable() error {
	m.logger.LogInfo("Creating comments table with schema", map[string]interface{}{
		"table": "comments",
		"columns": []string{
			"id uuid PRIMARY KEY",
			"video_id uuid",
			"user_id uuid",
			"content text",
			"created_at timestamp",
			"updated_at timestamp",
			"deleted_at timestamp",
			"parent_id uuid",
			"likes int",
			"dislikes int",
			"status text",
		},
	})

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
	m.logger.LogInfo("Creating comments_by_video table with schema", map[string]interface{}{
		"table": "comments_by_video",
		"columns": []string{
			"video_id uuid",
			"comment_id uuid",
			"created_at timestamp",
		},
		"primary_key":      "(video_id, created_at, comment_id)",
		"clustering_order": "created_at DESC, comment_id ASC",
	})

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
	m.logger.LogInfo("Creating replies table with schema", map[string]interface{}{
		"table": "replies",
		"columns": []string{
			"parent_id uuid",
			"comment_id uuid",
			"created_at timestamp",
		},
		"primary_key":      "(parent_id, created_at, comment_id)",
		"clustering_order": "created_at DESC, comment_id ASC",
	})

	query := `
	CREATE TABLE IF NOT EXISTS replies (
		parent_id uuid,
		comment_id uuid,
		created_at timestamp,
		PRIMARY KEY (parent_id, created_at, comment_id)
	) WITH CLUSTERING ORDER BY (created_at DESC, comment_id ASC)
	`
	return m.session.Query(query).Exec()
}

func (m *SchemaManager) createReactionsTable() error {
	m.logger.LogInfo("Creating reactions table with schema", map[string]interface{}{
		"table": "reactions",
		"columns": []string{
			"comment_id uuid",
			"user_id uuid",
			"type text",
			"created_at timestamp",
		},
		"primary_key": "(comment_id, user_id)",
	})

	query := `
	CREATE TABLE IF NOT EXISTS reactions (
		comment_id uuid,
		user_id uuid,
		type text,
		created_at timestamp,
		PRIMARY KEY (comment_id, user_id)
	)
	`
	return m.session.Query(query).Exec()
}
