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
	// Convert UUID to binary for ScyllaDB
	idBytes, _ := id.MarshalBinary()

	query := `
		SELECT id, video_id, user_id, content, created_at, updated_at, 
			   deleted_at, parent_id, likes, dislikes, status
		FROM comments
		WHERE id = ?
	`

	var c comment.Comment
	var status string
	var parentIDBytes []byte
	parentIDBytes = nil // Initialize as nil to properly handle NULL values

	// Use consistency level ONE (1)
	err := r.session.Query(query, idBytes).WithContext(ctx).Consistency(1).Scan(
		&c.ID, &c.VideoID, &c.UserID, &c.Content,
		&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
		&parentIDBytes, &c.Likes, &c.Dislikes, &status,
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

	// Convert parentIDBytes to UUID pointer if not nil
	if parentIDBytes != nil {
		parentID := uuid.UUID{}
		if err := parentID.UnmarshalBinary(parentIDBytes); err != nil {
			r.logger.LogError("Error unmarshaling parent ID", map[string]interface{}{
				"error":     err.Error(),
				"commentID": id,
			})
			return nil, fmt.Errorf("failed to unmarshal parent ID: %w", err)
		}
		c.ParentID = &parentID
	} else {
		c.ParentID = nil
	}

	c.Status = comment.Status(status)

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

	// Convert UUID to binary for ScyllaDB
	videoIDBytes, err := options.VideoID.MarshalBinary()
	if err != nil {
		r.logger.LogError("Error marshaling video ID", map[string]interface{}{"error": err.Error()})
		return result, err
	}

	// Step 1: Get comment IDs from comments_by_video table
	// This table is a denormalized index that allows us to find comments by video ID
	commentIDsQuery := `
		SELECT comment_id
		FROM comments_by_video
		WHERE video_id = ?
		LIMIT ?
	`

	// Execute query to get comment IDs with consistency level ONE (1)
	query := r.session.Query(commentIDsQuery, videoIDBytes, options.Limit).WithContext(ctx)
	query = query.Consistency(1) // Use consistency level ONE (1) instead of QUORUM
	commentIDsIter := query.PageSize(options.Limit).PageState(nil).Iter()

	// Skip to the desired page
	for i := 0; i < offset && commentIDsIter.Scanner().Next(); i++ {
		// Skip records
	}

	// Collect comment IDs
	var commentIDs []uuid.UUID
	for commentIDsIter.Scanner().Next() {
		var commentIDBytes []byte
		if err := commentIDsIter.Scanner().Scan(&commentIDBytes); err != nil {
			r.logger.LogError("Error scanning comment ID", map[string]interface{}{"error": err.Error()})
			return result, err
		}
		
		// Convert binary to UUID
		var commentID uuid.UUID
		if err := commentID.UnmarshalBinary(commentIDBytes); err != nil {
			r.logger.LogError("Error unmarshaling comment ID", map[string]interface{}{"error": err.Error()})
			return result, err
		}
		
		commentIDs = append(commentIDs, commentID)
	}

	if err := commentIDsIter.Close(); err != nil {
		r.logger.LogError("Error closing comment IDs iterator", map[string]interface{}{"error": err.Error()})
		return result, err
	}

	// Step 2: Fetch full comment data for each comment ID
	for _, commentID := range commentIDs {
		comment, err := r.GetByID(ctx, commentID)
		if err != nil {
			r.logger.LogError("Error fetching comment by ID", map[string]interface{}{
				"error": err.Error(),
				"commentID": commentID,
			})
			return result, err
		}

		// Only include non-deleted comments and top-level comments (no parent)
		if comment != nil && comment.DeletedAt == nil && comment.ParentID == nil {
			result.Comments = append(result.Comments, *comment)
		}
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

	// Convert UUID to binary for ScyllaDB
	parentIDBytes, err := options.ParentID.MarshalBinary()
	if err != nil {
		r.logger.LogError("Error marshaling parent ID", map[string]interface{}{"error": err.Error()})
		return result, err
	}

	// Calculate offset
	offset := (options.Page - 1) * options.Limit

	// Step 1: Get reply comment IDs from replies table
	repliesQuery := `
		SELECT comment_id
		FROM replies
		WHERE parent_id = ?
		LIMIT ?
	`

	// Execute query to get reply IDs with consistency level ONE (1)
	query := r.session.Query(repliesQuery, parentIDBytes, options.Limit).WithContext(ctx)
	query = query.Consistency(1) // Use consistency level ONE (1) instead of QUORUM
	repliesIter := query.PageSize(options.Limit).PageState(nil).Iter()

	// Skip to the desired page
	for i := 0; i < offset && repliesIter.Scanner().Next(); i++ {
		// Skip records
	}

	// Collect reply IDs
	var replyIDs []uuid.UUID
	for repliesIter.Scanner().Next() {
		var replyIDBytes []byte
		if err := repliesIter.Scanner().Scan(&replyIDBytes); err != nil {
			r.logger.LogError("Error scanning reply ID", map[string]interface{}{"error": err.Error()})
			return result, err
		}
		
		// Convert binary to UUID
		var replyID uuid.UUID
		if err := replyID.UnmarshalBinary(replyIDBytes); err != nil {
			r.logger.LogError("Error unmarshaling reply ID", map[string]interface{}{"error": err.Error()})
			return result, err
		}
		
		replyIDs = append(replyIDs, replyID)
	}

	if err := repliesIter.Close(); err != nil {
		r.logger.LogError("Error closing replies iterator", map[string]interface{}{"error": err.Error()})
		return result, err
	}

	// Step 2: Fetch full comment data for each reply ID
	for _, replyID := range replyIDs {
		reply, err := r.GetByID(ctx, replyID)
		if err != nil {
			r.logger.LogError("Error fetching reply by ID", map[string]interface{}{
				"error": err.Error(),
				"replyID": replyID,
			})
			return result, err
		}

		// Only include non-deleted replies
		if reply != nil && reply.DeletedAt == nil {
			result.Comments = append(result.Comments, *reply)
		}
	}

	// Count total replies
	countQuery := `
		SELECT COUNT(*)
		FROM replies
		WHERE parent_id = ?
	`

	var count int
	// Use consistency level ONE (1) for count query
	if err := r.session.Query(countQuery, parentIDBytes).WithContext(ctx).Consistency(1).Scan(&count); err != nil {
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
	fmt.Printf("DEBUG REPO: Starting Create for comment ID %s, videoID %s\n", c.ID.String(), c.VideoID.String())

	// Set default values if not provided
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
		fmt.Printf("DEBUG REPO: Generated new comment ID: %s\n", c.ID.String())
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

	// Log what we're about to do
	r.logger.LogInfo("Creating comment in ScyllaDB", map[string]interface{}{
		"commentID":   c.ID.String(),
		"videoID":     c.VideoID.String(),
		"userID":      c.UserID.String(),
		"hasParentID": c.ParentID != nil,
	})
	fmt.Printf("DEBUG REPO: User ID: %s, Content length: %d\n", c.UserID.String(), len(c.Content))
	if c.ParentID != nil {
		fmt.Printf("DEBUG REPO: This is a reply to comment: %s\n", c.ParentID.String())
	} else {
		fmt.Printf("DEBUG REPO: This is a top-level comment\n")
	}

	// Convert UUIDs to byte arrays for ScyllaDB
	commentIDBytes, _ := c.ID.MarshalBinary()
	videoIDBytes, _ := c.VideoID.MarshalBinary()
	userIDBytes, _ := c.UserID.MarshalBinary()

	var parentIDBytes interface{} = nil
	if c.ParentID != nil {
		// Only marshal the parent ID if it's not nil
		bytes, _ := c.ParentID.MarshalBinary()
		parentIDBytes = bytes
		fmt.Printf("DEBUG REPO: Marshaled parent ID to bytes\n")
	} else {
		fmt.Printf("DEBUG REPO: Parent ID is nil, using nil value directly\n")
	}

	fmt.Printf("DEBUG REPO: Successfully marshaled UUIDs to binary\n")

	// Create batch to insert comment and update indexes
	batch := r.session.NewBatch(gocql.LoggedBatch)
	batch.SetConsistency(1) // Set consistency level to ONE (1)

	// Insert comment
	commentQuery := `
		INSERT INTO comments (
			id, video_id, user_id, content, created_at, updated_at, 
			deleted_at, parent_id, likes, dislikes, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	fmt.Printf("DEBUG REPO: Comment query: %s\n", commentQuery)
	batch.Query(commentQuery,
		commentIDBytes, videoIDBytes, userIDBytes, c.Content,
		c.CreatedAt, c.UpdatedAt, c.DeletedAt,
		parentIDBytes, c.Likes, c.Dislikes, string(c.Status),
	)

	// Update comment_by_video index
	videoIndexQuery := `
		INSERT INTO comments_by_video (video_id, comment_id, created_at)
		VALUES (?, ?, ?)
	`
	fmt.Printf("DEBUG REPO: Video index query: %s\n", videoIndexQuery)
	batch.Query(videoIndexQuery, videoIDBytes, commentIDBytes, c.CreatedAt)

	// Update replies index if this is a reply
	if c.ParentID != nil {
		replyIndexQuery := `
			INSERT INTO replies (parent_id, comment_id, created_at)
			VALUES (?, ?, ?)
		`
		fmt.Printf("DEBUG REPO: Reply index query: %s\n", replyIndexQuery)
		batch.Query(replyIndexQuery, parentIDBytes, commentIDBytes, c.CreatedAt)
	}

	fmt.Printf("DEBUG REPO: Executing batch with %d statements\n", batch.Size())
	// Execute batch
	if err := r.session.ExecuteBatch(batch); err != nil {
		r.logger.LogError("Error creating comment", map[string]interface{}{
			"error":     err.Error(),
			"commentID": c.ID.String(),
			"videoID":   c.VideoID.String(),
			"errorType": fmt.Sprintf("%T", err),
		})
		fmt.Printf("DEBUG REPO: Batch execution failed: %v (type: %T)\n", err, err)
		return fmt.Errorf("failed to execute batch: %w", err)
	}

	r.logger.LogInfo("Comment created successfully", map[string]interface{}{
		"commentID": c.ID.String(),
	})
	fmt.Printf("DEBUG REPO: Comment created successfully\n")

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

	if err := r.session.Query(query, content, now, id).WithContext(ctx).Consistency(1).Exec(); err != nil {
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

	if err := r.session.Query(query, now, string(comment.StatusHidden), id).WithContext(ctx).Consistency(1).Exec(); err != nil {
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
	// Convert UUID to binary for ScyllaDB
	videoIDBytes, err := videoID.MarshalBinary()
	if err != nil {
		r.logger.LogError("Error marshaling video ID", map[string]interface{}{"error": err.Error()})
		return 0, err
	}

	query := `
		SELECT COUNT(*)
		FROM comments_by_video
		WHERE video_id = ?
	`

	var count int
	// Use consistency level ONE (1)
	if err := r.session.Query(query, videoIDBytes).WithContext(ctx).Consistency(1).Scan(&count); err != nil {
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

	// Execute query with consistency level ONE (1)
	iter := r.session.Query(query, args...).WithContext(ctx).Consistency(1).PageSize(options.Limit).PageState(nil).Iter()

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
	if err := r.session.Query(countQuery, countArgs...).WithContext(ctx).Consistency(1).Scan(&count); err != nil {
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

	err := r.session.Query(query, commentID, userID).WithContext(ctx).Consistency(1).Scan(
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
	batch.SetConsistency(1) // Set consistency level to ONE (1)

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
	batch.SetConsistency(1) // Set consistency level to ONE (1)

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
	if err := r.session.Query(query, commentID).WithContext(ctx).Consistency(1).Scan(&likes, &dislikes); err != nil {
		r.logger.LogError("Error getting reaction counts", map[string]interface{}{
			"error":     err.Error(),
			"commentID": commentID,
		})
		return 0, 0, err
	}

	return likes, dislikes, nil
}


