package auth

import (
	"errors"
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
	refreshToken := RefreshToken{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	return r.db.Create(&refreshToken).Error
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
