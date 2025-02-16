package config

// Service defines the interface for configuration operations
type Service interface {
	Load(path string) (*Config, error)
}

// Logger interface for logging operations
type Logger interface {
	LogInfo(msg string, fields map[string]interface{})
	LogError(err error, msg string) error
}
