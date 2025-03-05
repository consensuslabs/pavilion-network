package notification

import (
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/config"
)

// ServiceConfig contains configuration for the notification service
type ServiceConfig struct {
	// Pulsar connection settings
	PulsarURL          string
	PulsarWebServiceURL string
	TLSEnabled         bool
	TLSCertPath        string
	AuthToken          string
	OperationTimeout   time.Duration
	ConnectionTimeout  time.Duration
	Namespace          string
	
	// Enabled flag
	Enabled             bool
	
	// Topic names
	VideoEventsTopic    string
	CommentEventsTopic  string
	UserEventsTopic     string
	DeadLetterTopic     string
	RetryQueueTopic     string
	
	// Retention settings
	RetentionTimeHours int
	
	// Deduplication
	DeduplicationEnabled bool
	DeduplicationWindow  time.Duration
	
	// Resilience
	RetryEnabled      bool
	MaxRetries        int
	BackoffInitial    time.Duration
	BackoffMax        time.Duration
	BackoffMultiplier float64
}

// NewServiceConfigFromConfig creates a notification service config from the application config
func NewServiceConfigFromConfig(cfg *config.Config) *ServiceConfig {
	return &ServiceConfig{
		// Pulsar settings
		PulsarURL:           cfg.Pulsar.URL,
		PulsarWebServiceURL: cfg.Pulsar.WebServiceURL,
		TLSEnabled:          cfg.Pulsar.TLSEnabled,
		TLSCertPath:         cfg.Pulsar.TLSCertPath,
		AuthToken:           cfg.Pulsar.AuthToken, // Auth token from config (sourced from .env)
		OperationTimeout:    cfg.Pulsar.OperationTimeout,
		ConnectionTimeout:   cfg.Pulsar.ConnectionTimeout,
		Namespace:           cfg.Pulsar.Namespace,
		
		// Notification settings
		Enabled:             cfg.Notification.Enabled,
		VideoEventsTopic:    cfg.Notification.VideoEventsTopic,
		CommentEventsTopic:  cfg.Notification.CommentEventsTopic,
		UserEventsTopic:     cfg.Notification.UserEventsTopic,
		DeadLetterTopic:     cfg.Notification.DeadLetterTopic,
		RetryQueueTopic:     cfg.Notification.RetryQueueTopic,
		RetentionTimeHours:  cfg.Notification.RetentionTimeHours,
		DeduplicationEnabled: cfg.Notification.DeduplicationEnabled,
		DeduplicationWindow: cfg.Notification.DeduplicationWindow,
		RetryEnabled:        cfg.Notification.RetryEnabled,
		MaxRetries:          cfg.Notification.MaxRetries,
		BackoffInitial:      cfg.Notification.BackoffInitial,
		BackoffMax:          cfg.Notification.BackoffMax,
		BackoffMultiplier:   cfg.Notification.BackoffMultiplier,
	}
}

// DefaultConfig returns default configuration values
func DefaultConfig() *ServiceConfig {
	return &ServiceConfig{
		PulsarURL:           "pulsar://localhost:6650",
		PulsarWebServiceURL: "http://localhost:8083",
		TLSEnabled:          false,
		TLSCertPath:         "",
		AuthToken:           "", // No default auth token, should be loaded from env
		OperationTimeout:    30 * time.Second, 
		ConnectionTimeout:   30 * time.Second,
		Namespace:           "pavilion/notifications",
		
		Enabled:             true,
		VideoEventsTopic:    "persistent://pavilion/notifications/video-events",
		CommentEventsTopic:  "persistent://pavilion/notifications/comment-events",
		UserEventsTopic:     "persistent://pavilion/notifications/user-events",
		DeadLetterTopic:     "persistent://pavilion/notifications/dead-letter",
		RetryQueueTopic:     "persistent://pavilion/notifications/retry-queue",
		
		RetentionTimeHours:  48,
		
		DeduplicationEnabled: true,
		DeduplicationWindow:  2 * time.Hour,
		
		RetryEnabled:        true,
		MaxRetries:          5,
		BackoffInitial:      1 * time.Second,
		BackoffMax:          60 * time.Second,
		BackoffMultiplier:   2.0,
	}
}