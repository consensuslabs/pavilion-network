package types

import (
	"errors"
	"fmt"
)

// Common error types for the notification system
var (
	// ErrNotificationNotFound is returned when a notification is not found
	ErrNotificationNotFound = errors.New("notification not found")
	
	// ErrInvalidNotification is returned when a notification is invalid
	ErrInvalidNotification = errors.New("invalid notification")
	
	// ErrInvalidEventType is returned when an event type is invalid
	ErrInvalidEventType = errors.New("invalid event type")
	
	// ErrServiceDisabled is returned when the notification service is disabled
	ErrServiceDisabled = errors.New("notification service is disabled")
	
	// ErrConnectionFailed is returned when a connection to the message broker fails
	ErrConnectionFailed = errors.New("failed to connect to message broker")
)

// NotificationError represents an error in the notification system
type NotificationError struct {
	// The underlying error
	Err error
	// Additional context about the error
	Context string
	// The operation that caused the error
	Operation string
}

// Error returns the error message
func (e *NotificationError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("%s: %s: %v", e.Operation, e.Context, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Operation, e.Err)
}

// Unwrap returns the underlying error
func (e *NotificationError) Unwrap() error {
	return e.Err
}

// NewError creates a new NotificationError
func NewError(err error, operation, context string) *NotificationError {
	return &NotificationError{
		Err:       err,
		Context:   context,
		Operation: operation,
	}
} 