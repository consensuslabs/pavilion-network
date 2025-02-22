package auth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User model definition with authentication fields
// @Description User model
type User struct {
	// Unique user ID
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// Unique username
	Username string `gorm:"unique;not null" json:"username" example:"johndoe"`
	// User email address
	Email string `gorm:"unique;not null" json:"email" example:"user@example.com"`
	// Password hash (not exposed in JSON)
	Password string `gorm:"not null" json:"-"`
	// User's full name
	Name string `json:"name" example:"John Doe"`
	// Whether email is verified
	EmailVerified bool `gorm:"default:false" json:"emailVerified" example:"true"`
	// Last login timestamp
	LastLoginAt time.Time `json:"lastLoginAt,omitempty"`
	// Whether account is active
	Active bool `gorm:"default:true" json:"active" example:"true"`
	// Account creation timestamp
	CreatedAt time.Time `json:"createdAt"`
	// Last update timestamp
	UpdatedAt time.Time `json:"updatedAt"`
	// Refresh tokens (not exposed in JSON)
	RefreshTokens []RefreshToken `gorm:"foreignKey:UserID" json:"-"`
}

// RefreshToken model for storing refresh tokens
type RefreshToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null" json:"userId"`
	Token     string     `gorm:"unique;not null" json:"token"`
	ExpiresAt time.Time  `json:"expiresAt"`
	CreatedAt time.Time  `json:"createdAt"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
}

// BeforeCreate hook for User model
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = time.Now()
	}
	return nil
}

// BeforeUpdate hook for User model
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = time.Now()
	return nil
}
