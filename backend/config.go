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
	UploadDir string `mapstructure:"uploadDir"`
}

// P2PConfig contains P2P networking settings
type P2PConfig struct {
	Port       int    `mapstructure:"port"`
	Rendezvous string `mapstructure:"rendezvous"`
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
