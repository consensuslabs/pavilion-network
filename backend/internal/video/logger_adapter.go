package video

import "github.com/consensuslabs/pavilion-network/backend/internal/logger"

// LoggerAdapter adapts the internal logger to the video package's Logger interface
type LoggerAdapter struct {
	logger logger.Logger
}

// NewLoggerAdapter creates a new logger adapter
func NewLoggerAdapter(logger logger.Logger) Logger {
	return &LoggerAdapter{logger: logger}
}

// LogInfo implements the video.Logger interface
func (l *LoggerAdapter) LogInfo(message string, fields map[string]interface{}) {
	l.logger.LogInfo(message, fields)
}

// LogError implements the video.Logger interface
func (l *LoggerAdapter) LogError(message string, fields map[string]interface{}) {
	l.logger.LogError(nil, message)
}
