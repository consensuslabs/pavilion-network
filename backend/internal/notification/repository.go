package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// Repository implements the types.NotificationRepository interface
type Repository struct {
	session      *gocql.Session
	logger       logger.Logger
	keyspace     string
	table        string
	maxRetries   int
	retryBaseDelay time.Duration
}

// NewRepository creates a new ScyllaDB notification repository
func NewRepository(session *gocql.Session, logger logger.Logger, keyspace string, config *types.ServiceConfig) *Repository {
	// Default retry settings
	maxRetries := 3
	retryBaseDelay := 500 * time.Millisecond
	
	// Use config values if provided
	if config != nil {
		// If ScyllaDB config has retry settings, use them
		if config.ScyllaDB.MaxRetries > 0 {
			maxRetries = config.ScyllaDB.MaxRetries
		}
		if config.ScyllaDB.RetryInterval > 0 {
			retryBaseDelay = config.ScyllaDB.RetryInterval
		}
	}
	
	return &Repository{
		session:      session,
		logger:       logger,
		keyspace:     keyspace,
		table:        "notifications",
		maxRetries:   maxRetries,
		retryBaseDelay: retryBaseDelay,
	}
}

// executeWithRetry executes a function with retry logic
func (r *Repository) executeWithRetry(operation string, fn func() error) error {
	var err error
	var attempts int
	
	fmt.Printf("DEBUG executeWithRetry: Starting operation %s with max attempts %d\n", 
		operation, r.maxRetries)
	
	for attempts = 0; attempts < r.maxRetries; attempts++ {
		if attempts > 0 {
			// Wait before retrying with exponential backoff
			backoff := time.Duration(math.Pow(2, float64(attempts))) * r.retryBaseDelay
			fmt.Printf("DEBUG executeWithRetry: Attempt %d failed, retrying after %v\n", 
				attempts, backoff)
			time.Sleep(backoff)
		}
		
		// Execute the function
		err = fn()
		if err == nil {
			// Success
			if attempts > 0 {
				fmt.Printf("DEBUG executeWithRetry: Operation %s succeeded after %d attempts\n", 
					operation, attempts+1)
			}
			return nil
		}
		
		// Log the error
		r.logger.LogWarn(fmt.Sprintf("Operation %s failed (attempt %d/%d): %v", 
			operation, attempts+1, r.maxRetries, err), nil)
		fmt.Printf("DEBUG executeWithRetry: Attempt %d failed with error: %v\n", 
			attempts+1, err)
		
		// Check if error is retryable
		if !r.isRetryableError(err) {
			fmt.Printf("DEBUG executeWithRetry: Error is not retryable, giving up\n")
			break
		}
	}
	
	// All retries failed
	r.logger.LogError(err, fmt.Sprintf("Operation %s failed after %d attempts", 
		operation, attempts))
	fmt.Printf("DEBUG executeWithRetry: All %d attempts failed for operation %s\n", 
		attempts, operation)
	
	return fmt.Errorf("operation %s failed after %d attempts: %w", 
		operation, attempts, err)
}

// isRetryableError determines if an error is retryable
func (r *Repository) isRetryableError(err error) bool {
	// Connection errors, timeouts, and temporary database errors are retryable
	if err == gocql.ErrNoConnections || err == gocql.ErrTimeoutNoResponse || err == gocql.ErrConnectionClosed {
		return true
	}
	
	// Check for other transient errors
	if err == context.DeadlineExceeded || err == context.Canceled {
		return true
	}
	
	// For other types of errors, don't retry
	return false
}

// Close closes the repository connection
func (r *Repository) Close() error {
	// ScyllaDB session is managed externally, so we don't close it here
	return nil
}

// Ping checks if the repository connection is healthy
func (r *Repository) Ping(ctx context.Context) error {
	// Simple query to verify connection
	var version string
	err := r.session.Query("SELECT release_version FROM system.local").WithContext(ctx).Scan(&version)
	
	if err != nil {
		r.logger.LogError(err, "Failed to ping ScyllaDB")
		return fmt.Errorf("failed to ping ScyllaDB: %w", err)
	}
	
	return nil
}

// encodeToJSONBytes serializes an object to JSON bytes
func encodeToJSONBytes(obj interface{}) ([]byte, error) {
	if obj == nil {
		return nil, nil
	}
	return json.Marshal(obj)
}

// decodeFromJSONBytes deserializes JSON bytes to an object
func decodeFromJSONBytes(data []byte, obj interface{}) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, obj)
}

