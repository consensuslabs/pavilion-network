package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware creates a middleware for authentication
func AuthMiddleware(service *Service, responseHandler ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			responseHandler.UnauthorizedResponse(c, "Authorization header is required")
			c.Abort()
			return
		}

		// Check the Authorization header format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			responseHandler.UnauthorizedResponse(c, "Invalid authorization header format")
			c.Abort()
			return
		}

		token := parts[1]

		// Validate the token
		claims, err := service.ValidateToken(token)
		if err != nil {
			responseHandler.UnauthorizedResponse(c, "Invalid token")
			c.Abort()
			return
		}

		// Set user information in the context
		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)

		c.Next()
	}
}

// OptionalAuthMiddleware creates a middleware that attempts to authenticate but doesn't require it
func OptionalAuthMiddleware(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		token := parts[1]

		claims, err := service.ValidateToken(token)
		if err != nil {
			c.Next()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)

		c.Next()
	}
}
