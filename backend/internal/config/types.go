package config

import (
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/internal/video"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig       `yaml:"server"`
	Database DatabaseConfig     `yaml:"database"`
	Redis    RedisConfig        `yaml:"redis"`
	Storage  StorageConfig      `yaml:"storage"`
	Logging  logger.Config      `yaml:"logging"`
	Ffmpeg   video.FfmpegConfig `yaml:"ffmpeg"`
	Video    VideoConfig        `yaml:"video"`
}

// ServerConfig represents server configuration settings
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// DatabaseConfig represents database configuration settings
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Dbname   string `mapstructure:"dbname"`
	Port     int    `mapstructure:"port"`
	Sslmode  string `mapstructure:"sslmode"`
	Timezone string `mapstructure:"timezone"`
	Pool     struct {
		MaxOpen int `mapstructure:"maxOpen"`
		MaxIdle int `mapstructure:"maxIdle"`
	} `mapstructure:"pool"`
}

// StorageConfig represents storage configuration settings
type StorageConfig struct {
	UploadDir string     `mapstructure:"uploadDir"`
	TempDir   string     `mapstructure:"tempDir"`
	IPFS      IPFSConfig `mapstructure:"ipfs"`
	S3        S3Config   `mapstructure:"s3"`
}

// RedisConfig represents Redis configuration settings
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// VideoConfig represents video configuration settings
type VideoConfig struct {
	MaxSize        int64    `mapstructure:"maxSize"`
	MinTitleLength int      `mapstructure:"minTitleLength"`
	MaxTitleLength int      `mapstructure:"maxTitleLength"`
	MaxDescLength  int      `mapstructure:"maxDescLength"`
	AllowedFormats []string `mapstructure:"allowedFormats"`
}

// IPFSConfig represents IPFS configuration settings
type IPFSConfig struct {
	APIAddress string `mapstructure:"apiAddress"`
	Gateway    string `mapstructure:"gateway"`
}

// S3Config represents S3 configuration settings
type S3Config struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"accessKeyId"`
	SecretAccessKey string `mapstructure:"secretAccessKey"`
	UseSSL          bool   `mapstructure:"useSSL"`
	Region          string `mapstructure:"region"`
	Bucket          string `mapstructure:"bucket"`
}
