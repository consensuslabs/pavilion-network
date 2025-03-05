package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/google/uuid"
)

// Service implements the NotificationService interface
type Service struct {
	config       *ServiceConfig
	logger       logger.Logger
	pulsarClient pulsar.Client
	repository   NotificationRepository
	
	// Producers for different event types
	videoProducer   pulsar.Producer
	commentProducer pulsar.Producer
	userProducer    pulsar.Producer
	
	// DLQ and retry queue producers
	dlqProducer   pulsar.Producer
	retryProducer pulsar.Producer
}

// NewService creates a new Notification Service
func NewService(ctx context.Context, config *ServiceConfig, logger logger.Logger, repository NotificationRepository) (*Service, error) {
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

	// Initialize producers
	if err := service.initProducers(ctx); err != nil {
		client.Close()
		return nil, err
	}

	logger.LogInfo("Notification service initialized successfully", map[string]interface{}{
		"pulsar_url": config.PulsarURL,
		"tls_enabled": config.TLSEnabled,
		"auth_enabled": config.AuthToken != "",
	})

	return service, nil
}

// initProducers initializes all required Pulsar producers
func (s *Service) initProducers(ctx context.Context) error {
	var err error

	// Create video events producer
	s.videoProducer, err = s.createProducer(ctx, s.config.VideoEventsTopic)
	if err != nil {
		return fmt.Errorf("failed to create video events producer: %w", err)
	}

	// Create comment events producer
	s.commentProducer, err = s.createProducer(ctx, s.config.CommentEventsTopic)
	if err != nil {
		return fmt.Errorf("failed to create comment events producer: %w", err)
	}

	// Create user events producer
	s.userProducer, err = s.createProducer(ctx, s.config.UserEventsTopic)
	if err != nil {
		return fmt.Errorf("failed to create user events producer: %w", err)
	}

	// Create DLQ producer
	s.dlqProducer, err = s.createProducer(ctx, s.config.DeadLetterTopic)
	if err != nil {
		return fmt.Errorf("failed to create dead letter queue producer: %w", err)
	}

	// Create retry queue producer
	s.retryProducer, err = s.createProducer(ctx, s.config.RetryQueueTopic)
	if err != nil {
		return fmt.Errorf("failed to create retry queue producer: %w", err)
	}

	return nil
}

// createProducer creates a new Pulsar producer for the given topic
func (s *Service) createProducer(ctx context.Context, topic string) (pulsar.Producer, error) {
	return s.pulsarClient.CreateProducer(pulsar.ProducerOptions{
		Topic:                   topic,
		SendTimeout:             s.config.OperationTimeout,
		MaxPendingMessages:      100,
		DisableBatching:         false,
		BatchingMaxPublishDelay: 10 * time.Millisecond,
		BatchingMaxMessages:     1000,
	})
}

// PublishVideoEvent publishes a video-related notification event
func (s *Service) PublishVideoEvent(ctx context.Context, event *VideoEvent) error {
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
	if event.EventKey == "" {
		event.EventKey = event.VideoID.String()
	}
	if event.SequenceNumber == 0 {
		event.SequenceNumber = time.Now().UnixNano()
	}

	// Serialize the event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to serialize video event: %w", err)
	}

	// Create producer message
	msg := &pulsar.ProducerMessage{
		Payload: data,
		Key:     event.EventKey,
		Properties: map[string]string{
			"eventType": string(event.Type),
			"videoId":   event.VideoID.String(),
			"userId":    event.UserID.String(),
		},
		EventTime: event.CreatedAt,
	}

	// Send the message to Pulsar
	msgID, err := s.videoProducer.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to publish video event: %w", err)
	}

	s.logger.LogInfo("Published video event", map[string]interface{}{
		"eventType": event.Type,
		"videoId": event.VideoID.String(),
		"userId": event.UserID.String(),
		"messageId": msgID,
	})

	// Create a notification for persistent storage
	notification := &Notification{
		ID:        event.ID,
		UserID:    event.UserID,
		Type:      event.Type,
		Content:   createContentFromVideoEvent(event),
		Metadata:  createMetadataFromVideoEvent(event),
		CreatedAt: event.CreatedAt,
	}

	// Store the notification in the repository if we have one
	if s.repository != nil {
		if err := s.repository.SaveNotification(ctx, notification); err != nil {
			s.logger.LogError(err, "Failed to save video notification to repository")
			// We don't return the error as the message was already published to Pulsar
			// This is a non-critical error for the notification flow
		}
	}

	return nil
}

