# Logger Implementation Plan

## Overview

This document outlines the plan to unify and improve logging across the Pavilion Network backend. The goal is to establish consistent logging practices, improve debugging capabilities, and ensure proper log management across different environments.

## Current State

### Implemented Features
- Zap logger implementation in `backend/internal/logger`
- Structured logging support
- Multiple log levels (debug, info, warn, error)
- JSON encoding for log output
- Basic middleware logging for HTTP requests

### Issues Identified
1. Inconsistent logging practices
   - Mix of `fmt.Printf` and logger usage
   - Inconsistent error handling
   - Missing contextual information in logs
2. Direct console logging in production code
3. Lack of standardized log fields
4. No clear separation between development and production logging
5. Missing correlation IDs for request tracking

## Implementation Plan

### Phase 1: Logger Service Enhancement

1. **Extend Logger Interface**
   ```go
   type Logger interface {
       // Existing methods
       LogInfo(msg string, fields map[string]interface{})
       LogError(err error, msg string) error
       LogErrorf(err error, format string, args ...interface{}) error
       LogFatal(err error, context string)
       LogDebug(message string, fields map[string]interface{})
       LogWarn(message string, fields map[string]interface{})
       
       // New methods to add
       WithFields(fields map[string]interface{}) Logger
       WithContext(ctx context.Context) Logger
       WithRequestID(requestID string) Logger
       WithUserID(userID string) Logger
   }
   ```

2. **Standardize Log Fields**
   - Common fields across all logs:
     - `timestamp`
     - `level`
     - `service`
     - `environment`
     - `requestID` (for HTTP requests)
     - `userID` (when available)
     - `component` (module/package name)
     - `action` (operation being performed)

### Phase 2: Code Refactoring

1. **Auth Service Refactoring**
   - Replace all `fmt.Printf` calls in `auth/service.go`
   - Add structured logging for authentication events
   - Implement audit logging for security events

2. **Database Service Enhancement**
   - Add query logging with proper sanitization
   - Log slow queries and performance metrics
   - Structured logging for migrations

3. **Video Service Updates**
   - Replace direct console logging
   - Add structured logging for upload progress
   - Implement proper error logging

4. **Test Helper Updates**
   - Create dedicated test logger
   - Implement proper test output formatting
   - Add debug logging options for tests

### Phase 3: Middleware and Context

1. **Enhanced Request Logging**
   ```go
   func RequestLoggerMiddleware(logger Logger) gin.HandlerFunc {
       return func(c *gin.Context) {
           requestID := uuid.New().String()
           start := time.Now()
           
           // Add logger to context
           ctx := context.WithValue(c.Request.Context(), "logger",
               logger.WithRequestID(requestID))
           c.Request = c.Request.WithContext(ctx)
           
           c.Next()
           
           // Log request completion
           logger.WithFields(map[string]interface{}{
               "method":     c.Request.Method,
               "path":      c.Request.URL.Path,
               "status":    c.Writer.Status(),
               "latency":   time.Since(start),
               "requestID": requestID,
               "clientIP":  c.ClientIP(),
           }).LogInfo("Request completed")
       }
   }
   ```

2. **Context-Aware Logging**
   - Implement context propagation
   - Add trace ID support
   - Enable correlation across services

### Phase 4: Configuration and Environment Support

1. **Enhanced Configuration**
   ```yaml
   logging:
     level: info
     format: json
     output: stdout
     file:
       enabled: true
       path: /var/log/pavilion
       rotate: true
       maxSize: 100MB
       maxAge: 30d
     development: false
     sampling:
       initial: 100
       thereafter: 100
   ```

2. **Environment-Specific Settings**
   - Development: Pretty console output
   - Testing: Minimal logging
   - Production: JSON structured logging

### Phase 5: Monitoring and Alerting Integration

1. **Metrics Collection**
   - Error rate tracking
   - Request latency monitoring
   - Resource usage logging

2. **Alert Integration**
   - Critical error alerting
   - Performance threshold alerts
   - Security event notifications

## Implementation Timeline

1. Phase 1: 1 week
   - Logger interface enhancement
   - Field standardization

2. Phase 2: 2 weeks
   - Service-by-service refactoring
   - Code cleanup and testing

3. Phase 3: 1 week
   - Middleware implementation
   - Context integration

4. Phase 4: 1 week
   - Configuration updates
   - Environment setup

5. Phase 5: 1 week
   - Monitoring setup
   - Alert configuration

Total Timeline: 6 weeks

## Success Criteria

1. No direct `fmt` usage in production code
2. All logs following standardized format
3. Complete context propagation
4. Environment-appropriate logging
5. Proper error tracking and alerting
6. Improved debugging capabilities
7. Comprehensive logging documentation

## Future Enhancements

1. Log aggregation system integration
2. Advanced log analysis tools
3. Custom log viewers for development
4. Performance optimization
5. Advanced sampling strategies
6. Machine learning for log analysis

## Documentation Updates Required

1. Update API documentation with logging examples
2. Create logging style guide
3. Document log levels and their usage
4. Add troubleshooting guide
5. Create log analysis documentation 