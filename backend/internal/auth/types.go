package auth

import (
	"time"

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

// App represents the application context needed by auth handlers
type App struct {
	Config          *Config
	Logger          Logger
	Auth            AuthService
	Token           TokenService
	ResponseHandler ResponseHandler
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	User         User   `json:"user"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	TokenType    string `json:"tokenType"`
	ExpiresIn    int    `json:"expiresIn"`
}

// TokenClaims represents the JWT claims
type TokenClaims struct {
	UserID int64  `json:"userId"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}