// PublishCommentEvent publishes a comment-related notification event
func (s *Service) PublishCommentEvent(ctx context.Context, event *CommentEvent) error {
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
	if event.EventKey == "" {
		event.EventKey = event.CommentID.String()
	}
	if event.SequenceNumber == 0 {
		event.SequenceNumber = time.Now().UnixNano()
	}

	// Serialize the event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to serialize comment event: %w", err)
	}

	// Create producer message
	msg := &pulsar.ProducerMessage{
		Payload: data,
		Key:     event.EventKey,
		Properties: map[string]string{
			"eventType": string(event.Type),
			"commentId": event.CommentID.String(),
			"userId":    event.UserID.String(),
		},
		EventTime: event.CreatedAt,
	}

	// Send the message
	msgID, err := s.commentProducer.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to publish comment event: %w", err)
	}

	s.logger.LogInfo("Published comment event", map[string]interface{}{
		"eventType": event.Type,
		"commentId": event.CommentID.String(),
		"userId": event.UserID.String(),
		"messageId": msgID,
	})

	// Create a notification for persistent storage
	notification := &Notification{
		ID:        event.ID,
		UserID:    event.UserID,
		Type:      event.Type,
		Content:   createContentFromCommentEvent(event),
		Metadata:  createMetadataFromCommentEvent(event),
		CreatedAt: event.CreatedAt,
	}

	// Store the notification in the repository if we have one
	if s.repository != nil {
		if err := s.repository.SaveNotification(ctx, notification); err != nil {
			s.logger.LogError(err, "Failed to save comment notification to repository")
			// We don't return the error as the message was already published to Pulsar
			// This is a non-critical error for the notification flow
		}
	}

	return nil
}

// PublishUserEvent publishes a user-related notification event
func (s *Service) PublishUserEvent(ctx context.Context, event *UserEvent) error {
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
	if event.EventKey == "" {
		event.EventKey = fmt.Sprintf("%s-%s", event.UserID.String(), event.TargetUserID.String())
	}
	if event.SequenceNumber == 0 {
		event.SequenceNumber = time.Now().UnixNano()
	}

	// Serialize the event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to serialize user event: %w", err)
	}

	// Create producer message
	msg := &pulsar.ProducerMessage{
		Payload: data,
		Key:     event.EventKey,
		Properties: map[string]string{
			"eventType":    string(event.Type),
			"userId":       event.UserID.String(),
			"targetUserId": event.TargetUserID.String(),
		},
		EventTime: event.CreatedAt,
	}

	// Send the message
	msgID, err := s.userProducer.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to publish user event: %w", err)
	}

	s.logger.LogInfo("Published user event", map[string]interface{}{
		"eventType": event.Type,
		"userId": event.UserID.String(),
		"targetUserId": event.TargetUserID.String(),
		"messageId": msgID,
	})

	// Create a notification for the target user
	notification := &Notification{
		ID:        event.ID,
		UserID:    event.TargetUserID, // The notification is for the target user
		Type:      event.Type,
		Content:   createContentFromUserEvent(event),
		Metadata:  createMetadataFromUserEvent(event),
		CreatedAt: event.CreatedAt,
	}

	// Store the notification in the repository if we have one
	if s.repository != nil {
		if err := s.repository.SaveNotification(ctx, notification); err != nil {
			s.logger.LogError(err, "Failed to save user notification to repository")
			// We don't return the error as the message was already published to Pulsar
			// This is a non-critical error for the notification flow
		}
	}

	return nil
}

// GetUserNotifications gets notifications for a user
func (s *Service) GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Notification, error) {
	if s.repository == nil {
		s.logger.LogWarn("Repository not initialized, returning empty notifications list", map[string]interface{}{
			"userID": userID.String(),
		})
		return []*Notification{}, nil
	}

	notifications, err := s.repository.GetNotificationsByUserID(ctx, userID, limit, offset)
	if err != nil {
		s.logger.LogError(err, "Failed to get notifications for user")
		return nil, fmt.Errorf("failed to get notifications for user: %w", err)
	}

	s.logger.LogInfo("Retrieved notifications for user", map[string]interface{}{
		"userID": userID.String(),
		"count": len(notifications),
	})

	return notifications, nil
}

// GetUnreadCount gets the count of unread notifications
func (s *Service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	if s.repository == nil {
		s.logger.LogWarn("Repository not initialized, returning zero unread count", map[string]interface{}{
			"userID": userID.String(),
		})
		return 0, nil
	}

	count, err := s.repository.GetUnreadCount(ctx, userID)
	if err != nil {
		s.logger.LogError(err, "Failed to get unread count for user")
		return 0, fmt.Errorf("failed to get unread count for user: %w", err)
	}

	s.logger.LogInfo("Retrieved unread count for user", map[string]interface{}{
		"userID": userID.String(),
		"count": count,
	})

	return count, nil
}

// MarkAsRead marks a notification as read
func (s *Service) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	if s.repository == nil {
		s.logger.LogWarn("Repository not initialized, cannot mark notification as read", map[string]interface{}{
			"notificationID": notificationID.String(),
		})
		return nil
	}

	err := s.repository.MarkAsRead(ctx, notificationID)
	if err != nil {
		s.logger.LogError(err, "Failed to mark notification as read")
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	s.logger.LogInfo("Marked notification as read", map[string]interface{}{
		"notificationID": notificationID.String(),
	})

	return nil
}

