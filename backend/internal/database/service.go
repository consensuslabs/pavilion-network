package database

import (
	"fmt"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/auth"
	"github.com/consensuslabs/pavilion-network/backend/internal/config"
	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DatabaseService implements the Service interface
type DatabaseService struct {
	config          *config.DatabaseConfig
	logger          Logger
	db              *gorm.DB
	migrationConfig *MigrationConfig
}

// NewDatabaseService creates a new database service instance
func NewDatabaseService(config *config.DatabaseConfig, logger Logger) *DatabaseService {
	service := &DatabaseService{
		config: config,
		logger: logger,
	}
	return service
}

// Connect establishes a connection to the database
func (s *DatabaseService) Connect() (*gorm.DB, error) {
	s.logger.LogInfo("Attempting to connect to database", map[string]interface{}{
		"database": s.config.Dbname,
		"host":     s.config.Host,
		"port":     s.config.Port,
	})

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

	// Configure GORM with CockroachDB-specific settings
	gormConfig := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, // CockroachDB handles foreign keys differently
		PrepareStmt:                              true, // Enable prepared statement cache
		Logger:                                   NewGormLogger(s.logger, 200*time.Millisecond),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		s.logger.LogError(err, "Failed to connect to database")
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Explicitly set the database
	if err := db.Exec("USE " + s.config.Dbname).Error; err != nil {
		s.logger.LogError(err, "Failed to switch database")
		return nil, fmt.Errorf("failed to switch to database %s: %v", s.config.Dbname, err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		s.logger.LogError(err, "Failed to get database instance")
		return nil, fmt.Errorf("failed to get database instance: %v", err)
	}

	// Debug: Log current database
	var currentDB string
	if err := sqlDB.QueryRow("SELECT current_database()").Scan(&currentDB); err != nil {
		s.logger.LogWarn("Failed to get current database", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		s.logger.LogInfo("Connected to database", map[string]interface{}{
			"database": currentDB,
		})
	}

	// Configure connection pool using values from config
	sqlDB.SetMaxOpenConns(s.config.Pool.MaxOpen)
	sqlDB.SetMaxIdleConns(s.config.Pool.MaxIdle)

	s.logger.LogInfo("Configured connection pool", map[string]interface{}{
		"maxOpenConns": s.config.Pool.MaxOpen,
		"maxIdleConns": s.config.Pool.MaxIdle,
	})

	// Initialize migration config now that we have the database connection
	s.migrationConfig = NewMigrationConfig(db, s.logger)
	s.logger.LogInfo("Initialized migration config", map[string]interface{}{
		"environment": s.migrationConfig.Environment,
		"autoMigrate": s.migrationConfig.AutoMigrate,
		"forceRun":    s.migrationConfig.ForceRun,
	})

	// Create enum type using a transaction to handle CockroachDB's transaction retry logic
	err = db.Transaction(func(tx *gorm.DB) error {
		return tx.Exec(`CREATE TYPE IF NOT EXISTS upload_status AS ENUM ('pending', 'uploading', 'completed', 'failed')`).Error
	})
	if err != nil {
		s.logger.LogError(err, "Failed to create upload_status enum")
		return nil, fmt.Errorf("failed to create upload_status enum: %v", err)
	}

	// Initialize migration tracking table
	if err := s.migrationConfig.InitializeMigrationTable(); err != nil {
		s.logger.LogError(err, "Failed to initialize migration tracking")
		return nil, fmt.Errorf("failed to initialize migration tracking: %v", err)
	}

	// Get list of applied migrations
	migrations, err := s.migrationConfig.GetAppliedMigrations()
	if err != nil {
		s.logger.LogError(err, "Failed to get applied migrations")
		return nil, fmt.Errorf("failed to get applied migrations: %v", err)
	}

	s.logger.LogInfo("Found applied migrations", map[string]interface{}{
		"count": len(migrations),
	})
	for _, migration := range migrations {
		s.logger.LogDebug("Applied migration", map[string]interface{}{
			"name":       migration.Name,
			"applied_at": migration.AppliedAt,
			"batch_no":   migration.BatchNo,
		})
	}

	// Only run auto-migration in development or when explicitly enabled
	if s.migrationConfig.ShouldAutoMigrate() {
		s.logger.LogInfo("Running auto-migration", nil)
		if err = db.AutoMigrate(&auth.User{}, &auth.RefreshToken{}, &video.Video{}, &video.Transcode{}, &video.TranscodeSegment{}); err != nil {
			s.logger.LogError(err, "Auto-migration failed")
			return nil, fmt.Errorf("auto migration failed: %v", err)
		}
		s.logger.LogInfo("Auto-migration completed successfully", nil)
	} else {
		s.logger.LogInfo("Skipping auto-migration", map[string]interface{}{
			"environment": s.migrationConfig.Environment,
		})
	}

	s.db = db
	return db, nil
}

// Close closes the database connection
func (s *DatabaseService) Close() error {
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err != nil {
			s.logger.LogError(err, "Failed to get database instance during close")
			return fmt.Errorf("failed to get database instance: %v", err)
		}
		if err := sqlDB.Close(); err != nil {
			s.logger.LogError(err, "Failed to close database connection")
			return fmt.Errorf("failed to close database connection: %v", err)
		}
		s.logger.LogInfo("Database connection closed successfully", nil)
	}
	return nil
}
