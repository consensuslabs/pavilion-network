package auth

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"github.com/google/uuid"
)

// Service handles authentication-related business logic
type Service struct {
	db           *gorm.DB
	tokenService TokenService
}

// NewService creates a new auth service instance
func NewService(db *gorm.DB, ts TokenService) *Service {
	return &Service{
		db:           db,
		tokenService: ts,
	}
}

// Login handles user authentication
func (s *Service) Login(identifier, password string) (*LoginResponse, error) {
	var user User
	// Lookup user by email or username
	if err := s.db.Where("email = ? OR username = ?", identifier, identifier).First(&user).Error; err != nil {
		return nil, ErrInvalidCredentials
	}

	// Check if user's email is verified
	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	if !checkPasswordHash(password, user.Password) {
		return nil, ErrInvalidCredentials
	}

	// Update last login timestamp
	user.LastLoginAt = time.Now()
	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}

	accessToken, err := s.tokenService.GenerateAccessToken(&user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken(&user)
	if err != nil {
		return nil, err
	}

	response := &LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}
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
	return errors.New("not implemented")
}

// RefreshToken generates a new access token using the provided refresh token.
func (s *Service) RefreshToken(refreshToken string) (*LoginResponse, error) {
	return nil, errors.New("not implemented")
}

// ValidateToken validates the provided token and returns its claims if valid.
func (s *Service) ValidateToken(token string) (*TokenClaims, error) {
	return nil, errors.New("not implemented")
}
