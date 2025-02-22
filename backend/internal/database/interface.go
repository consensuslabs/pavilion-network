package database

import (
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"gorm.io/gorm"
)

// Service defines the interface for database operations
type Service interface {
	Connect() (*gorm.DB, error)
	Close() error
}

// Logger interface for logging operations
type Logger = logger.Logger
