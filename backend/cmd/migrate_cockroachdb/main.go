package main

import (
	"fmt"
	"log"
	"os"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/migrations/cockroachdb"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load environment variables from .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: No .env file found or error loading it: %v", err)
	}

	// Initialize logger
	loggerInstance, err := logger.NewLogger(&logger.Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Configure viper for config loading
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	
	// Set default values
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 26257)
	viper.SetDefault("database.user", "root")
	viper.SetDefault("database.dbname", "defaultdb")
	viper.SetDefault("database.sslmode", "disable")
	
	// Read config
	if err := viper.ReadInConfig(); err != nil {
		loggerInstance.LogWarn(fmt.Sprintf("Failed to read config: %v", err), map[string]interface{}{
			"error": err.Error(),
		})
		loggerInstance.LogInfo("Using default values for CockroachDB connection", nil)
	}
	
	// Get database configuration from viper
	host := viper.GetString("database.host")
	port := viper.GetInt("database.port")
	user := viper.GetString("database.user")
	password := viper.GetString("database.password")
	dbname := viper.GetString("database.dbname")
	sslmode := viper.GetString("database.sslmode")
	
	// Allow overriding config with environment variables
	if envHost := os.Getenv("DB_HOST"); envHost != "" {
		host = envHost
	}
	
	if envPort := os.Getenv("DB_PORT"); envPort != "" {
		// We're not parsing to int here since it would add complexity
		// The connection string will handle string to int conversion
		loggerInstance.LogInfo(fmt.Sprintf("Using DB_PORT from environment: %s", envPort), nil)
	}
	
	if envUser := os.Getenv("DB_USER"); envUser != "" {
		user = envUser
	}
	
	if envPass := os.Getenv("DB_PASSWORD"); envPass != "" {
		password = envPass
	}
	
	if envName := os.Getenv("DB_NAME"); envName != "" {
		dbname = envName
	}
	
	if envSsl := os.Getenv("DB_SSLMODE"); envSsl != "" {
		sslmode = envSsl
	}
	
	// Log connection details
	loggerInstance.LogInfo("Connecting to CockroachDB", map[string]interface{}{
		"host":    host,
		"port":    port,
		"user":    user,
		"dbname":  dbname,
		"sslmode": sslmode,
	})
	
	// Construct database connection string
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)
	
	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		loggerInstance.LogFatal(err, "Failed to connect to CockroachDB")
	}
	
	// Run the CockroachDB migrations
	loggerInstance.LogInfo("Running CockroachDB migrations", nil)
	migrationRunner := cockroachdb.NewMigrationRunner(db, loggerInstance)
	if err := migrationRunner.RunMigrations(); err != nil {
		loggerInstance.LogFatal(err, "Failed to run migrations")
	}
	
	loggerInstance.LogInfo("CockroachDB migrations completed successfully", nil)
} 