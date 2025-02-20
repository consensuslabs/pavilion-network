package middleware

import (
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/gin-gonic/gin"
)

// GetLogger retrieves the logger from the gin context
func GetLogger(c *gin.Context) logger.Logger {
	if log, exists := c.Get("logger"); exists {
		if contextLogger, ok := log.(logger.Logger); ok {
			return contextLogger
		}
	}
	// Return a default logger if none found in context
	defaultLogger, _ := logger.NewLogger(&logger.Config{
		Level:       logger.Level("info"),
		Format:      "json",
		Output:      "stdout",
		Development: false,
	})
	return defaultLogger
}
