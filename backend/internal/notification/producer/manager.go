package producer

import (
	"fmt"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
)

// Manager manages all notification producers
type Manager struct {
	client     pulsar.Client
	logger     logger.Logger
	repository types.NotificationRepository
	config     *types.ServiceConfig
	
	userProducer    *UserProducer
	commentProducer *CommentProducer
	videoProducer   *VideoProducer
}

// NewManager creates a new producer manager
func NewManager(
	client pulsar.Client,
	logger logger.Logger,
	repository types.NotificationRepository,
	config *types.ServiceConfig,
) (*Manager, error) {
	return &Manager{
		client:     client,
		logger:     logger,
		repository: repository,
		config:     config,
	}, nil
}

// Start initializes and starts all producers
func (m *Manager) Start() error {
	var err error

	// Initialize user producer
	m.userProducer, err = NewUserProducer(m.client, m.logger, m.repository, m.config)
	if err != nil {
		return fmt.Errorf("failed to create user producer: %w", err)
	}

	// Initialize comment producer
	m.commentProducer, err = NewCommentProducer(m.client, m.logger, m.repository, m.config)
	if err != nil {
		return fmt.Errorf("failed to create comment producer: %w", err)
	}

	// Initialize video producer
	m.videoProducer, err = NewVideoProducer(m.client, m.logger, m.repository, m.config)
	if err != nil {
		return fmt.Errorf("failed to create video producer: %w", err)
	}

	m.logger.LogInfo("All notification producers started successfully", nil)
	return nil
}

// Stop closes all producers
func (m *Manager) Stop() error {
	if m.userProducer != nil {
		if err := m.userProducer.Close(); err != nil {
			m.logger.LogError(err, "Failed to close user producer")
		}
	}

	if m.commentProducer != nil {
		if err := m.commentProducer.Close(); err != nil {
			m.logger.LogError(err, "Failed to close comment producer")
		}
	}

	if m.videoProducer != nil {
		if err := m.videoProducer.Close(); err != nil {
			m.logger.LogError(err, "Failed to close video producer")
		}
	}

	m.logger.LogInfo("All notification producers stopped successfully", nil)
	return nil
}

// GetUserProducer returns the user producer
func (m *Manager) GetUserProducer() *UserProducer {
	return m.userProducer
}

// GetCommentProducer returns the comment producer
func (m *Manager) GetCommentProducer() *CommentProducer {
	return m.commentProducer
}

// GetVideoProducer returns the video producer
func (m *Manager) GetVideoProducer() *VideoProducer {
	return m.videoProducer
}