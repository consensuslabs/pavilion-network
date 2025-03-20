package comment

import (
	"context"
	"errors"
	"fmt"
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
	repo               Repository
	notificationService interface{}
}

// NewService creates a new comment service
func NewService(repo Repository) Service {
	return &serviceImpl{
		repo: repo,
	}
}

// SetNotificationService sets the notification service
func (s *serviceImpl) SetNotificationService(notificationService interface{}) {
	s.notificationService = notificationService
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
	fmt.Printf("DEBUG SERVICE: Starting CreateComment for videoID %s\n", comment.VideoID.String())

	// Validate comment
	if comment.VideoID == uuid.Nil {
		fmt.Printf("DEBUG SERVICE: Invalid videoID - nil UUID\n")
		return errors.New("video ID is required")
	}
	if comment.UserID == uuid.Nil {
		fmt.Printf("DEBUG SERVICE: Invalid userID - nil UUID\n")
		return errors.New("user ID is required")
	}
	if comment.Content == "" {
		fmt.Printf("DEBUG SERVICE: Empty content\n")
		return errors.New("content is required")
	}

	// Set default values
	if comment.ID == uuid.Nil {
		comment.ID = uuid.New()
		fmt.Printf("DEBUG SERVICE: Generated new comment ID: %s\n", comment.ID.String())
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
		fmt.Printf("DEBUG SERVICE: This is a reply to comment %s\n", comment.ParentID.String())
		parent, err := s.repo.GetByID(ctx, *comment.ParentID)
		if err != nil {
			fmt.Printf("DEBUG SERVICE: Error validating parent comment: %v\n", err)
			return fmt.Errorf("error validating parent comment: %w", err)
		}
		if parent == nil {
			fmt.Printf("DEBUG SERVICE: Parent comment not found\n")
			return ErrCommentNotFound
		}

		// Ensure parent is not itself a reply
		if parent.ParentID != nil {
			fmt.Printf("DEBUG SERVICE: Cannot reply to a reply\n")
			return errors.New("cannot reply to a reply")
		}
	}

	fmt.Printf("DEBUG SERVICE: Calling repository.Create\n")
	// Save the comment to the repository
	err := s.repo.Create(ctx, comment)
	if err != nil {
		fmt.Printf("DEBUG SERVICE: Repository error: %v\n", err)
		return fmt.Errorf("error creating comment in repository: %w", err)
	}

	// Publish notification event if notification service is available
	if s.notificationService != nil {
		fmt.Printf("DEBUG SERVICE: Publishing comment notification event\n")
		
		// Check if this is a reply or a new comment
		if comment.ParentID != nil {
			// This is a reply
			if adapter, ok := s.notificationService.(interface {
				PublishCommentReplyEvent(ctx context.Context, userID, videoID, commentID, parentID uuid.UUID, content string) error
			}); ok {
				// Get the parent comment to find the parent author
				parentComment, err := s.repo.GetByID(ctx, *comment.ParentID)
				if err == nil && parentComment != nil {
					// Only notify if the parent comment author is different from the replier
					if parentComment.UserID != comment.UserID {
						if err := adapter.PublishCommentReplyEvent(ctx, parentComment.UserID, comment.VideoID, comment.ID, *comment.ParentID, comment.Content); err != nil {
							fmt.Printf("DEBUG SERVICE: Error publishing comment reply notification: %v\n", err)
							// Don't return error, just log it
						}
					}
				}
			}
		} else {
			// This is a new comment on a video
			if adapter, ok := s.notificationService.(interface {
				PublishCommentCreatedEvent(ctx context.Context, userID, videoID, commentID uuid.UUID, content string) error
			}); ok {
				// TODO: Get video owner ID from video service and only notify if the comment author is different from the video owner
				// For now, use the comment user ID as a placeholder
				// We'll need to fetch the video owner ID from the video service
				if err := adapter.PublishCommentCreatedEvent(ctx, comment.UserID, comment.VideoID, comment.ID, comment.Content); err != nil {
					fmt.Printf("DEBUG SERVICE: Error publishing comment created notification: %v\n", err)
					// Don't return error, just log it
				}
			}
		}
	}

	fmt.Printf("DEBUG SERVICE: Comment created successfully\n")
	return nil
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
