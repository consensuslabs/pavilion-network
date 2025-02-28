package scylladb

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"github.com/consensuslabs/pavilion-network/backend/internal/comment"
	"github.com/consensuslabs/pavilion-network/backend/internal/video"
)

// CommentRepository implements the comment.Repository interface using ScyllaDB
type CommentRepository struct {
	session *gocql.Session
	logger  video.Logger
}

// NewCommentRepository creates a new ScyllaDB repository for comments
func NewCommentRepository(session *gocql.Session, logger video.Logger) comment.Repository {
	return &CommentRepository{
		session: session,
		logger:  logger,
	}
}

// GetByID retrieves a comment by its ID
func (r *CommentRepository) GetByID(ctx context.Context, id uuid.UUID) (*comment.Comment, error) {
	query := `
		SELECT id, video_id, user_id, content, created_at, updated_at, 
			   deleted_at, parent_id, likes, dislikes, status
		FROM comments
		WHERE id = ?
	`

	var c comment.Comment
	var status string
	var parentID *uuid.UUID

	err := r.session.Query(query, id).WithContext(ctx).Scan(
		&c.ID, &c.VideoID, &c.UserID, &c.Content,
		&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
		&parentID, &c.Likes, &c.Dislikes, &status,
	)

	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, nil
		}
		r.logger.LogError("Error getting comment by ID", map[string]interface{}{
			"error":     err.Error(),
			"commentID": id,
		})
		return nil, err
	}

	c.Status = comment.Status(status)
	c.ParentID = parentID

	return &c, nil
}

// GetByVideoID retrieves comments for a video with pagination
func (r *CommentRepository) GetByVideoID(ctx context.Context, options comment.CommentFilterOptions) (comment.PaginatedComments, error) {
	// Initialize result
	result := comment.PaginatedComments{
		Comments:    []comment.Comment{},
		CurrentPage: options.Page,
		TotalCount:  0,
	}

	// Calculate offset
	offset := (options.Page - 1) * options.Limit

	// Query to get comments by video ID
	query := `
		SELECT c.id, c.video_id, c.user_id, c.content, c.created_at, c.updated_at, 
			   c.deleted_at, c.parent_id, c.likes, c.dislikes, c.status
		FROM comments c
		JOIN comments_by_video cv ON c.id = cv.comment_id
		WHERE cv.video_id = ? AND c.parent_id IS NULL AND c.deleted_at IS NULL
		LIMIT ?
	`

	// Execute query
	iter := r.session.Query(query, options.VideoID, options.Limit).WithContext(ctx).PageSize(options.Limit).PageState(nil).Iter()

	// Skip to the desired page
	for i := 0; i < offset && iter.Scanner().Next(); i++ {
		// Skip records
	}

	// Process results
	for iter.Scanner().Next() {
		var c comment.Comment
		var status string
		var parentID *uuid.UUID

		err := iter.Scanner().Scan(
			&c.ID, &c.VideoID, &c.UserID, &c.Content,
			&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
			&parentID, &c.Likes, &c.Dislikes, &status,
		)
		if err != nil {
			r.logger.LogError("Error scanning comment", map[string]interface{}{"error": err.Error()})
			return result, err
		}

		c.Status = comment.Status(status)
		c.ParentID = parentID

		result.Comments = append(result.Comments, c)
	}

	if err := iter.Close(); err != nil {
		r.logger.LogError("Error closing iterator", map[string]interface{}{"error": err.Error()})
		return result, err
	}

	// Get total count
	count, err := r.Count(ctx, options.VideoID)
	if err != nil {
		return result, err
	}

	// Calculate pagination info
	result.TotalCount = count
	result.TotalPages = int(math.Ceil(float64(count) / float64(options.Limit)))
	result.HasNextPage = options.Page < result.TotalPages
	result.HasPrevPage = options.Page > 1

	return result, nil
}

