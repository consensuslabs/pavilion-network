package database

import "gorm.io/gorm"

// Service defines the interface for database operations
type Service interface {
	Connect() (*gorm.DB, error)
	Close() error
}

// Logger interface for logging operations
type Logger interface {
	LogInfo(msg string, fields map[string]interface{})
	LogError(err error, msg string) error
}
