package main

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// initDatabase initializes the PostgreSQL connection using GORM
func initDatabase(config *DatabaseConfig) (*gorm.DB, error) {
	// Construct DSN from configuration
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		config.Host,
		config.User,
		config.Password,
		config.Dbname,
		config.Port,
		config.Sslmode,
		config.Timezone,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Create the upload_status enum type if it doesn't exist
	if err := db.Exec(`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'upload_status') THEN
				CREATE TYPE upload_status AS ENUM ('pending', 'uploading', 'completed', 'failed');
			END IF;
		END
		$$;`).Error; err != nil {
		return nil, fmt.Errorf("failed to create upload_status enum: %v", err)
	}

	// Auto-migrate the schema
	if err = db.AutoMigrate(&User{}, &Video{}, &Transcode{}, &TranscodeSegment{}); err != nil {
		return nil, fmt.Errorf("auto migration failed: %v", err)
	}

	return db, nil
}
