package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for auth endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new auth handler instance
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// RegisterRoutes registers all auth routes
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", h.handleLogin)
	}
}

// handleLogin handles the login endpoint
func (h *Handler) handleLogin(c *gin.Context) {
	identifier := c.PostForm("identifier")
	password := c.PostForm("password")

	user, err := h.service.Login(identifier, password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_DB",
				"message": "Failed to save user",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user": user,
		},
		"message": "OAuth login stub - token: dummy-token",
	})
}
