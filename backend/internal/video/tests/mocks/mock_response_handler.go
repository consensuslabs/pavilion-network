package mocks

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
)

// MockResponseHandler is a mock implementation of the ResponseHandler interface
type MockResponseHandler struct {
	mock.Mock
}

// ErrorResponse mocks the error response handler method
func (m *MockResponseHandler) ErrorResponse(c *gin.Context, code int, errorCode string, message string, err error) {
	m.Called(c, code, errorCode, message, err)
	c.JSON(code, gin.H{
		"success": false,
		"error": gin.H{
			"code":    errorCode,
			"message": message,
		},
	})
}

// SuccessResponse mocks the success response handler method
func (m *MockResponseHandler) SuccessResponse(c *gin.Context, data interface{}, message string) {
	m.Called(c, data, message)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}
