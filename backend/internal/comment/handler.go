package comment

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/auth"
	httpHandler "github.com/consensuslabs/pavilion-network/backend/internal/http"
	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler defines the HTTP handler for comment operations
type Handler struct {
	service  Service
	response httpHandler.ResponseHandler
	logger   video.Logger
}

// NewHandler creates a new comment handler
func NewHandler(service Service, response httpHandler.ResponseHandler, logger video.Logger) *Handler {
	return &Handler{
		service:  service,
		response: response,
		logger:   logger,
	}
}

// RegisterRoutes registers the comment API routes
func (h *Handler) RegisterRoutes(router *gin.Engine, authService *auth.Service) {
	// Unprotected routes
	router.GET("/video/:id/comments", h.GetCommentsByVideoID)
	router.GET("/comment/:id/replies", h.GetRepliesByCommentID)

	// Protected routes
	protected := router.Group("")
	protected.Use(auth.AuthMiddleware(authService, h.response))
	{
		protected.POST("/video/:id/comment", h.CreateComment)
		protected.PUT("/comment/:id", h.UpdateComment)
		protected.DELETE("/comment/:id", h.DeleteComment)
		protected.POST("/comment/:id/reaction", h.AddReaction)
		protected.DELETE("/comment/:id/reaction", h.RemoveReaction)
	}
}

