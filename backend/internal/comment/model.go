package comment

import (
	"time"

	"github.com/google/uuid"
)

// Comment represents a comment on a video
type Comment struct {
	ID        uuid.UUID  `json:"id"`
	VideoID   uuid.UUID  `json:"video_id"`
	UserID    uuid.UUID  `json:"user_id"`
	Content   string     `json:"content"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	Likes     int        `json:"likes"`
	Dislikes  int        `json:"dislikes"`
	Status    Status     `json:"status"`
}

// Reaction represents a user's reaction to a comment
type Reaction struct {
	CommentID uuid.UUID `json:"comment_id"`
	UserID    uuid.UUID `json:"user_id"`
	Type      Type      `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Status represents the possible statuses of a comment
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
type Type string

const (
	// TypeLike represents a positive reaction
	TypeLike Type = "LIKE"

	// TypeDislike represents a negative reaction
	TypeDislike Type = "DISLIKE"
)

// CommentFilterOptions provides filtering options for comment queries
type CommentFilterOptions struct {
	VideoID   uuid.UUID
	ParentID  *uuid.UUID
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}

// ReactionFilterOptions provides filtering options for reaction queries
type ReactionFilterOptions struct {
	CommentID uuid.UUID
	Type      Type
	Page      int
	Limit     int
}

// PaginatedComments represents a paginated list of comments
type PaginatedComments struct {
	Comments    []Comment `json:"comments"`
	TotalCount  int       `json:"total_count"`
	CurrentPage int       `json:"current_page"`
	TotalPages  int       `json:"total_pages"`
	HasNextPage bool      `json:"has_next_page"`
	HasPrevPage bool      `json:"has_prev_page"`
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
