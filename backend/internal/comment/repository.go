package comment

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for comment data access
type Repository interface {
	// Comment operations
	GetByID(ctx context.Context, id uuid.UUID) (*Comment, error)
	GetByVideoID(ctx context.Context, options CommentFilterOptions) (PaginatedComments, error)
	GetReplies(ctx context.Context, options CommentFilterOptions) (PaginatedComments, error)
	Create(ctx context.Context, comment *Comment) error
	Update(ctx context.Context, id uuid.UUID, content string) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context, videoID uuid.UUID) (int, error)

	// Reaction operations
	GetReactions(ctx context.Context, options ReactionFilterOptions) ([]Reaction, int, error)
	GetReactionByUser(ctx context.Context, commentID, userID uuid.UUID) (*Reaction, error)
	CreateOrUpdateReaction(ctx context.Context, reaction *Reaction) error
	DeleteReaction(ctx context.Context, commentID, userID uuid.UUID) error
	GetReactionCounts(ctx context.Context, commentID uuid.UUID) (int, int, error)
}

// Service defines the business logic interface for comment operations
type Service interface {
	// Comment operations
	GetCommentByID(ctx context.Context, id uuid.UUID) (*Comment, error)
	GetCommentsByVideoID(ctx context.Context, options CommentFilterOptions) (PaginatedComments, error)
	GetRepliesByCommentID(ctx context.Context, options CommentFilterOptions) (PaginatedComments, error)
	CreateComment(ctx context.Context, comment *Comment) error
	UpdateComment(ctx context.Context, id uuid.UUID, content string) error
	DeleteComment(ctx context.Context, id uuid.UUID) error

	// Reaction operations
	GetUserReaction(ctx context.Context, commentID, userID uuid.UUID) (*Reaction, error)
	AddReaction(ctx context.Context, reaction *Reaction) error
	RemoveReaction(ctx context.Context, commentID, userID uuid.UUID) error
	
	// Set notification service
	SetNotificationService(notificationService interface{})
}