// @Summary Get comments for a video
// @Description Retrieves a paginated list of comments for a specific video
// @Tags comment
// @Accept json
// @Produce json
// @Param id path string true "Video ID (UUID)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Number of comments per page (default: 20, max: 100)"
// @Param sort query string false "Sort order (options: newest, oldest, most_liked; default: newest)"
// @Success 200 {object} http.Response{data=PaginatedComments} "Comments retrieved successfully"
// @Failure 400 {object} http.Response{error=http.Error} "Invalid video ID format"
// @Failure 500 {object} http.Response{error=http.Error} "Internal server error"
// @Router /video/{id}/comments [get]
func (h *Handler) GetCommentsByVideoID(c *gin.Context) {
	videoIDStr := c.Param("id")
	videoID, err := uuid.Parse(videoIDStr)
	if err != nil {
		h.response.ErrorResponse(c, http.StatusBadRequest, "invalid_video_id", "Invalid video ID format", err)
		return
	}

	// Parse query parameters with defaults
	page, limit, sortBy, sortOrder := getPaginationParams(c)

	options := CommentFilterOptions{
		VideoID:   videoID,
		Page:      page,
		Limit:     limit,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	comments, err := h.service.GetCommentsByVideoID(c.Request.Context(), options)
	if err != nil {
		h.response.InternalErrorResponse(c, "Failed to retrieve comments", err)
		return
	}

	h.response.SuccessResponse(c, comments, "Comments retrieved successfully")
}

// @Summary Get replies to a comment
// @Description Retrieves a paginated list of replies for a specific comment
// @Tags comment
// @Accept json
// @Produce json
// @Param id path string true "Comment ID (UUID)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Number of replies per page (default: 20, max: 100)"
// @Success 200 {object} http.Response{data=PaginatedComments} "Replies retrieved successfully"
// @Failure 400 {object} http.Response{error=http.Error} "Invalid comment ID format"
// @Failure 500 {object} http.Response{error=http.Error} "Internal server error"
// @Router /comment/{id}/replies [get]
func (h *Handler) GetRepliesByCommentID(c *gin.Context) {
	commentIDStr := c.Param("id")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		h.response.ErrorResponse(c, http.StatusBadRequest, "invalid_comment_id", "Invalid comment ID format", err)
		return
	}

	// Parse query parameters with defaults
	page, limit, _, _ := getPaginationParams(c)

	options := CommentFilterOptions{
		ParentID:  &commentID,
		Page:      page,
		Limit:     limit,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	replies, err := h.service.GetRepliesByCommentID(c.Request.Context(), options)
	if err != nil {
		h.response.InternalErrorResponse(c, "Failed to retrieve replies", err)
		return
	}

	h.response.SuccessResponse(c, replies, "Replies retrieved successfully")
}

// @Summary Create a new comment
// @Description Creates a new comment for a video
// @Tags Comment
// @Accept json
// @Produce json
// @Param id path string true "Video ID"
// @Param Authorization header string true "Bearer token"
// @Param comment body CreateCommentRequest true "Comment data"
// @Success 200 {object} response.SuccessResponse{data=Comment} "Comment created successfully"
// @Failure 400 {object} response.ErrorResponse "Invalid video ID format or invalid comment"
// @Failure 401 {object} response.ErrorResponse "User not authenticated"
// @Failure 500 {object} response.ErrorResponse "Failed to create comment"
// @Router /video/{id}/comment [post]
func (h *Handler) CreateComment(c *gin.Context) {
	fmt.Printf("DEBUG HANDLER: Starting CreateComment handler\n")

	// Try-catch equivalent to prevent panics from bringing down the server
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("DEBUG HANDLER: PANIC RECOVERED in CreateComment: %v\n", r)
			h.response.InternalErrorResponse(c, "Internal server error", fmt.Errorf("panic: %v", r))
		}
	}()

	videoIDStr := c.Param("id")
	fmt.Printf("DEBUG HANDLER: Video ID string: %s\n", videoIDStr)

	videoID, err := uuid.Parse(videoIDStr)
	if err != nil {
		fmt.Printf("DEBUG HANDLER: Failed to parse video ID: %v\n", err)
		h.response.ErrorResponse(c, http.StatusBadRequest, "invalid_video_id", "Invalid video ID format", err)
		return
	}
	fmt.Printf("DEBUG HANDLER: Parsed video ID: %s\n", videoID.String())

	// Get user ID from context (set by auth middleware)
	userIDRaw, exists := c.Get("userID")
	fmt.Printf("DEBUG HANDLER: User ID exists: %v, userIDRaw: %v, type: %T\n", exists, userIDRaw, userIDRaw)

	if !exists {
		fmt.Printf("DEBUG HANDLER: User not authenticated\n")
		h.response.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Check the type and try to convert userID
	fmt.Printf("DEBUG HANDLER: Attempting to convert userID from type %T\n", userIDRaw)
	var userID uuid.UUID

	switch v := userIDRaw.(type) {
	case string:
		fmt.Printf("DEBUG HANDLER: userID is string: %s\n", v)
		parsedID, parseErr := uuid.Parse(v)
		if parseErr != nil {
			fmt.Printf("DEBUG HANDLER: Failed to parse userID string: %v\n", parseErr)
			h.response.InternalErrorResponse(c, "Invalid user ID format", parseErr)
			return
		}
		userID = parsedID
	case uuid.UUID:
		fmt.Printf("DEBUG HANDLER: userID is already UUID\n")
		userID = v
	default:
		fmt.Printf("DEBUG HANDLER: userID is unknown type: %T\n", v)
		h.response.InternalErrorResponse(c, "Invalid user ID type", fmt.Errorf("unexpected user ID type: %T", v))
		return
	}

	fmt.Printf("DEBUG HANDLER: Final userID: %s\n", userID.String())

	// Parse request body
	var req CreateCommentRequest
	fmt.Printf("DEBUG HANDLER: About to bind JSON request\n")

	if err := c.ShouldBindJSON(&req); err != nil {
		// Include full error details in the response
		fmt.Printf("DEBUG HANDLER: JSON binding error: %v\n", err)
		h.response.ErrorResponse(c, http.StatusBadRequest, "binding_error",
			fmt.Sprintf("JSON binding error: %v", err), err)
		return
	}

	fmt.Printf("DEBUG HANDLER: JSON bound successfully, content: %s, parentID: %v\n",
		req.Content, req.ParentID)

	// Create comment
	comment := NewComment(videoID, userID, req.Content, req.ParentID)
	fmt.Printf("DEBUG HANDLER: Comment created, about to save: ID=%s\n", comment.ID.String())

	fmt.Printf("DEBUG HANDLER: Attempting to create comment - VideoID: %s, UserID: %s, Content length: %d\n",
		videoID.String(), userID.String(), len(req.Content))

	if err := h.service.CreateComment(c.Request.Context(), comment); err != nil {
		// Include full error details in the response
		fmt.Printf("DEBUG HANDLER: Service.CreateComment failed: %v\n", err)
		h.response.ErrorResponse(c, http.StatusInternalServerError, "create_comment_error",
			fmt.Sprintf("Failed to create comment: %v", err), err)
		return
	}

	fmt.Printf("DEBUG HANDLER: Comment created successfully\n")
	h.response.SuccessResponse(c, comment, "Comment created successfully")
}

