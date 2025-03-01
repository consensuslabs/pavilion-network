package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// ConfigService implements the Service interface
type ConfigService struct {
	logger ConfigLogger
}

// NewConfigService creates a new configuration service
func NewConfigService(logger ConfigLogger) *ConfigService {
	return &ConfigService{
		logger: logger,
	}
}

// Load loads the configuration from the specified path
func (s *ConfigService) Load(path string) (*Config, error) {
	viper.AddConfigPath(path)

	// Determine environment and config file
	env := os.Getenv("ENV")
	s.logger.LogInfo("Loading configuration", map[string]interface{}{
		"environment": env,
		"path":        path,
	})

	if env == "test" {
		viper.SetConfigName("config_test")
		// Load test environment variables
		if err := loadEnvFile(path, ".env.test"); err != nil {
			s.logger.LogError(err, "Failed to load test environment variables")
			return nil, fmt.Errorf("failed to load test environment variables: %v", err)
		}
		s.logger.LogInfo("Loaded test configuration file", map[string]interface{}{
			"config_file": "config_test.yaml",
			"env_file":    ".env.test",
		})
	} else {
		viper.SetConfigName("config")
		// Load regular environment variables
		if err := loadEnvFile(path, ".env"); err != nil {
			s.logger.LogError(err, "Failed to load environment variables")
			return nil, fmt.Errorf("failed to load environment variables: %v", err)
		}
		s.logger.LogInfo("Loaded regular configuration file", map[string]interface{}{
			"config_file": "config.yaml",
			"env_file":    ".env",
		})
	}
	viper.SetConfigType("yaml")

	// Set default values
	s.setDefaults()

	// Configure environment variable handling
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AllowEmptyEnv(true)

	// Enable environment variable substitution
	viper.SetEnvPrefix("")

	// Bind environment variables
	viper.BindEnv("environment", "ENV") // Bind ENV to environment field

	// Bind migration control variables
	viper.BindEnv("auto_migrate", "AUTO_MIGRATE")
	viper.BindEnv("force_migration", "FORCE_MIGRATION")

	// Bind logging configuration variables
	viper.BindEnv("logging.level", "LOG_LEVEL")
	viper.BindEnv("logging.format", "LOG_FORMAT")
	viper.BindEnv("logging.output", "LOG_OUTPUT")
	viper.BindEnv("logging.file.enabled", "LOG_FILE_ENABLED")
	viper.BindEnv("logging.file.path", "LOG_FILE_PATH")
	viper.BindEnv("logging.file.rotate", "LOG_FILE_ROTATE")
	viper.BindEnv("logging.file.maxSize", "LOG_FILE_MAX_SIZE")
	viper.BindEnv("logging.file.maxAge", "LOG_FILE_MAX_AGE")
	viper.BindEnv("logging.development", "LOG_ENV_DEVELOPMENT")
	viper.BindEnv("logging.sampling.initial", "LOG_SAMPLING_INITIAL")
	viper.BindEnv("logging.sampling.thereafter", "LOG_SAMPLING_THEREAFTER")

	// Bind database credentials
	viper.BindEnv("database.password", "DB_PASSWORD")

	// Bind Redis credentials
	viper.BindEnv("redis.password", "REDIS_PASSWORD")

	// Bind ScyllaDB password only
	viper.BindEnv("scylladb.password", "SCYLLA_PASSWORD")

	// Bind JWT configuration
	viper.BindEnv("auth.jwt.secret", "JWT_SECRET")

	// Bind S3 configuration variables - using camelCase to match the mapstructure tags
	// Only bind credentials from environment variables, let region and bucket come from config.yaml
	viper.BindEnv("storage.s3.accessKeyId", "S3_ACCESS_KEY_ID")
	viper.BindEnv("storage.s3.secretAccessKey", "S3_SECRET_ACCESS_KEY")

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Debug: Print S3 credentials from Viper after reading config file
	s.logger.LogInfo("S3 credentials from config file", map[string]interface{}{
		"accessKeyId_direct":      viper.GetString("storage.s3.accessKeyId"),
		"secretAccessKey_direct":  viper.GetString("storage.s3.secretAccessKey"),
		"access_key_id_snake":     viper.GetString("storage.s3.access_key_id"),
		"secret_access_key_snake": viper.GetString("storage.s3.secret_access_key"),
		"env_access_key":          os.Getenv("S3_ACCESS_KEY_ID"),
		"env_secret_key":          os.Getenv("S3_SECRET_ACCESS_KEY"),
	})

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// Debug: Print S3 credentials from the unmarshaled config
	s.logger.LogInfo("S3 credentials after unmarshal", map[string]interface{}{
		"accessKeyId":     config.Storage.S3.AccessKeyID,
		"secretAccessKey": config.Storage.S3.SecretAccessKey,
	})

	// Validate the configuration
	if err := s.validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	// Convert relative paths to absolute
	if err := s.resolveStoragePaths(&config, path); err != nil {
		return nil, fmt.Errorf("failed to resolve storage paths: %v", err)
	}

	s.logger.LogInfo("Configuration loaded successfully", map[string]interface{}{
		"environment": env,
		"config_file": viper.ConfigFileUsed(),
	})
	return &config, nil
}

// loadEnvFile loads environment variables from the specified file
func loadEnvFile(path, filename string) error {
	envFile := filepath.Join(path, filename)
	if _, err := os.Stat(envFile); err == nil {
		if err := godotenv.Load(envFile); err != nil {
			return fmt.Errorf("error loading %s: %v", filename, err)
		}
	}
	return nil
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
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.development", false)
	viper.SetDefault("logging.file.enabled", false)
	viper.SetDefault("logging.file.path", "/var/log/pavilion")
	viper.SetDefault("logging.file.rotate", true)
	viper.SetDefault("logging.file.maxSize", "100MB")
	viper.SetDefault("logging.file.maxAge", "30d")
	viper.SetDefault("logging.sampling.initial", 100)
	viper.SetDefault("logging.sampling.thereafter", 100)
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