// GetReplies retrieves replies for a comment with pagination
func (r *CommentRepository) GetReplies(ctx context.Context, options comment.CommentFilterOptions) (comment.PaginatedComments, error) {
	// Initialize result
	result := comment.PaginatedComments{
		Comments:    []comment.Comment{},
		CurrentPage: options.Page,
		TotalCount:  0,
	}

	if options.ParentID == nil {
		return result, fmt.Errorf("parent comment ID is required")
	}

	// Calculate offset
	offset := (options.Page - 1) * options.Limit

	// Query to get replies
	query := `
		SELECT c.id, c.video_id, c.user_id, c.content, c.created_at, c.updated_at, 
			   c.deleted_at, c.parent_id, c.likes, c.dislikes, c.status
		FROM comments c
		JOIN replies r ON c.id = r.comment_id
		WHERE r.parent_id = ? AND c.deleted_at IS NULL
		LIMIT ?
	`

	// Execute query
	iter := r.session.Query(query, options.ParentID, options.Limit).WithContext(ctx).PageSize(options.Limit).PageState(nil).Iter()

	// Skip to the desired page
	for i := 0; i < offset && iter.Scanner().Next(); i++ {
		// Skip records
	}

	// Process results
	for iter.Scanner().Next() {
		var c comment.Comment
		var status string
		var parentID *uuid.UUID

		err := iter.Scanner().Scan(
			&c.ID, &c.VideoID, &c.UserID, &c.Content,
			&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
			&parentID, &c.Likes, &c.Dislikes, &status,
		)
		if err != nil {
			r.logger.LogError("Error scanning reply", map[string]interface{}{"error": err.Error()})
			return result, err
		}

		c.Status = comment.Status(status)
		c.ParentID = parentID

		result.Comments = append(result.Comments, c)
	}

	if err := iter.Close(); err != nil {
		r.logger.LogError("Error closing iterator", map[string]interface{}{"error": err.Error()})
		return result, err
	}

	// Count total replies
	countQuery := `
		SELECT COUNT(*)
		FROM replies
		WHERE parent_id = ?
	`

	var count int
	if err := r.session.Query(countQuery, options.ParentID).WithContext(ctx).Scan(&count); err != nil {
		r.logger.LogError("Error counting replies", map[string]interface{}{
			"error":     err.Error(),
			"commentID": *options.ParentID,
		})
		return result, err
	}

	// Calculate pagination info
	result.TotalCount = count
	result.TotalPages = int(math.Ceil(float64(count) / float64(options.Limit)))
	result.HasNextPage = options.Page < result.TotalPages
	result.HasPrevPage = options.Page > 1

	return result, nil
}

// Create creates a new comment
func (r *CommentRepository) Create(ctx context.Context, c *comment.Comment) error {
	// Set default values if not provided
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = c.CreatedAt
	}
	if c.Status == "" {
		c.Status = comment.StatusActive
	}

	// Create batch to insert comment and update indexes
	batch := r.session.NewBatch(gocql.LoggedBatch)

	// Insert comment
	commentQuery := `
		INSERT INTO comments (
			id, video_id, user_id, content, created_at, updated_at, 
			deleted_at, parent_id, likes, dislikes, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	batch.Query(commentQuery,
		c.ID, c.VideoID, c.UserID, c.Content,
		c.CreatedAt, c.UpdatedAt, c.DeletedAt,
		c.ParentID, c.Likes, c.Dislikes, string(c.Status),
	)

	// Update comment_by_video index
	videoIndexQuery := `
		INSERT INTO comments_by_video (video_id, comment_id, created_at)
		VALUES (?, ?, ?)
	`
	batch.Query(videoIndexQuery, c.VideoID, c.ID, c.CreatedAt)

	// Update replies index if this is a reply
	if c.ParentID != nil {
		replyIndexQuery := `
			INSERT INTO replies (parent_id, comment_id, created_at)
			VALUES (?, ?, ?)
		`
		batch.Query(replyIndexQuery, c.ParentID, c.ID, c.CreatedAt)
	}

	// Execute batch
	if err := r.session.ExecuteBatch(batch); err != nil {
		r.logger.LogError("Error creating comment", map[string]interface{}{
			"error":     err.Error(),
			"commentID": c.ID,
		})
		return err
	}

	return nil
}

// Update updates a comment's content
func (r *CommentRepository) Update(ctx context.Context, id uuid.UUID, content string) error {
	now := time.Now().UTC()

	query := `
		UPDATE comments
		SET content = ?, updated_at = ?
		WHERE id = ?
	`

	if err := r.session.Query(query, content, now, id).WithContext(ctx).Exec(); err != nil {
		r.logger.LogError("Error updating comment", map[string]interface{}{
			"error":     err.Error(),
			"commentID": id,
		})
		return err
	}

	return nil
}

// Delete soft-deletes a comment
func (r *CommentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()

	query := `
		UPDATE comments
		SET deleted_at = ?, status = ?
		WHERE id = ?
	`

	if err := r.session.Query(query, now, string(comment.StatusHidden), id).WithContext(ctx).Exec(); err != nil {
		r.logger.LogError("Error soft deleting comment", map[string]interface{}{
			"error":     err.Error(),
			"commentID": id,
		})
		return err
	}

	return nil
}

// Count gets total number of comments for a video
func (r *CommentRepository) Count(ctx context.Context, videoID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM comments_by_video
		WHERE video_id = ?
	`

	var count int
	if err := r.session.Query(query, videoID).WithContext(ctx).Scan(&count); err != nil {
		r.logger.LogError("Error counting comments", map[string]interface{}{
			"error":   err.Error(),
			"videoID": videoID,
		})
		return 0, err
	}

	return count, nil
}

