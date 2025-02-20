package database

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"gorm.io/gorm"
)

// MigrationConfig holds configuration for database migrations
type MigrationConfig struct {
	Environment string
	AutoMigrate bool
	ForceRun    bool
	db          *gorm.DB
}

// NewMigrationConfig creates a new migration configuration
func NewMigrationConfig(db *gorm.DB) *MigrationConfig {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	// Check AUTO_MIGRATE env var, defaults to true in development, false otherwise
	autoMigrate := false
	if autoMigrateEnv := os.Getenv("AUTO_MIGRATE"); autoMigrateEnv != "" {
		autoMigrate = autoMigrateEnv == "true"
	} else {
		// Default behavior if AUTO_MIGRATE is not set
		autoMigrate = env == "development"
	}

	return &MigrationConfig{
		Environment: env,
		AutoMigrate: autoMigrate,
		ForceRun:    os.Getenv("FORCE_MIGRATION") == "true",
		db:          db,
	}
}

// InitializeMigrationTable creates the migrations tracking table
func (c *MigrationConfig) InitializeMigrationTable() error {
	// Check if table exists
	if !c.db.Migrator().HasTable(&MigrationRecord{}) {
		// Create table first
		createTableSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		hash TEXT NOT NULL,
		applied_at TIMESTAMP NOT NULL,
		batch_no INT NOT NULL
		)`
		if err := c.db.Exec(createTableSQL).Error; err != nil {
			return fmt.Errorf("failed to create schema_migrations table: %v", err)
		}

		// Then create the index
		createIndexSQL := `CREATE UNIQUE INDEX IF NOT EXISTS idx_schema_migrations_name ON schema_migrations(name)`
		if err := c.db.Exec(createIndexSQL).Error; err != nil {
			return fmt.Errorf("failed to create index on schema_migrations: %v", err)
		}
	}

	// Table exists, no need to do anything
	return nil
}

// HasMigrationBeenApplied checks if a specific migration has already been run
func (c *MigrationConfig) HasMigrationBeenApplied(name string) (bool, error) {
	var count int64
	err := c.db.Model(&MigrationRecord{}).Where("name = ?", name).Count(&count).Error
	return count > 0, err
}

// RecordMigration records a successful migration
func (c *MigrationConfig) RecordMigration(name string, content string) error {
	// Calculate hash of migration content
	hash := sha256.Sum256([]byte(content))
	hashStr := hex.EncodeToString(hash[:])

	// Get the current batch number
	var batchNo int
	err := c.db.Model(&MigrationRecord{}).Select("COALESCE(MAX(batch_no), 0) + 1").Row().Scan(&batchNo)
	if err != nil {
		return fmt.Errorf("failed to determine batch number: %v", err)
	}

	// Record the migration
	record := MigrationRecord{
		Name:      name,
		Hash:      hashStr,
		AppliedAt: time.Now(),
		BatchNo:   batchNo,
	}

	return c.db.Create(&record).Error
}

// GetAppliedMigrations returns a list of all applied migrations
func (c *MigrationConfig) GetAppliedMigrations() ([]MigrationRecord, error) {
	var migrations []MigrationRecord
	err := c.db.Order("applied_at").Find(&migrations).Error
	return migrations, err
}

// ShouldRunMigration determines if migrations should be executed
func (c *MigrationConfig) ShouldRunMigration() bool {
	// If FORCE_MIGRATION is true, always run migrations regardless of environment
	if c.ForceRun {
		return true
	}

	// In development or test, check AUTO_MIGRATE
	if c.Environment == "development" || c.Environment == "test" {
		return c.AutoMigrate
	}

	// In production or other environments, don't run migrations unless forced
	return false
}

// ShouldAutoMigrate determines if auto-migration should run
func (c *MigrationConfig) ShouldAutoMigrate() bool {
	return c.AutoMigrate
}

// ValidateMigration validates if a migration should run in the current environment
func (c *MigrationConfig) ValidateMigration(migrationName string) error {
	if !c.ShouldRunMigration() {
		return fmt.Errorf("migrations are disabled in %s environment. Use FORCE_MIGRATION=true to override", c.Environment)
	}

	if c.Environment == "production" && !c.ForceRun {
		return fmt.Errorf("attempting to run migration %s in production without force flag", migrationName)
	}

	// Check if migration has already been applied
	applied, err := c.HasMigrationBeenApplied(migrationName)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %v", err)
	}
	if applied {
		return fmt.Errorf("migration %s has already been applied", migrationName)
	}

	return nil
}
