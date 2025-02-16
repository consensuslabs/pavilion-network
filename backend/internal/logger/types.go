package logger

// Config represents logging configuration
type Config struct {
	Level string `mapstructure:"level" yaml:"level"`
}
