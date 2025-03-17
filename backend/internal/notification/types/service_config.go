package types

import (
	"time"
)

// ServiceConfig contains configuration for the notification service
type ServiceConfig struct {
	// General settings
	// Whether the notification service is enabled
	Enabled bool
	// Whether notification consumers are enabled
	ConsumersEnabled bool

	// Pulsar connection settings
	// URL for the Pulsar broker
	PulsarURL string
	// Timeout for Pulsar operations
	OperationTimeout time.Duration
	// Timeout for establishing connection to Pulsar
	ConnectionTimeout time.Duration
	
	// Security settings
	// Whether TLS is enabled for Pulsar connection
	TLSEnabled bool
	// Path to TLS certificate file
	TLSCertPath string
	// Authentication token for Pulsar
	AuthToken string

	// Topic configuration
	Topics struct {
		// Topic for video-related events
		VideoEvents string
		// Topic for comment-related events
		CommentEvents string
		// Topic for user-related events
		UserEvents string
		// Topic for messages that couldn't be processed
		DeadLetter string
		// Topic for retrying failed messages
		RetryQueue string
	}
	
	// Retention settings
	// How long to retain notifications in hours
	RetentionTimeHours int
	
	// Deduplication settings
	// Whether to enable deduplication of notifications
	DeduplicationEnabled bool
	// Time window for deduplication
	DeduplicationWindow time.Duration
	
	// Resilience settings
	// Whether to enable retries for failed operations
	RetryEnabled bool
	// Maximum number of retry attempts
	MaxRetries int
	// Initial backoff duration for retries
	BackoffInitial time.Duration
	// Maximum backoff duration for retries
	BackoffMax time.Duration
	// Multiplier for backoff duration between retries
	BackoffMultiplier float64
	
	// ScyllaDB settings for the notification repository
	ScyllaDB struct {
		// Connection pool settings
		// Maximum number of connections in the pool
		MaxConnections int
		// Maximum number of idle connections in the pool
		MaxIdleConnections int
		// Timeout for establishing database connections
		ConnectTimeout time.Duration
		// Timeout for database operations
		Timeout time.Duration
		
		// Retry settings for database operations
		// Maximum number of retry attempts for database operations
		MaxRetries int
		// Interval between retry attempts
		RetryInterval time.Duration
	}
} 