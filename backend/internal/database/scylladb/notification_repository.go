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

	// Convert notification ID to binary for ScyllaDB
	idBytes, err := notification.ID.MarshalBinary()
	if err != nil {
		r.logger.LogError(err, "Error marshaling notification ID")
		return fmt.Errorf("error marshaling notification ID: %w", err)
	}

	// Convert user ID to binary for ScyllaDB
	userIDBytes, err := notification.UserID.MarshalBinary()
	if err != nil {
		r.logger.LogError(err, "Error marshaling user ID")
		return fmt.Errorf("error marshaling user ID: %w", err)
	}

	// Execute the query
	if err := r.session.Query(query,
		idBytes,
		userIDBytes,
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
	// Convert UUID to binary for ScyllaDB
	userIDBytes, err := userID.MarshalBinary()
	if err != nil {
		r.logger.LogError(err, "Error marshaling user ID")
		return nil, fmt.Errorf("error marshaling user ID: %w", err)
	}

	query := `SELECT id, user_id, type, content, metadata, read_at, created_at FROM notifications 
			WHERE user_id = ? ORDER BY created_at DESC LIMIT ?`

	// Execute the query with consistency level ONE
	scanner := r.session.Query(query, userIDBytes, limit+offset).WithContext(ctx).Consistency(1).Iter().Scanner()
	
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
	// Convert UUID to binary for ScyllaDB
	userIDBytes, err := userID.MarshalBinary()
	if err != nil {
		r.logger.LogError(err, "Error marshaling user ID")
		return 0, fmt.Errorf("error marshaling user ID: %w", err)
	}

	query := `SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read_at IS NULL`

	var count int
	if err := r.session.Query(query, userIDBytes).WithContext(ctx).Consistency(1).Scan(&count); err != nil {
		r.logger.LogError(err, "Failed to get unread count")
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

// MarkAsRead marks a notification as read
func (r *NotificationRepository) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	// Convert notification ID to binary for ScyllaDB
	notificationIDBytes, err := notificationID.MarshalBinary()
	if err != nil {
		r.logger.LogError(err, "Error marshaling notification ID")
		return fmt.Errorf("error marshaling notification ID: %w", err)
	}

	// First check if the notification exists
	checkQuery := `SELECT id FROM notifications WHERE id = ?`
	var idBytes []byte
	if err := r.session.Query(checkQuery, notificationIDBytes).WithContext(ctx).Scan(&idBytes); err != nil {
		if err == gocql.ErrNotFound {
			return fmt.Errorf("notification not found")
		}
		r.logger.LogError(err, "Failed to check notification existence")
		return fmt.Errorf("failed to check notification existence: %w", err)
	}

	// Update the notification
	query := `UPDATE notifications SET read_at = ? WHERE id = ?`
	now := time.Now()
	if err := r.session.Query(query, gocql.UUIDFromTime(now), notificationIDBytes).WithContext(ctx).Exec(); err != nil {
		r.logger.LogError(err, "Failed to mark notification as read")
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return nil
}

// MarkAllAsRead marks all notifications for a user as read
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	// Convert UUID to binary for ScyllaDB
	userIDBytes, err := userID.MarshalBinary()
	if err != nil {
		r.logger.LogError(err, "Error marshaling user ID")
		return fmt.Errorf("error marshaling user ID: %w", err)
	}

	query := `UPDATE notifications SET read_at = ? WHERE user_id = ? AND read_at IS NULL`
	now := time.Now()
	if err := r.session.Query(query, gocql.UUIDFromTime(now), userIDBytes).WithContext(ctx).Consistency(1).Exec(); err != nil {
		r.logger.LogError(err, "Failed to mark all notifications as read")
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	return nil
}