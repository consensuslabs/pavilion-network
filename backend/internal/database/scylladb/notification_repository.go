package scylladb

import (
	"context"
	"fmt"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// NotificationRepository implements the notification.NotificationRepository interface
type NotificationRepository struct {
	session *gocql.Session
	logger  logger.Logger
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(session *gocql.Session, logger logger.Logger) *NotificationRepository {
	return &NotificationRepository{
		session: session,
		logger:  logger,
	}
}

// SaveNotification saves a notification to the database
func (r *NotificationRepository) SaveNotification(ctx context.Context, notification *notification.Notification) error {
	query := `INSERT INTO notifications (id, user_id, type, content, metadata, created_at) 
			VALUES (?, ?, ?, ?, ?, ?)`

	// Serialize metadata to bytes for storage
	metadataBytes, err := encodeToJSONBytes(notification.Metadata)
	if err != nil {
		r.logger.LogError(err, "Failed to serialize notification metadata")
		return fmt.Errorf("failed to serialize notification metadata: %w", err)
	}

	// Execute the query
	if err := r.session.Query(query,
		notification.ID,
		notification.UserID,
		notification.Type,
		notification.Content,
		metadataBytes,
		notification.CreatedAt,
	).WithContext(ctx).Exec(); err != nil {
		r.logger.LogError(err, "Failed to save notification")
		return fmt.Errorf("failed to save notification: %w", err)
	}

	return nil
}

// GetNotificationsByUserID retrieves notifications for a user
func (r *NotificationRepository) GetNotificationsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*notification.Notification, error) {
	query := `SELECT id, user_id, type, content, metadata, read_at, created_at FROM notifications 
			WHERE user_id = ? ORDER BY created_at DESC LIMIT ?`

	// Execute the query
	scanner := r.session.Query(query, userID, limit+offset).WithContext(ctx).Iter().Scanner()
	
	notifications := make([]*notification.Notification, 0)
	
	// Skip offset rows
	skipped := 0
	for scanner.Next() && skipped < offset {
		skipped++
	}
	
	count := 0
	for scanner.Next() && count < limit {
		var notif notification.Notification
		var metadataBytes []byte
		var readAt gocql.UUID
		var readAtPtr *time.Time

		// Scan values from the row
		if err := scanner.Scan(
			&notif.ID,
			&notif.UserID,
			&notif.Type,
			&notif.Content,
			&metadataBytes,
			&readAt,
			&notif.CreatedAt,
		); err != nil {
			r.logger.LogError(err, "Failed to scan notification row")
			return nil, fmt.Errorf("failed to scan notification row: %w", err)
		}

		// Deserialize metadata from bytes
		if len(metadataBytes) > 0 {
			if err := decodeFromJSONBytes(metadataBytes, &notif.Metadata); err != nil {
				r.logger.LogError(err, "Failed to deserialize notification metadata")
				// Continue with empty metadata rather than failing the whole request
				notif.Metadata = make(map[string]interface{})
			}
		} else {
			notif.Metadata = make(map[string]interface{})
		}

		// Convert readAt UUID to time.Time if it's not nil
		var emptyUUID gocql.UUID
		if readAt != emptyUUID {
			t := readAt.Time()
			readAtPtr = &t
		}
		notif.ReadAt = readAtPtr

		notifications = append(notifications, &notif)
		count++
	}

	if err := scanner.Err(); err != nil {
		r.logger.LogError(err, "Error iterating notification results")
		return nil, fmt.Errorf("error iterating notification results: %w", err)
	}

	return notifications, nil
}

// GetUnreadCount gets the count of unread notifications for a user
func (r *NotificationRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read_at IS NULL`

	var count int
	if err := r.session.Query(query, userID).WithContext(ctx).Scan(&count); err != nil {
		r.logger.LogError(err, "Failed to get unread count")
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

// MarkAsRead marks a notification as read
func (r *NotificationRepository) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	// First check if the notification exists
	checkQuery := `SELECT id FROM notifications WHERE id = ?`
	var id uuid.UUID
	if err := r.session.Query(checkQuery, notificationID).WithContext(ctx).Scan(&id); err != nil {
		if err == gocql.ErrNotFound {
			return fmt.Errorf("notification not found")
		}
		r.logger.LogError(err, "Failed to check notification existence")
		return fmt.Errorf("failed to check notification existence: %w", err)
	}

	// Update the notification
	query := `UPDATE notifications SET read_at = ? WHERE id = ?`
	now := time.Now()
	if err := r.session.Query(query, gocql.UUIDFromTime(now), notificationID).WithContext(ctx).Exec(); err != nil {
		r.logger.LogError(err, "Failed to mark notification as read")
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return nil
}

// MarkAllAsRead marks all notifications for a user as read
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE notifications SET read_at = ? WHERE user_id = ? AND read_at IS NULL`
	now := time.Now()
	if err := r.session.Query(query, gocql.UUIDFromTime(now), userID).WithContext(ctx).Exec(); err != nil {
		r.logger.LogError(err, "Failed to mark all notifications as read")
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	return nil
}