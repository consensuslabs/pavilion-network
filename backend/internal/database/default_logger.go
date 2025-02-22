package database

import (
	"context"
	"fmt"
	"log"
)

// defaultLogger provides a simple logger implementation for cases where a full logger is not available
type defaultLogger struct{}

// NewDefaultLogger creates a new default logger instance
func NewDefaultLogger() (Logger, error) {
	return &defaultLogger{}, nil
}

func (l *defaultLogger) LogInfo(msg string, fields map[string]interface{}) {
	log.Printf("INFO: %s %v", msg, fields)
}

func (l *defaultLogger) LogError(err error, msg string) error {
	log.Printf("ERROR: %s: %v", msg, err)
	return err
}

func (l *defaultLogger) LogErrorf(err error, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	log.Printf("ERROR: %s: %v", msg, err)
	return err
}

func (l *defaultLogger) LogWarn(message string, fields map[string]interface{}) {
	log.Printf("WARN: %s %v", message, fields)
}

func (l *defaultLogger) LogDebug(message string, fields map[string]interface{}) {
	log.Printf("DEBUG: %s %v", message, fields)
}

func (l *defaultLogger) LogFatal(err error, context string) {
	log.Printf("FATAL: %s: %v", context, err)
}

func (l *defaultLogger) WithContext(ctx context.Context) Logger {
	return l
}

func (l *defaultLogger) WithFields(fields map[string]interface{}) Logger {
	return l
}

func (l *defaultLogger) WithRequestID(requestID string) Logger {
	return l
}

func (l *defaultLogger) WithUserID(userID string) Logger {
	return l
}
