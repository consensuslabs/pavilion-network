package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// User model definition with authentication fields
type User struct {
	ID            int64          `gorm:"primaryKey;autoIncrement:false;type:bigint;default:unique_rowid()" json:"id"`
	Name          string         `json:"name"`
	Email         string         `gorm:"unique;not null" json:"email"`
	Password      string         `gorm:"not null" json:"-"` // Password hash, not exposed in JSON
	LastLoginAt   time.Time      `json:"lastLoginAt,omitempty"`
	Active        bool           `gorm:"default:true" json:"active"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	RefreshTokens []RefreshToken `gorm:"foreignKey:UserID" json:"-"`
}

// RefreshToken model for storing refresh tokens
type RefreshToken struct {
	ID        int64      `gorm:"primaryKey;autoIncrement:false;type:bigint;default:unique_rowid()" json:"id"`
	UserID    int64      `json:"userId"`
	Token     string     `gorm:"unique;not null" json:"token"`
	ExpiresAt time.Time  `json:"expiresAt"`
	CreatedAt time.Time  `json:"createdAt"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
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
	ExpiresIn    int    `json:"expiresIn"` // Seconds until the access token expires
}

// TokenClaims represents the JWT claims
type TokenClaims struct {
	UserID int64  `json:"userId"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
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
