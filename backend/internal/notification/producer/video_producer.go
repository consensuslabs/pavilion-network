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

// VideoProducer is responsible for producing video events
type VideoProducer struct {
	*BaseProducer
}

// NewVideoProducer creates a new video producer
func NewVideoProducer(
	client pulsar.Client,
	logger logger.Logger,
	repository types.NotificationRepository,
	config *types.ServiceConfig,
) (*VideoProducer, error) {
	baseProducer, err := NewBaseProducer(
		client,
		config.Topics.VideoEvents,
		logger,
		repository,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create base producer: %w", err)
	}

	return &VideoProducer{
		BaseProducer: baseProducer,
	}, nil
}

// Publish publishes a video event
func (p *VideoProducer) Publish(ctx context.Context, event interface{}) error {
	videoEvent, ok := event.(types.VideoEvent)
	if !ok {
		return fmt.Errorf("invalid event type: expected VideoEvent")
	}

	// Ensure event has an ID and timestamp
	videoEvent.ID = util.GenerateEventID(videoEvent.ID)
	videoEvent.CreatedAt = util.GenerateEventTime(videoEvent.CreatedAt)

	// Create properties for the message
	properties := map[string]string{
		"event_type": string(videoEvent.Type),
		"user_id":    videoEvent.UserID.String(),
		"video_id":   videoEvent.VideoID.String(),
	}

	// Create producer message
	msg, err := util.CreateProducerMessage(
		videoEvent,
		videoEvent.VideoID.String(),
		properties,
		videoEvent.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create producer message: %w", err)
	}

	// Send the message
	if _, err := p.producer.Send(ctx, msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	p.logger.LogInfo("Published video event", map[string]interface{}{
		"event_id": videoEvent.ID,
		"event_type": string(videoEvent.Type),
		"user_id": videoEvent.UserID,
		"video_id": videoEvent.VideoID,
	})

	return nil
}

// PublishVideoUploadEvent publishes a video upload event
func (p *VideoProducer) PublishVideoUploadEvent(
	ctx context.Context,
	userID, videoID uuid.UUID,
	title string,
) error {
	event := types.VideoEvent{
		BaseEvent: types.BaseEvent{
			Type:      types.VideoUploaded,
			CreatedAt: time.Now(),
		},
		VideoID: videoID,
		UserID:  userID,
		Title:   title,
	}

	return p.Publish(ctx, event)
}

// PublishVideoLikeEvent publishes a video like event
func (p *VideoProducer) PublishVideoLikeEvent(
	ctx context.Context,
	userID, videoID uuid.UUID,
) error {
	event := types.VideoEvent{
		BaseEvent: types.BaseEvent{
			Type:      types.VideoLiked,
			CreatedAt: time.Now(),
		},
		VideoID: videoID,
		UserID:  userID,
	}

	return p.Publish(ctx, event)
}

// PublishVideoUnlikeEvent publishes a video unlike event
func (p *VideoProducer) PublishVideoUnlikeEvent(
	ctx context.Context,
	userID, videoID uuid.UUID,
) error {
	event := types.VideoEvent{
		BaseEvent: types.BaseEvent{
			Type:      types.VideoUnliked,
			CreatedAt: time.Now(),
		},
		VideoID: videoID,
		UserID:  userID,
	}

	return p.Publish(ctx, event)
} 