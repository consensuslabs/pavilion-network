package testhelper

import (
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/consensuslabs/pavilion-network/backend/internal/auth"
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
}

// LoadTestConfig loads the test configuration from config_test.yaml.
func LoadTestConfig() (*Config, error) {
	// Load environment variables from .env.test from alternative locations
	if err := godotenv.Load("../../.env.test"); err != nil {
		if err2 := godotenv.Load("../.env.test"); err2 != nil {
			fmt.Println("Warning: .env.test not loaded from either ../../.env.test or ../.env.test, proceeding without it")
		}
	}

	v := viper.New()
	v.AutomaticEnv()

	// Check if TEST_CONFIG_FILE environment variable is set to explicitly specify config file location.
	if cfgFile := os.Getenv("TEST_CONFIG_FILE"); cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		// Use the intended test config file with test database settings
		v.SetConfigName("config_test")
		v.SetConfigType("yaml")
		// Since tests run from testhelper folder, use parent directory as config path
		v.AddConfigPath("..")
		v.AddConfigPath("../..")
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

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

	// Insert logging statements to debug configuration.
	fmt.Printf("Loaded Test Config: Environment=%s, Database: host=%s, port=%d, user=%s, dbname=%s, sslmode=%s\n", cfg.Environment, cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Name, cfg.Database.SSLMode)
	fmt.Printf("Full Config: %+v\n", cfg)

	if testDB := os.Getenv("TEST_DB"); testDB != "" {
		fmt.Printf("Overriding database name with TEST_DB: %s\n", testDB)
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
	fmt.Printf("After USE query, current database: %s\n", currentDB)

	// Run migrations using our migration runner.
	if err := migrations.RunMigrations(db, "up"); err != nil {
		t.Fatalf("failed to run test migrations: %v", err)
	}

	// Auto migrate auth models.
	if err := db.AutoMigrate(&auth.User{}, &auth.RefreshToken{}); err != nil {
		t.Fatalf("failed auto migrating auth models: %v", err)
	}

	return db
}
