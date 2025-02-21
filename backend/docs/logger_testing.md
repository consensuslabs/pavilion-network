# Logger Testing Documentation

## Overview

This document outlines the testing strategy and implementation details for the logging system in the Pavilion Network backend. The logging tests cover request logging, error handling, context propagation, and field management.

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
     level: debug
     format: json
     output: stdout
     development: true
     sampling:
       initial: 100
       thereafter: 100
   ```

## Test Cases

### 1. Request Logger Middleware (`TestRequestLoggerMiddleware`)

Tests the HTTP request logging middleware functionality:

```go
func TestRequestLoggerMiddleware(t *testing.T) {
    // Basic request logging
    t.Run("Basic Request Logging", func(t *testing.T) {...})
    
    // Error status code logging
    t.Run("Error Status Code Logging", func(t *testing.T) {...})
    
    // Warning status code logging
    t.Run("Warning Status Code Logging", func(t *testing.T) {...})
    
    // Context logger injection
    t.Run("Context Logger Injection", func(t *testing.T) {...})
    
    // User ID logging
    t.Run("User ID Logging", func(t *testing.T) {...})
    
    // Latency tracking
    t.Run("Latency Tracking", func(t *testing.T) {...})
}
```

Key test scenarios:
- HTTP request field capture (method, path, status)
- Request ID generation and propagation
- Response status code based logging levels
- Request timing and latency tracking
- User ID context propagation
- Client IP and User Agent logging

### 2. Logger Context Management

Tests the logger's context handling capabilities:
- Request ID propagation
- User ID propagation
- Field inheritance
- Context chaining

### 3. Log Level Management

Tests different logging levels:
- Info level for normal requests
- Warning level for 4xx errors
- Error level for 5xx errors
- Debug level for development
- Fatal level handling

## Test Utilities

### Mock Logger

The test suite includes a mock logger implementation:

```go
type mockLogger struct {
    infoMessages  []string
    errorMessages []string
    warnMessages  []string
    fields        map[string]interface{}
    contextLogger *mockLogger
}

func newMockLogger() *mockLogger
func (m *mockLogger) LogInfo(msg string, fields map[string]interface{})
func (m *mockLogger) LogError(err error, msg string) error
func (m *mockLogger) LogWarn(message string, fields map[string]interface{})
```

### Test Router Setup

Utility for setting up test HTTP routers:

```go
func setupTestRouter(mockLogger *mockLogger) *gin.Engine
```

## Best Practices

1. **Log Message Verification**
   - Verify correct message content
   - Check log level appropriateness
   - Validate structured field data

2. **Context Management**
   - Test context propagation
   - Verify field inheritance
   - Check context chain integrity

3. **Field Handling**
   - Validate required fields presence
   - Check field value accuracy
   - Test field merging behavior

4. **Performance Considerations**
   - Verify latency tracking
   - Check sampling behavior
   - Test high-volume logging

## Common Test Scenarios

### 1. HTTP Request Logging

```go
// Test successful request logging
t.Run("Successful Request", func(t *testing.T) {...})

// Test error request logging
t.Run("Error Request", func(t *testing.T) {...})

// Test request timing
t.Run("Request Timing", func(t *testing.T) {...})
```

### 2. Context Propagation

```go
// Test request ID propagation
t.Run("Request ID", func(t *testing.T) {...})

// Test user ID propagation
t.Run("User ID", func(t *testing.T) {...})
```

### 3. Field Management

```go
// Test field inheritance
t.Run("Field Inheritance", func(t *testing.T) {...})

// Test field merging
t.Run("Field Merging", func(t *testing.T) {...})
```

## Running Logger Tests

To run only logger tests:

```bash
go test -v ./internal/http/middleware/...
```

To run a specific test:

```bash
go test -v ./internal/http/middleware/... -run TestRequestLoggerMiddleware
```

## Troubleshooting

Common issues and solutions:

1. **Message Propagation Issues**
   - Check context logger setup
   - Verify message passing in mock
   - Ensure proper logger chaining

2. **Field Inheritance Problems**
   - Verify field copying logic
   - Check context logger fields
   - Validate field merging

3. **Context Management Issues**
   - Check context logger creation
   - Verify context propagation
   - Ensure proper cleanup

## Future Improvements

Planned enhancements to logger testing:

1. Add performance benchmarks for logging operations
2. Implement stress testing for high-volume logging
3. Add integration tests with log aggregation systems
4. Enhance field validation testing
5. Add concurrent logging tests
6. Implement log format validation
7. Add tests for log rotation and file output
8. Enhance error condition coverage

## Test Coverage

Current test coverage includes:
- HTTP request logging
- Error and warning logging
- Context propagation
- Field management
- Request timing
- User identification

Areas for additional coverage:
- Log rotation
- File output
- Sampling behavior
- Concurrent logging
- Log format validation
- Error recovery
- Resource cleanup 