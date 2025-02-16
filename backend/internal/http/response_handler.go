package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// responseHandler implements the ResponseHandler interface
type responseHandler struct {
	logger Logger
}

// NewResponseHandler creates a new instance of ResponseHandler
func NewResponseHandler(logger Logger) ResponseHandler {
	return &responseHandler{
		logger: logger,
	}
}

// SuccessResponse sends a success response with optional data and message
func (h *responseHandler) SuccessResponse(c *gin.Context, data interface{}, message string) {
	response := Response{
		Success: true,
		Message: message,
		Data:    data,
	}
	c.JSON(http.StatusOK, response)
}

// ErrorResponse sends an error response with status code, error code, and message
func (h *responseHandler) ErrorResponse(c *gin.Context, status int, code, message string, err error) {
	if err != nil {
		h.logger.LogError(err, message)
	}

	response := Response{
		Success: false,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
	c.JSON(status, response)
}

// ValidationErrorResponse sends a validation error response
func (h *responseHandler) ValidationErrorResponse(c *gin.Context, field, message string) {
	response := Response{
		Success: false,
		Error: &Error{
			Code:    "VALIDATION_ERROR",
			Message: message,
			Field:   field,
		},
	}
	c.JSON(http.StatusBadRequest, response)
}

// NotFoundResponse sends a not found error response
func (h *responseHandler) NotFoundResponse(c *gin.Context, message string) {
	response := Response{
		Success: false,
		Error: &Error{
			Code:    "NOT_FOUND",
			Message: message,
		},
	}
	c.JSON(http.StatusNotFound, response)
}

// UnauthorizedResponse sends an unauthorized error response
func (h *responseHandler) UnauthorizedResponse(c *gin.Context, message string) {
	response := Response{
		Success: false,
		Error: &Error{
			Code:    "UNAUTHORIZED",
			Message: message,
		},
	}
	c.JSON(http.StatusUnauthorized, response)
}

// ForbiddenResponse sends a forbidden error response
func (h *responseHandler) ForbiddenResponse(c *gin.Context, message string) {
	response := Response{
		Success: false,
		Error: &Error{
			Code:    "FORBIDDEN",
			Message: message,
		},
	}
	c.JSON(http.StatusForbidden, response)
}

// InternalErrorResponse sends an internal server error response
func (h *responseHandler) InternalErrorResponse(c *gin.Context, message string, err error) {
	if err != nil {
		h.logger.LogError(err, message)
	}

	response := Response{
		Success: false,
		Error: &Error{
			Code:    "INTERNAL_ERROR",
			Message: message,
		},
	}
	c.JSON(http.StatusInternalServerError, response)
}
