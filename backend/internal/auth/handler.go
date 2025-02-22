package auth

import (
	stdhttp "net/http"

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
		// Public routes
		auth.POST("/login", h.handleLogin)
		auth.POST("/register", h.handleRegister)
		auth.POST("/refresh", h.handleRefresh)

		// Protected routes (require authentication)
		protected := auth.Group("")
		protected.Use(AuthMiddleware(h.service, h.responseHandler))
		protected.POST("/logout", h.handleLogout)
	}
}

// @Summary Login user
// @Description Authenticate user and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} http.APIResponse{data=LoginResponse} "Login successful"
// @Failure 400 {object} http.APIResponse{error=http.APIError} "Invalid request format"
// @Failure 401 {object} http.APIResponse{error=http.APIError} "Invalid credentials"
// @Router /auth/login [post]
func (h *Handler) handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responseHandler.ValidationErrorResponse(c, "request", "Invalid request format")
		return
	}

	response, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		h.responseHandler.ErrorResponse(c, stdhttp.StatusUnauthorized, "AUTH_ERROR", err.Error(), err)
		return
	}

	h.responseHandler.SuccessResponse(c, response, "Login successful")
}

// @Summary Register new user
// @Description Register a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 200 {object} http.APIResponse{data=User} "Registration successful"
// @Failure 400 {object} http.APIResponse{error=http.APIError} "Invalid request format or user already exists"
// @Router /auth/register [post]
func (h *Handler) handleRegister(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responseHandler.ValidationErrorResponse(c, "request", "Invalid request format")
		return
	}

	user, err := h.service.Register(req)
	if err != nil {
		h.responseHandler.ErrorResponse(c, stdhttp.StatusBadRequest, "REGISTRATION_ERROR", err.Error(), err)
		return
	}

	h.responseHandler.SuccessResponse(c, user, "Registration successful")
}

// @Summary Refresh access token
// @Description Get a new access token using a valid refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} http.APIResponse{data=LoginResponse} "Token refresh successful"
// @Failure 400 {object} http.APIResponse{error=http.APIError} "Invalid request format"
// @Failure 401 {object} http.APIResponse{error=http.APIError} "Invalid or expired refresh token"
// @Router /auth/refresh [post]
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
		h.responseHandler.ErrorResponse(c, stdhttp.StatusUnauthorized, "REFRESH_ERROR", err.Error(), err)
		return
	}

	h.responseHandler.SuccessResponse(c, response, "Token refresh successful")
}

// @Summary Logout user
// @Description Invalidate refresh token and end user session
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RefreshTokenRequest true "Refresh token to invalidate"
// @Success 200 {object} http.APIResponse "Logout successful"
// @Failure 400 {object} http.APIResponse{error=http.APIError} "Invalid request format"
// @Failure 401 {object} http.APIResponse{error=http.APIError} "Unauthorized or invalid token"
// @Router /auth/logout [post]
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
		h.responseHandler.ErrorResponse(c, stdhttp.StatusInternalServerError, "INTERNAL_ERROR", "Invalid user ID format", err)
		return
	}

	if err := h.service.Logout(userID, req.RefreshToken); err != nil {
		h.responseHandler.ErrorResponse(c, stdhttp.StatusUnauthorized, "LOGOUT_ERROR", err.Error(), err)
		return
	}

	h.responseHandler.SuccessResponse(c, nil, "Logout successful")
}
