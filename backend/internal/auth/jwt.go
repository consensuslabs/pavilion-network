package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTService implements the TokenService interface using JWT tokens
type JWTService struct {
	config *Config
}

// NewJWTService creates a new JWT token service
func NewJWTService(config *Config) TokenService {
	return &JWTService{
		config: config,
	}
}

// GenerateAccessToken generates a new JWT access token for a user
func (s *JWTService) GenerateAccessToken(user *User) (string, error) {
	claims := &TokenClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.config.JWT.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.Secret))
}

// GenerateRefreshToken generates a new JWT refresh token for a user
func (s *JWTService) GenerateRefreshToken(user *User) (string, error) {
	claims := &TokenClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.config.JWT.RefreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.Secret))
}

// ValidateAccessToken validates a JWT access token and returns its claims
func (s *JWTService) ValidateAccessToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWT.Secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// ValidateRefreshToken validates a JWT refresh token and returns its claims
func (s *JWTService) ValidateRefreshToken(tokenString string) (*TokenClaims, error) {
	// For refresh tokens, we also need to check if the token exists in the database
	// and hasn't been revoked
	claims, err := s.ValidateAccessToken(tokenString) // Reuse validation logic
	if err != nil {
		return nil, err
	}

	// Convert string ID back to UUID and validate it's a valid UUID
	_, err = uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token: %v", err)
	}

	// TODO: Check if refresh token exists in database and hasn't been revoked
	// This will be implemented when we add the refresh token table and repository

	return claims, nil
}