// GetReactions retrieves reactions for a comment with pagination
func (r *CommentRepository) GetReactions(ctx context.Context, options comment.ReactionFilterOptions) ([]comment.Reaction, int, error) {
	// Calculate offset
	offset := (options.Page - 1) * options.Limit

	// Build query
	query := `
		SELECT comment_id, user_id, type, created_at, updated_at
		FROM reactions
		WHERE comment_id = ?
	`
	args := []interface{}{options.CommentID}

	// Add reaction type filter if provided
	if options.Type != "" {
		query += " AND type = ?"
		args = append(args, string(options.Type))
	}

	query += " LIMIT ?"
	args = append(args, options.Limit)

	// Execute query
	iter := r.session.Query(query, args...).WithContext(ctx).PageSize(options.Limit).PageState(nil).Iter()

	// Skip to the desired page
	for i := 0; i < offset && iter.Scanner().Next(); i++ {
		// Skip records
	}

	// Process results
	var reactions []comment.Reaction

	for iter.Scanner().Next() {
		var reaction comment.Reaction
		var typeStr string

		err := iter.Scanner().Scan(
			&reaction.CommentID, &reaction.UserID, &typeStr,
			&reaction.CreatedAt, &reaction.UpdatedAt,
		)
		if err != nil {
			r.logger.LogError("Error scanning reaction", map[string]interface{}{"error": err.Error()})
			return nil, 0, err
		}

		reaction.Type = comment.Type(typeStr)
		reactions = append(reactions, reaction)
	}

	if err := iter.Close(); err != nil {
		r.logger.LogError("Error closing iterator", map[string]interface{}{"error": err.Error()})
		return nil, 0, err
	}

	// Count total reactions
	var countQuery string
	countArgs := []interface{}{options.CommentID}

	if options.Type != "" {
		countQuery = `
			SELECT COUNT(*)
			FROM reactions
			WHERE comment_id = ? AND type = ?
		`
		countArgs = append(countArgs, string(options.Type))
	} else {
		countQuery = `
			SELECT COUNT(*)
			FROM reactions
			WHERE comment_id = ?
		`
	}

	var count int
	if err := r.session.Query(countQuery, countArgs...).WithContext(ctx).Scan(&count); err != nil {
		r.logger.LogError("Error counting reactions", map[string]interface{}{
			"error":     err.Error(),
			"commentID": options.CommentID,
		})
		return reactions, 0, err
	}

	return reactions, count, nil
}

// GetReactionByUser retrieves a user's reaction to a comment
func (r *CommentRepository) GetReactionByUser(ctx context.Context, commentID, userID uuid.UUID) (*comment.Reaction, error) {
	query := `
		SELECT comment_id, user_id, type, created_at, updated_at
		FROM reactions
		WHERE comment_id = ? AND user_id = ?
	`

	var reaction comment.Reaction
	var typeStr string

	err := r.session.Query(query, commentID, userID).WithContext(ctx).Scan(
		&reaction.CommentID, &reaction.UserID, &typeStr,
		&reaction.CreatedAt, &reaction.UpdatedAt,
	)

	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, nil
		}
		r.logger.LogError("Error getting reaction", map[string]interface{}{
			"error":     err.Error(),
			"commentID": commentID,
			"userID":    userID,
		})
		return nil, err
	}

	reaction.Type = comment.Type(typeStr)
	return &reaction, nil
}

