package migrations

import (
	"fmt"

	"github.com/consensuslabs/pavilion-network/backend/internal/database"
	"gorm.io/gorm"
)

type Migrator interface {
	Up() error
	Down() error
}

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

	migrations := []struct {
		Name     string
		Migrator Migrator
	}{
		{"001_create_video_uploads.go", NewMigration(db)},
		{"002_create_videos.go", NewVideoMigration(db)},
	}

	if direction == "up" {
		for i, migration := range migrations {
			// Check if migration has already been applied
			applied, err := migrationConfig.HasMigrationBeenApplied(migration.Name)
			if err != nil {
				return fmt.Errorf("failed to check migration status: %v", err)
			}
			if applied {
				migrationConfig.Logger.LogInfo("Migration already applied", map[string]interface{}{
					"migration": migration.Name,
				})
				continue
			}

			migrationConfig.Logger.LogInfo("Running migration up", map[string]interface{}{
				"index": i + 1,
				"name":  migration.Name,
			})
			if err := migration.Migrator.Up(); err != nil {
				return fmt.Errorf("failed to run migration %d up: %v", i+1, err)
			}

			// Record successful migration
			if err := migrationConfig.RecordMigration(migration.Name, fmt.Sprintf("Migration %s executed successfully", migration.Name)); err != nil {
				return fmt.Errorf("failed to record migration %s: %v", migration.Name, err)
			}
			migrationConfig.Logger.LogInfo("Successfully recorded migration", map[string]interface{}{
				"migration": migration.Name,
			})
		}
	} else if direction == "down" {
		// Run migrations in reverse order
		for i := len(migrations) - 1; i >= 0; i-- {
			migrationConfig.Logger.LogInfo("Running migration down", map[string]interface{}{
				"index": i + 1,
				"name":  migrations[i].Name,
			})
			if err := migrations[i].Migrator.Down(); err != nil {
				return fmt.Errorf("failed to run migration %d down: %v", i+1, err)
			}
			// Note: We don't remove records for down migrations to maintain history
		}
	} else {
		return fmt.Errorf("invalid migration direction: %s", direction)
	}

	return nil
}
