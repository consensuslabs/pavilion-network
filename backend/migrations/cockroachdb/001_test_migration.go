package cockroachdb

import (
	"fmt"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"gorm.io/gorm"
)

// TestMigration is a simple test migration to verify the migration system works
type TestMigration struct {
	logger logger.Logger
}

// NewTestMigration creates a new test migration
func NewTestMigration(logger logger.Logger) *TestMigration {
	return &TestMigration{
		logger: logger,
	}
}

// Execute runs the migration
func (m *TestMigration) Execute(db *gorm.DB) error {
	m.logger.LogInfo("Executing test migration for CockroachDB", nil)

	// Create a temporary test table if it doesn't exist
	sql := `
		CREATE TABLE IF NOT EXISTS migration_test (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`

	if err := db.Exec(sql).Error; err != nil {
		m.logger.LogError(err, "Failed to create test table")
		return fmt.Errorf("failed to create test table: %w", err)
	}

	// Insert a test record to confirm migration ran
	insertSQL := `
		INSERT INTO migration_test (name) 
		VALUES ('Test migration executed at ' || NOW()::text)
	`

	if err := db.Exec(insertSQL).Error; err != nil {
		m.logger.LogError(err, "Failed to insert test record")
		return fmt.Errorf("failed to insert test record: %w", err)
	}

	m.logger.LogInfo("Test migration completed successfully", nil)
	return nil
} 