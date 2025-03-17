package scylladb

import (
	"time"
)

// Config holds the configuration for ScyllaDB
type Config struct {
	Hosts          []string      `mapstructure:"hosts"`
	Port           int           `mapstructure:"port"`
	Keyspace       string        `mapstructure:"keyspace"`
	Username       string        `mapstructure:"username"`
	Password       string        `mapstructure:"password"`
	Consistency    string        `mapstructure:"consistency"`
	Replication    Replication   `mapstructure:"replication"`
	Timeout        time.Duration `mapstructure:"timeout"`
	ConnectTimeout time.Duration `mapstructure:"connectTimeout"`
	
	// Connection pool settings
	Pool PoolConfig `mapstructure:"pool" yaml:"pool"`
	
	// Retry settings
	Retry RetryConfig `mapstructure:"retry" yaml:"retry"`
}

// Replication config for ScyllaDB
type Replication struct {
	Class             string `mapstructure:"class"`
	ReplicationFactor int    `mapstructure:"replicationFactor"`
}

// PoolConfig represents the connection pool configuration
type PoolConfig struct {
	MaxConnections     int `mapstructure:"maxConnections" yaml:"maxConnections"`
	MaxIdleConnections int `mapstructure:"maxIdleConnections" yaml:"maxIdleConnections"`
}

// RetryConfig represents the retry configuration
type RetryConfig struct {
	MaxRetries    int           `mapstructure:"maxRetries" yaml:"maxRetries"`
	RetryInterval time.Duration `mapstructure:"retryInterval" yaml:"retryInterval"`
}
