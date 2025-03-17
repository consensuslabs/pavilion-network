package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/consumer"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/producer"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
	"github.com/google/uuid"
)

// Service implements the NotificationService interface
type Service struct {
	config       *types.ServiceConfig
	logger       logger.Logger
	pulsarClient pulsar.Client
	repository   types.NotificationRepository
	
	// Producer manager
	producerManager *producer.Manager
	
	// Consumer manager
	consumerManager *consumer.Manager
}

// NewService creates a new Notification Service
func NewService(ctx context.Context, config *types.ServiceConfig, logger logger.Logger, repository types.NotificationRepository) (*Service, error) {
	if !config.Enabled {
		logger.LogInfo("Notification service is disabled", nil)
		return &Service{config: config, logger: logger, repository: repository}, nil
	}

	// Create Pulsar client
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

	client, err := pulsar.NewClient(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create Pulsar client: %w", err)
	}

	service := &Service{
		config:       config,
		logger:       logger,
		pulsarClient: client,
		repository:   repository,
	}

	// Initialize producer manager
	service.producerManager, err = producer.NewManager(client, logger, repository, config)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create producer manager: %w", err)
	}

	// Start producers
	if err := service.producerManager.Start(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to start producers: %w", err)
	}

	// Initialize consumer manager
	service.consumerManager = consumer.NewManager(
		client,
		config,
		logger,
		repository,
	)

	// Start consumers if enabled
	if config.ConsumersEnabled {
		if err := service.consumerManager.Start(ctx); err != nil {
			logger.LogError(err, "Failed to start notification consumers")
			logger.LogWarn("Continuing without notification consumers", nil)
		} else {
			logger.LogInfo("Notification consumers started successfully", nil)
		}
	} else {
		logger.LogInfo("Notification consumers are disabled", nil)
	}

	logger.LogInfo("Notification service initialized successfully", map[string]interface{}{
		"pulsar_url": config.PulsarURL,
		"tls_enabled": config.TLSEnabled,
		"auth_enabled": config.AuthToken != "",
		"consumers_enabled": config.ConsumersEnabled,
	})

	return service, nil
}

// PublishVideoEvent publishes a video event to Pulsar
func (s *Service) PublishVideoEvent(ctx context.Context, event *types.VideoEvent) error {
	if !s.config.Enabled {
		s.logger.LogDebug("Notification service is disabled, ignoring video event", nil)
		return nil
	}

	// Set default values if not provided
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	// Use the appropriate producer based on event type
	switch event.Type {
	case VideoUploaded:
		return s.producerManager.GetVideoProducer().PublishVideoUploadEvent(
			ctx,
			event.UserID,
			event.VideoID,
			event.Title,
		)
	case VideoLiked:
		return s.producerManager.GetVideoProducer().PublishVideoLikeEvent(
			ctx,
			event.UserID,
			event.VideoID,
		)
	case VideoUnliked:
		return s.producerManager.GetVideoProducer().PublishVideoUnlikeEvent(
			ctx,
			event.UserID,
			event.VideoID,
		)
	default:
		return fmt.Errorf("unsupported video event type: %s", event.Type)
	}
}

// PublishCommentEvent publishes a comment event to Pulsar
func (s *Service) PublishCommentEvent(ctx context.Context, event *types.CommentEvent) error {
	if !s.config.Enabled {
		s.logger.LogDebug("Notification service is disabled, ignoring comment event", nil)
		return nil
	}

	// Set default values if not provided
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	// Use the appropriate producer based on event type
	switch event.Type {
	case CommentCreated:
		return s.producerManager.GetCommentProducer().PublishCommentCreatedEvent(
			ctx,
			event.UserID,
			event.VideoID,
			event.CommentID,
			event.Content,
		)
	case CommentReplied:
		return s.producerManager.GetCommentProducer().PublishCommentReplyEvent(
			ctx,
			event.UserID,
			event.VideoID,
			event.CommentID,
			event.ParentID,
			event.Content,
		)
	case CommentMention:
		// Extract mentioned user IDs from metadata
		var mentionedIDs []uuid.UUID
		if event.Metadata != nil {
			if mentionedIDsRaw, ok := event.Metadata["mentionedUserIds"]; ok {
				if mentionedIDsSlice, ok := mentionedIDsRaw.([]interface{}); ok {
					for _, id := range mentionedIDsSlice {
						if idStr, ok := id.(string); ok {
							if uid, err := uuid.Parse(idStr); err == nil {
								mentionedIDs = append(mentionedIDs, uid)
							}
						}
					}
				}
			}
		}
		
		return s.producerManager.GetCommentProducer().PublishMentionEvent(
			ctx,
			event.UserID,
			event.VideoID,
			event.CommentID,
			event.Content,
			mentionedIDs,
		)
	default:
		return fmt.Errorf("unsupported comment event type: %s", event.Type)
	}
}

// PublishUserEvent publishes a user event to Pulsar
func (s *Service) PublishUserEvent(ctx context.Context, event *types.UserEvent) error {
	if !s.config.Enabled {
		s.logger.LogDebug("Notification service is disabled, ignoring user event", nil)
		return nil
	}

	// Set default values if not provided
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	// Use the appropriate producer based on event type
	switch event.Type {
	case UserFollowed:
		return s.producerManager.GetUserProducer().PublishFollowEvent(
			ctx,
			event.UserID,
			event.TargetUserID,
		)
	case UserUnfollowed:
		return s.producerManager.GetUserProducer().PublishUnfollowEvent(
			ctx,
			event.UserID,
			event.TargetUserID,
		)
	default:
		return fmt.Errorf("unsupported user event type: %s", event.Type)
	}
}

// GetUserNotifications gets notifications for a user with pagination
func (s *Service) GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*types.Notification, error) {
	return s.repository.GetNotificationsByUserID(ctx, userID, limit, offset)
}

// GetUnreadCount gets the count of unread notifications for a user
func (s *Service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repository.GetUnreadCount(ctx, userID)
}

// MarkAsRead marks a notification as read
func (s *Service) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	return s.repository.MarkAsRead(ctx, notificationID)
}

// MarkAllAsRead marks all notifications for a user as read
func (s *Service) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	return s.repository.MarkAllAsRead(ctx, userID)
}

// Close closes the service and releases resources
func (s *Service) Close() error {
	if !s.config.Enabled {
		return nil
	}

	// Stop consumers
	if s.consumerManager != nil && s.consumerManager.IsRunning() {
		if err := s.consumerManager.Stop(); err != nil {
			s.logger.LogError(err, "Error stopping notification consumers")
			// Continue with cleanup despite errors
		}
	}

	// Stop producers
	if s.producerManager != nil {
		if err := s.producerManager.Stop(); err != nil {
			s.logger.LogError(err, "Error stopping notification producers")
			// Continue with cleanup despite errors
		}
	}

	// Close Pulsar client
	if s.pulsarClient != nil {
		s.pulsarClient.Close()
	}

	s.logger.LogInfo("Notification service closed", nil)
	return nil
}