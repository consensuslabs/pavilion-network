package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Service handles authentication-related business logic
type Service struct {
	db            *gorm.DB
	tokenService  TokenService
	refreshTokens RefreshTokenService
	config        *Config
}

// NewService creates a new auth service instance
func NewService(db *gorm.DB, ts TokenService, rt RefreshTokenService, config *Config) *Service {
	return &Service{
		db:            db,
		tokenService:  ts,
		refreshTokens: rt,
		config:        config,
	}
}

// Login handles user authentication
func (s *Service) Login(identifier, password string) (*LoginResponse, error) {
	fmt.Printf("Login attempt for identifier: %s\n", identifier)

	var user User
	// Lookup user by email or username
	if err := s.db.Where("email = ? OR username = ?", identifier, identifier).First(&user).Error; err != nil {
		fmt.Printf("User lookup failed: %v\n", err)
		return nil, ErrInvalidCredentials
	}

	// Check if user's email is verified
	if !user.EmailVerified {
		fmt.Printf("Login attempt for unverified email: %s\n", user.Email)
		return nil, ErrEmailNotVerified
	}

	if !checkPasswordHash(password, user.Password) {
		fmt.Printf("Invalid password for user: %s\n", user.Email)
		return nil, ErrInvalidCredentials
	}

	fmt.Printf("Generating tokens for user: %s (ID: %s)\n", user.Email, user.ID)

	// Generate tokens
	accessToken, err := s.tokenService.GenerateAccessToken(&user)
	if err != nil {
		fmt.Printf("Failed to generate access token: %v\n", err)
		return nil, fmt.Errorf("failed to generate access token: %v", err)
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken(&user)
	if err != nil {
		fmt.Printf("Failed to generate refresh token: %v\n", err)
		return nil, fmt.Errorf("failed to generate refresh token: %v", err)
	}

	fmt.Printf("Generated refresh token for user %s: %s\n", user.ID, refreshToken)

	// Store refresh token
	if err := s.refreshTokens.Create(user.ID, refreshToken, time.Now().Add(s.config.JWT.RefreshTokenTTL)); err != nil {
		fmt.Printf("Failed to store refresh token: %v\n", err)
		return nil, fmt.Errorf("failed to store refresh token: %v", err)
	}

	// Update last login timestamp
	user.LastLoginAt = time.Now()
	if err := s.db.Save(&user).Error; err != nil {
		fmt.Printf("Failed to update last login timestamp: %v\n", err)
		return nil, err
	}

	response := &LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.JWT.AccessTokenTTL.Seconds()),
	}
	fmt.Printf("Login successful for user: %s\n", user.Email)
	return response, nil
}

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrEmailNotVerified = errors.New("email not verified")

func checkPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name"`
}

func (s *Service) Register(req RegisterRequest) (*User, error) {
	// Check if a user with given email or username already exists
	var count int64
	if err := s.db.Model(&User{}).Where("email = ? OR username = ?", req.Email, req.Username).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("user already exists")
	}

	// Hash the password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create the user record
	user := User{
		Username:      req.Username,
		Email:         req.Email,
		Password:      string(hashed),
		Name:          req.Name,
		EmailVerified: false,
		Active:        true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	// TODO: Send verification email to the user

	return &user, nil
}

// Logout invalidates the provided refresh token and ends the session for the user.
func (s *Service) Logout(userID uuid.UUID, refreshToken string) error {
	// Validate the refresh token
	claims, err := s.tokenService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return fmt.Errorf("invalid refresh token: %v", err)
	}

	// Verify the token belongs to the user
	tokenUserID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID in token: %v", err)
	}

	if tokenUserID != userID {
		return errors.New("token does not belong to user")
	}

	// Revoke the refresh token
	if err := s.refreshTokens.RevokeByToken(refreshToken); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %v", err)
	}

	return nil
}

// RefreshToken generates a new access token using the provided refresh token.
func (s *Service) RefreshToken(refreshToken string) (*LoginResponse, error) {
	// Validate the refresh token
	claims, err := s.tokenService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %v", err)
	}

	// Get user from claims
	var user User
	if err := s.db.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found: %v", err)
	}

	// Verify refresh token exists and is valid
	storedToken, err := s.refreshTokens.GetByToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("refresh token not found or expired: %v", err)
	}
	if storedToken.RevokedAt != nil {
		return nil, fmt.Errorf("refresh token has been revoked")
	}

	// Generate only new access token
	accessToken, err := s.tokenService.GenerateAccessToken(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %v", err)
	}

	return &LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken, // Reuse the same refresh token
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.JWT.AccessTokenTTL.Seconds()),
	}, nil
}

// ValidateToken validates the provided token and returns its claims if valid.
func (s *Service) ValidateToken(token string) (*TokenClaims, error) {
	// First try to validate as access token
	claims, err := s.tokenService.ValidateAccessToken(token)
	if err == nil {
		return claims, nil
	}

	// If not an access token, try as refresh token
	claims, err = s.tokenService.ValidateRefreshToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	return claims, nil
}
