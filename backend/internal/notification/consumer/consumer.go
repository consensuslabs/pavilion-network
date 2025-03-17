package consumer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
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

// BaseConsumer provides common functionality for all consumers
type BaseConsumer struct {
	client           pulsar.Client
	consumer         pulsar.Consumer
	topic            string
	subscriptionName string
	logger           logger.Logger
	repository       types.NotificationRepository
	running          bool
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	mutex            sync.RWMutex
	processMessage   func(msg pulsar.Message) error
	deadLetterTopic  string
	retryLetterTopic string
}

// NewBaseConsumer creates a new base consumer
func NewBaseConsumer(
	client pulsar.Client,
	topic string,
	subscriptionName string,
	logger logger.Logger,
	repository types.NotificationRepository,
	deadLetterTopic string,
	retryLetterTopic string,
) *BaseConsumer {
	consumer := &BaseConsumer{
		client:           client,
		topic:            topic,
		subscriptionName: subscriptionName,
		logger:           logger,
		repository:       repository,
		running:          false,
		deadLetterTopic:  deadLetterTopic,
		retryLetterTopic: retryLetterTopic,
	}
	
	// Set default message processor
	consumer.processMessage = consumer.defaultProcessMessage
	
	return consumer
}

// Start starts consuming messages from the topic
func (c *BaseConsumer) Start(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.running {
		return fmt.Errorf("consumer already running")
	}

	// Create a new context with cancellation
	c.ctx, c.cancel = context.WithCancel(ctx)

	// Create consumer
	consumer, err := c.client.Subscribe(pulsar.ConsumerOptions{
		Topic:            c.topic,
		SubscriptionName: c.subscriptionName,
		Type:             pulsar.Shared, // Use shared subscription for scalability
		RetryEnable:      true,
		DLQ: &pulsar.DLQPolicy{
			MaxDeliveries:    3,
			DeadLetterTopic:  c.deadLetterTopic,
			RetryLetterTopic: c.retryLetterTopic,
		},
	})
	if err != nil {
		c.cancel()
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	c.consumer = consumer
	c.running = true

	// Start consuming messages in a goroutine
	c.wg.Add(1)
	go c.consumeMessages()

	c.logger.LogInfo("Consumer started", map[string]interface{}{
		"topic":          c.topic,
		"subscription":   c.subscriptionName,
	})

	return nil
}

// Stop stops the consumer and releases resources
func (c *BaseConsumer) Stop() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.running {
		return nil
	}

	// Cancel the context to signal the consumer to stop
	c.cancel()

	// Close the consumer
	c.consumer.Close()

	// Wait for the consumer goroutine to finish
	c.wg.Wait()

	c.running = false
	c.logger.LogInfo("Consumer stopped", map[string]interface{}{
		"topic":        c.topic,
		"subscription": c.subscriptionName,
	})

	return nil
}

// IsRunning returns true if the consumer is running
func (c *BaseConsumer) IsRunning() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.running
}

// consumeMessages continuously consumes messages from the topic
func (c *BaseConsumer) consumeMessages() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// Receive message with a timeout
			msg, err := c.consumer.Receive(c.ctx)
			if err != nil {
				if c.ctx.Err() == nil {
					c.logger.LogError(err, "Error receiving message")
					// Sleep briefly to avoid tight loop on error
					time.Sleep(100 * time.Millisecond)
				}
				continue
			}

			// Process the message
			if err := c.processMessage(msg); err != nil {
				c.logger.LogError(err, "Error processing message")
				// Negative acknowledge to trigger retry
				c.consumer.Nack(msg)
			} else {
				// Acknowledge the message
				c.consumer.Ack(msg)
			}
		}
	}
}

// defaultProcessMessage is the default implementation for processing messages
func (c *BaseConsumer) defaultProcessMessage(msg pulsar.Message) error {
	// Default implementation just logs the message
	c.logger.LogInfo("Received message", map[string]interface{}{
		"messageId":   msg.ID(),
		"topic":       c.topic,
		"publishTime": msg.PublishTime(),
		"properties":  msg.Properties(),
	})
	return nil
} 