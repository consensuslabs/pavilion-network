package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
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
	logger        logger.Logger
}

// NewService creates a new auth service instance
func NewService(db *gorm.DB, ts TokenService, rt RefreshTokenService, config *Config, logger logger.Logger) *Service {
	return &Service{
		db:            db,
		tokenService:  ts,
		refreshTokens: rt,
		config:        config,
		logger:        logger,
	}
}

// Login handles user authentication
func (s *Service) Login(identifier, password string) (*LoginResponse, error) {
	s.logger.LogInfo("Login attempt", map[string]interface{}{
		"identifier": identifier,
	})

	var user User
	// Lookup user by email or username
	if err := s.db.Where("email = ? OR username = ?", identifier, identifier).First(&user).Error; err != nil {
		s.logger.LogWarn("User lookup failed", map[string]interface{}{
			"identifier": identifier,
			"error":      err.Error(),
		})
		return nil, ErrInvalidCredentials
	}

	// Check if user's email is verified
	if !user.EmailVerified {
		s.logger.LogWarn("Login attempt with unverified email", map[string]interface{}{
			"email":  user.Email,
			"userID": user.ID,
		})
		return nil, ErrEmailNotVerified
	}

	if !checkPasswordHash(password, user.Password) {
		s.logger.LogWarn("Invalid password attempt", map[string]interface{}{
			"email":  user.Email,
			"userID": user.ID,
		})
		return nil, ErrInvalidCredentials
	}

	s.logger.LogInfo("Generating tokens", map[string]interface{}{
		"userID": user.ID,
		"email":  user.Email,
	})

	// Generate tokens
	accessToken, err := s.tokenService.GenerateAccessToken(&user)
	if err != nil {
		s.logger.LogError(err, "Failed to generate access token")
		return nil, fmt.Errorf("failed to generate access token: %v", err)
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken(&user)
	if err != nil {
		s.logger.LogError(err, "Failed to generate refresh token")
		return nil, fmt.Errorf("failed to generate refresh token: %v", err)
	}

	s.logger.LogInfo("Generated refresh token", map[string]interface{}{
		"userID": user.ID,
	})

	// Store refresh token
	if err := s.refreshTokens.Create(user.ID, refreshToken, time.Now().Add(s.config.JWT.RefreshTokenTTL)); err != nil {
		s.logger.LogError(err, "Failed to store refresh token")
		return nil, fmt.Errorf("failed to store refresh token: %v", err)
	}

	// Update last login timestamp
	user.LastLoginAt = time.Now()
	if err := s.db.Save(&user).Error; err != nil {
		s.logger.LogError(err, "Failed to update last login timestamp")
		return nil, err
	}

	response := &LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.JWT.AccessTokenTTL.Seconds()),
	}

	s.logger.LogInfo("Login successful", map[string]interface{}{
		"userID": user.ID,
		"email":  user.Email,
	})

	return response, nil
}

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrEmailNotVerified = errors.New("email not verified")

func checkPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// RegisterRequest represents the registration request payload
// @Description Registration request payload
type RegisterRequest struct {
	// Unique username
	Username string `json:"username" binding:"required" example:"johndoe"`
	// User email address
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
	// User password (min 8 characters)
	Password string `json:"password" binding:"required,min=6" example:"Pass123!"`
	// User's full name
	Name string `json:"name" example:"John Doe"`
}

func (s *Service) Register(req RegisterRequest) (*User, error) {
	s.logger.LogInfo("Registration attempt", map[string]interface{}{
		"username": req.Username,
		"email":    req.Email,
	})

	// Check if a user with given email or username already exists
	var count int64
	if err := s.db.Model(&User{}).Where("email = ? OR username = ?", req.Email, req.Username).Count(&count).Error; err != nil {
		s.logger.LogError(err, "Failed to check existing user")
		return nil, err
	}
	if count > 0 {
		s.logger.LogWarn("Registration attempt with existing credentials", map[string]interface{}{
			"username": req.Username,
			"email":    req.Email,
		})
		return nil, errors.New("user already exists")
	}

	// Hash the password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.LogError(err, "Failed to hash password")
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
		s.logger.LogError(err, "Failed to create user")
		return nil, err
	}

	s.logger.LogInfo("User registered successfully", map[string]interface{}{
		"userID":   user.ID,
		"username": user.Username,
		"email":    user.Email,
	})

	// TODO: Send verification email to the user

	return &user, nil
}

// Logout invalidates the provided refresh token and ends the session for the user.
func (s *Service) Logout(userID uuid.UUID, refreshToken string) error {
	s.logger.LogInfo("Logout attempt", map[string]interface{}{
		"userID": userID,
	})

	// Validate the refresh token
	claims, err := s.tokenService.ValidateRefreshToken(refreshToken)
	if err != nil {
		s.logger.LogWarn("Invalid refresh token during logout", map[string]interface{}{
			"userID": userID,
			"error":  err.Error(),
		})
		return fmt.Errorf("invalid refresh token: %v", err)
	}

	// Verify the token belongs to the user
	tokenUserID, err := uuid.Parse(claims.UserID)
	if err != nil {
		s.logger.LogError(err, "Invalid user ID in token")
		return fmt.Errorf("invalid user ID in token: %v", err)
	}

	if tokenUserID != userID {
		s.logger.LogWarn("Token does not belong to user", map[string]interface{}{
			"userID":      userID,
			"tokenUserID": tokenUserID,
		})
		return errors.New("token does not belong to user")
	}

	// Revoke the refresh token
	if err := s.refreshTokens.RevokeByToken(refreshToken); err != nil {
		s.logger.LogError(err, "Failed to revoke refresh token")
		return fmt.Errorf("failed to revoke refresh token: %v", err)
	}

	s.logger.LogInfo("Logout successful", map[string]interface{}{
		"userID": userID,
	})

	return nil
}

// RefreshToken generates a new access token using the provided refresh token.
func (s *Service) RefreshToken(refreshToken string) (*LoginResponse, error) {
	s.logger.LogInfo("Token refresh attempt", nil)

	// Validate the refresh token
	claims, err := s.tokenService.ValidateRefreshToken(refreshToken)
	if err != nil {
		s.logger.LogWarn("Invalid refresh token", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("invalid refresh token: %v", err)
	}

	// Get user from claims
	var user User
	if err := s.db.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		s.logger.LogError(err, "User not found during token refresh")
		return nil, fmt.Errorf("user not found: %v", err)
	}

	// Verify refresh token exists and is valid
	storedToken, err := s.refreshTokens.GetByToken(refreshToken)
	if err != nil {
		s.logger.LogWarn("Refresh token not found or expired", map[string]interface{}{
			"userID": user.ID,
			"error":  err.Error(),
		})
		return nil, fmt.Errorf("refresh token not found or expired: %v", err)
	}
	if storedToken.RevokedAt != nil {
		s.logger.LogWarn("Attempt to use revoked refresh token", map[string]interface{}{
			"userID": user.ID,
		})
		return nil, fmt.Errorf("refresh token has been revoked")
	}

	// Generate only new access token
	accessToken, err := s.tokenService.GenerateAccessToken(&user)
	if err != nil {
		s.logger.LogError(err, "Failed to generate new access token")
		return nil, fmt.Errorf("failed to generate access token: %v", err)
	}

	s.logger.LogInfo("Token refresh successful", map[string]interface{}{
		"userID": user.ID,
	})

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
