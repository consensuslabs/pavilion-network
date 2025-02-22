package config

import (
	"os"
	"testing"
)

// mockLogger provides a simple logger implementation for testing
type mockLogger struct {
	infoMessages  []string
	errorMessages []string
}

func newMockLogger() *mockLogger {
	return &mockLogger{}
}

func (m *mockLogger) LogInfo(msg string, fields map[string]interface{}) {
	m.infoMessages = append(m.infoMessages, msg)
}

func (m *mockLogger) LogError(err error, msg string) error {
	m.errorMessages = append(m.errorMessages, msg)
	return err
}

func TestLoadConfig(t *testing.T) {
	// Create a test logger
	logger := newMockLogger()

	// Create config service
	configService := NewConfigService(logger)

	tests := []struct {
		name       string
		env        string
		wantEnv    string
		wantDBName string
	}{
		{
			name:       "Test Environment",
			env:        "test",
			wantEnv:    "test",
			wantDBName: "pavilion_test",
		},
		{
			name:       "Development Environment",
			env:        "development",
			wantEnv:    "development",
			wantDBName: "pavilion_db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment
			os.Setenv("ENV", tt.env)
			defer os.Unsetenv("ENV")

			// Load configuration
			cfg, err := configService.Load("../..")
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// Check environment
			if cfg.Environment != tt.wantEnv {
				t.Errorf("Expected environment %s, got %s", tt.wantEnv, cfg.Environment)
			}

			// Check database name
			if cfg.Database.Dbname != tt.wantDBName {
				t.Errorf("Expected database name %s, got %s", tt.wantDBName, cfg.Database.Dbname)
			}

			// Verify that we got some info messages
			if len(logger.infoMessages) == 0 {
				t.Error("Expected some info messages to be logged")
			}
		})
	}
}
