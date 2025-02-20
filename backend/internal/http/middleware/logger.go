package middleware

import (
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestLoggerMiddleware creates a middleware for logging HTTP requests
func RequestLoggerMiddleware(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate request ID
		requestID := uuid.New().String()
		start := time.Now()

		// Add logger to context
		contextLogger := log.WithRequestID(requestID)
		c.Set("logger", contextLogger)

		// Process request
		c.Next()

		// Log request completion
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// Prepare log fields
		fields := map[string]interface{}{
			"method":    c.Request.Method,
			"path":      c.Request.URL.Path,
			"status":    statusCode,
			"latency":   duration,
			"requestID": requestID,
			"clientIP":  c.ClientIP(),
			"userAgent": c.Request.UserAgent(),
		}

		// Add user ID if available
		if userID, exists := c.Get("userID"); exists {
			fields["userID"] = userID
		}

		// Log with appropriate level based on status code
		switch {
		case statusCode >= 500:
			contextLogger.LogError(nil, "Server error processing request")
		case statusCode >= 400:
			contextLogger.LogWarn("Client error processing request", fields)
		default:
			contextLogger.LogInfo("Request completed", fields)
		}
	}
}
