package mocks

import (
	"github.com/consensuslabs/pavilion-network/backend/testhelper"
	"github.com/stretchr/testify/mock"
)

// MockLogger implements the video.Logger interface for testing
type MockLogger struct {
	mock.Mock
}

// LogInfo implements the Logger interface
func (m *MockLogger) LogInfo(message string, fields map[string]interface{}) {
	m.Called(message, fields)
}

// LogError implements the Logger interface
func (m *MockLogger) LogError(message string, fields map[string]interface{}) {
	m.Called(message, fields)
}

// TestLoggerAdapter adapts testhelper.TestLogger to implement video.Logger
type TestLoggerAdapter struct {
	Logger *testhelper.TestLogger
}

// LogInfo adapts the testhelper logger's LogInfo method to match our interface
func (a *TestLoggerAdapter) LogInfo(message string, fields map[string]interface{}) {
	// Pass the message and fields to the test logger
	a.Logger.LogInfo(message, fields)
}

// LogError adapts the testhelper logger's LogError method to match our interface
func (a *TestLoggerAdapter) LogError(message string, fields map[string]interface{}) {
	// The testhelper.TestLogger expects an error and a message
	// We'll create a mock error with the message for compatibility
	a.Logger.LogError(nil, message)
}

// NewTestLoggerAdapter creates a new adapter around testhelper.TestLogger
func NewTestLoggerAdapter(logger *testhelper.TestLogger) *TestLoggerAdapter {
	return &TestLoggerAdapter{Logger: logger}
}
