package auth

import (
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// Config represents authentication configuration
type Config struct {
	JWT struct {
		Secret          string
		AccessTokenTTL  time.Duration
		RefreshTokenTTL time.Duration
	}
	Password struct {
		MinLength  int
		MaxLength  int
		MinDigits  int
		MinSymbols int
	}
}

// NewConfigFromAuthConfig creates an auth.Config from config.AuthConfig
func NewConfigFromAuthConfig(cfg *config.AuthConfig) *Config {
	authConfig := &Config{}
	authConfig.JWT.Secret = cfg.JWT.Secret
	authConfig.JWT.AccessTokenTTL = cfg.JWT.AccessTokenTTL
	authConfig.JWT.RefreshTokenTTL = cfg.JWT.RefreshTokenTTL

	authConfig.Password.MinLength = 8  // Default password requirements
	authConfig.Password.MaxLength = 72 // bcrypt max length
	authConfig.Password.MinDigits = 1
	authConfig.Password.MinSymbols = 1

	return authConfig
}

// App represents the application context needed by auth handlers
type App struct {
	Config          *Config
	Logger          Logger
	Auth            AuthService
	Token           TokenService
	ResponseHandler ResponseHandler
}

// LoginRequest represents the login request payload
// @Description Login request payload
type LoginRequest struct {
	// User email address
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
	// User password
	Password string `json:"password" binding:"required,min=8" example:"Pass123!"`
}

// RefreshTokenRequest represents the refresh token request payload
// @Description Refresh token request payload
type RefreshTokenRequest struct {
	// Valid refresh token
	RefreshToken string `json:"refreshToken" binding:"required" example:"eyJhbGciOiJIUzI1NiIs..."`
}

// LoginResponse represents the login response
// @Description Login response payload
type LoginResponse struct {
	// User information
	User User `json:"user"`
	// JWT access token
	AccessToken string `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIs..."`
	// JWT refresh token
	RefreshToken string `json:"refreshToken" example:"eyJhbGciOiJIUzI1NiIs..."`
	// Token type (always "Bearer")
	TokenType string `json:"tokenType" example:"Bearer"`
	// Token expiration time in seconds
	ExpiresIn int `json:"expiresIn" example:"3600"`
}

// TokenClaims represents the JWT claims
// @Description JWT claims structure
type TokenClaims struct {
	// User ID
	UserID string `json:"userId" example:"550e8400-e29b-41d4-a716-446655440000"`
	// User email
	Email string `json:"email" example:"user@example.com"`
	jwt.RegisteredClaims
}
