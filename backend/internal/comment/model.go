package comment

import (
	"time"

	"github.com/google/uuid"
)

// Comment represents a comment on a video
// @Description A comment on a video with metadata and reaction counts
type Comment struct {
	ID        uuid.UUID  `json:"id" example:"123e4567-e89b-12d3-a456-426614174000" swaggertype:"string" format:"uuid"`
	VideoID   uuid.UUID  `json:"video_id" example:"123e4567-e89b-12d3-a456-426614174001" swaggertype:"string" format:"uuid"`
	UserID    uuid.UUID  `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174002" swaggertype:"string" format:"uuid"`
	Content   string     `json:"content" example:"This is a great video!"`
	CreatedAt time.Time  `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt time.Time  `json:"updated_at" example:"2023-01-01T12:00:00Z"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" example:"2023-01-02T12:00:00Z"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty" example:"123e4567-e89b-12d3-a456-426614174003" swaggertype:"string" format:"uuid"`
	Likes     int        `json:"likes" example:"5"`
	Dislikes  int        `json:"dislikes" example:"1"`
	Status    Status     `json:"status" example:"ACTIVE" enums:"ACTIVE,FLAGGED,HIDDEN"`
}

// Reaction represents a user's reaction to a comment
// @Description A user's reaction (like or dislike) to a comment
type Reaction struct {
	CommentID uuid.UUID `json:"comment_id" example:"123e4567-e89b-12d3-a456-426614174000" swaggertype:"string" format:"uuid"`
	UserID    uuid.UUID `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174002" swaggertype:"string" format:"uuid"`
	Type      Type      `json:"type" example:"LIKE" enums:"LIKE,DISLIKE"`
	CreatedAt time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`
}

// Status represents the possible statuses of a comment
// @Description Status of a comment (ACTIVE, FLAGGED, or HIDDEN)
type Status string

const (
	// StatusActive represents a normal, visible comment
	StatusActive Status = "ACTIVE"

	// StatusFlagged represents a comment flagged for review
	StatusFlagged Status = "FLAGGED"

	// StatusHidden represents a comment hidden from general view
	StatusHidden Status = "HIDDEN"
)

// Type represents the possible reaction types
// @Description Type of reaction (LIKE or DISLIKE)
type Type string

const (
	// TypeLike represents a positive reaction
	TypeLike Type = "LIKE"

	// TypeDislike represents a negative reaction
	TypeDislike Type = "DISLIKE"
)

// CommentFilterOptions provides filtering options for comment queries
// @Description Options for filtering and paginating comments
type CommentFilterOptions struct {
	VideoID   uuid.UUID  `json:"video_id" example:"123e4567-e89b-12d3-a456-426614174000" swaggertype:"string" format:"uuid"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty" example:"123e4567-e89b-12d3-a456-426614174003" swaggertype:"string" format:"uuid"`
	Page      int        `json:"page" example:"1"`
	Limit     int        `json:"limit" example:"20"`
	SortBy    string     `json:"sort_by" example:"created_at"`
	SortOrder string     `json:"sort_order" example:"desc"`
}

// ReactionFilterOptions provides filtering options for reaction queries
// @Description Options for filtering and paginating reactions
type ReactionFilterOptions struct {
	CommentID uuid.UUID `json:"comment_id" example:"123e4567-e89b-12d3-a456-426614174000" swaggertype:"string" format:"uuid"`
	Type      Type      `json:"type,omitempty" example:"LIKE" enums:"LIKE,DISLIKE"`
	Page      int       `json:"page" example:"1"`
	Limit     int       `json:"limit" example:"20"`
}

// PaginatedComments represents a paginated list of comments
// @Description A paginated list of comments with metadata about the pagination
type PaginatedComments struct {
	Comments    []Comment `json:"comments"`
	TotalCount  int       `json:"total_count" example:"42"`
	CurrentPage int       `json:"current_page" example:"1"`
	TotalPages  int       `json:"total_pages" example:"3"`
	HasNextPage bool      `json:"has_next_page" example:"true"`
	HasPrevPage bool      `json:"has_prev_page" example:"false"`
}

// CreateCommentRequest represents the request body for creating a new comment
// For top-level comments, omit parent_id or set it to null (not "null" and not an empty string)
// For replies, set parent_id to the UUID of the parent comment
type CreateCommentRequest struct {
	Content  string     `json:"content" binding:"required" example:"This is a comment"`
	ParentID *uuid.UUID `json:"parent_id" example:null`
}

// UpdateCommentRequest represents the request body for updating a comment
type UpdateCommentRequest struct {
	Content string `json:"content" binding:"required" example:"Updated comment content"`
}

// ReactionRequest represents the request body for adding a reaction to a comment
type ReactionRequest struct {
	Type string `json:"type" binding:"required" example:"like"`
}

// NewComment creates a new comment with default values
func NewComment(videoID, userID uuid.UUID, content string, parentID *uuid.UUID) *Comment {
	now := time.Now().UTC()
	return &Comment{
		ID:        uuid.New(),
		VideoID:   videoID,
		UserID:    userID,
		Content:   content,
		ParentID:  parentID,
		CreatedAt: now,
		UpdatedAt: now,
		Likes:     0,
		Dislikes:  0,
		Status:    StatusActive,
	}
}

// NewReaction creates a new reaction with default values
func NewReaction(commentID, userID uuid.UUID, reactionType Type) *Reaction {
	now := time.Now().UTC()
	return &Reaction{
		CommentID: commentID,
		UserID:    userID,
		Type:      reactionType,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
