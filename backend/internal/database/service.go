package database

import (
	"fmt"

	"github.com/consensuslabs/pavilion-network/backend/internal/auth"
	"github.com/consensuslabs/pavilion-network/backend/internal/config"
	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DatabaseService implements the Service interface
type DatabaseService struct {
	config *config.DatabaseConfig
	logger Logger
	db     *gorm.DB
}

// NewDatabaseService creates a new database service instance
func NewDatabaseService(config *config.DatabaseConfig, logger Logger) *DatabaseService {
	return &DatabaseService{
		config: config,
		logger: logger,
	}
}

// Connect establishes a connection to the database
func (s *DatabaseService) Connect() (*gorm.DB, error) {
	s.logger.LogInfo(fmt.Sprintf("Attempting to connect to database: %s", s.config.Dbname), nil)

	// Construct DSN from configuration
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		s.config.Host,
		s.config.User,
		s.config.Password,
		s.config.Dbname,
		s.config.Port,
		s.config.Sslmode,
		s.config.Timezone,
	)

	s.logger.LogInfo(fmt.Sprintf("Using database connection string (without credentials): host=%s dbname=%s port=%d",
		s.config.Host, s.config.Dbname, s.config.Port), nil)

	// Configure GORM with CockroachDB-specific settings
	gormConfig := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, // CockroachDB handles foreign keys differently
		PrepareStmt:                              true, // Enable prepared statement cache
		Logger:                                   logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Explicitly set the database
	if err := db.Exec("USE " + s.config.Dbname).Error; err != nil {
		return nil, fmt.Errorf("failed to switch to database %s: %v", s.config.Dbname, err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %v", err)
	}

	// Debug: Log current database
	var currentDB string
	if err := sqlDB.QueryRow("SELECT current_database()").Scan(&currentDB); err != nil {
		s.logger.LogInfo(fmt.Sprintf("Failed to get current database: %v", err), nil)
	} else {
		s.logger.LogInfo(fmt.Sprintf("Connected to database: %s", currentDB), nil)
	}

	// Configure connection pool using values from config
	sqlDB.SetMaxOpenConns(s.config.Pool.MaxOpen)
	sqlDB.SetMaxIdleConns(s.config.Pool.MaxIdle)

	// Create enum type using a transaction to handle CockroachDB's transaction retry logic
	err = db.Transaction(func(tx *gorm.DB) error {
		return tx.Exec(`CREATE TYPE IF NOT EXISTS upload_status AS ENUM ('pending', 'uploading', 'completed', 'failed')`).Error
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create upload_status enum: %v", err)
	}

	// Auto-migrate the schema
	if err = db.AutoMigrate(&auth.User{}, &auth.RefreshToken{}, &video.Video{}, &video.Transcode{}, &video.TranscodeSegment{}); err != nil {
		return nil, fmt.Errorf("auto migration failed: %v", err)
	}

	s.db = db
	return db, nil
}

// Close closes the database connection
func (s *DatabaseService) Close() error {
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err != nil {
			return fmt.Errorf("failed to get database instance: %v", err)
		}
		if err := sqlDB.Close(); err != nil {
			return fmt.Errorf("failed to close database connection: %v", err)
		}
	}
	return nil
}
