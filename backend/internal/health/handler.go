package health

import (
	"github.com/gin-gonic/gin"
)

// Handler handles health check related endpoints
type Handler struct {
	responseHandler ResponseHandler
}

// NewHandler creates a new health check handler
func NewHandler(responseHandler ResponseHandler) *Handler {
	return &Handler{
		responseHandler: responseHandler,
	}
}

// HandleHealthCheck handles the health check endpoint
func (h *Handler) HandleHealthCheck(c *gin.Context) {
	h.responseHandler.SuccessResponse(c, nil, "Health check successful")
}
