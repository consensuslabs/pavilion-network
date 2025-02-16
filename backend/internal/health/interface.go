package health

import (
	"github.com/gin-gonic/gin"
)

// ResponseHandler defines the interface for handling HTTP responses
type ResponseHandler interface {
	SuccessResponse(c *gin.Context, data interface{}, message string)
	ErrorResponse(c *gin.Context, status int, code, message string, err error)
}
