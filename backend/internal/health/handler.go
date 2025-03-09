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

// @Summary Health check endpoint
// @Description Checks if the API server is running properly
// @Tags health
// @Produce json
// @Success 200 {object} interface{} "Health check successful"
// @Router /health [get]
func (h *Handler) HandleHealthCheck(c *gin.Context) {
	h.responseHandler.SuccessResponse(c, nil, "Health check successful")
}
