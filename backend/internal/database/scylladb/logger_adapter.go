package scylladb

import (
	"github.com/consensuslabs/pavilion-network/backend/internal/video"
)

// LoggerAdapter adapts any logger to the ScyllaDB package requirements
type LoggerAdapter struct {
	logger video.Logger
}

// NewLoggerAdapter creates a new logger adapter
func NewLoggerAdapter(logger video.Logger) *LoggerAdapter {
	return &LoggerAdapter{
		logger: logger,
	}
}

// LogInfo logs informational messages
func (l *LoggerAdapter) LogInfo(message string, fields map[string]interface{}) {
	// Add a prefix to clearly identify ScyllaDB logs
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["component"] = "scylladb"

	l.logger.LogInfo(message, fields)
}

// LogError logs error messages
func (l *LoggerAdapter) LogError(message string, fields map[string]interface{}) {
	// Add a prefix to clearly identify ScyllaDB logs
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["component"] = "scylladb"

	l.logger.LogError(message, fields)
}
