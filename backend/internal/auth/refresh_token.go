package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshTokenRepository handles refresh token storage and retrieval
type RefreshTokenRepository struct {
	db *gorm.DB
}

// NewRefreshTokenRepository creates a new refresh token repository
func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		db: db,
	}
}

// Create stores a new refresh token
func (r *RefreshTokenRepository) Create(userID uuid.UUID, token string, expiresAt time.Time) error {
	fmt.Printf("Attempting to create refresh token - UserID: %s, Token: %s, ExpiresAt: %v\n", userID, token, expiresAt)

	// Check if token already exists
	var count int64
	if err := r.db.Model(&RefreshToken{}).Where("token = ?", token).Count(&count).Error; err != nil {
		fmt.Printf("Error checking existing token: %v\n", err)
		return err
	}
	if count > 0 {
		fmt.Printf("WARNING: Token already exists in database!\n")
		return fmt.Errorf("refresh token already exists")
	}

	refreshToken := RefreshToken{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	err := r.db.Create(&refreshToken).Error
	if err != nil {
		fmt.Printf("Error creating refresh token: %v\n", err)
	} else {
		fmt.Printf("Successfully created refresh token for user %s\n", userID)
	}
	return err
}

// GetByToken retrieves a refresh token by its token string
func (r *RefreshTokenRepository) GetByToken(token string) (*RefreshToken, error) {
	var refreshToken RefreshToken
	err := r.db.Where("token = ? AND revoked_at IS NULL AND expires_at > ?", token, time.Now()).First(&refreshToken).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("refresh token not found or expired")
		}
		return nil, err
	}
	return &refreshToken, nil
}

// RevokeByToken revokes a refresh token
func (r *RefreshTokenRepository) RevokeByToken(token string) error {
	now := time.Now()
	result := r.db.Model(&RefreshToken{}).
		Where("token = ? AND revoked_at IS NULL", token).
		Update("revoked_at", now)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("refresh token not found or already revoked")
	}

	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (r *RefreshTokenRepository) RevokeAllUserTokens(userID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

// DeleteExpired deletes all expired refresh tokens
func (r *RefreshTokenRepository) DeleteExpired() error {
	return r.db.Where("expires_at < ? OR revoked_at IS NOT NULL", time.Now()).
		Delete(&RefreshToken{}).Error
}
