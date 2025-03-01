package config

import (
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/video"
)

// Config represents the application configuration
type Config struct {
	Environment string             `mapstructure:"environment" yaml:"environment"`
	Server      ServerConfig       `yaml:"server"`
	Database    DatabaseConfig     `yaml:"database"`
	Redis       RedisConfig        `yaml:"redis"`
	Storage     StorageConfig      `yaml:"storage"`
	Logging     LoggingConfig      `yaml:"logging"`
	Ffmpeg      video.FfmpegConfig `yaml:"ffmpeg"`
	Video       VideoConfig        `yaml:"video"`
	Auth        AuthConfig         `yaml:"auth"`
	ScyllaDB    ScyllaDBConfig     `yaml:"scylladb"`
}

// AuthConfig represents authentication configuration settings
type AuthConfig struct {
	JWT struct {
		Secret          string        `mapstructure:"secret"`
		AccessTokenTTL  time.Duration `mapstructure:"accessTokenTTL"`
		RefreshTokenTTL time.Duration `mapstructure:"refreshTokenTTL"`
	} `mapstructure:"jwt"`
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

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level       string `mapstructure:"level" yaml:"level"`
	Format      string `mapstructure:"format" yaml:"format"`
	Output      string `mapstructure:"output" yaml:"output"`
	Development bool   `mapstructure:"development" yaml:"development"`

	File struct {
		Enabled bool   `mapstructure:"enabled" yaml:"enabled"`
		Path    string `mapstructure:"path" yaml:"path"`
		Rotate  bool   `mapstructure:"rotate" yaml:"rotate"`
		MaxSize string `mapstructure:"maxSize" yaml:"maxSize"`
		MaxAge  string `mapstructure:"maxAge" yaml:"maxAge"`
	} `mapstructure:"file" yaml:"file"`

	Sampling struct {
		Initial    int `mapstructure:"initial" yaml:"initial"`
		Thereafter int `mapstructure:"thereafter" yaml:"thereafter"`
	} `mapstructure:"sampling" yaml:"sampling"`
}

// ScyllaDBConfig represents ScyllaDB configuration settings
type ScyllaDBConfig struct {
	Hosts       []string `mapstructure:"hosts" yaml:"hosts"`
	Port        int      `mapstructure:"port" yaml:"port"`
	Keyspace    string   `mapstructure:"keyspace" yaml:"keyspace"`
	Username    string   `mapstructure:"username" yaml:"username"`
	Password    string   `mapstructure:"password" yaml:"password"`
	Consistency string   `mapstructure:"consistency" yaml:"consistency"`
	Replication struct {
		Class             string `mapstructure:"class" yaml:"class"`
		ReplicationFactor int    `mapstructure:"replicationFactor" yaml:"replicationFactor"`
	} `mapstructure:"replication" yaml:"replication"`
	Timeout        time.Duration `mapstructure:"timeout" yaml:"timeout"`
	ConnectTimeout time.Duration `mapstructure:"connectTimeout" yaml:"connectTimeout"`
}