// MarkAllAsRead marks all notifications as read
func (s *Service) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	if s.repository == nil {
		s.logger.LogWarn("Repository not initialized, cannot mark all notifications as read", map[string]interface{}{
			"userID": userID.String(),
		})
		return nil
	}

	err := s.repository.MarkAllAsRead(ctx, userID)
	if err != nil {
		s.logger.LogError(err, "Failed to mark all notifications as read")
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	s.logger.LogInfo("Marked all notifications as read for user", map[string]interface{}{
		"userID": userID.String(),
	})

	return nil
}

// Close closes the notification service and releases resources
func (s *Service) Close() error {
	if !s.config.Enabled {
		return nil
	}

	// Close all producers
	if s.videoProducer != nil {
		s.videoProducer.Close()
	}
	if s.commentProducer != nil {
		s.commentProducer.Close()
	}
	if s.userProducer != nil {
		s.userProducer.Close()
	}
	if s.dlqProducer != nil {
		s.dlqProducer.Close()
	}
	if s.retryProducer != nil {
		s.retryProducer.Close()
	}

	// Close the Pulsar client
	if s.pulsarClient != nil {
		s.pulsarClient.Close()
	}

	s.logger.LogInfo("Notification service closed", nil)
	return nil
}

// Helper methods for creating notification content and metadata

// createContentFromVideoEvent creates a human-readable notification content from a video event
func createContentFromVideoEvent(event *VideoEvent) string {
	switch event.Type {
	case VideoUploaded:
		return fmt.Sprintf("Your video '%s' has been uploaded successfully", event.Title)
	case VideoProcessed:
		return fmt.Sprintf("Your video '%s' has been processed and is now available", event.Title)
	case VideoUpdated:
		return fmt.Sprintf("Your video '%s' has been updated", event.Title)
	case VideoDeleted:
		return fmt.Sprintf("Your video '%s' has been deleted", event.Title)
	default:
		return fmt.Sprintf("Video notification: %s", event.Title)
	}
}

// createMetadataFromVideoEvent extracts metadata from a video event
func createMetadataFromVideoEvent(event *VideoEvent) map[string]interface{} {
	metadata := make(map[string]interface{})
	
	// Add basic video information
	metadata["videoId"] = event.VideoID.String()
	metadata["userId"] = event.UserID.String()
	metadata["eventType"] = string(event.Type)
	
	// Add title if available
	if event.Title != "" {
		metadata["title"] = event.Title
	}
	
	// Add custom metadata if available
	if event.Metadata != nil {
		for k, v := range event.Metadata {
			metadata[k] = v
		}
	}
	
	return metadata
}

// createContentFromCommentEvent creates a human-readable notification content from a comment event
func createContentFromCommentEvent(event *CommentEvent) string {
	switch event.Type {
	case CommentCreated:
		return "New comment on your video"
	case CommentReplied:
		return "Someone replied to your comment"
	case CommentReaction:
		return "Someone reacted to your comment"
	default:
		return "Comment notification"
	}
}

// createMetadataFromCommentEvent extracts metadata from a comment event
func createMetadataFromCommentEvent(event *CommentEvent) map[string]interface{} {
	metadata := make(map[string]interface{})
	
	// Add basic comment information
	metadata["commentId"] = event.CommentID.String()
	metadata["userId"] = event.UserID.String()
	metadata["eventType"] = string(event.Type)
	
	// Add video ID if available
	if event.VideoID != uuid.Nil {
		metadata["videoId"] = event.VideoID.String()
	}
	
	// Add parent comment ID if available
	if event.ParentID != uuid.Nil {
		metadata["parentId"] = event.ParentID.String()
	}
	
	// Add content preview if available
	if event.Content != "" {
		// Truncate content for preview if it's too long
		if len(event.Content) > 50 {
			metadata["contentPreview"] = event.Content[:47] + "..."
		} else {
			metadata["contentPreview"] = event.Content
		}
	}
	
	// Add custom metadata if available
	if event.Metadata != nil {
		for k, v := range event.Metadata {
			metadata[k] = v
		}
	}
	
	return metadata
}

// createContentFromUserEvent creates a human-readable notification content from a user event
func createContentFromUserEvent(event *UserEvent) string {
	switch event.Type {
	case UserFollowed:
		return "Someone started following you"
	case UserMentioned:
		return "You were mentioned in a comment"
	case AuthEvent:
		return "Security alert: new login detected"
	default:
		return "User notification"
	}
}

// createMetadataFromUserEvent extracts metadata from a user event
func createMetadataFromUserEvent(event *UserEvent) map[string]interface{} {
	metadata := make(map[string]interface{})
	
	// Add basic user information
	metadata["userId"] = event.UserID.String()
	metadata["targetUserId"] = event.TargetUserID.String()
	metadata["eventType"] = string(event.Type)
	
	// Add content if available
	if event.Content != "" {
		metadata["content"] = event.Content
	}
	
	// Add custom metadata if available
	if event.Metadata != nil {
		for k, v := range event.Metadata {
			metadata[k] = v
		}
	}
	
	return metadata
}