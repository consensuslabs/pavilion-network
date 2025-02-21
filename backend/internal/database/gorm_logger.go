package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// GormLogger implements GORM's logger.Interface and integrates with our logging system
type GormLogger struct {
	logger     Logger
	slowQuery  time.Duration
	skipErrors []error
}

// NewGormLogger creates a new GORM logger instance
func NewGormLogger(logger Logger, slowQuery time.Duration) gormlogger.Interface {
	return &GormLogger{
		logger:    logger,
		slowQuery: slowQuery,
		skipErrors: []error{
			gorm.ErrRecordNotFound,
		},
	}
}

// LogMode implements GORM's logger.Interface
func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return l
}

// Info implements GORM's logger.Interface
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.logger.LogInfo(fmt.Sprintf(msg, data...), map[string]interface{}{
		"source": "gorm",
	})
}

// Warn implements GORM's logger.Interface
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.logger.LogWarn(fmt.Sprintf(msg, data...), map[string]interface{}{
		"source": "gorm",
	})
}

// Error implements GORM's logger.Interface
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.logger.LogError(fmt.Errorf(msg, data...), "GORM error")
}

// Trace implements GORM's logger.Interface
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	// Create base fields
	fields := map[string]interface{}{
		"source":        "gorm",
		"duration":      elapsed.String(),
		"rows_affected": rows,
		"sql":           sql,
	}

	// Extract additional context from the query
	if ctx != nil {
		if traceID, ok := ctx.Value("trace_id").(string); ok {
			fields["trace_id"] = traceID
		}
		if requestID, ok := ctx.Value("request_id").(string); ok {
			fields["request_id"] = requestID
		}
	}

	// Log slow queries
	if elapsed > l.slowQuery {
		l.logger.LogWarn("SLOW SQL >= "+l.slowQuery.String(), fields)
		return
	}

	// Log errors
	if err != nil {
		// Skip certain errors
		for _, skipErr := range l.skipErrors {
			if err == skipErr {
				return
			}
		}

		fields["error"] = err.Error()
		l.logger.LogError(err, "SQL error", fields)
		return
	}

	// Log successful queries at debug level
	l.logger.LogDebug("SQL query", fields)
}
