package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthService handles authentication operations
type AuthService interface {
	Login(identifier, password string) (*LoginResponse, error)
	Logout(userID uuid.UUID, refreshToken string) error
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
	ValidationErrorResponse(c *gin.Context, field, message string)
	UnauthorizedResponse(c *gin.Context, message string)
	ForbiddenResponse(c *gin.Context, message string)
	NotFoundResponse(c *gin.Context, message string)
	InternalErrorResponse(c *gin.Context, message string, err error)
}

// Logger interface for logging operations
type Logger interface {
	LogInfo(msg string, fields map[string]interface{})
	LogError(err error, msg string) error
}

// RefreshTokenService handles refresh token operations
type RefreshTokenService interface {
	Create(userID uuid.UUID, token string, expiresAt time.Time) error
	GetByToken(token string) (*RefreshToken, error)
	RevokeByToken(token string) error
	RevokeAllUserTokens(userID uuid.UUID) error
	DeleteExpired() error
}
