package notification

import (
	"net/http"
	"strconv"

	httpHandler "github.com/consensuslabs/pavilion-network/backend/internal/http"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for notification endpoints
type Handler struct {
	service         NotificationService
	responseHandler httpHandler.ResponseHandler
	logger          logger.Logger
}

// NewHandler creates a new notification handler instance
func NewHandler(service NotificationService, responseHandler httpHandler.ResponseHandler, logger logger.Logger) *Handler {
	return &Handler{
		service:         service,
		responseHandler: responseHandler,
		logger:          logger,
	}
}

// RegisterRoutes registers all notification routes
func (h *Handler) RegisterRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) {
	// All notification routes require authentication
	notifications := router.Group("/api/v1/notifications")
	notifications.Use(authMiddleware)
	{
		notifications.GET("/", h.handleGetNotifications)
		notifications.GET("/unread-count", h.handleGetUnreadCount)
		notifications.PUT("/:id/read", h.handleMarkAsRead)
		notifications.PUT("/read-all", h.handleMarkAllAsRead)
	}
}

// @Summary Get user notifications
// @Description Retrieve a paginated list of notifications for the authenticated user
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Number of notifications to return (default: 10)"
// @Param page query int false "Page number for pagination (default: 1)"
// @Success 200 {object} httpHandler.APIResponse{data=[]Notification} "Notifications retrieved successfully"
// @Failure 401 {object} httpHandler.APIResponse{error=httpHandler.APIError} "Unauthorized"
// @Failure 500 {object} httpHandler.APIResponse{error=httpHandler.APIError} "Internal server error"
// @Router /api/v1/notifications/ [get]
func (h *Handler) handleGetNotifications(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	
	// Get user ID from context (set by auth middleware)
	userIDStr, exists := c.Get("userID")
	if !exists {
		h.responseHandler.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		h.logger.LogInfo("Invalid user ID format", map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		})
		h.responseHandler.ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Invalid user ID format", err)
		return
	}

	// Parse pagination parameters
	limit := 10 // Default limit
	page := 1   // Default page

	if limitParam := c.Query("limit"); limitParam != "" {
		parsedLimit, err := strconv.Atoi(limitParam)
		if err != nil || parsedLimit <= 0 {
			h.logger.LogInfo("Invalid limit parameter", map[string]interface{}{
				"request_id": requestID,
				"limit":      limitParam,
			})
			h.responseHandler.ErrorResponse(c, http.StatusBadRequest, "INVALID_PARAMETER", "Invalid limit parameter, must be a positive integer", nil)
			return
		}
		limit = parsedLimit
	}

	if pageParam := c.Query("page"); pageParam != "" {
		parsedPage, err := strconv.Atoi(pageParam)
		if err != nil || parsedPage <= 0 {
			h.logger.LogInfo("Invalid page parameter", map[string]interface{}{
				"request_id": requestID,
				"page":       pageParam,
			})
			h.responseHandler.ErrorResponse(c, http.StatusBadRequest, "INVALID_PARAMETER", "Invalid page parameter, must be a positive integer", nil)
			return
		}
		page = parsedPage
	}

	// Calculate offset based on page and limit
	offset := (page - 1) * limit

	// Get notifications
	notifications, err := h.service.GetUserNotifications(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.logger.LogInfo("Failed to get notifications", map[string]interface{}{
			"request_id": requestID,
			"user_id":    userID.String(),
			"error":      err.Error(),
		})
		h.responseHandler.ErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve notifications", err)
		return
	}

	h.logger.LogInfo("Notifications retrieved successfully", map[string]interface{}{
		"request_id": requestID,
		"user_id":    userID.String(),
		"count":      len(notifications),
	})

	h.responseHandler.SuccessResponse(c, notifications, "Notifications retrieved successfully")
}

