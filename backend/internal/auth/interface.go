package auth

import (
	"github.com/gin-gonic/gin"
)

// AuthService handles authentication operations
type AuthService interface {
	Login(email, password string) (*LoginResponse, error)
	Logout(userID int64, refreshToken string) error
	RefreshToken(refreshToken string) (*LoginResponse, error)
	ValidateToken(token string) (*TokenClaims, error)
}

// TokenService handles JWT operations
type TokenService interface {
	GenerateAccessToken(user *User) (string, error)
	GenerateRefreshToken(user *User) (string, error)
	ValidateAccessToken(token string) (*TokenClaims, error)
	ValidateRefreshToken(token string) (*TokenClaims, error)
}

// ResponseHandler handles HTTP responses
type ResponseHandler interface {
	SuccessResponse(c *gin.Context, data interface{}, message string)
	ErrorResponse(c *gin.Context, status int, code, message string, err error)
}

// Logger interface for logging operations
type Logger interface {
	LogInfo(msg string, fields map[string]interface{})
	LogError(err error, msg string) error
}
