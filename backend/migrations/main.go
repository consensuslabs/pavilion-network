package migrations

import (
	"fmt"

	"github.com/consensuslabs/pavilion-network/backend/internal/database"
	"gorm.io/gorm"
)

// MigrationRunner handles database migrations
type MigrationRunner struct {
	db     *gorm.DB
	config *database.MigrationConfig
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *gorm.DB, config *database.MigrationConfig) *MigrationRunner {
	return &MigrationRunner{
		db:     db,
		config: config,
	}
}

// RunMigrations runs all migrations in the specified direction
func RunMigrations(db *gorm.DB, direction string) error {
	// Initialize migration config with a default logger
	defaultLogger, err := database.NewDefaultLogger()
	if err != nil {
		return fmt.Errorf("failed to create default logger: %v", err)
	}

	// Initialize migration config
	migrationConfig := database.NewMigrationConfig(db, defaultLogger)

	// Log migration configuration
	migrationConfig.Logger.LogInfo("Migration Configuration", map[string]interface{}{
		"environment":     migrationConfig.Environment,
		"auto_migrate":    migrationConfig.AutoMigrate,
		"force_migration": migrationConfig.ForceRun,
	})

	// Initialize migration table before any other operations
	if err := migrationConfig.InitializeMigrationTable(); err != nil {
		return fmt.Errorf("failed to initialize migration table: %v", err)
	}

	// Check if migrations should run
	if !migrationConfig.ShouldRunMigration() {
		if migrationConfig.Environment == "development" {
			migrationConfig.Logger.LogInfo("Skipping migrations", map[string]interface{}{
				"auto_migrate":    migrationConfig.AutoMigrate,
				"force_migration": migrationConfig.ForceRun,
			})
		} else {
			migrationConfig.Logger.LogInfo("Skipping migrations in non-development environment", map[string]interface{}{
				"environment":     migrationConfig.Environment,
				"force_migration": migrationConfig.ForceRun,
			})
		}
		return nil
	}

	// For now, we rely on auto-migration in development/testing
	// Add versioned migrations here when needed for production
	return nil
}