// @Summary Get unread notification count
// @Description Get the count of unread notifications for the authenticated user
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Success 200 {object} httpHandler.APIResponse{data=map[string]int} "Unread count retrieved successfully"
// @Failure 401 {object} httpHandler.APIResponse{error=httpHandler.APIError} "Unauthorized"
// @Failure 500 {object} httpHandler.APIResponse{error=httpHandler.APIError} "Internal server error"
// @Router /api/v1/notifications/unread-count [get]
func (h *Handler) handleGetUnreadCount(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	
	// Get user ID from context (set by auth middleware)
	userIDStr, exists := c.Get("userID")
	if !exists {
		h.responseHandler.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		h.logger.LogInfo("Invalid user ID format", map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		})
		h.responseHandler.ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Invalid user ID format", err)
		return
	}

	// Get unread count
	count, err := h.service.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		h.logger.LogInfo("Failed to get unread notification count", map[string]interface{}{
			"request_id": requestID,
			"user_id":    userID.String(),
			"error":      err.Error(),
		})
		h.responseHandler.ErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve unread notification count", err)
		return
	}

	h.logger.LogInfo("Unread notification count retrieved successfully", map[string]interface{}{
		"request_id": requestID,
		"user_id":    userID.String(),
		"count":      count,
	})

	h.responseHandler.SuccessResponse(c, map[string]int{"count": count}, "Unread count retrieved successfully")
}

// @Summary Mark notification as read
// @Description Mark a specific notification as read
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Param id path string true "Notification ID (UUID)"
// @Success 200 {object} httpHandler.APIResponse "Notification marked as read"
// @Failure 400 {object} httpHandler.APIResponse{error=httpHandler.APIError} "Invalid notification ID"
// @Failure 401 {object} httpHandler.APIResponse{error=httpHandler.APIError} "Unauthorized"
// @Failure 404 {object} httpHandler.APIResponse{error=httpHandler.APIError} "Notification not found"
// @Failure 500 {object} httpHandler.APIResponse{error=httpHandler.APIError} "Internal server error"
// @Router /api/v1/notifications/{id}/read [put]
func (h *Handler) handleMarkAsRead(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	notificationID := c.Param("id")
	
	// Parse UUID from string
	notifUUID, err := uuid.Parse(notificationID)
	if err != nil {
		h.logger.LogInfo("Invalid notification ID format", map[string]interface{}{
			"request_id":      requestID,
			"notification_id": notificationID,
			"error":           err.Error(),
		})
		h.responseHandler.ErrorResponse(c, http.StatusBadRequest, "INVALID_ID", "Invalid notification ID format", err)
		return
	}

	// Mark notification as read
	if err := h.service.MarkAsRead(c.Request.Context(), notifUUID); err != nil {
		h.logger.LogInfo("Failed to mark notification as read", map[string]interface{}{
			"request_id":      requestID,
			"notification_id": notificationID,
			"error":           err.Error(),
		})
		
		// Check for specific error types
		if err.Error() == "notification not found" {
			h.responseHandler.NotFoundResponse(c, "Notification not found")
			return
		}
		
		h.responseHandler.ErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to mark notification as read", err)
		return
	}

	h.logger.LogInfo("Notification marked as read successfully", map[string]interface{}{
		"request_id":      requestID,
		"notification_id": notificationID,
	})

	h.responseHandler.SuccessResponse(c, nil, "Notification marked as read")
}

// @Summary Mark all notifications as read
// @Description Mark all notifications as read for the authenticated user
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Success 200 {object} httpHandler.APIResponse "All notifications marked as read"
// @Failure 401 {object} httpHandler.APIResponse{error=httpHandler.APIError} "Unauthorized"
// @Failure 500 {object} httpHandler.APIResponse{error=httpHandler.APIError} "Internal server error"
// @Router /api/v1/notifications/read-all [put]
func (h *Handler) handleMarkAllAsRead(c *gin.Context) {
	requestID, _ := c.Get("request_id")
	
	// Get user ID from context (set by auth middleware)
	userIDStr, exists := c.Get("userID")
	if !exists {
		h.responseHandler.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		h.logger.LogInfo("Invalid user ID format", map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		})
		h.responseHandler.ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Invalid user ID format", err)
		return
	}

	// Mark all notifications as read
	if err := h.service.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		h.logger.LogInfo("Failed to mark all notifications as read", map[string]interface{}{
			"request_id": requestID,
			"user_id":    userID.String(),
			"error":      err.Error(),
		})
		h.responseHandler.ErrorResponse(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to mark all notifications as read", err)
		return
	}

	h.logger.LogInfo("All notifications marked as read successfully", map[string]interface{}{
		"request_id": requestID,
		"user_id":    userID.String(),
	})

	h.responseHandler.SuccessResponse(c, nil, "All notifications marked as read")
}