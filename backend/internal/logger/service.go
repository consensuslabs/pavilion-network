package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Service wraps zap logger with our custom methods
type Service struct {
	*zap.Logger
}

// NewService creates a new logger instance
func NewService(config *Config) (*Service, error) {
	var level zapcore.Level
	if config != nil {
		switch config.Level {
		case "debug":
			level = zapcore.DebugLevel
		case "info":
			level = zapcore.InfoLevel
		case "warn":
			level = zapcore.WarnLevel
		case "error":
			level = zapcore.ErrorLevel
		default:
			level = zapcore.InfoLevel
		}
	} else {
		level = zapcore.InfoLevel
	}

	zapConfig := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %v", err)
	}

	return &Service{logger}, nil
}

// LogError logs an error with context and returns an error that can be returned to the client
func (s *Service) LogError(err error, context string) error {
	s.With(zap.Error(err)).Error(context)
	return err
}

// LogErrorf logs a formatted error message with context and returns an error
func (s *Service) LogErrorf(err error, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	s.With(zap.Error(err)).Error(msg)
	return err
}

// LogFatal logs a fatal error and exits the application
func (s *Service) LogFatal(err error, context string) {
	s.With(zap.Error(err)).Fatal(context)
}

// LogInfo logs an informational message with optional fields
func (s *Service) LogInfo(message string, fields map[string]interface{}) {
	if fields != nil {
		s.With(zap.Any("fields", fields)).Info(message)
	} else {
		s.Info(message)
	}
}

// LogDebug logs a debug message with optional fields
func (s *Service) LogDebug(message string, fields map[string]interface{}) {
	if fields != nil {
		s.With(zap.Any("fields", fields)).Debug(message)
	} else {
		s.Debug(message)
	}
}

// LogWarn logs a warning message with optional fields
func (s *Service) LogWarn(message string, fields map[string]interface{}) {
	if fields != nil {
		s.With(zap.Any("fields", fields)).Warn(message)
	} else {
		s.Warn(message)
	}
}
