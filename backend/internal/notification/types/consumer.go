package types

import (
	"context"
)

// Consumer defines the interface for notification consumers
type Consumer interface {
	// Start starts consuming messages from the topic
	Start(ctx context.Context) error
	
	// Stop stops the consumer and releases resources
	Stop() error
	
	// IsRunning returns true if the consumer is running
	IsRunning() bool
}

// ConsumerManager defines the interface for managing notification consumers
type ConsumerManager interface {
	// Start starts all consumers
	Start(ctx context.Context) error
	
	// Stop stops all consumers
	Stop() error
	
	// IsRunning returns true if the manager is running
	IsRunning() bool
} 