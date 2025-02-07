package main

import (
	"github.com/sirupsen/logrus"
)

// Logger wraps logrus.Logger to provide consistent logging across the application
type Logger struct {
	*logrus.Logger
}

// NewLogger creates a new logger instance with the specified configuration
func NewLogger(config LoggingConfig) (*Logger, error) {
	logger := &Logger{logrus.New()}

	// Set the log formatter
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Set log level from config
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		return nil, err
	}
	logger.SetLevel(level)

	return logger, nil
}

// LogError logs an error with context and returns an error that can be returned to the client
func (l *Logger) LogError(err error, context string) error {
	l.WithError(err).Error(context)
	return err
}

// LogErrorf logs a formatted error message with context and returns an error
func (l *Logger) LogErrorf(err error, format string, args ...interface{}) error {
	l.WithError(err).Errorf(format, args...)
	return err
}

// LogFatal logs a fatal error and exits the application
func (l *Logger) LogFatal(err error, context string) {
	l.WithError(err).Fatal(context)
}

// LogInfo logs an informational message with optional fields
func (l *Logger) LogInfo(message string, fields map[string]interface{}) {
	if fields != nil {
		l.WithFields(fields).Info(message)
	} else {
		l.Info(message)
	}
}

// LogDebug logs a debug message with optional fields
func (l *Logger) LogDebug(message string, fields map[string]interface{}) {
	if fields != nil {
		l.WithFields(fields).Debug(message)
	} else {
		l.Debug(message)
	}
}

// LogWarn logs a warning message with optional fields
func (l *Logger) LogWarn(message string, fields map[string]interface{}) {
	if fields != nil {
		l.WithFields(fields).Warn(message)
	} else {
		l.Warn(message)
	}
}
