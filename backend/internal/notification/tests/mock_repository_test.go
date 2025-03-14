package tests

import (
	"context"
	"sync"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/notification"
	"github.com/google/uuid"
)

// MockRepository is a simple in-memory repository for testing
type MockRepository struct {
	notifications map[uuid.UUID]*notification.Notification
	unreadCount   map[uuid.UUID]int
	mutex         sync.RWMutex
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		notifications: make(map[uuid.UUID]*notification.Notification),
		unreadCount:   make(map[uuid.UUID]int),
	}
}

// SaveNotification saves a notification
func (r *MockRepository) SaveNotification(ctx context.Context, notification *notification.Notification) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.notifications[notification.ID] = notification

	// Update unread count
	if notification.ReadAt == nil {
		r.unreadCount[notification.UserID]++
	}

	return nil
}

// GetNotificationsByUserID gets notifications for a user
func (r *MockRepository) GetNotificationsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*notification.Notification, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []*notification.Notification
	for _, n := range r.notifications {
		if n.UserID == userID {
			result = append(result, n)
		}
	}

	// Apply offset and limit
	if offset >= len(result) {
		return []*notification.Notification{}, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}

// GetUnreadCount gets the count of unread notifications
func (r *MockRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.unreadCount[userID], nil
}

// MarkAsRead marks a notification as read
func (r *MockRepository) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	n, exists := r.notifications[notificationID]
	if !exists {
		return nil
	}

	// Check if it's already read
	if n.ReadAt != nil {
		return nil
	}

	// Mark as read
	now := time.Now()
	n.ReadAt = &now

	// Update unread count
	r.unreadCount[n.UserID]--
	if r.unreadCount[n.UserID] < 0 {
		r.unreadCount[n.UserID] = 0
	}

	return nil
}

// MarkAllAsRead marks all notifications for a user as read
func (r *MockRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := ctx.Value("now").(notification.TimeFunc)()

	for _, n := range r.notifications {
		if n.UserID == userID && n.ReadAt == nil {
			n.ReadAt = &now
		}
	}

	r.unreadCount[userID] = 0

	return nil
}