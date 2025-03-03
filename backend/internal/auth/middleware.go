package auth

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware creates a middleware for authentication
func AuthMiddleware(service *Service, responseHandler ResponseHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Printf("DEBUG AUTH: Starting auth middleware check for path: %s\n", c.Request.URL.Path)

		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			fmt.Printf("DEBUG AUTH: Missing Authorization header\n")
			responseHandler.UnauthorizedResponse(c, "Authorization header is required")
			c.Abort()
			return
		}

		// Check the Authorization header format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			fmt.Printf("DEBUG AUTH: Invalid authorization format: %s\n", authHeader)
			responseHandler.UnauthorizedResponse(c, "Invalid authorization header format")
			c.Abort()
			return
		}

		token := parts[1]
		fmt.Printf("DEBUG AUTH: Validating token (first 10 chars): %s...\n", token[:10])

		// Validate the token
		claims, err := service.ValidateToken(token)
		if err != nil {
			fmt.Printf("DEBUG AUTH: Token validation failed: %v\n", err)
			responseHandler.UnauthorizedResponse(c, "Invalid token")
			c.Abort()
			return
		}

		fmt.Printf("DEBUG AUTH: Token valid for userID: %v\n", claims.UserID)
		// Set user information in the context
		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)

		fmt.Printf("DEBUG AUTH: Auth middleware completed successfully\n")
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
