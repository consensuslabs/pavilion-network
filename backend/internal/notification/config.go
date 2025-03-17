package notification

import (
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/config"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
)

// NewServiceConfigFromConfig creates a notification service config from the application config
func NewServiceConfigFromConfig(cfg *config.Config) *types.ServiceConfig {
	svcConfig := &types.ServiceConfig{
		// General settings
		Enabled:          cfg.Notification.Enabled,
		ConsumersEnabled: cfg.Notification.ConsumersEnabled,

		// Pulsar settings
		PulsarURL:         cfg.Notification.PulsarURL,
		OperationTimeout:  cfg.Notification.OperationTimeout,
		ConnectionTimeout: cfg.Notification.ConnectionTimeout,
		
		// Security settings
		TLSEnabled:        cfg.Notification.TLSEnabled,
		TLSCertPath:       cfg.Notification.TLSCertPath,
		AuthToken:         cfg.Notification.AuthToken,
		
		// Retention settings
		RetentionTimeHours: cfg.Notification.RetentionTimeHours,
		
		// Deduplication
		DeduplicationEnabled: cfg.Notification.DeduplicationEnabled,
		DeduplicationWindow:  cfg.Notification.DeduplicationWindow,
		
		// Resilience
		RetryEnabled:      cfg.Notification.RetryEnabled,
		MaxRetries:        cfg.Notification.MaxRetries,
		BackoffInitial:    cfg.Notification.BackoffInitial,
		BackoffMax:        cfg.Notification.BackoffMax,
		BackoffMultiplier: cfg.Notification.BackoffMultiplier,
	}
	
	// Topic configuration
	svcConfig.Topics.VideoEvents = cfg.Notification.VideoEventsTopic
	svcConfig.Topics.CommentEvents = cfg.Notification.CommentEventsTopic
	svcConfig.Topics.UserEvents = cfg.Notification.UserEventsTopic
	svcConfig.Topics.DeadLetter = cfg.Notification.DeadLetterTopic
	svcConfig.Topics.RetryQueue = cfg.Notification.RetryQueueTopic
	
	// ScyllaDB configuration
	svcConfig.ScyllaDB.MaxConnections = cfg.ScyllaDB.Pool.MaxConnections
	svcConfig.ScyllaDB.MaxIdleConnections = cfg.ScyllaDB.Pool.MaxIdleConnections
	svcConfig.ScyllaDB.ConnectTimeout = cfg.ScyllaDB.ConnectTimeout
	svcConfig.ScyllaDB.Timeout = cfg.ScyllaDB.Timeout
	svcConfig.ScyllaDB.MaxRetries = cfg.ScyllaDB.Retry.MaxRetries
	svcConfig.ScyllaDB.RetryInterval = cfg.ScyllaDB.Retry.RetryInterval
	
	return svcConfig
}

// DefaultConfig returns a default configuration for the notification service
func DefaultConfig() *types.ServiceConfig {
	svcConfig := &types.ServiceConfig{
		// General settings
		Enabled:          true,
		ConsumersEnabled: true,
		
		// Pulsar connection settings
		PulsarURL:        "pulsar://localhost:6650",
		OperationTimeout:  30 * time.Second,
		ConnectionTimeout: 30 * time.Second,
		
		// Security settings
		TLSEnabled:        false,
		TLSCertPath:       "",
		AuthToken:         "",
		
		// Retention settings
		RetentionTimeHours:  48,
		
		// Deduplication
		DeduplicationEnabled: true,
		DeduplicationWindow:  2 * time.Hour,
		
		// Resilience
		RetryEnabled:        true,
		MaxRetries:          5,
		BackoffInitial:      1 * time.Second,
		BackoffMax:          60 * time.Second,
		BackoffMultiplier:   2.0,
	}
	
	// Topic configuration
	svcConfig.Topics.VideoEvents = "persistent://pavilion/notifications/video-events"
	svcConfig.Topics.CommentEvents = "persistent://pavilion/notifications/comment-events"
	svcConfig.Topics.UserEvents = "persistent://pavilion/notifications/user-events"
	svcConfig.Topics.DeadLetter = "persistent://pavilion/notifications/dead-letter"
	svcConfig.Topics.RetryQueue = "persistent://pavilion/notifications/retry-queue"
	
	// ScyllaDB connection pool settings
	svcConfig.ScyllaDB.MaxConnections = 20
	svcConfig.ScyllaDB.MaxIdleConnections = 5
	svcConfig.ScyllaDB.ConnectTimeout = 10 * time.Second
	svcConfig.ScyllaDB.Timeout = 5 * time.Second
	svcConfig.ScyllaDB.MaxRetries = 3
	svcConfig.ScyllaDB.RetryInterval = 500 * time.Millisecond
	
	return svcConfig
}