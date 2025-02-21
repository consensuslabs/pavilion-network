package testhelper

import (
	"context"
	"errors"
	"testing"
)

func TestTestLogger(t *testing.T) {
	t.Run("Basic Logging", func(t *testing.T) {
		logger := NewTestLogger(true)

		logger.LogInfo("test info", map[string]interface{}{"key": "value"})
		logger.LogError(errors.New("test error"), "error message")
		logger.LogWarn("test warning", nil)
		logger.LogDebug("test debug", nil)

		if len(logger.GetInfoMessages()) != 1 {
			t.Error("Expected 1 info message")
		}
		if len(logger.GetErrorMessages()) != 1 {
			t.Error("Expected 1 error message")
		}
		if len(logger.GetWarnMessages()) != 1 {
			t.Error("Expected 1 warning message")
		}
		if len(logger.GetDebugMessages()) != 1 {
			t.Error("Expected 1 debug message")
		}
	})

	t.Run("Debug Enable/Disable", func(t *testing.T) {
		logger := NewTestLogger(false)

		logger.LogDebug("test debug 1", nil)
		if len(logger.GetDebugMessages()) != 0 {
			t.Error("Expected no debug messages when debug is disabled")
		}

		logger.EnableDebug()
		logger.LogDebug("test debug 2", nil)
		if len(logger.GetDebugMessages()) != 1 {
			t.Error("Expected 1 debug message after enabling debug")
		}

		logger.DisableDebug()
		logger.LogDebug("test debug 3", nil)
		if len(logger.GetDebugMessages()) != 1 {
			t.Error("Expected no additional debug messages after disabling debug")
		}
	})

	t.Run("Field Merging", func(t *testing.T) {
		logger := NewTestLogger(true)
		withFields := logger.WithFields(map[string]interface{}{
			"base": "value",
		}).(*TestLogger)

		withFields.LogInfo("test", map[string]interface{}{
			"additional": "value",
		})

		messages := withFields.GetInfoMessages()
		if len(messages) != 1 {
			t.Fatal("Expected 1 message")
		}

		fields := messages[0].Fields
		if fields["base"] != "value" || fields["additional"] != "value" {
			t.Error("Expected both base and additional fields to be present")
		}
	})

	t.Run("Context Propagation", func(t *testing.T) {
		logger := NewTestLogger(true)
		ctx := context.Background()

		contextLogger := logger.WithContext(ctx)
		if contextLogger == nil {
			t.Error("Expected non-nil context logger")
		}

		requestLogger := logger.WithRequestID("123").(*TestLogger)
		requestLogger.LogInfo("test", nil)

		messages := requestLogger.GetInfoMessages()
		if len(messages) != 1 {
			t.Fatal("Expected 1 message")
		}

		if messages[0].Fields["requestID"] != "123" {
			t.Error("Expected requestID to be present in fields")
		}
	})

	t.Run("Message Clearing", func(t *testing.T) {
		logger := NewTestLogger(true)

		logger.LogInfo("test info", nil)
		logger.LogError(errors.New("test error"), "error message")
		logger.LogWarn("test warning", nil)
		logger.LogDebug("test debug", nil)

		logger.ClearMessages()

		if len(logger.GetInfoMessages()) != 0 {
			t.Error("Expected no info messages after clearing")
		}
		if len(logger.GetErrorMessages()) != 0 {
			t.Error("Expected no error messages after clearing")
		}
		if len(logger.GetWarnMessages()) != 0 {
			t.Error("Expected no warning messages after clearing")
		}
		if len(logger.GetDebugMessages()) != 0 {
			t.Error("Expected no debug messages after clearing")
		}
	})

	t.Run("Fatal Logging", func(t *testing.T) {
		logger := NewTestLogger(true)

		logger.LogFatal(errors.New("fatal error"), "fatal context")

		messages := logger.GetErrorMessages()
		if len(messages) != 1 {
			t.Fatal("Expected 1 fatal message")
		}

		if messages[0].Fields["error"] != "fatal error" {
			t.Error("Expected error field in fatal message")
		}
		if messages[0].Fields["context"] != "fatal context" {
			t.Error("Expected context field in fatal message")
		}
	})

	t.Run("Error Formatting", func(t *testing.T) {
		logger := NewTestLogger(true)

		err := errors.New("formatted error")
		logger.LogErrorf(err, "error with %s", "formatting")

		messages := logger.GetErrorMessages()
		if len(messages) != 1 {
			t.Fatal("Expected 1 error message")
		}

		if messages[0].Message != "error with formatting" {
			t.Error("Expected formatted error message")
		}
	})
}
