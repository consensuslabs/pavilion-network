package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// initDatabase initializes the PostgreSQL connection using GORM
func initDatabase(config *DatabaseConfig) (*gorm.DB, error) {
	log.Printf("Attempting to connect to database: %s", config.Dbname)

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

	log.Printf("Using database connection string (without credentials): host=%s dbname=%s port=%d",
		config.Host, config.Dbname, config.Port)

	// Configure GORM with CockroachDB-specific settings
	gormConfig := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, // CockroachDB handles foreign keys differently
		PrepareStmt:                              true, // Enable prepared statement cache
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Explicitly set the database
	if err := db.Exec("USE " + config.Dbname).Error; err != nil {
		return nil, fmt.Errorf("failed to switch to database %s: %v", config.Dbname, err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %v", err)
	}

	// Debug: Log current database
	var currentDB string
	if err := sqlDB.QueryRow("SELECT current_database()").Scan(&currentDB); err != nil {
		log.Printf("Failed to get current database: %v", err)
	} else {
		log.Printf("Connected to database: %s", currentDB)
	}

	// Configure connection pool using values from config
	sqlDB.SetMaxIdleConns(config.Pool.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.Pool.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.Pool.ConnMaxLifetime)

	// Create enum type using a transaction to handle CockroachDB's transaction retry logic
	err = db.Transaction(func(tx *gorm.DB) error {
		return tx.Exec(`CREATE TYPE IF NOT EXISTS upload_status AS ENUM ('pending', 'uploading', 'completed', 'failed')`).Error
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create upload_status enum: %v", err)
	}

	// Auto-migrate the schema
	if err = db.AutoMigrate(&User{}, &Video{}, &Transcode{}, &TranscodeSegment{}); err != nil {
		return nil, fmt.Errorf("auto migration failed: %v", err)
	}

	return db, nil
}
