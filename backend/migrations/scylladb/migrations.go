package scylladb

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/gocql/gocql"
)

// MigrationRunner manages ScyllaDB migrations
type MigrationRunner struct {
	session  *gocql.Session
	keyspace string
	logger   logger.Logger
}

// NewMigrationRunner creates a new ScyllaDB migration runner
func NewMigrationRunner(session *gocql.Session, keyspace string, logger logger.Logger) *MigrationRunner {
	return &MigrationRunner{
		session:  session,
		keyspace: keyspace,
		logger:   logger,
	}
}

// RunMigrations executes all ScyllaDB migrations in order
func (r *MigrationRunner) RunMigrations() error {
	r.logger.LogInfo("Starting ScyllaDB migrations", nil)

	// Initialize migrations tracking table
	if err := r.initializeMigrationsTable(); err != nil {
		r.logger.LogError(err, "Failed to initialize migrations table")
		return err
	}

	// Run test migration
	migrationName := "001_test_migration"
	migrationContent := "Test migration to demonstrate how migrations work"
	
	// Check if this migration has already been applied
	applied, err := r.hasMigrationBeenApplied(migrationName)
	if err != nil {
		r.logger.LogError(err, "Failed to check if migration has been applied")
		return err
	}
	
	if applied {
		r.logger.LogInfo(fmt.Sprintf("Migration %s already applied, skipping", migrationName), nil)
	} else {
		r.logger.LogInfo("Running test migration", nil)
		testMigration := NewTestMigration(r.logger)
		if err := testMigration.Execute(r.session, r.keyspace); err != nil {
			r.logger.LogError(err, "Failed to run test migration")
			return err
		}
		
		// Record the migration as successful
		if err := r.recordMigration(migrationName, migrationContent); err != nil {
			r.logger.LogError(err, "Failed to record migration")
			return err
		}
	}

	// Add more migrations here following the same pattern
	// Example: 
	// migrationName = "002_next_migration"
	// migrationContent = "Description of what this migration does"
	// 
	// applied, err = r.hasMigrationBeenApplied(migrationName)
	// if err != nil {
	//     r.logger.LogError(err, "Failed to check if migration has been applied")
	//     return err
	// }
	// 
	// if !applied {
	//     r.logger.LogInfo("Running next migration", nil)
	//     nextMigration := NewNextMigration(r.logger)
	//     if err := nextMigration.Execute(r.session, r.keyspace); err != nil {
	//         r.logger.LogError(err, "Failed to run next migration")
	//         return err
	//     }
	//     
	//     if err := r.recordMigration(migrationName, migrationContent); err != nil {
	//         r.logger.LogError(err, "Failed to record migration")
	//         return err
	//     }
	// }

	r.logger.LogInfo("ScyllaDB migrations completed successfully", nil)
	return nil
}

// initializeMigrationsTable creates the schema_migrations table for tracking migrations
func (r *MigrationRunner) initializeMigrationsTable() error {
	r.logger.LogInfo("Initializing ScyllaDB migrations table", nil)
	
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.schema_migrations (
			id uuid,
			name text,
			hash text,
			applied_at timestamp,
			batch_no int,
			PRIMARY KEY (name)
		)`,
		r.keyspace,
	)
	
	if err := r.session.Query(query).Exec(); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}
	
	r.logger.LogInfo("ScyllaDB migrations table initialized", nil)
	return nil
}

// hasMigrationBeenApplied checks if a specific migration has already been run
func (r *MigrationRunner) hasMigrationBeenApplied(name string) (bool, error) {
	r.logger.LogInfo(fmt.Sprintf("Checking if migration %s has been applied", name), nil)
	
	var id gocql.UUID
	query := fmt.Sprintf(`
		SELECT id FROM %s.schema_migrations WHERE name = ? LIMIT 1
	`, r.keyspace)
	
	err := r.session.Query(query, name).Scan(&id)
	if err != nil {
		// Not found is not an error in this context
		if err == gocql.ErrNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check migration status: %w", err)
	}
	
	// If we get here, the migration exists in the table
	return true, nil
}

// recordMigration records a successful migration
func (r *MigrationRunner) recordMigration(name, content string) error {
	r.logger.LogInfo(fmt.Sprintf("Recording migration %s", name), nil)
	
	// Calculate hash of migration content
	hash := sha256.Sum256([]byte(content))
	hashStr := hex.EncodeToString(hash[:])
	
	// Get the current batch number (maximum batch_no + 1)
	var batchNo int
	batchQuery := fmt.Sprintf(`
		SELECT MAX(batch_no) FROM %s.schema_migrations
	`, r.keyspace)
	
	err := r.session.Query(batchQuery).Scan(&batchNo)
	if err != nil && err != gocql.ErrNotFound {
		return fmt.Errorf("failed to determine batch number: %w", err)
	}
	
	// Increment batch number (or start at 1 if no migrations exist)
	batchNo++
	
	// Generate a random UUID for the migration record
	id, err := gocql.RandomUUID()
	if err != nil {
		return fmt.Errorf("failed to generate UUID: %w", err)
	}
	now := time.Now()
	
	// Insert the migration record
	insertQuery := fmt.Sprintf(`
		INSERT INTO %s.schema_migrations 
		(id, name, hash, applied_at, batch_no) 
		VALUES (?, ?, ?, ?, ?)
	`, r.keyspace)
	
	if err := r.session.Query(insertQuery, id, name, hashStr, now, batchNo).Exec(); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}
	
	r.logger.LogInfo(fmt.Sprintf("Migration %s recorded successfully", name), nil)
	return nil
} 