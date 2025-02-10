package main

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all configuration values from config.yaml.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	IPFS     IPFSConfig     `mapstructure:"ipfs"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Ffmpeg   FfmpegConfig   `mapstructure:"ffmpeg"`
	Storage  StorageConfig  `mapstructure:"storage"`
	P2P      P2PConfig      `mapstructure:"p2p"`
	Video    VideoConfig    `mapstructure:"video"`
}

// ServerConfig contains server specific configurations.
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// DatabaseConfig contains database connection settings.
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Dbname   string `mapstructure:"dbname"`
	Port     int    `mapstructure:"port"`
	Sslmode  string `mapstructure:"sslmode"`
	Timezone string `mapstructure:"timezone"`
}

// RedisConfig contains Redis connection settings.
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// IPFSConfig contains the configuration for IPFS access.
type IPFSConfig struct {
	Host       string `mapstructure:"host"`
	GatewayURL string `mapstructure:"gateway_url"`
}

// LoggingConfig contains logging level settings.
type LoggingConfig struct {
	Level string `mapstructure:"level"`
}

// FfmpegConfig contains ffmpeg configuration settings.
type FfmpegConfig struct {
	Path            string `mapstructure:"path"`
	VideoCodec      string `mapstructure:"videoCodec"`
	AudioCodec      string `mapstructure:"audioCodec"`
	Preset          string `mapstructure:"preset"`
	HLSTime         int    `mapstructure:"hlsTime"`
	HLSPlaylistType string `mapstructure:"hlsPlaylistType"`
}

// StorageConfig contains storage configuration settings.
type StorageConfig struct {
	UploadDir string     `mapstructure:"uploadDir"`
	IPFS      IPFSConfig `mapstructure:"ipfs"`
	S3        S3Config   `mapstructure:"s3"`
}

// P2PConfig contains P2P networking settings
type P2PConfig struct {
	Port       int    `mapstructure:"port"`
	Rendezvous string `mapstructure:"rendezvous"`
}

// VideoConfig contains video upload and processing settings
type VideoConfig struct {
	MaxSize        int64    `mapstructure:"maxSize"`        // Maximum file size in bytes
	MinTitleLength int      `mapstructure:"minTitleLength"` // Minimum title length
	MaxTitleLength int      `mapstructure:"maxTitleLength"` // Maximum title length
	MaxDescLength  int      `mapstructure:"maxDescLength"`  // Maximum description length
	AllowedFormats []string `mapstructure:"allowedFormats"` // List of allowed video formats
}

// S3Config contains S3-related configuration settings
type S3Config struct {
	Bucket          string            `mapstructure:"bucket"`
	Region          string            `mapstructure:"region"`
	AccessKeyId     string            `mapstructure:"accessKeyId"`
	SecretAccessKey string            `mapstructure:"secretAccessKey"`
	Directories     S3DirectoryConfig `mapstructure:"directories"`
}

// S3DirectoryConfig contains S3 directory paths
type S3DirectoryConfig struct {
	VideoPost        string `mapstructure:"videoPost"`
	MeetingRecording string `mapstructure:"meetingRecording"`
	ChatAttachments  string `mapstructure:"chatAttachments"`
	ProfilePhoto     string `mapstructure:"profilePhoto"`
}

// loadConfig loads configuration from config.yaml using Viper.
func loadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %v", err)
	}

	return config, nil
}

// LoadConfig loads the configuration from file and environment variables
func LoadConfig(path string) (*Config, error) {
	// Create a new Viper instance to avoid conflicts
	v := viper.New()

	// Load .env file first
	v.SetConfigFile(".env")
	err := v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("error reading .env file: %v", err)
	}

	// Get values from .env file
	accessKeyID := v.GetString("S3_ACCESS_KEY_ID")
	secretAccessKey := v.GetString("S3_SECRET_ACCESS_KEY")
	region := v.GetString("S3_REGION")
	bucket := v.GetString("S3_BUCKET_NAME")

	// Create a new Viper instance for yaml config
	v = viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	// Set S3 values from .env
	v.Set("storage.s3.accessKeyId", accessKeyID)
	v.Set("storage.s3.secretAccessKey", secretAccessKey)
	v.Set("storage.s3.region", region)
	v.Set("storage.s3.bucket", bucket)

	// Debug: Print configuration values
	fmt.Printf("Configuration Values:\n")
	fmt.Printf("S3_ACCESS_KEY_ID: %s\n", v.GetString("storage.s3.accessKeyId"))
	fmt.Printf("S3_REGION: %s\n", v.GetString("storage.s3.region"))
	fmt.Printf("S3_BUCKET_NAME: %s\n", v.GetString("storage.s3.bucket"))
	fmt.Printf("Has S3_SECRET_ACCESS_KEY: %v\n", v.IsSet("storage.s3.secretAccessKey"))

	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %v", err)
	}

	// Verify S3 credentials are present
	if config.Storage.S3.AccessKeyId == "" || config.Storage.S3.SecretAccessKey == "" {
		return nil, fmt.Errorf("S3 credentials are missing. Please check your environment variables")
	}

	return config, nil
}
