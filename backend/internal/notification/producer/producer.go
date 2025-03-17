package producer

import (
	"context"
	"fmt"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
)

// Producer defines the interface for notification producers
type Producer interface {
	// Publish publishes an event to the topic
	Publish(ctx context.Context, event interface{}) error
	
	// Close closes the producer and releases resources
	Close() error
}

// BaseProducer provides common functionality for all producers
type BaseProducer struct {
	client    pulsar.Client
	producer  pulsar.Producer
	topic     string
	logger    logger.Logger
	repository types.NotificationRepository
}

// NewBaseProducer creates a new base producer
func NewBaseProducer(
	client pulsar.Client,
	topic string,
	logger logger.Logger,
	repository types.NotificationRepository,
) (*BaseProducer, error) {
	// Create producer
	producer, err := client.CreateProducer(pulsar.ProducerOptions{
		Topic:                   topic,
		SendTimeout:             30 * time.Second,
		MaxPendingMessages:      100,
		DisableBatching:         false,
		BatchingMaxPublishDelay: 10 * time.Millisecond,
		BatchingMaxMessages:     1000,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	return &BaseProducer{
		client:    client,
		producer:  producer,
		topic:     topic,
		logger:    logger,
		repository: repository,
	}, nil
}

// Close closes the producer and releases resources
func (p *BaseProducer) Close() error {
	if p.producer != nil {
		p.producer.Close()
	}
	return nil
} 