package logger

// Logger defines the interface for logging operations
type Logger interface {
	LogInfo(msg string, fields map[string]interface{})
	LogError(err error, msg string) error
	LogErrorf(err error, format string, args ...interface{}) error
	LogFatal(err error, context string)
	LogDebug(message string, fields map[string]interface{})
	LogWarn(message string, fields map[string]interface{})
}