// SaveNotification saves a notification to the database
func (r *Repository) SaveNotification(ctx context.Context, notification *types.Notification) error {
	// Generate a new UUID if not set
	if notification.ID == uuid.Nil {
		notification.ID = uuid.New()
	}
	
	// Set created_at if not set
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}
	
	fmt.Printf("DEBUG SaveNotification: Saving notification ID=%s, UserID=%s, Type=%s\n", 
		notification.ID, notification.UserID, notification.Type)
	
	// Convert metadata to map[string]string for ScyllaDB
	metadataMap := make(map[string]string)
	for k, v := range notification.Metadata {
		// Convert all values to strings
		metadataMap[k] = fmt.Sprintf("%v", v)
	}
	
	fmt.Printf("DEBUG SaveNotification: Metadata converted to map[string]string: %v\n", metadataMap)

	// Convert notification ID to binary for ScyllaDB
	idBytes, err := notification.ID.MarshalBinary()
	if err != nil {
		r.logger.LogError(err, "Error marshaling notification ID")
		fmt.Printf("DEBUG SaveNotification ERROR: Failed to marshal ID: %v\n", err)
		return fmt.Errorf("error marshaling notification ID: %w", err)
	}

	// Convert user ID to binary for ScyllaDB
	userIDBytes, err := notification.UserID.MarshalBinary()
	if err != nil {
		r.logger.LogError(err, "Error marshaling user ID")
		fmt.Printf("DEBUG SaveNotification ERROR: Failed to marshal user ID: %v\n", err)
		return fmt.Errorf("error marshaling user ID: %w", err)
	}

	// Execute the insert query with retry
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
	
	fmt.Printf("DEBUG SaveNotification: Executing query: %s\n", query)
	fmt.Printf("DEBUG SaveNotification: Using keyspace=%s, table=%s\n", r.keyspace, r.table)

	return r.executeWithRetry("SaveNotification", func() error {
		err := r.session.Query(query,
			idBytes,
			userIDBytes,
			notification.Type,
			notification.Content,
			metadataMap,
			notification.ReadAt,
			notification.CreatedAt,
		).WithContext(ctx).Exec()
		
		if err != nil {
			fmt.Printf("DEBUG SaveNotification ERROR: Query execution failed: %v\n", err)
		} else {
			fmt.Printf("DEBUG SaveNotification SUCCESS: Notification saved successfully\n")
		}
		
		return err
	})
}

