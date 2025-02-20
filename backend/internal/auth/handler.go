package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for auth endpoints
type Handler struct {
	service         *Service
	responseHandler ResponseHandler
}

// NewHandler creates a new auth handler instance
func NewHandler(service *Service, responseHandler ResponseHandler) *Handler {
	return &Handler{
		service:         service,
		responseHandler: responseHandler,
	}
}

// RegisterRoutes registers all auth routes
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", h.handleLogin)
		auth.POST("/register", h.handleRegister)
		auth.POST("/refresh", h.handleRefresh)
		auth.POST("/logout", h.handleLogout)
	}
}

// handleLogin handles the login endpoint
func (h *Handler) handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responseHandler.ValidationErrorResponse(c, "request", "Invalid request format")
		return
	}

	response, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		h.responseHandler.ErrorResponse(c, http.StatusUnauthorized, "AUTH_ERROR", err.Error(), err)
		return
	}

	h.responseHandler.SuccessResponse(c, response, "Login successful")
}

// handleRegister handles the registration endpoint
func (h *Handler) handleRegister(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responseHandler.ValidationErrorResponse(c, "request", "Invalid request format")
		return
	}

	user, err := h.service.Register(req)
	if err != nil {
		h.responseHandler.ErrorResponse(c, http.StatusBadRequest, "REGISTRATION_ERROR", err.Error(), err)
		return
	}

	h.responseHandler.SuccessResponse(c, user, "Registration successful")
}

// handleRefresh handles the token refresh endpoint
func (h *Handler) handleRefresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responseHandler.ValidationErrorResponse(c, "refreshToken", "Refresh token is required")
		return
	}

	response, err := h.service.RefreshToken(req.RefreshToken)
	if err != nil {
		h.responseHandler.ErrorResponse(c, http.StatusUnauthorized, "REFRESH_ERROR", err.Error(), err)
		return
	}

	h.responseHandler.SuccessResponse(c, response, "Token refresh successful")
}

// handleLogout handles the logout endpoint
func (h *Handler) handleLogout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responseHandler.ValidationErrorResponse(c, "refreshToken", "Refresh token is required")
		return
	}

	// Get user ID from the context (set by auth middleware)
	userIDStr, exists := c.Get("userID")
	if !exists {
		h.responseHandler.UnauthorizedResponse(c, "User not authenticated")
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		h.responseHandler.ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Invalid user ID format", err)
		return
	}

	if err := h.service.Logout(userID, req.RefreshToken); err != nil {
		h.responseHandler.ErrorResponse(c, http.StatusUnauthorized, "LOGOUT_ERROR", err.Error(), err)
		return
	}

	h.responseHandler.SuccessResponse(c, nil, "Logout successful")
}
