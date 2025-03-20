package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/migrations/scylladb"
	"github.com/gocql/gocql"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
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
	viper.SetDefault("scylladb.hosts", []string{"localhost"})
	viper.SetDefault("scylladb.port", 9042)
	viper.SetDefault("scylladb.keyspace", "pavilion")
	viper.SetDefault("scylladb.timeout", 30 * time.Second)
	viper.SetDefault("scylladb.connectTimeout", 30 * time.Second)
	viper.SetDefault("scylladb.consistency", "quorum")
	
	// Read config
	if err := viper.ReadInConfig(); err != nil {
		loggerInstance.LogWarn(fmt.Sprintf("Failed to read config: %v", err), map[string]interface{}{
			"error": err.Error(),
		})
		loggerInstance.LogInfo("Using default values for ScyllaDB connection", nil)
	}
	
	// Get ScyllaDB configuration from viper
	hosts := viper.GetStringSlice("scylladb.hosts")
	port := viper.GetInt("scylladb.port")
	keyspace := viper.GetString("scylladb.keyspace")
	timeout := viper.GetDuration("scylladb.timeout")
	connectTimeout := viper.GetDuration("scylladb.connectTimeout")
	consistency := viper.GetString("scylladb.consistency")
	
	// Allow overriding config with environment variables
	if os.Getenv("SCYLLADB_HOSTS") != "" {
		hosts = []string{os.Getenv("SCYLLADB_HOSTS")}
	}
	
	if os.Getenv("SCYLLADB_KEYSPACE") != "" {
		keyspace = os.Getenv("SCYLLADB_KEYSPACE")
	}
	
	// Log connection details
	loggerInstance.LogInfo("Connecting to ScyllaDB", map[string]interface{}{
		"hosts":    hosts,
		"port":     port,
		"keyspace": keyspace,
	})
	
	// Create cluster config
	cluster := gocql.NewCluster(hosts...)
	cluster.Port = port
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.ParseConsistency(consistency)
	cluster.Timeout = timeout
	cluster.ConnectTimeout = connectTimeout
	
	// Connect to ScyllaDB
	session, err := cluster.CreateSession()
	if err != nil {
		loggerInstance.LogFatal(err, "Failed to connect to ScyllaDB")
	}
	defer session.Close()
	
	// Run the notification metadata migration
	loggerInstance.LogInfo("Running notification metadata migration", nil)
	migration := scylladb.NewNotificationMetadataMigration(loggerInstance)
	if err := migration.Execute(session, keyspace); err != nil {
		loggerInstance.LogFatal(err, "Failed to run migration")
	}
	
	loggerInstance.LogInfo("Migration completed successfully", nil)
}