// GetNotificationsByUserID returns a list of notifications for a user
func (r *Repository) GetNotificationsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*types.Notification, error) {
	// Validate input
	if limit <= 0 {
		limit = 10 // Default limit
	}
	
	if offset < 0 {
		offset = 0
	}

	// Convert user ID to binary for ScyllaDB
	userIDBytes, err := userID.MarshalBinary()
	if err != nil {
		r.logger.LogError(err, "Error marshaling user ID")
		return nil, fmt.Errorf("error marshaling user ID: %w", err)
	}

	// Construct the query
	query := fmt.Sprintf(`
		SELECT id, user_id, type, content, metadata, read_at, created_at
		FROM %s.notifications
		WHERE user_id = ?
		ORDER BY created_at DESC, id ASC
		LIMIT ?`, r.keyspace)

	fmt.Printf("DEBUG GetNotificationsByUserID: Executing query: %s with userID=%s, limit=%d\n", 
		query, userID, limit)
	
	// Initialize slice for results
	notifications := make([]*types.Notification, 0)
	
	// Execute the query with retry
	err = r.executeWithRetry("GetNotificationsByUserID", func() error {
		// Create a new iterator
		iter := r.session.Query(query, userIDBytes, limit).WithContext(ctx).Iter()
		
		// Prepare variables to scan into
		var (
			idBytes      []byte
			userIDBytes  []byte
			notifType    string
			content      string
			metadataMap  map[string]string
			readAt       *time.Time  // Change to *time.Time instead of gocql.UUID
			createdAt    time.Time
		)

		fmt.Printf("DEBUG GetNotificationsByUserID: Starting scan of results\n")
		
		// Scan rows
		for iter.Scan(&idBytes, &userIDBytes, &notifType, &content, &metadataMap, &readAt, &createdAt) {
			fmt.Printf("DEBUG GetNotificationsByUserID: Found notification with type=%s\n", notifType)
			
			// Convert ID from bytes to UUID
			var id uuid.UUID
			if err := id.UnmarshalBinary(idBytes); err != nil {
				r.logger.LogError(err, "Error unmarshaling notification ID")
				fmt.Printf("DEBUG GetNotificationsByUserID ERROR: Failed to unmarshal ID: %v\n", err)
				continue
			}

			// Convert user ID from bytes to UUID
			var uid uuid.UUID
			if err := uid.UnmarshalBinary(userIDBytes); err != nil {
				r.logger.LogError(err, "Error unmarshaling user ID")
				fmt.Printf("DEBUG GetNotificationsByUserID ERROR: Failed to unmarshal user ID: %v\n", err)
				continue
			}

			// Convert metadata map[string]string to map[string]interface{}
			metadata := make(map[string]interface{})
			for k, v := range metadataMap {
				metadata[k] = v
			}

			// Create notification
			notification := &types.Notification{
				ID:        id,
				UserID:    uid,
				Type:      notifType,
				Content:   content,
				Metadata:  metadata,
				CreatedAt: createdAt,
			}

			// Handle read_at if present (directly use the *time.Time value)
			if readAt != nil {
				notification.ReadAt = readAt
				fmt.Printf("DEBUG GetNotificationsByUserID: Notification was read at %v\n", *readAt)
			}

			// Add to results
			notifications = append(notifications, notification)
		}

		// Check for errors in iteration
		if err := iter.Close(); err != nil {
			r.logger.LogError(err, "Error iterating notifications")
			fmt.Printf("DEBUG GetNotificationsByUserID ERROR: Iterator error: %v\n", err)
			return err
		}

		fmt.Printf("DEBUG GetNotificationsByUserID: Found %d notifications\n", len(notifications))
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}

	return notifications, nil
}

// GetUnreadCount gets the count of unread notifications for a user
func (r *Repository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	// Convert userID to binary for ScyllaDB
	userIDBytes, err := userID.MarshalBinary()
	if err != nil {
		r.logger.LogError(err, "Error marshaling user ID")
		return 0, fmt.Errorf("error marshaling user ID: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s.%s 
		WHERE user_id = ? AND read_at IS NULL`,
		r.keyspace, r.table,
	)

	var count int
	var countErr error

	// Execute query with retry
	err = r.executeWithRetry("GetUnreadCount", func() error {
		if err := r.session.Query(query, userIDBytes).WithContext(ctx).Consistency(gocql.One).Scan(&count); err != nil {
			countErr = fmt.Errorf("failed to get unread count: %w", err)
			return countErr
		}
		return nil
	})

	if err != nil {
		r.logger.LogError(err, "Failed to get unread count")
		return 0, err
	}

	if countErr != nil {
		return 0, countErr
	}

	return count, nil
}

// MarkAsRead marks a notification as read
func (r *Repository) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	// Convert notificationID to binary for ScyllaDB
	notificationIDBytes, err := notificationID.MarshalBinary()
	if err != nil {
		r.logger.LogError(err, "Error marshaling notification ID")
		return fmt.Errorf("error marshaling notification ID: %w", err)
	}

	// First check if the notification exists
	checkQuery := fmt.Sprintf(`
		SELECT id 
		FROM %s.%s 
		WHERE id = ?`,
		r.keyspace, r.table,
	)
	
	var idBytes []byte
	var checkErr error

	// Check existence with retry
	err = r.executeWithRetry("CheckNotificationExists", func() error {
		if err := r.session.Query(checkQuery, notificationIDBytes).WithContext(ctx).Scan(&idBytes); err != nil {
			if err == gocql.ErrNotFound {
				checkErr = fmt.Errorf("notification not found")
				// Don't retry for NotFound errors
				return checkErr
			}
			checkErr = fmt.Errorf("failed to check notification existence: %w", err)
			return checkErr
		}
		return nil
	})

	if err != nil || checkErr != nil {
		if checkErr != nil {
			return checkErr
		}
		return err
	}

	now := time.Now()
	query := fmt.Sprintf(`
		UPDATE %s.%s 
		SET read_at = ? 
		WHERE id = ?`,
		r.keyspace, r.table,
	)

	// Execute update with retry
	return r.executeWithRetry("MarkAsRead", func() error {
		return r.session.Query(query, gocql.UUIDFromTime(now), notificationIDBytes).WithContext(ctx).Exec()
	})
}

// MarkAllAsRead marks all notifications for a user as read
func (r *Repository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	// Convert userID to binary for ScyllaDB
	userIDBytes, err := userID.MarshalBinary()
	if err != nil {
		r.logger.LogError(err, "Error marshaling user ID")
		return fmt.Errorf("error marshaling user ID: %w", err)
	}

	now := time.Now()
	query := fmt.Sprintf(`
		UPDATE %s.%s 
		SET read_at = ? 
		WHERE user_id = ? AND read_at IS NULL`,
		r.keyspace, r.table,
	)

	// Execute update with retry
	return r.executeWithRetry("MarkAllAsRead", func() error {
		return r.session.Query(query, gocql.UUIDFromTime(now), userIDBytes).WithContext(ctx).Consistency(gocql.One).Exec()
	})
}