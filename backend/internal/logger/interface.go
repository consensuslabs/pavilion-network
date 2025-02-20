package logger

import "context"

// Logger defines the interface for logging operations
type Logger interface {
	// Core logging methods
	LogInfo(msg string, fields map[string]interface{})
	LogError(err error, msg string) error
	LogErrorf(err error, format string, args ...interface{}) error
	LogFatal(err error, context string)
	LogDebug(message string, fields map[string]interface{})
	LogWarn(message string, fields map[string]interface{})

	// Context and field management
	WithFields(fields map[string]interface{}) Logger
	WithContext(ctx context.Context) Logger
	WithRequestID(requestID string) Logger
	WithUserID(userID string) Logger
}
