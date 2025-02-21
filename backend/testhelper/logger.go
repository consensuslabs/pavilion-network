package testhelper

import (
	"context"
	"fmt"
	"sync"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
)

// TestLogger provides a logger implementation for testing with debug capabilities
type TestLogger struct {
	mu            sync.RWMutex
	infoMessages  []LogEntry
	errorMessages []LogEntry
	warnMessages  []LogEntry
	debugMessages []LogEntry
	fields        map[string]interface{}
	contextLogger *TestLogger
	debugEnabled  bool
}

// LogEntry represents a log entry with its message and fields
type LogEntry struct {
	Message string
	Fields  map[string]interface{}
}

// NewTestLogger creates a new test logger instance
func NewTestLogger(debugEnabled bool) *TestLogger {
	return &TestLogger{
		fields:       make(map[string]interface{}),
		debugEnabled: debugEnabled,
	}
}

// LogInfo implements logger.Logger
func (t *TestLogger) LogInfo(msg string, fields map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.contextLogger != nil {
		t.contextLogger.LogInfo(msg, fields)
		return
	}

	t.infoMessages = append(t.infoMessages, LogEntry{Message: msg, Fields: t.mergeFields(fields)})
}

// LogError implements logger.Logger
func (t *TestLogger) LogError(err error, msg string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.contextLogger != nil {
		return t.contextLogger.LogError(err, msg)
	}

	fields := map[string]interface{}{}
	if err != nil {
		fields["error"] = err.Error()
	}
	t.errorMessages = append(t.errorMessages, LogEntry{Message: msg, Fields: t.mergeFields(fields)})
	return err
}

// LogErrorf implements logger.Logger
func (t *TestLogger) LogErrorf(err error, format string, args ...interface{}) error {
	return t.LogError(err, fmt.Sprintf(format, args...))
}

// LogFatal implements logger.Logger
func (t *TestLogger) LogFatal(err error, context string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	fields := map[string]interface{}{
		"context": context,
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	// In test mode, we don't want to actually exit
	t.errorMessages = append(t.errorMessages, LogEntry{Message: "FATAL: " + context, Fields: t.mergeFields(fields)})
}

// LogDebug implements logger.Logger
func (t *TestLogger) LogDebug(message string, fields map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.debugEnabled {
		return
	}

	if t.contextLogger != nil {
		t.contextLogger.LogDebug(message, fields)
		return
	}

	t.debugMessages = append(t.debugMessages, LogEntry{Message: message, Fields: t.mergeFields(fields)})
}

// LogWarn implements logger.Logger
func (t *TestLogger) LogWarn(message string, fields map[string]interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.contextLogger != nil {
		t.contextLogger.LogWarn(message, fields)
		return
	}

	t.warnMessages = append(t.warnMessages, LogEntry{Message: message, Fields: t.mergeFields(fields)})
}

// WithFields implements logger.Logger
func (t *TestLogger) WithFields(fields map[string]interface{}) logger.Logger {
	t.mu.Lock()
	defer t.mu.Unlock()

	newLogger := NewTestLogger(t.debugEnabled)
	newLogger.fields = t.mergeFields(fields)
	newLogger.infoMessages = t.infoMessages
	newLogger.errorMessages = t.errorMessages
	newLogger.warnMessages = t.warnMessages
	newLogger.debugMessages = t.debugMessages
	t.contextLogger = newLogger
	return newLogger
}

// WithContext implements logger.Logger
func (t *TestLogger) WithContext(ctx context.Context) logger.Logger {
	return t.WithFields(nil)
}

// WithRequestID implements logger.Logger
func (t *TestLogger) WithRequestID(requestID string) logger.Logger {
	return t.WithFields(map[string]interface{}{
		"requestID": requestID,
	})
}

// WithUserID implements logger.Logger
func (t *TestLogger) WithUserID(userID string) logger.Logger {
	return t.WithFields(map[string]interface{}{
		"userID": userID,
	})
}

// GetInfoMessages returns all info level messages
func (t *TestLogger) GetInfoMessages() []LogEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.infoMessages
}

// GetErrorMessages returns all error level messages
func (t *TestLogger) GetErrorMessages() []LogEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.errorMessages
}

// GetWarnMessages returns all warning level messages
func (t *TestLogger) GetWarnMessages() []LogEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.warnMessages
}

// GetDebugMessages returns all debug level messages
func (t *TestLogger) GetDebugMessages() []LogEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.debugMessages
}

// ClearMessages clears all logged messages
func (t *TestLogger) ClearMessages() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.infoMessages = nil
	t.errorMessages = nil
	t.warnMessages = nil
	t.debugMessages = nil
}

// EnableDebug enables debug logging
func (t *TestLogger) EnableDebug() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.debugEnabled = true
}

// DisableDebug disables debug logging
func (t *TestLogger) DisableDebug() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.debugEnabled = false
}

// mergeFields merges the logger's base fields with the provided fields
func (t *TestLogger) mergeFields(fields map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{}, len(t.fields)+len(fields))
	for k, v := range t.fields {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}
	return merged
}
