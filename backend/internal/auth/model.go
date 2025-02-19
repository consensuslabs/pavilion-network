package auth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User model definition with authentication fields
type User struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Username      string         `gorm:"unique;not null" json:"username"`
	Email         string         `gorm:"unique;not null" json:"email"`
	Password      string         `gorm:"not null" json:"-"` // Password hash, not exposed in JSON
	Name          string         `json:"name"`
	EmailVerified bool           `gorm:"default:false" json:"emailVerified"`
	LastLoginAt   time.Time      `json:"lastLoginAt,omitempty"`
	Active        bool           `gorm:"default:true" json:"active"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
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
