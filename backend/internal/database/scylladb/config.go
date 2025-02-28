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
}

// Replication config for ScyllaDB
type Replication struct {
	Class             string `mapstructure:"class"`
	ReplicationFactor int    `mapstructure:"replicationFactor"`
}
