package migrations

import (
	"fmt"
	"log"

	"github.com/consensuslabs/pavilion-network/backend/internal/database"
	"gorm.io/gorm"
)

type Migrator interface {
	Up() error
	Down() error
}

func RunMigrations(db *gorm.DB, direction string) error {

	// Initialize migration config
	migrationConfig := database.NewMigrationConfig(db)

	// Log migration configuration
	log.Printf("Migration Configuration:")
	log.Printf("- Environment: %s", migrationConfig.Environment)
	log.Printf("- Auto Migrate: %v", migrationConfig.AutoMigrate)
	log.Printf("- Force Migration: %v", migrationConfig.ForceRun)

	// Check if migrations should run
	if !migrationConfig.ShouldRunMigration() {
		if migrationConfig.Environment == "development" {
			log.Printf("Skipping migrations (AUTO_MIGRATE=%v, FORCE_MIGRATION=%v)",
				migrationConfig.AutoMigrate, migrationConfig.ForceRun)
		} else {
			log.Printf("Skipping migrations in %s environment (FORCE_MIGRATION=%v)",
				migrationConfig.Environment, migrationConfig.ForceRun)
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
				log.Printf("Migration %s already applied, skipping", migration.Name)
				continue
			}

			log.Printf("Running migration %d up: %s", i+1, migration.Name)
			if err := migration.Migrator.Up(); err != nil {
				return fmt.Errorf("failed to run migration %d up: %v", i+1, err)
			}

			// Record successful migration
			if err := migrationConfig.RecordMigration(migration.Name, fmt.Sprintf("Migration %s executed successfully", migration.Name)); err != nil {
				return fmt.Errorf("failed to record migration %s: %v", migration.Name, err)
			}
			log.Printf("Successfully recorded migration: %s", migration.Name)
		}
	} else if direction == "down" {
		// Run migrations in reverse order
		for i := len(migrations) - 1; i >= 0; i-- {
			log.Printf("Running migration %d down: %s", i+1, migrations[i].Name)
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