// @Summary Update a comment
// @Description Updates the content of an existing comment
// @Tags comment
// @Accept json
// @Produce json
// @Param id path string true "Comment ID (UUID)"
// @Security BearerAuth
// @Param comment body UpdateCommentRequest true "Updated comment data"
// @Success 200 {object} http.Response{message=string} "Comment updated successfully"
// @Failure 400 {object} http.Response{error=http.Error} "Invalid comment ID format or invalid comment data"
// @Failure 401 {object} http.Response{error=http.Error} "Unauthorized - user not authenticated"
// @Failure 404 {object} http.Response{error=http.Error} "Comment not found"
// @Failure 500 {object} http.Response{error=http.Error} "Internal server error"
// @Router /comment/{id} [put]
func (h *Handler) UpdateComment(c *gin.Context) {
	commentIDStr := c.Param("id")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		h.response.ErrorResponse(c, http.StatusBadRequest, "invalid_comment_id", "Invalid comment ID format", err)
		return
	}

	// Parse request body
	var req UpdateCommentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationErrorResponse(c, "content", "Invalid request body")
		return
	}

	// Update comment
	if err := h.service.UpdateComment(c.Request.Context(), commentID, req.Content); err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			h.response.NotFoundResponse(c, "Comment not found")
			return
		}
		h.response.InternalErrorResponse(c, "Failed to update comment", err)
		return
	}

	h.response.SuccessResponse(c, nil, "Comment updated successfully")
}

// @Summary Delete a comment
// @Description Deletes an existing comment
// @Tags comment
// @Accept json
// @Produce json
// @Param id path string true "Comment ID (UUID)"
// @Security BearerAuth
// @Success 200 {object} http.Response{message=string} "Comment deleted successfully"
// @Failure 400 {object} http.Response{error=http.Error} "Invalid comment ID format"
// @Failure 401 {object} http.Response{error=http.Error} "Unauthorized - user not authenticated"
// @Failure 404 {object} http.Response{error=http.Error} "Comment not found"
// @Failure 500 {object} http.Response{error=http.Error} "Internal server error"
// @Router /comment/{id} [delete]
func (h *Handler) DeleteComment(c *gin.Context) {
	commentIDStr := c.Param("id")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		h.response.ErrorResponse(c, http.StatusBadRequest, "invalid_comment_id", "Invalid comment ID format", err)
		return
	}

	// Delete comment
	if err := h.service.DeleteComment(c.Request.Context(), commentID); err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			h.response.NotFoundResponse(c, "Comment not found")
			return
		}
		h.response.InternalErrorResponse(c, "Failed to delete comment", err)
		return
	}

	h.response.SuccessResponse(c, nil, "Comment deleted successfully")
}

