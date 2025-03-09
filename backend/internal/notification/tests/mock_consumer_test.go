package tests

import (
	"context"
	"sync"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification"
)

// MockConsumer is a simple consumer for testing
type MockConsumer struct {
	client      pulsar.Client
	consumer    pulsar.Consumer
	topic       string
	messages    []pulsar.Message
	messageChan chan pulsar.Message
	ctx         context.Context
	cancel      context.CancelFunc
	logger      logger.Logger
	wg          sync.WaitGroup
	mutex       sync.Mutex
}

// NewMockConsumer creates a new test consumer
func NewMockConsumer(config *notification.ServiceConfig, topic string, logger logger.Logger) (*MockConsumer, error) {
	// Create client options
	clientOptions := pulsar.ClientOptions{
		URL:               config.PulsarURL,
		OperationTimeout:  config.OperationTimeout,
		ConnectionTimeout: config.ConnectionTimeout,
	}

	// Configure TLS if enabled
	if config.TLSEnabled && config.TLSCertPath != "" {
		clientOptions.TLSTrustCertsFilePath = config.TLSCertPath
		clientOptions.TLSAllowInsecureConnection = false
	}

	// Configure authentication if token is provided
	if config.AuthToken != "" {
		clientOptions.Authentication = pulsar.NewAuthenticationToken(config.AuthToken)
	}

	// Create client
	client, err := pulsar.NewClient(clientOptions)
	if err != nil {
		return nil, err
	}

	// Create consumer
	ctx, cancel := context.WithCancel(context.Background())
	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:            topic,
		SubscriptionName: "test-subscription-" + time.Now().Format("20060102150405"),
		Type:             pulsar.Exclusive,
	})
	if err != nil {
		cancel()
		client.Close()
		return nil, err
	}

	m := &MockConsumer{
		client:      client,
		consumer:    consumer,
		topic:       topic,
		messages:    make([]pulsar.Message, 0),
		messageChan: make(chan pulsar.Message, 100),
		ctx:         ctx,
		cancel:      cancel,
		logger:      logger,
	}

	// Start the consumer go routine
	m.wg.Add(1)
	go m.consumeMessages()

	return m, nil
}

// consumeMessages continuously consumes messages
func (m *MockConsumer) consumeMessages() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
		default:
			msg, err := m.consumer.Receive(m.ctx)
			if err != nil {
				if m.ctx.Err() == nil {
					m.logger.LogError(err, "Error receiving message")
				}
				return
			}

			m.mutex.Lock()
			m.messages = append(m.messages, msg)
			m.mutex.Unlock()

			// Send to channel for async processing
			select {
			case m.messageChan <- msg:
			default:
				m.logger.LogWarn("Message channel full, dropping message", nil)
			}

			// Acknowledge the message
			m.consumer.Ack(msg)
		}
	}
}

// GetMessages returns received messages
func (m *MockConsumer) GetMessages() []pulsar.Message {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.messages
}

// Close closes the consumer
func (m *MockConsumer) Close() {
	m.cancel()
	m.consumer.Close()
	m.client.Close()
	m.wg.Wait()
	close(m.messageChan)
}