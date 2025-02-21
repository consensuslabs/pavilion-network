package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestGormLogger(t *testing.T) {
	testLogger := newMockLogger()
	gormLogger := NewGormLogger(testLogger, 200*time.Millisecond)

	t.Run("Info Logging", func(t *testing.T) {
		gormLogger.Info(context.Background(), "test info message")
		messages := testLogger.GetInfoMessages()
		if len(messages) == 0 {
			t.Error("Expected info message to be logged")
		}
		if messages[len(messages)-1].Message != "test info message" {
			t.Errorf("Expected message 'test info message', got '%s'", messages[len(messages)-1].Message)
		}
	})

	t.Run("Warn Logging", func(t *testing.T) {
		gormLogger.Warn(context.Background(), "test warn message")
		messages := testLogger.GetWarnMessages()
		if len(messages) == 0 {
			t.Error("Expected warning message to be logged")
		}
		if messages[len(messages)-1].Message != "test warn message" {
			t.Errorf("Expected message 'test warn message', got '%s'", messages[len(messages)-1].Message)
		}
	})

	t.Run("Error Logging", func(t *testing.T) {
		gormLogger.Error(context.Background(), "test error message")
		messages := testLogger.GetErrorMessages()
		if len(messages) == 0 {
			t.Error("Expected error message to be logged")
		}
		if messages[len(messages)-1].Message != "GORM error" {
			t.Errorf("Expected message 'GORM error', got '%s'", messages[len(messages)-1].Message)
		}
	})

	t.Run("Trace Normal Query", func(t *testing.T) {
		testLogger.ClearMessages()
		ctx := context.Background()
		begin := time.Now()
		fc := func() (string, int64) {
			return "SELECT * FROM users", 10
		}

		gormLogger.Trace(ctx, begin, fc, nil)
		messages := testLogger.GetDebugMessages()
		if len(messages) == 0 {
			t.Error("Expected debug message for normal query")
		}

		lastMsg := messages[len(messages)-1]
		if lastMsg.Fields["sql"] != "SELECT * FROM users" {
			t.Errorf("Expected SQL query in fields, got %v", lastMsg.Fields["sql"])
		}
		if lastMsg.Fields["rows_affected"] != int64(10) {
			t.Errorf("Expected 10 rows affected, got %v", lastMsg.Fields["rows_affected"])
		}
	})

	t.Run("Trace Slow Query", func(t *testing.T) {
		testLogger.ClearMessages()
		ctx := context.Background()
		begin := time.Now().Add(-300 * time.Millisecond) // Make it a slow query
		fc := func() (string, int64) {
			return "SELECT * FROM large_table", 1000
		}

		gormLogger.Trace(ctx, begin, fc, nil)
		messages := testLogger.GetWarnMessages()
		if len(messages) == 0 {
			t.Error("Expected warning message for slow query")
		}

		lastMsg := messages[len(messages)-1]
		if lastMsg.Fields["sql"] != "SELECT * FROM large_table" {
			t.Errorf("Expected SQL query in fields, got %v", lastMsg.Fields["sql"])
		}
		if lastMsg.Fields["rows_affected"] != int64(1000) {
			t.Errorf("Expected 1000 rows affected, got %v", lastMsg.Fields["rows_affected"])
		}
	})

	t.Run("Trace Query Error", func(t *testing.T) {
		testLogger.ClearMessages()
		ctx := context.Background()
		begin := time.Now()
		fc := func() (string, int64) {
			return "SELECT * FROM nonexistent_table", 0
		}
		err := errors.New("table does not exist")

		gormLogger.Trace(ctx, begin, fc, err)
		messages := testLogger.GetErrorMessages()
		if len(messages) == 0 {
			t.Error("Expected error message for failed query")
		}

		lastMsg := messages[len(messages)-1]
		if lastMsg.Fields["sql"] != "SELECT * FROM nonexistent_table" {
			t.Errorf("Expected SQL query in fields, got %v", lastMsg.Fields["sql"])
		}
		if lastMsg.Fields["error"] != "table does not exist" {
			t.Errorf("Expected error message in fields, got %v", lastMsg.Fields["error"])
		}
	})

	t.Run("Trace with Context Values", func(t *testing.T) {
		testLogger.ClearMessages()
		ctx := context.WithValue(context.Background(), "request_id", "test-request-id")
		ctx = context.WithValue(ctx, "trace_id", "test-trace-id")
		begin := time.Now()
		fc := func() (string, int64) {
			return "SELECT * FROM users", 5
		}

		gormLogger.Trace(ctx, begin, fc, nil)
		messages := testLogger.GetDebugMessages()
		if len(messages) == 0 {
			t.Error("Expected debug message with context values")
		}

		lastMsg := messages[len(messages)-1]
		if lastMsg.Fields["request_id"] != "test-request-id" {
			t.Errorf("Expected request_id in fields, got %v", lastMsg.Fields["request_id"])
		}
		if lastMsg.Fields["trace_id"] != "test-trace-id" {
			t.Errorf("Expected trace_id in fields, got %v", lastMsg.Fields["trace_id"])
		}
	})

	t.Run("Skip Record Not Found Error", func(t *testing.T) {
		testLogger.ClearMessages()
		ctx := context.Background()
		begin := time.Now()
		fc := func() (string, int64) {
			return "SELECT * FROM users WHERE id = 1", 0
		}

		gormLogger.Trace(ctx, begin, fc, gorm.ErrRecordNotFound)
		messages := testLogger.GetErrorMessages()
		if len(messages) > 0 {
			t.Error("Expected no error message for record not found")
		}
	})
}
