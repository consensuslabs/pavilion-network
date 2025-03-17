package consumer

import (
	"context"
	"fmt"
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
)

// Manager manages all notification consumers
type Manager struct {
	client     pulsar.Client
	config     *types.ServiceConfig
	logger     logger.Logger
	repository types.NotificationRepository
	consumers  []types.Consumer
	running    bool
	mutex      sync.RWMutex
}

// NewManager creates a new consumer manager
func NewManager(
	client pulsar.Client,
	config *types.ServiceConfig,
	logger logger.Logger,
	repository types.NotificationRepository,
) *Manager {
	return &Manager{
		client:     client,
		config:     config,
		logger:     logger,
		repository: repository,
		consumers:  make([]types.Consumer, 0),
		running:    false,
	}
}

// Start starts all consumers
func (m *Manager) Start(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.running {
		return fmt.Errorf("consumer manager already running")
	}

	// Register consumers if not already done
	if len(m.consumers) == 0 {
		// Create and register video consumer
		videoConsumer := NewVideoConsumer(m.client, m.logger, m.repository, m.config)
		m.consumers = append(m.consumers, videoConsumer)

		// Create and register comment consumer
		commentConsumer := NewCommentConsumer(m.client, m.logger, m.repository, m.config)
		m.consumers = append(m.consumers, commentConsumer)

		// Create and register user consumer
		userConsumer := NewUserConsumer(m.client, m.logger, m.repository, m.config)
		m.consumers = append(m.consumers, userConsumer)

		m.logger.LogInfo("Registered notification consumers", map[string]interface{}{
			"count": len(m.consumers),
		})
	}

	// Start each consumer
	for i, consumer := range m.consumers {
		if err := consumer.Start(ctx); err != nil {
			// Stop any consumers that were already started
			for j := 0; j < i; j++ {
				m.consumers[j].Stop()
			}
			return fmt.Errorf("failed to start consumer %d: %w", i, err)
		}
	}

	m.running = true
	m.logger.LogInfo("Started all notification consumers", map[string]interface{}{
		"count": len(m.consumers),
	})

	return nil
}

// Stop stops all consumers
func (m *Manager) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.running {
		return nil
	}

	var lastErr error
	// Stop each consumer
	for i, consumer := range m.consumers {
		if err := consumer.Stop(); err != nil {
			m.logger.LogError(err, fmt.Sprintf("Failed to stop consumer %d", i))
			lastErr = err
		}
	}

	m.running = false
	m.logger.LogInfo("Stopped all notification consumers", map[string]interface{}{
		"count": len(m.consumers),
	})

	// Return the last error if any
	return lastErr
}

// IsRunning returns true if the manager is running
func (m *Manager) IsRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.running
} 