# Logger Testing Documentation

## Overview

This document outlines the testing strategy and implementation details for the logging system in the Pavilion Network backend. The logging system consists of multiple logger implementations to support different use cases and testing scenarios.

## Logger Implementations

### 1. Default Logger (`defaultLogger`)

Located in `internal/database/default_logger.go`, this is a simple logger implementation for cases where a full logger is not available:

- Used primarily in database migrations
- Provides basic console output using Go's standard `log` package
- Implements the core `Logger` interface
- Requires no configuration
- Useful for standalone operations and initial setup

```go
type defaultLogger struct{}

func NewDefaultLogger() (Logger, error)
func (l *defaultLogger) LogInfo(msg string, fields map[string]interface{})
func (l *defaultLogger) LogError(err error, msg string, fields ...map[string]interface{}) error
// ... other methods
```

### 2. Mock Logger (`mockLogger`)

Located in `internal/database/mock_logger_test.go`, this implementation is specifically for testing:

- Captures and stores log messages for verification
- Thread-safe with mutex protection
- Supports field merging and context propagation
- Maintains separate message queues for different log levels

```go
type mockLogger struct {
    mu            sync.RWMutex
    infoMessages  []mockLogEntry
    errorMessages []mockLogEntry
    warnMessages  []mockLogEntry
    debugMessages []mockLogEntry
    fatalMessages []mockLogEntry
    fields        map[string]interface{}
}
```

### 3. Test Logger (`TestLogger`)

Located in `testhelper/logger.go`, this is a more sophisticated testing logger:

- Supports debug capabilities
- Provides message querying and clearing
- Implements context chaining
- Thread-safe operations
- Used in integration tests

## Test Environment Setup

### Configuration

The logger tests use a dedicated test environment with:

- Console output for easy debugging
- Debug level logging enabled
- JSON format for structured logging
- In-memory logging for tests

### Required Test Files

1. `config_test.yaml`:
   ```yaml
   logging:
     level: error  # Default to error level for tests
     format: console  # Default to console for better test output
     output: stdout  # Always use stdout for tests
     development: true  # Always true for test environment
     sampling:
       initial: 100  # No sampling in tests
       thereafter: 100
   ```

2. `.env.test`:
   ```env
   LOG_LEVEL=error
   LOG_FORMAT=console
   ```

## Test Cases

### 1. Default Logger Tests

Test basic logging functionality:
```go
func TestDefaultLogger(t *testing.T) {
    logger, _ := NewDefaultLogger()
    logger.LogInfo("test message", nil)
    logger.LogError(errors.New("test error"), "error message")
}
```

### 2. Mock Logger Tests

Test message capture and field handling:
```go
func TestMockLogger(t *testing.T) {
    logger := newMockLogger()
    logger.LogInfo("test info", map[string]interface{}{"key": "value"})
    messages := logger.GetInfoMessages()
    // Verify message content and fields
}
```

### 3. GORM Logger Integration Tests

Test database operation logging:
```go
func TestGormLogger(t *testing.T) {
    logger := newMockLogger()
    gormLogger := NewGormLogger(logger, 200*time.Millisecond)
    // Test various database operations and verify logs
}
```

## Best Practices

1. **Logger Selection**
   - Use `defaultLogger` for standalone operations
   - Use `mockLogger` for unit tests
   - Use `TestLogger` for integration tests

2. **Test Coverage**
   - Test all log levels (Info, Error, Warn, Debug, Fatal)
   - Verify field merging and context propagation
   - Test concurrent logging operations
   - Validate structured logging format

3. **Field Validation**
   - Check required fields presence
   - Verify field inheritance in context chains
   - Validate field merging behavior

4. **Error Handling**
   - Test error propagation
   - Verify error field capture
   - Check error formatting

## Running Logger Tests

To run specific logger tests:

```bash
# Run all logger tests
go test -v ./internal/database/... -run "TestGormLogger|TestDefaultLogger"

# Run mock logger tests
go test -v ./internal/database/... -run TestMockLogger

# Run test logger tests
go test -v ./testhelper/... -run TestTestLogger
```

## Common Test Patterns

### 1. Field Verification
```go
if fields["error"] != "test error" {
    t.Errorf("Expected error field, got %v", fields["error"])
}
```

### 2. Message Capture
```go
messages := logger.GetInfoMessages()
if len(messages) == 0 {
    t.Error("Expected messages to be logged")
}
```

### 3. Context Propagation
```go
logger := logger.WithContext(ctx)
logger.LogInfo("test", nil)
// Verify context fields are present
```

## Future Improvements

1. Add performance benchmarks
2. Implement log format validation
3. Add concurrent logging tests
4. Enhance error condition coverage
5. Add integration with log aggregation systems
6. Implement log rotation testing
7. Add structured logging validation
8. Enhance context propagation testing

## Troubleshooting

Common issues and solutions:

1. **Missing Log Messages**
   - Check log level configuration
   - Verify logger implementation
   - Check mutex locking

2. **Field Inheritance Issues**
   - Verify field merging logic
   - Check context propagation
   - Validate field maps

3. **Concurrent Access Issues**
   - Ensure proper mutex usage
   - Check thread safety
   - Verify message queue handling 