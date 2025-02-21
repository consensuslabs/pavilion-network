package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshTokenRepository handles refresh token storage and retrieval
type RefreshTokenRepository struct {
	db     *gorm.DB
	logger logger.Logger
}

// NewRefreshTokenRepository creates a new refresh token repository
func NewRefreshTokenRepository(db *gorm.DB, logger logger.Logger) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		db:     db,
		logger: logger,
	}
}

// Create stores a new refresh token
func (r *RefreshTokenRepository) Create(userID uuid.UUID, token string, expiresAt time.Time) error {
	r.logger.LogInfo("Creating refresh token", map[string]interface{}{
		"userID":    userID,
		"expiresAt": expiresAt,
	})

	// Check if token already exists
	var count int64
	if err := r.db.Model(&RefreshToken{}).Where("token = ?", token).Count(&count).Error; err != nil {
		r.logger.LogError(err, "Error checking existing token")
		return err
	}
	if count > 0 {
		r.logger.LogWarn("Token already exists in database", map[string]interface{}{
			"userID": userID,
		})
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
		r.logger.LogError(err, "Failed to create refresh token")
	} else {
		r.logger.LogInfo("Successfully created refresh token", map[string]interface{}{
			"userID": userID,
		})
	}
	return err
}

// GetByToken retrieves a refresh token by its token string
func (r *RefreshTokenRepository) GetByToken(token string) (*RefreshToken, error) {
	var refreshToken RefreshToken
	err := r.db.Where("token = ? AND revoked_at IS NULL AND expires_at > ?", token, time.Now()).First(&refreshToken).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.LogWarn("Refresh token not found or expired", nil)
			return nil, errors.New("refresh token not found or expired")
		}
		r.logger.LogError(err, "Error retrieving refresh token")
		return nil, err
	}
	return &refreshToken, nil
}

// RevokeByToken revokes a refresh token
func (r *RefreshTokenRepository) RevokeByToken(token string) error {
	r.logger.LogInfo("Revoking refresh token", nil)

	result := r.db.Model(&RefreshToken{}).
		Where("token = ? AND revoked_at IS NULL", token).
		Update("revoked_at", time.Now())

	if result.Error != nil {
		r.logger.LogError(result.Error, "Failed to revoke refresh token")
		return result.Error
	}

	if result.RowsAffected == 0 {
		r.logger.LogWarn("No active token found to revoke", nil)
		return errors.New("token not found or already revoked")
	}

	r.logger.LogInfo("Successfully revoked refresh token", nil)
	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (r *RefreshTokenRepository) RevokeAllUserTokens(userID uuid.UUID) error {
	r.logger.LogInfo("Revoking all refresh tokens for user", map[string]interface{}{
		"userID": userID,
	})

	result := r.db.Model(&RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now())

	if result.Error != nil {
		r.logger.LogError(result.Error, "Failed to revoke all user tokens")
		return result.Error
	}

	r.logger.LogInfo("Successfully revoked all user tokens", map[string]interface{}{
		"userID":        userID,
		"tokensRevoked": result.RowsAffected,
	})
	return nil
}

// DeleteExpired deletes all expired refresh tokens
func (r *RefreshTokenRepository) DeleteExpired() error {
	r.logger.LogInfo("Deleting expired refresh tokens", nil)

	result := r.db.Where("expires_at < ? OR revoked_at IS NOT NULL", time.Now()).
		Delete(&RefreshToken{})

	if result.Error != nil {
		r.logger.LogError(result.Error, "Failed to delete expired tokens")
		return result.Error
	}

	r.logger.LogInfo("Successfully deleted expired tokens", map[string]interface{}{
		"tokensDeleted": result.RowsAffected,
	})
	return nil
}