// CreateOrUpdateReaction creates or updates a reaction
func (r *CommentRepository) CreateOrUpdateReaction(ctx context.Context, reaction *comment.Reaction) error {
	// Get existing reaction if any
	existingReaction, err := r.GetReactionByUser(ctx, reaction.CommentID, reaction.UserID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	batch := r.session.NewBatch(gocql.LoggedBatch)

	// Set default values
	if reaction.CreatedAt.IsZero() {
		reaction.CreatedAt = now
	}
	reaction.UpdatedAt = now

	// Update reactions table
	reactionQuery := `
		INSERT INTO reactions (comment_id, user_id, type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`
	batch.Query(reactionQuery,
		reaction.CommentID, reaction.UserID, string(reaction.Type),
		reaction.CreatedAt, reaction.UpdatedAt,
	)

	// Handle comment likes/dislikes counters
	if existingReaction != nil && existingReaction.Type != reaction.Type {
		// If changing reaction type, decrement the old counter
		if existingReaction.Type == comment.TypeLike {
			batch.Query("UPDATE comments SET likes = likes - 1 WHERE id = ?", reaction.CommentID)
		} else if existingReaction.Type == comment.TypeDislike {
			batch.Query("UPDATE comments SET dislikes = dislikes - 1 WHERE id = ?", reaction.CommentID)
		}
	}

	// Increment the appropriate counter
	if reaction.Type == comment.TypeLike {
		// Only increment if it's a new reaction or changed type
		if existingReaction == nil || existingReaction.Type != reaction.Type {
			batch.Query("UPDATE comments SET likes = likes + 1 WHERE id = ?", reaction.CommentID)
		}
	} else if reaction.Type == comment.TypeDislike {
		// Only increment if it's a new reaction or changed type
		if existingReaction == nil || existingReaction.Type != reaction.Type {
			batch.Query("UPDATE comments SET dislikes = dislikes + 1 WHERE id = ?", reaction.CommentID)
		}
	}

	// Execute batch
	if err := r.session.ExecuteBatch(batch); err != nil {
		r.logger.LogError("Error creating/updating reaction", map[string]interface{}{
			"error":     err.Error(),
			"commentID": reaction.CommentID,
			"userID":    reaction.UserID,
		})
		return err
	}

	return nil
}

// DeleteReaction removes a reaction
func (r *CommentRepository) DeleteReaction(ctx context.Context, commentID, userID uuid.UUID) error {
	// Get existing reaction to determine type for counter updates
	existingReaction, err := r.GetReactionByUser(ctx, commentID, userID)
	if err != nil {
		return err
	}

	if existingReaction == nil {
		// No reaction to delete
		return nil
	}

	batch := r.session.NewBatch(gocql.LoggedBatch)

	// Delete from reactions table
	deleteQuery := `
		DELETE FROM reactions
		WHERE comment_id = ? AND user_id = ?
	`
	batch.Query(deleteQuery, commentID, userID)

	// Update the appropriate counter
	if existingReaction.Type == comment.TypeLike {
		batch.Query("UPDATE comments SET likes = likes - 1 WHERE id = ?", commentID)
	} else if existingReaction.Type == comment.TypeDislike {
		batch.Query("UPDATE comments SET dislikes = dislikes - 1 WHERE id = ?", commentID)
	}

	// Execute batch
	if err := r.session.ExecuteBatch(batch); err != nil {
		r.logger.LogError("Error deleting reaction", map[string]interface{}{
			"error":     err.Error(),
			"commentID": commentID,
			"userID":    userID,
		})
		return err
	}

	return nil
}

// GetReactionCounts gets the count of likes and dislikes for a comment
func (r *CommentRepository) GetReactionCounts(ctx context.Context, commentID uuid.UUID) (int, int, error) {
	// We can either query the comments table directly for the cached counters
	// or count from the reactions table for accuracy

	// Using the cached counters for performance
	query := `
		SELECT likes, dislikes
		FROM comments
		WHERE id = ?
	`

	var likes, dislikes int
	if err := r.session.Query(query, commentID).WithContext(ctx).Scan(&likes, &dislikes); err != nil {
		r.logger.LogError("Error getting reaction counts", map[string]interface{}{
			"error":     err.Error(),
			"commentID": commentID,
		})
		return 0, 0, err
	}

	return likes, dislikes, nil
}
