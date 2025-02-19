package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// ConfigService implements the Service interface
type ConfigService struct {
	logger Logger
}

// NewConfigService creates a new configuration service
func NewConfigService(logger Logger) *ConfigService {
	return &ConfigService{
		logger: logger,
	}
}

// Load loads the configuration from the specified path
func (s *ConfigService) Load(path string) (*Config, error) {
	viper.AddConfigPath(path)
	// Use test configuration file if ENV is set to test
	if os.Getenv("ENV") == "test" {
		viper.SetConfigName("config_test")
	} else {
		viper.SetConfigName("config")
	}
	viper.SetConfigType("yaml")

	// Set default values
	s.setDefaults()

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// Validate the configuration
	if err := s.validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	// Convert relative paths to absolute
	if err := s.resolveStoragePaths(&config, path); err != nil {
		return nil, fmt.Errorf("failed to resolve storage paths: %v", err)
	}

	s.logger.LogInfo("Configuration loaded successfully", nil)
	return &config, nil
}

// setDefaults sets default values for configuration
func (s *ConfigService) setDefaults() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.timezone", "UTC")
	viper.SetDefault("database.pool.maxOpen", 100)
	viper.SetDefault("database.pool.maxIdle", 10)
	viper.SetDefault("storage.uploadDir", "uploads")
	viper.SetDefault("storage.tempDir", "temp")
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("p2p.port", 6000)
	viper.SetDefault("p2p.rendezvous", "/pavilion")
	viper.SetDefault("video.maxSize", 1024*1024*1024) // 1GB
	viper.SetDefault("video.minTitleLength", 3)
	viper.SetDefault("video.maxTitleLength", 100)
	viper.SetDefault("video.maxDescLength", 5000)
	viper.SetDefault("video.allowedFormats", []string{".mp4", ".mov", ".avi"})
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("storage.ipfs.apiAddress", "/ip4/127.0.0.1/tcp/5001")
	viper.SetDefault("storage.ipfs.gateway", "http://localhost:8080")
}

// validate performs validation on the configuration
func (s *ConfigService) validate(config *Config) error {
	if config.Server.Port <= 0 {
		return fmt.Errorf("invalid server port")
	}

	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if config.Database.User == "" {
		return fmt.Errorf("database user is required")
	}

	if config.Database.Dbname == "" {
		return fmt.Errorf("database name is required")
	}

	if config.Database.Port <= 0 {
		return fmt.Errorf("invalid database port")
	}

	return nil
}

// resolveStoragePaths converts relative paths to absolute paths
func (s *ConfigService) resolveStoragePaths(config *Config, basePath string) error {
	uploadDir := config.Storage.UploadDir
	if !filepath.IsAbs(uploadDir) {
		absPath, err := filepath.Abs(filepath.Join(basePath, uploadDir))
		if err != nil {
			return fmt.Errorf("failed to resolve upload directory path: %v", err)
		}
		config.Storage.UploadDir = absPath
	}

	tempDir := config.Storage.TempDir
	if !filepath.IsAbs(tempDir) {
		absPath, err := filepath.Abs(filepath.Join(basePath, tempDir))
		if err != nil {
			return fmt.Errorf("failed to resolve temp directory path: %v", err)
		}
		config.Storage.TempDir = absPath
	}

	return nil
}
