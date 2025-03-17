package producer

import (
	"context"
	"fmt"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/util"
	"github.com/google/uuid"
)

// UserProducer is responsible for producing user events
type UserProducer struct {
	*BaseProducer
}

// NewUserProducer creates a new user producer
func NewUserProducer(
	client pulsar.Client,
	logger logger.Logger,
	repository types.NotificationRepository,
	config *types.ServiceConfig,
) (*UserProducer, error) {
	baseProducer, err := NewBaseProducer(
		client,
		config.Topics.UserEvents,
		logger,
		repository,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create base producer: %w", err)
	}

	return &UserProducer{
		BaseProducer: baseProducer,
	}, nil
}

// Publish publishes a user event
func (p *UserProducer) Publish(ctx context.Context, event interface{}) error {
	userEvent, ok := event.(types.UserEvent)
	if !ok {
		return fmt.Errorf("invalid event type: expected UserEvent")
	}

	// Ensure event has an ID and timestamp
	userEvent.ID = util.GenerateEventID(userEvent.ID)
	userEvent.CreatedAt = util.GenerateEventTime(userEvent.CreatedAt)

	// Create properties for the message
	properties := map[string]string{
		"event_type":     string(userEvent.Type),
		"user_id":        userEvent.UserID.String(),
		"target_user_id": userEvent.TargetUserID.String(),
	}

	// Create producer message
	msg, err := util.CreateProducerMessage(
		userEvent,
		userEvent.UserID.String(),
		properties,
		userEvent.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create producer message: %w", err)
	}

	// Send the message
	if _, err := p.producer.Send(ctx, msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	p.logger.LogInfo("Published user event", map[string]interface{}{
		"event_id":       userEvent.ID,
		"event_type":     string(userEvent.Type),
		"user_id":        userEvent.UserID,
		"target_user_id": userEvent.TargetUserID,
	})

	return nil
}

// PublishFollowEvent publishes a follow event
func (p *UserProducer) PublishFollowEvent(ctx context.Context, userID, targetUserID uuid.UUID) error {
	event := types.UserEvent{
		BaseEvent: types.BaseEvent{
			Type:      types.UserFollowed,
			CreatedAt: time.Now(),
		},
		UserID:       userID,
		TargetUserID: targetUserID,
	}

	return p.Publish(ctx, event)
}

// PublishUnfollowEvent publishes an unfollow event
func (p *UserProducer) PublishUnfollowEvent(ctx context.Context, userID, targetUserID uuid.UUID) error {
	event := types.UserEvent{
		BaseEvent: types.BaseEvent{
			Type:      types.UserUnfollowed,
			CreatedAt: time.Now(),
		},
		UserID:       userID,
		TargetUserID: targetUserID,
	}

	return p.Publish(ctx, event)
} 