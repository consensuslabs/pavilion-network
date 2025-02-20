package logger

// Level represents the logging level
type Level string

const (
	// Log levels
	DebugLevel Level = "debug"
	InfoLevel  Level = "info"
	WarnLevel  Level = "warn"
	ErrorLevel Level = "error"
	FatalLevel Level = "fatal"
)

// Config holds the logger configuration
type Config struct {
	// Log level configuration
	Level Level `mapstructure:"level" yaml:"level"`

	// Output format (json or console)
	Format string `mapstructure:"format" yaml:"format"`

	// Output destination (stdout or file path)
	Output string `mapstructure:"output" yaml:"output"`

	// Development mode flag
	Development bool `mapstructure:"development" yaml:"development"`

	// File output configuration
	File struct {
		Enabled bool   `mapstructure:"enabled" yaml:"enabled"`
		Path    string `mapstructure:"path" yaml:"path"`
		Rotate  bool   `mapstructure:"rotate" yaml:"rotate"`
		MaxSize string `mapstructure:"maxSize" yaml:"maxSize"`
		MaxAge  string `mapstructure:"maxAge" yaml:"maxAge"`
	} `mapstructure:"file" yaml:"file"`

	// Sampling configuration
	Sampling struct {
		Initial    int `mapstructure:"initial" yaml:"initial"`
		Thereafter int `mapstructure:"thereafter" yaml:"thereafter"`
	} `mapstructure:"sampling" yaml:"sampling"`
}

// StandardFields represents the standard fields that should be included in all logs
type StandardFields struct {
	Timestamp   string                 `json:"timestamp"`
	Level       Level                  `json:"level"`
	Service     string                 `json:"service"`
	Environment string                 `json:"environment"`
	RequestID   string                 `json:"requestID,omitempty"`
	UserID      string                 `json:"userID,omitempty"`
	Component   string                 `json:"component"`
	Action      string                 `json:"action"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}
