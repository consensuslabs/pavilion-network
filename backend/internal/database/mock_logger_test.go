package database

import (
	"context"
	"fmt"
	"sync"
)

// mockLogEntry represents a log entry with its message and fields
type mockLogEntry struct {
	Message string
	Fields  map[string]interface{}
}

// mockLogger provides a logger implementation for testing
type mockLogger struct {
	mu            sync.RWMutex
	infoMessages  []mockLogEntry
	errorMessages []mockLogEntry
	warnMessages  []mockLogEntry
	debugMessages []mockLogEntry
	fatalMessages []mockLogEntry
	fields        map[string]interface{}
}

// newMockLogger creates a new mock logger instance
func newMockLogger() *mockLogger {
	return &mockLogger{
		fields: make(map[string]interface{}),
	}
}

// LogInfo implements Logger interface
func (m *mockLogger) LogInfo(msg string, fields map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infoMessages = append(m.infoMessages, mockLogEntry{Message: msg, Fields: m.mergeFields(fields)})
}

// LogError implements Logger interface
func (m *mockLogger) LogError(err error, msg string, fields ...map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Start with error field
	mergedFields := map[string]interface{}{}
	if err != nil {
		mergedFields["error"] = err.Error()
	}

	// Merge additional fields if provided
	if len(fields) > 0 {
		for k, v := range fields[0] {
			mergedFields[k] = v
		}
	}

	m.errorMessages = append(m.errorMessages, mockLogEntry{Message: msg, Fields: m.mergeFields(mergedFields)})
	return err
}

// LogErrorf implements Logger interface
func (m *mockLogger) LogErrorf(err error, format string, args ...interface{}) error {
	return m.LogError(err, fmt.Sprintf(format, args...))
}

// LogWarn implements Logger interface
func (m *mockLogger) LogWarn(message string, fields map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.warnMessages = append(m.warnMessages, mockLogEntry{Message: message, Fields: m.mergeFields(fields)})
}

// LogDebug implements Logger interface
func (m *mockLogger) LogDebug(message string, fields map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debugMessages = append(m.debugMessages, mockLogEntry{Message: message, Fields: m.mergeFields(fields)})
}

// LogFatal implements Logger interface
func (m *mockLogger) LogFatal(err error, context string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fields := map[string]interface{}{
		"context": context,
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	m.fatalMessages = append(m.fatalMessages, mockLogEntry{Message: "FATAL: " + context, Fields: m.mergeFields(fields)})
}

// WithContext implements Logger interface
func (m *mockLogger) WithContext(ctx context.Context) Logger {
	fields := make(map[string]interface{})
	if ctx != nil {
		if traceID, ok := ctx.Value("trace_id").(string); ok {
			fields["trace_id"] = traceID
		}
		if requestID, ok := ctx.Value("request_id").(string); ok {
			fields["request_id"] = requestID
		}
	}
	return m.WithFields(fields)
}

// WithFields creates a new logger with the given fields
func (m *mockLogger) WithFields(fields map[string]interface{}) Logger {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a new logger
	newLogger := &mockLogger{
		fields:        make(map[string]interface{}),
		infoMessages:  m.infoMessages,
		errorMessages: m.errorMessages,
		warnMessages:  m.warnMessages,
		debugMessages: m.debugMessages,
		fatalMessages: m.fatalMessages,
	}

	// Copy existing fields
	for k, v := range m.fields {
		newLogger.fields[k] = v
	}

	// Add new fields
	if fields != nil {
		for k, v := range fields {
			newLogger.fields[k] = v
		}
	}

	return newLogger
}

// GetInfoMessages returns all info level messages
func (m *mockLogger) GetInfoMessages() []mockLogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.infoMessages
}

// GetErrorMessages returns all error level messages
func (m *mockLogger) GetErrorMessages() []mockLogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errorMessages
}

// GetWarnMessages returns all warning level messages
func (m *mockLogger) GetWarnMessages() []mockLogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.warnMessages
}

// GetDebugMessages returns all debug level messages
func (m *mockLogger) GetDebugMessages() []mockLogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.debugMessages
}

// GetFatalMessages returns all fatal level messages
func (m *mockLogger) GetFatalMessages() []mockLogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.fatalMessages
}

// ClearMessages clears all logged messages
func (m *mockLogger) ClearMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infoMessages = nil
	m.errorMessages = nil
	m.warnMessages = nil
	m.debugMessages = nil
	m.fatalMessages = nil
}

// mergeFields merges the logger's base fields with the provided fields
func (m *mockLogger) mergeFields(fields map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	// Copy base fields
	for k, v := range m.fields {
		merged[k] = v
	}

	// Add new fields, overwriting base fields if they exist
	if fields != nil {
		for k, v := range fields {
			merged[k] = v
		}
	}

	return merged
}
