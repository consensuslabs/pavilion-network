package auth

import (
	"time"

	"gorm.io/gorm"
)

// Service handles authentication-related business logic
type Service struct {
	db *gorm.DB
}

// NewService creates a new auth service instance
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

// Login handles user authentication
func (s *Service) Login(email string) (*User, error) {
	user := &User{
		Name:      "Test User",
		Email:     email,
		CreatedAt: time.Now(),
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}
