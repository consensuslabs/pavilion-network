package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/consensuslabs/pavilion-network/backend/migrations"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func loadConfig() (*struct {
	Database struct {
		Host     string
		User     string
		Password string
		Dbname   string
		Port     int
		Sslmode  string
	}
}, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../../../") // Look for config in the backend directory

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var config struct {
		Database struct {
			Host     string
			User     string
			Password string
			Dbname   string
			Port     int
			Sslmode  string
		}
	}

	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %v", err)
	}

	return &config, nil
}

func main() {
	// Load .env file
	if err := godotenv.Load("../../../.env"); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Parse command line arguments
	direction := flag.String("direction", "up", "Migration direction (up/down)")
	flag.Parse()

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override password from environment if present
	if envPass := os.Getenv("DB_PASSWORD"); envPass != "" {
		config.Database.Password = envPass
	}

	// Construct database connection string
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		config.Database.Host,
		config.Database.User,
		config.Database.Password,
		config.Database.Dbname,
		config.Database.Port,
		config.Database.Sslmode,
	)

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := migrations.RunMigrations(db, *direction); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Printf("Successfully ran migrations in %s direction", *direction)
}
