package cockroachdb

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"gorm.io/gorm"
)

// SchemaVersion represents a migration record in the database
type SchemaVersion struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"uniqueIndex;not null"`
	Hash      string    `gorm:"not null"`
	AppliedAt time.Time `gorm:"not null"`
	BatchNo   int       `gorm:"not null"`
}

// TableName overrides the table name
func (SchemaVersion) TableName() string {
	return "schema_migrations"
}

// MigrationRunner manages CockroachDB migrations
type MigrationRunner struct {
	db     *gorm.DB
	logger logger.Logger
}

// NewMigrationRunner creates a new CockroachDB migration runner
func NewMigrationRunner(db *gorm.DB, logger logger.Logger) *MigrationRunner {
	return &MigrationRunner{
		db:     db,
		logger: logger,
	}
}

// RunMigrations executes all CockroachDB migrations in order
func (r *MigrationRunner) RunMigrations() error {
	r.logger.LogInfo("Starting CockroachDB migrations", nil)

	// Initialize migrations tracking table
	if err := r.initializeMigrationsTable(); err != nil {
		r.logger.LogError(err, "Failed to initialize migrations table")
		return err
	}

	// Run test migration
	migrationName := "001_test_migration"
	migrationContent := "Create migration_test table to verify migrations work"
	
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
		if err := testMigration.Execute(r.db); err != nil {
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
	//     if err := nextMigration.Execute(r.db); err != nil {
	//         r.logger.LogError(err, "Failed to run next migration")
	//         return err
	//     }
	//     
	//     if err := r.recordMigration(migrationName, migrationContent); err != nil {
	//         r.logger.LogError(err, "Failed to record migration")
	//         return err
	//     }
	// }

	r.logger.LogInfo("CockroachDB migrations completed successfully", nil)
	return nil
}

// initializeMigrationsTable creates the schema_migrations table for tracking migrations
func (r *MigrationRunner) initializeMigrationsTable() error {
	r.logger.LogInfo("Initializing migrations table for CockroachDB migrations", nil)
	
	// Check if the table already exists
	var count int64
	err := r.db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'schema_migrations'").Count(&count).Error
	if err != nil {
		r.logger.LogError(err, "Failed to check if schema_migrations table exists")
		return fmt.Errorf("failed to check if schema_migrations table exists: %w", err)
	}
	
	if count == 0 {
		// Create the table if it doesn't exist
		createTableSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			name STRING NOT NULL UNIQUE,
			hash STRING NOT NULL,
			applied_at TIMESTAMP NOT NULL,
			batch_no INT NOT NULL
		)
		`
		
		if err := r.db.Exec(createTableSQL).Error; err != nil {
			r.logger.LogError(err, "Failed to create schema_migrations table")
			return fmt.Errorf("failed to create schema_migrations table: %w", err)
		}
	}
	
	r.logger.LogInfo("Migrations table initialized", nil)
	return nil
}

// hasMigrationBeenApplied checks if a specific migration has already been run
func (r *MigrationRunner) hasMigrationBeenApplied(name string) (bool, error) {
	r.logger.LogInfo(fmt.Sprintf("Checking if migration %s has been applied", name), nil)
	
	var count int64
	err := r.db.Model(&SchemaVersion{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check migration status: %w", err)
	}
	
	return count > 0, nil
}

// recordMigration records a successful migration
func (r *MigrationRunner) recordMigration(name, content string) error {
	r.logger.LogInfo(fmt.Sprintf("Recording migration %s", name), nil)
	
	// Calculate hash of migration content
	hash := sha256.Sum256([]byte(content))
	hashStr := hex.EncodeToString(hash[:])
	
	// Get the current batch number (maximum batch_no + 1)
	var batchNo int
	err := r.db.Model(&SchemaVersion{}).Select("COALESCE(MAX(batch_no), 0) + 1").Row().Scan(&batchNo)
	if err != nil {
		return fmt.Errorf("failed to determine batch number: %w", err)
	}
	
	// Create migration record
	record := SchemaVersion{
		Name:      name,
		Hash:      hashStr,
		AppliedAt: time.Now(),
		BatchNo:   batchNo,
	}
	
	// Insert the record
	if err := r.db.Create(&record).Error; err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}
	
	r.logger.LogInfo(fmt.Sprintf("Migration %s recorded successfully", name), nil)
	return nil
} 