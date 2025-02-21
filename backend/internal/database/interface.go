package database

import (
	"context"

	"gorm.io/gorm"
)

// Service defines the interface for database operations
type Service interface {
	Connect() (*gorm.DB, error)
	Close() error
}

// Logger interface for logging operations
type Logger interface {
	LogInfo(msg string, fields map[string]interface{})
	LogError(err error, msg string, fields ...map[string]interface{}) error
	LogErrorf(err error, format string, args ...interface{}) error
	LogWarn(message string, fields map[string]interface{})
	LogDebug(message string, fields map[string]interface{})
	LogFatal(err error, context string)
	WithContext(ctx context.Context) Logger
}
