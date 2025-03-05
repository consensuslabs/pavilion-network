package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// Repository implements the NotificationRepository interface
type Repository struct {
	session *gocql.Session
	logger  logger.Logger
	keyspace string
	table    string
}

// NewRepository creates a new ScyllaDB notification repository
func NewRepository(session *gocql.Session, logger logger.Logger, keyspace string) *Repository {
	return &Repository{
		session:  session,
		logger:   logger,
		keyspace: keyspace,
		table:    "notifications",
	}
}

// SaveNotification saves a notification to ScyllaDB
func (r *Repository) SaveNotification(ctx context.Context, notification *Notification) error {
	// If ID is not set, generate a new one
	if notification.ID == uuid.Nil {
		notification.ID = uuid.New()
	}

	// If CreatedAt is not set, set it to now
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}

	// Execute the insert query
	query := fmt.Sprintf(`
		INSERT INTO %s.%s (
			id, 
			user_id, 
			type, 
			content, 
			metadata, 
			read_at, 
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.keyspace, r.table,
	)

	err := r.session.Query(query,
		notification.ID,
		notification.UserID,
		notification.Type,
		notification.Content,
		notification.Metadata,
		notification.ReadAt,
		notification.CreatedAt,
	).Exec()

	if err != nil {
		r.logger.LogError(err, "Failed to save notification")
		return fmt.Errorf("failed to save notification: %w", err)
	}

	r.logger.LogInfo("Notification saved successfully", map[string]interface{}{
		"id":     notification.ID.String(),
		"userId": notification.UserID.String(),
		"type":   string(notification.Type),
	})

	return nil
}

// GetNotificationsByUserID retrieves notifications for a user with pagination
func (r *Repository) GetNotificationsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Notification, error) {
	// Validate input
	if limit <= 0 {
		limit = 10 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	// Execute the query with pagination
	query := fmt.Sprintf(`
		SELECT id, user_id, type, content, metadata, read_at, created_at 
		FROM %s.%s 
		WHERE user_id = ? 
		ORDER BY created_at DESC 
		LIMIT ?`,
		r.keyspace, r.table,
	)

	// Execute query
	iter := r.session.Query(query, userID, limit).PageSize(limit).Iter()

	// Process results
	var notifications []*Notification
	var id, uid gocql.UUID
	var notificationType string
	var content string
	var metadata map[string]interface{}
	var readAt *time.Time
	var createdAt time.Time

	// Skip offset records
	for i := 0; i < offset && iter.Scan(&id, &uid, &notificationType, &content, &metadata, &readAt, &createdAt); i++ {
		// Just skipping records for offset
	}

	// Process the remaining records up to the limit
	for iter.Scan(&id, &uid, &notificationType, &content, &metadata, &readAt, &createdAt) {
		notif := &Notification{
			ID:        uuid.UUID(id),
			UserID:    uuid.UUID(uid),
			Type:      EventType(notificationType),
			Content:   content,
			Metadata:  metadata,
			ReadAt:    readAt,
			CreatedAt: createdAt,
		}
		notifications = append(notifications, notif)
	}

	if err := iter.Close(); err != nil {
		r.logger.LogError(err, "Failed to get notifications by user ID")
		return nil, fmt.Errorf("failed to get notifications by user ID: %w", err)
	}

	return notifications, nil
}

// GetUnreadCount gets the count of unread notifications for a user
func (r *Repository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s.%s 
		WHERE user_id = ? AND read_at IS NULL`,
		r.keyspace, r.table,
	)

	var count int
	if err := r.session.Query(query, userID).Scan(&count); err != nil {
		r.logger.LogError(err, "Failed to get unread count")
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

// MarkAsRead marks a notification as read
func (r *Repository) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	now := time.Now()
	query := fmt.Sprintf(`
		UPDATE %s.%s 
		SET read_at = ? 
		WHERE id = ?`,
		r.keyspace, r.table,
	)

	if err := r.session.Query(query, now, notificationID).Exec(); err != nil {
		r.logger.LogError(err, "Failed to mark notification as read")
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return nil
}

// MarkAllAsRead marks all notifications for a user as read
func (r *Repository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	query := fmt.Sprintf(`
		UPDATE %s.%s 
		SET read_at = ? 
		WHERE user_id = ? AND read_at IS NULL`,
		r.keyspace, r.table,
	)

	if err := r.session.Query(query, now, userID).Exec(); err != nil {
		r.logger.LogError(err, "Failed to mark all notifications as read")
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	return nil
}