package comment

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Common errors
var (
	ErrCommentNotFound  = errors.New("comment not found")
	ErrInvalidComment   = errors.New("invalid comment")
	ErrInvalidReaction  = errors.New("invalid reaction")
	ErrPermissionDenied = errors.New("permission denied")
	ErrInvalidPage      = errors.New("invalid page number")
	ErrInvalidLimit     = errors.New("invalid limit number")
)

// serviceImpl implements the Service interface
type serviceImpl struct {
	repo Repository
}

// NewService creates a new comment service
func NewService(repo Repository) Service {
	return &serviceImpl{
		repo: repo,
	}
}

// GetCommentByID retrieves a comment by its ID
func (s *serviceImpl) GetCommentByID(ctx context.Context, id uuid.UUID) (*Comment, error) {
	return s.repo.GetByID(ctx, id)
}

// GetCommentsByVideoID retrieves comments for a video with pagination
func (s *serviceImpl) GetCommentsByVideoID(ctx context.Context, options CommentFilterOptions) (PaginatedComments, error) {
	// Validate options
	if options.Page < 1 {
		options.Page = 1
	}
	if options.Limit < 1 {
		options.Limit = 10
	} else if options.Limit > 100 {
		options.Limit = 100
	}

	// Ensure we're only getting top-level comments
	options.ParentID = nil

	return s.repo.GetByVideoID(ctx, options)
}

// GetRepliesByCommentID retrieves replies for a comment with pagination
func (s *serviceImpl) GetRepliesByCommentID(ctx context.Context, options CommentFilterOptions) (PaginatedComments, error) {
	// Validate options
	if options.Page < 1 {
		options.Page = 1
	}
	if options.Limit < 1 {
		options.Limit = 10
	} else if options.Limit > 100 {
		options.Limit = 100
	}

	// Validate commentID
	if options.ParentID == nil {
		return PaginatedComments{}, errors.New("parent comment ID is required")
	}

	return s.repo.GetReplies(ctx, options)
}

// CreateComment creates a new comment
func (s *serviceImpl) CreateComment(ctx context.Context, comment *Comment) error {
	// Validate comment
	if comment.VideoID == uuid.Nil {
		return errors.New("video ID is required")
	}
	if comment.UserID == uuid.Nil {
		return errors.New("user ID is required")
	}
	if comment.Content == "" {
		return errors.New("content is required")
	}

	// Set default values
	if comment.ID == uuid.Nil {
		comment.ID = uuid.New()
	}
	now := time.Now().UTC()
	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = now
	}
	if comment.UpdatedAt.IsZero() {
		comment.UpdatedAt = now
	}
	if comment.Status == "" {
		comment.Status = StatusActive
	}

	// If this is a reply, validate parent comment exists
	if comment.ParentID != nil {
		parent, err := s.repo.GetByID(ctx, *comment.ParentID)
		if err != nil {
			return err
		}
		if parent == nil {
			return ErrCommentNotFound
		}

		// Ensure parent is not itself a reply
		if parent.ParentID != nil {
			return errors.New("cannot reply to a reply")
		}
	}

	return s.repo.Create(ctx, comment)
}

// UpdateComment updates a comment's content
func (s *serviceImpl) UpdateComment(ctx context.Context, id uuid.UUID, content string) error {
	// Validate content
	if content == "" {
		return errors.New("content is required")
	}

	// Get comment to update
	comment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if comment == nil {
		return ErrCommentNotFound
	}

	return s.repo.Update(ctx, id, content)
}

// DeleteComment soft-deletes a comment
func (s *serviceImpl) DeleteComment(ctx context.Context, id uuid.UUID) error {
	// Check if comment exists
	comment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if comment == nil {
		return ErrCommentNotFound
	}

	return s.repo.Delete(ctx, id)
}

// GetUserReaction gets a user's reaction to a comment
func (s *serviceImpl) GetUserReaction(ctx context.Context, commentID, userID uuid.UUID) (*Reaction, error) {
	if commentID == uuid.Nil || userID == uuid.Nil {
		return nil, errors.New("comment ID and user ID are required")
	}

	return s.repo.GetReactionByUser(ctx, commentID, userID)
}

// AddReaction adds or updates a user's reaction to a comment
func (s *serviceImpl) AddReaction(ctx context.Context, reaction *Reaction) error {
	// Validate reaction
	if reaction.CommentID == uuid.Nil {
		return errors.New("comment ID is required")
	}
	if reaction.UserID == uuid.Nil {
		return errors.New("user ID is required")
	}
	if reaction.Type != TypeLike && reaction.Type != TypeDislike {
		return errors.New("invalid reaction type")
	}

	// Check if comment exists
	comment, err := s.repo.GetByID(ctx, reaction.CommentID)
	if err != nil {
		return err
	}
	if comment == nil {
		return ErrCommentNotFound
	}

	// Set timestamps
	now := time.Now().UTC()
	if reaction.CreatedAt.IsZero() {
		reaction.CreatedAt = now
	}
	reaction.UpdatedAt = now

	return s.repo.CreateOrUpdateReaction(ctx, reaction)
}

// RemoveReaction removes a user's reaction from a comment
func (s *serviceImpl) RemoveReaction(ctx context.Context, commentID, userID uuid.UUID) error {
	if commentID == uuid.Nil || userID == uuid.Nil {
		return errors.New("comment ID and user ID are required")
	}

	// Check if comment exists
	comment, err := s.repo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment == nil {
		return ErrCommentNotFound
	}

	return s.repo.DeleteReaction(ctx, commentID, userID)
}
