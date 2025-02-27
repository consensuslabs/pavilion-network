package testhelper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/consensuslabs/pavilion-network/backend/internal/auth"
	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/consensuslabs/pavilion-network/backend/migrations"
)

// Config represents the minimal configuration needed for the test environment.
type Config struct {
	Environment string `mapstructure:"environment"`
	Server      struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"server"`
	Database struct {
		Driver   string `mapstructure:"driver"`
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		Name     string `mapstructure:"dbname"`
		SSLMode  string `mapstructure:"sslmode"`
	} `mapstructure:"database"`
	Storage struct {
		UploadDir string `mapstructure:"uploadDir"`
		TempDir   string `mapstructure:"tempDir"`
		IPFS      struct {
			APIAddress string `mapstructure:"apiAddress"`
			Gateway    string `mapstructure:"gateway"`
		} `mapstructure:"ipfs"`
		S3 struct {
			Endpoint        string `mapstructure:"endpoint"`
			Bucket          string `mapstructure:"bucket"`
			Region          string `mapstructure:"region"`
			AccessKeyID     string `mapstructure:"accessKeyId"`
			SecretAccessKey string `mapstructure:"secretAccessKey"`
			UseSSL          bool   `mapstructure:"useSSL"`
			RootDirectory   string `mapstructure:"root_directory"`
		} `mapstructure:"s3"`
	} `mapstructure:"storage"`
	FFmpeg struct {
		Path        string   `mapstructure:"path"`
		ProbePath   string   `mapstructure:"probePath"`
		VideoCodec  string   `mapstructure:"videoCodec"`
		AudioCodec  string   `mapstructure:"audioCodec"`
		Preset      string   `mapstructure:"preset"`
		OutputPath  string   `mapstructure:"outputPath"`
		Resolutions []string `mapstructure:"resolutions"`
	} `mapstructure:"ffmpeg"`
	Video struct {
		MaxSize        int64    `mapstructure:"maxSize"`
		MinTitleLength int      `mapstructure:"minTitleLength"`
		MaxTitleLength int      `mapstructure:"maxTitleLength"`
		MaxDescLength  int      `mapstructure:"maxDescLength"`
		AllowedFormats []string `mapstructure:"allowedFormats"`
	} `mapstructure:"video"`
	Logging struct {
		Level       string `mapstructure:"level"`
		Format      string `mapstructure:"format"`
		Output      string `mapstructure:"output"`
		Development bool   `mapstructure:"development"`
	} `mapstructure:"logging"`
}

// LoadTestConfig loads the test configuration from config_test.yaml.
func LoadTestConfig() (*Config, error) {
	// Load environment variables from .env.test from alternative locations
	envFiles := []string{
		".env.test",
		"../.env.test",
		"../../.env.test",
		"../../../.env.test",
		"../../../../.env.test",
	}

	// Print current working directory for debugging
	cwd, _ := os.Getwd()
	fmt.Println("Current working directory for config loading:", cwd)
	
	envLoaded := false
	for _, envFile := range envFiles {
		absPath, _ := filepath.Abs(envFile)
		if _, err := os.Stat(absPath); err == nil {
			fmt.Printf("Found .env.test at: %s\n", absPath)
		}
		
		if err := godotenv.Load(envFile); err == nil {
			envLoaded = true
			fmt.Printf("Successfully loaded env from: %s\n", envFile)
			break
		}
	}

	if !envLoaded {
		fmt.Println("Warning: .env.test not loaded from any location, proceeding without it")
	}
	
	// Try looking for an env file at the project root level
	rootSearchPaths := []string{
		"/Users/umitdogan/workout/dev/pavilion-network-mvp/pavilion-network/backend/.env.test",
		"/Users/umitdogan/workout/dev/pavilion-network-mvp/pavilion-network/.env.test",
	}
	
	for _, envFile := range rootSearchPaths {
		if _, err := os.Stat(envFile); err == nil {
			fmt.Printf("Found additional .env.test at: %s\n", envFile)
			if err := godotenv.Load(envFile); err == nil {
				fmt.Printf("Successfully loaded env from additional path: %s\n", envFile)
				envLoaded = true
			}
		}
	}

	v := viper.New()
	
	// Set up environment variables mapping
	v.SetEnvPrefix("")  // No prefix for environment variables
	v.AutomaticEnv()
	
	// Map specific environment variables to config keys for S3
	v.BindEnv("storage.s3.accessKeyId", "S3_ACCESS_KEY_ID")
	v.BindEnv("storage.s3.secretAccessKey", "S3_SECRET_ACCESS_KEY")
	
	// Debug output of environment variables
	fmt.Println("Environment variables:")
	fmt.Printf("S3_ACCESS_KEY_ID set: %v\n", os.Getenv("S3_ACCESS_KEY_ID") != "")
	fmt.Printf("S3_SECRET_ACCESS_KEY set: %v\n", os.Getenv("S3_SECRET_ACCESS_KEY") != "")

	// Check if TEST_CONFIG_FILE environment variable is set to explicitly specify config file location.
	if cfgFile := os.Getenv("TEST_CONFIG_FILE"); cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		// Use the intended test config file with test database settings
		v.SetConfigName("config_test")
		v.SetConfigType("yaml")

		// Add paths for different test directory depths
		// For auth tests (2 levels deep)
		v.AddConfigPath("..")
		v.AddConfigPath("../..")

		// For video e2e tests (4 levels deep)
		v.AddConfigPath("../../..")
		v.AddConfigPath("../../../..")

		// Also try the current directory
		v.AddConfigPath(".")

		// Try to find the project root by looking for go.mod
		if wd, err := os.Getwd(); err == nil {
			// Start from current directory and go up to find project root
			dir := wd
			for i := 0; i < 5; i++ { // Look up to 5 levels up
				// Check if this directory contains config_test.yaml
				if _, err := os.Stat(filepath.Join(dir, "config_test.yaml")); err == nil {
					v.AddConfigPath(dir)
					break
				}

				// Check if this is the project root (contains go.mod)
				if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
					v.AddConfigPath(dir) // Found project root
					break
				}

				// Go up one directory
				parentDir := filepath.Dir(dir)
				if parentDir == dir {
					// We've reached the filesystem root
					break
				}
				dir = parentDir
			}
		}
	}

	// Print all config paths for debugging
	fmt.Println("Looking for config_test.yaml in these paths:")
	// Viper doesn't expose a method to get all config paths, so we'll just list the ones we added
	fmt.Println(" - Current directory")
	fmt.Println(" - Parent directory (..)")
	fmt.Println(" - Grandparent directory (../..)")
	fmt.Println(" - Great-grandparent directory (../../..)")
	fmt.Println(" - Great-great-grandparent directory (../../../..)")
	fmt.Println(" - Project root (if found)")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	fmt.Println("Loaded config from:", v.ConfigFileUsed())

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SetupTestDB connects to the test database and runs migrations.
func SetupTestDB(t *testing.T) *gorm.DB {
	cfg, err := LoadTestConfig()
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}

	logger := NewTestLogger(true)

	// Log configuration using structured logging
	logger.LogInfo("Loaded Test Config", map[string]interface{}{
		"environment": cfg.Environment,
		"database": map[string]interface{}{
			"host":    cfg.Database.Host,
			"port":    cfg.Database.Port,
			"user":    cfg.Database.User,
			"dbname":  cfg.Database.Name,
			"sslmode": cfg.Database.SSLMode,
		},
	})

	logger.LogDebug("Full Config", map[string]interface{}{
		"config": cfg,
	})

	if testDB := os.Getenv("TEST_DB"); testDB != "" {
		logger.LogInfo("Overriding database name", map[string]interface{}{
			"test_db": testDB,
		})
		cfg.Database.Name = testDB
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	// Use the DSN to open a database connection.
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// Explicitly switch to the specified test database to avoid connecting to default_db
	if err := db.Exec("USE " + cfg.Database.Name).Error; err != nil {
		t.Fatalf("failed to execute USE query on test database: %v", err)
	}

	// Debug: Verify the current database is correct
	var currentDB string
	if err := db.Raw("SELECT current_database()").Scan(&currentDB).Error; err != nil {
		t.Fatalf("failed to get current database: %v", err)
	}
	logger.LogInfo("Current database", map[string]interface{}{
		"database": currentDB,
	})

	// Run migrations using our migration runner.
	if err := migrations.RunMigrations(db, "up"); err != nil {
		t.Fatalf("failed to run test migrations: %v", err)
	}

	// Auto migrate auth models.
	if err := db.AutoMigrate(&auth.User{}, &auth.RefreshToken{}); err != nil {
		t.Fatalf("failed auto migrating auth models: %v", err)
	}

	// Create upload_status enum type if it doesn't exist
	// This must be done BEFORE auto-migrating video models
	var exists int
	if err := db.Raw("SELECT 1 FROM pg_type WHERE typname = 'upload_status'").Scan(&exists).Error; err != nil {
		t.Fatalf("failed to check if upload_status type exists: %v", err)
	}

	if exists == 0 {
		logger.LogInfo("Creating upload_status enum type", nil)
		if err := db.Exec("CREATE TYPE upload_status AS ENUM ('pending', 'uploading', 'completed', 'failed')").Error; err != nil {
			// Ignore error if type already exists
			if !strings.Contains(err.Error(), "already exists") {
				t.Fatalf("failed to create upload_status enum type: %v", err)
			}
		}
	}

	// Import video package for models
	videoModels := []interface{}{
		&video.Video{},
		&video.VideoUpload{},
		&video.Transcode{},
		&video.TranscodeSegment{},
	}

	// Auto migrate video models
	if err := db.AutoMigrate(videoModels...); err != nil {
		t.Fatalf("failed auto migrating video models: %v", err)
	}

	return db
}
