package main

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// initDatabase initializes the PostgreSQL connection using GORM
func initDatabase(config DatabaseConfig) (*gorm.DB, error) {
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

	// Auto-migrate the schema
	if err = db.AutoMigrate(&User{}, &Video{}, &Transcode{}, &TranscodeSegment{}); err != nil {
		return nil, fmt.Errorf("auto migration failed: %v", err)
	}

	return db, nil
}
