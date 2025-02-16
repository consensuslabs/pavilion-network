package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

type Migrator interface {
	Up() error
	Down() error
}

func RunMigrations(db *gorm.DB, direction string) error {
	// Ensure we're using the correct database
	if err := db.Exec("USE pavilion_db").Error; err != nil {
		return fmt.Errorf("failed to switch to pavilion_db: %v", err)
	}

	migrations := []Migrator{
		NewMigration(db),      // 001_create_video_uploads
		NewVideoMigration(db), // 002_create_videos
	}

	if direction == "up" {
		for i, migration := range migrations {
			log.Printf("Running migration %d up", i+1)
			if err := migration.Up(); err != nil {
				return fmt.Errorf("failed to run migration %d up: %v", i+1, err)
			}
		}
	} else if direction == "down" {
		// Run migrations in reverse order
		for i := len(migrations) - 1; i >= 0; i-- {
			log.Printf("Running migration %d down", i+1)
			if err := migrations[i].Down(); err != nil {
				return fmt.Errorf("failed to run migration %d down: %v", i+1, err)
			}
		}
	} else {
		return fmt.Errorf("invalid migration direction: %s", direction)
	}

	return nil
}
