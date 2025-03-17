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

// CommentProducer is responsible for producing comment events
type CommentProducer struct {
	*BaseProducer
}

// NewCommentProducer creates a new comment producer
func NewCommentProducer(
	client pulsar.Client,
	logger logger.Logger,
	repository types.NotificationRepository,
	config *types.ServiceConfig,
) (*CommentProducer, error) {
	baseProducer, err := NewBaseProducer(
		client,
		config.Topics.CommentEvents,
		logger,
		repository,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create base producer: %w", err)
	}

	return &CommentProducer{
		BaseProducer: baseProducer,
	}, nil
}

// Publish publishes a comment event
func (p *CommentProducer) Publish(ctx context.Context, event interface{}) error {
	commentEvent, ok := event.(types.CommentEvent)
	if !ok {
		return fmt.Errorf("invalid event type: expected CommentEvent")
	}

	// Ensure event has an ID and timestamp
	commentEvent.ID = util.GenerateEventID(commentEvent.ID)
	commentEvent.CreatedAt = util.GenerateEventTime(commentEvent.CreatedAt)

	// Create properties for the message
	properties := map[string]string{
		"event_type": string(commentEvent.Type),
		"user_id":    commentEvent.UserID.String(),
		"video_id":   commentEvent.VideoID.String(),
		"comment_id": commentEvent.CommentID.String(),
	}

	if commentEvent.ParentID != uuid.Nil {
		properties["parent_id"] = commentEvent.ParentID.String()
	}

	// Create producer message
	msg, err := util.CreateProducerMessage(
		commentEvent,
		commentEvent.CommentID.String(),
		properties,
		commentEvent.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create producer message: %w", err)
	}

	// Send the message
	if _, err := p.producer.Send(ctx, msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	p.logger.LogInfo("Published comment event", map[string]interface{}{
		"event_id": commentEvent.ID,
		"event_type": string(commentEvent.Type),
		"user_id": commentEvent.UserID,
		"video_id": commentEvent.VideoID,
		"comment_id": commentEvent.CommentID,
	})

	return nil
}

// PublishCommentCreatedEvent publishes a comment creation event
func (p *CommentProducer) PublishCommentCreatedEvent(
	ctx context.Context,
	userID, videoID, commentID uuid.UUID,
	content string,
) error {
	event := types.CommentEvent{
		BaseEvent: types.BaseEvent{
			Type:      types.CommentCreated,
			CreatedAt: time.Now(),
		},
		CommentID: commentID,
		UserID:    userID,
		VideoID:   videoID,
		Content:   content,
	}

	return p.Publish(ctx, event)
}

// PublishCommentReplyEvent publishes a comment reply event
func (p *CommentProducer) PublishCommentReplyEvent(
	ctx context.Context,
	userID, videoID, commentID, parentID uuid.UUID,
	content string,
) error {
	event := types.CommentEvent{
		BaseEvent: types.BaseEvent{
			Type:      types.CommentReplied,
			CreatedAt: time.Now(),
		},
		CommentID: commentID,
		UserID:    userID,
		VideoID:   videoID,
		ParentID:  parentID,
		Content:   content,
	}

	return p.Publish(ctx, event)
}

// PublishMentionEvent publishes a mention event
func (p *CommentProducer) PublishMentionEvent(
	ctx context.Context,
	userID, videoID, commentID uuid.UUID,
	content string,
	mentionedIDs []uuid.UUID,
) error {
	event := types.CommentEvent{
		BaseEvent: types.BaseEvent{
			Type:      types.CommentMention,
			CreatedAt: time.Now(),
		},
		CommentID: commentID,
		UserID:    userID,
		VideoID:   videoID,
		Content:   content,
		Metadata: map[string]interface{}{
			"mentionedIds": mentionedIDs,
		},
	}

	return p.Publish(ctx, event)
} 