// @Summary Add a reaction to a comment
// @Description Adds a reaction (like/dislike) to a comment
// @Tags comment
// @Accept json
// @Produce json
// @Param id path string true "Comment ID (UUID)"
// @Security BearerAuth
// @Param reaction body ReactionRequest true "Reaction data"
// @Success 200 {object} http.Response{message=string} "Reaction added successfully"
// @Failure 400 {object} http.Response{error=http.Error} "Invalid comment ID format or invalid reaction type"
// @Failure 401 {object} http.Response{error=http.Error} "Unauthorized - user not authenticated"
// @Failure 404 {object} http.Response{error=http.Error} "Comment not found"
// @Failure 500 {object} http.Response{error=http.Error} "Internal server error"
// @Router /comment/{id}/reaction [post]
func (h *Handler) AddReaction(c *gin.Context) {
	commentIDStr := c.Param("id")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		h.response.ErrorResponse(c, http.StatusBadRequest, "invalid_comment_id", "Invalid comment ID format", err)
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		h.response.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Parse request body
	var req ReactionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.response.ValidationErrorResponse(c, "type", "Invalid request body")
		return
	}

	// Validate reaction type
	var reactionType Type
	if req.Type == string(TypeLike) {
		reactionType = TypeLike
	} else if req.Type == string(TypeDislike) {
		reactionType = TypeDislike
	} else {
		h.response.ValidationErrorResponse(c, "type", "Invalid reaction type")
		return
	}

	// Create reaction
	reaction := &Reaction{
		CommentID: commentID,
		UserID:    userID.(uuid.UUID),
		Type:      reactionType,
		CreatedAt: getNowUTC(),
		UpdatedAt: getNowUTC(),
	}

	if err := h.service.AddReaction(c.Request.Context(), reaction); err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			h.response.NotFoundResponse(c, "Comment not found")
			return
		}
		h.response.InternalErrorResponse(c, "Failed to add reaction", err)
		return
	}

	h.response.SuccessResponse(c, nil, "Reaction added successfully")
}

// @Summary Remove a reaction from a comment
// @Description Removes a user's reaction from a comment
// @Tags comment
// @Accept json
// @Produce json
// @Param id path string true "Comment ID (UUID)"
// @Security BearerAuth
// @Success 200 {object} http.Response{message=string} "Reaction removed successfully"
// @Failure 400 {object} http.Response{error=http.Error} "Invalid comment ID format"
// @Failure 401 {object} http.Response{error=http.Error} "Unauthorized - user not authenticated"
// @Failure 404 {object} http.Response{error=http.Error} "Comment or reaction not found"
// @Failure 500 {object} http.Response{error=http.Error} "Internal server error"
// @Router /comment/{id}/reaction [delete]
func (h *Handler) RemoveReaction(c *gin.Context) {
	commentIDStr := c.Param("id")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		h.response.ErrorResponse(c, http.StatusBadRequest, "invalid_comment_id", "Invalid comment ID format", err)
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		h.response.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	// Remove reaction
	if err := h.service.RemoveReaction(c.Request.Context(), commentID, userID.(uuid.UUID)); err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			h.response.NotFoundResponse(c, "Comment not found")
			return
		}
		h.response.InternalErrorResponse(c, "Failed to remove reaction", err)
		return
	}

	h.response.SuccessResponse(c, nil, "Reaction removed successfully")
}

// Helper functions

// getPaginationParams extracts and validates pagination parameters from request
func getPaginationParams(c *gin.Context) (page, limit int, sortBy, sortOrder string) {
	// Default values
	page = 1
	limit = 20
	sortBy = "created_at"
	sortOrder = "desc"

	// Parse page parameter
	if pageStr := c.Query("page"); pageStr != "" {
		if val, err := strconv.Atoi(pageStr); err == nil && val > 0 {
			page = val
		}
	}

	// Parse limit parameter
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}

	// Parse sort parameter
	if sort := c.Query("sort"); sort != "" {
		switch sort {
		case "newest":
			sortBy = "created_at"
			sortOrder = "desc"
		case "oldest":
			sortBy = "created_at"
			sortOrder = "asc"
		case "most_liked":
			sortBy = "likes"
			sortOrder = "desc"
		}
	}

	return page, limit, sortBy, sortOrder
}

// getNowUTC returns the current time in UTC
func getNowUTC() time.Time {
	return time.Now().UTC()
}
