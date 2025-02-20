package logger

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLoggerService struct {
	logger *zap.Logger
	fields map[string]interface{}
}

// NewLogger creates a new Logger instance
func NewLogger(config *Config) (Logger, error) {
	var zapConfig zap.Config

	if config.Development {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	// Configure log level
	level, err := zapcore.ParseLevel(string(config.Level))
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %v", err)
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Configure encoding
	zapConfig.Encoding = config.Format

	// Configure output paths
	if config.File.Enabled {
		zapConfig.OutputPaths = []string{config.File.Path}
	} else {
		zapConfig.OutputPaths = []string{config.Output}
	}

	// Create the logger
	zapLogger, err := zapConfig.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %v", err)
	}

	return &zapLoggerService{
		logger: zapLogger,
		fields: make(map[string]interface{}),
	}, nil
}

func (l *zapLoggerService) LogInfo(msg string, fields map[string]interface{}) {
	l.logger.Info(msg, l.convertFields(fields)...)
}

func (l *zapLoggerService) LogError(err error, msg string) error {
	if err != nil {
		l.logger.Error(msg, zap.Error(err))
	}
	return err
}

func (l *zapLoggerService) LogErrorf(err error, format string, args ...interface{}) error {
	if err != nil {
		msg := fmt.Sprintf(format, args...)
		l.logger.Error(msg, zap.Error(err))
	}
	return err
}

func (l *zapLoggerService) LogFatal(err error, context string) {
	l.logger.Fatal(context, zap.Error(err))
}

func (l *zapLoggerService) LogDebug(message string, fields map[string]interface{}) {
	l.logger.Debug(message, l.convertFields(fields)...)
}

func (l *zapLoggerService) LogWarn(message string, fields map[string]interface{}) {
	l.logger.Warn(message, l.convertFields(fields)...)
}

func (l *zapLoggerService) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{}, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	return &zapLoggerService{
		logger: l.logger,
		fields: newFields,
	}
}

func (l *zapLoggerService) WithContext(ctx context.Context) Logger {
	return l.WithFields(map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (l *zapLoggerService) WithRequestID(requestID string) Logger {
	return l.WithFields(map[string]interface{}{
		"requestID": requestID,
	})
}

func (l *zapLoggerService) WithUserID(userID string) Logger {
	return l.WithFields(map[string]interface{}{
		"userID": userID,
	})
}

func (l *zapLoggerService) convertFields(fields map[string]interface{}) []zap.Field {
	zapFields := make([]zap.Field, 0, len(l.fields)+len(fields))

	// Add base fields
	for k, v := range l.fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	// Add additional fields
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	return zapFields
}
