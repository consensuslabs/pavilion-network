package config

// Service defines the interface for configuration operations
type Service interface {
	Load(path string) (*Config, error)
}

// ConfigLogger defines the logging interface used by the config package
type ConfigLogger interface {
	LogInfo(msg string, fields map[string]interface{})
	LogError(err error, msg string) error
}
