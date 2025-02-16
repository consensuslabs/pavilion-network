package http

import (
	"github.com/gin-gonic/gin"
)

// ResponseHandler defines the interface for handling HTTP responses
type ResponseHandler interface {
	SuccessResponse(c *gin.Context, data interface{}, message string)
	ErrorResponse(c *gin.Context, status int, code, message string, err error)
	ValidationErrorResponse(c *gin.Context, field, message string)
	NotFoundResponse(c *gin.Context, message string)
	UnauthorizedResponse(c *gin.Context, message string)
	ForbiddenResponse(c *gin.Context, message string)
	InternalErrorResponse(c *gin.Context, message string, err error)
}

// Logger interface for logging operations
type Logger interface {
	LogInfo(msg string, fields map[string]interface{})
	LogError(err error, msg string) error
}
