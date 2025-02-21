package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// mockLogger implements logger.Logger interface for testing
type mockLogger struct {
	infoMessages  []string
	errorMessages []string
	warnMessages  []string
	fields        map[string]interface{}
	contextLogger *mockLogger // Add this field to track the context logger
}

func newMockLogger() *mockLogger {
	return &mockLogger{
		fields: make(map[string]interface{}),
	}
}

func (m *mockLogger) LogInfo(msg string, fields map[string]interface{}) {
	if m.contextLogger != nil {
		m.contextLogger.LogInfo(msg, fields)
		return
	}
	m.infoMessages = append(m.infoMessages, msg)
	// Merge the existing fields with the new fields
	mergedFields := make(map[string]interface{})
	for k, v := range m.fields {
		mergedFields[k] = v
	}
	for k, v := range fields {
		mergedFields[k] = v
	}
	m.fields = mergedFields
}

func (m *mockLogger) LogError(err error, msg string) error {
	if m.contextLogger != nil {
		return m.contextLogger.LogError(err, msg)
	}
	m.errorMessages = append(m.errorMessages, msg)
	return err
}

func (m *mockLogger) LogErrorf(err error, format string, args ...interface{}) error {
	return err
}

func (m *mockLogger) LogFatal(err error, context string) {
	// No-op for testing
}

func (m *mockLogger) LogDebug(message string, fields map[string]interface{}) {
	// No-op for testing
}

func (m *mockLogger) LogWarn(message string, fields map[string]interface{}) {
	if m.contextLogger != nil {
		m.contextLogger.LogWarn(message, fields)
		return
	}
	m.warnMessages = append(m.warnMessages, message)
	// Merge the existing fields with the new fields
	mergedFields := make(map[string]interface{})
	for k, v := range m.fields {
		mergedFields[k] = v
	}
	for k, v := range fields {
		mergedFields[k] = v
	}
	m.fields = mergedFields
}

func (m *mockLogger) WithFields(fields map[string]interface{}) logger.Logger {
	newLogger := newMockLogger()
	newLogger.infoMessages = m.infoMessages
	newLogger.errorMessages = m.errorMessages
	newLogger.warnMessages = m.warnMessages
	for k, v := range m.fields {
		newLogger.fields[k] = v
	}
	if fields != nil {
		for k, v := range fields {
			newLogger.fields[k] = v
		}
	}
	m.contextLogger = newLogger // Track the new logger
	return newLogger
}

func (m *mockLogger) WithContext(ctx context.Context) logger.Logger {
	return m.WithFields(nil)
}

func (m *mockLogger) WithRequestID(requestID string) logger.Logger {
	return m.WithFields(map[string]interface{}{
		"requestID": requestID,
	})
}

func (m *mockLogger) WithUserID(userID string) logger.Logger {
	return m.WithFields(map[string]interface{}{
		"userID": userID,
	})
}

func setupTestRouter(mockLogger *mockLogger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestLoggerMiddleware(mockLogger))
	return router
}

func TestRequestLoggerMiddleware(t *testing.T) {
	t.Run("Basic Request Logging", func(t *testing.T) {
		mockLogger := newMockLogger()
		router := setupTestRouter(mockLogger)

		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		t.Logf("Info messages: %v", mockLogger.contextLogger.infoMessages)
		t.Logf("Context logger fields: %+v", mockLogger.contextLogger.fields)

		if len(mockLogger.contextLogger.infoMessages) == 0 {
			t.Error("Expected request to be logged")
		}

		// Use the context logger's fields for assertions
		fields := mockLogger.contextLogger.fields
		if fields["method"] != "GET" {
			t.Errorf("Expected method GET, got %v", fields["method"])
		}

		if fields["path"] != "/test" {
			t.Errorf("Expected path /test, got %v", fields["path"])
		}

		if fields["status"] != 200 {
			t.Errorf("Expected status 200, got %v", fields["status"])
		}

		if fields["requestID"] == "" {
			t.Error("Expected requestID to be set")
		}
	})

	t.Run("Error Status Code Logging", func(t *testing.T) {
		mockLogger := newMockLogger()
		router := setupTestRouter(mockLogger)

		router.GET("/error", func(c *gin.Context) {
			c.Status(http.StatusInternalServerError)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/error", nil)
		router.ServeHTTP(w, req)

		t.Logf("Error messages: %v", mockLogger.contextLogger.errorMessages)
		t.Logf("Context logger fields: %+v", mockLogger.contextLogger.fields)

		if len(mockLogger.contextLogger.errorMessages) == 0 {
			t.Error("Expected error to be logged for 5xx status")
		}
	})

	t.Run("Warning Status Code Logging", func(t *testing.T) {
		mockLogger := newMockLogger()
		router := setupTestRouter(mockLogger)

		router.GET("/warning", func(c *gin.Context) {
			c.Status(http.StatusBadRequest)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/warning", nil)
		router.ServeHTTP(w, req)

		t.Logf("Warning messages: %v", mockLogger.contextLogger.warnMessages)
		t.Logf("Context logger fields: %+v", mockLogger.contextLogger.fields)

		if len(mockLogger.contextLogger.warnMessages) == 0 {
			t.Error("Expected warning to be logged for 4xx status")
		}
	})

	t.Run("Context Logger Injection", func(t *testing.T) {
		mockLogger := newMockLogger()
		router := setupTestRouter(mockLogger)

		router.GET("/context", func(c *gin.Context) {
			logger := GetLogger(c)
			if logger == nil {
				t.Error("Expected logger to be available in context")
			}
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/context", nil)
		router.ServeHTTP(w, req)
	})

	t.Run("User ID Logging", func(t *testing.T) {
		mockLogger := newMockLogger()
		router := setupTestRouter(mockLogger)

		userID := uuid.New()
		router.GET("/user", func(c *gin.Context) {
			c.Set("userID", userID)
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/user", nil)
		router.ServeHTTP(w, req)

		t.Logf("Context logger fields: %+v", mockLogger.contextLogger.fields)

		fields := mockLogger.contextLogger.fields
		if fields["userID"] != userID {
			t.Errorf("Expected userID %v to be logged, got %v", userID, fields["userID"])
		}
	})

	t.Run("Latency Tracking", func(t *testing.T) {
		mockLogger := newMockLogger()
		router := setupTestRouter(mockLogger)

		router.GET("/latency", func(c *gin.Context) {
			time.Sleep(10 * time.Millisecond)
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/latency", nil)
		router.ServeHTTP(w, req)

		t.Logf("Context logger fields: %+v", mockLogger.contextLogger.fields)

		fields := mockLogger.contextLogger.fields
		latency, ok := fields["latency"].(time.Duration)
		if !ok {
			t.Error("Expected latency to be logged as duration")
		}

		if latency < 10*time.Millisecond {
			t.Error("Expected latency to be at least 10ms")
		}
	})
}
