package errors

import "fmt"

// Error method implementation for ValidationError
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Error method implementation for StorageError
func (e *StorageError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Error method implementation for ProcessingError
func (e *ProcessingError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// New creates a new ValidationError
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// NewStorageError creates a new StorageError
func NewStorageError(message string, cause error) *StorageError {
	return &StorageError{
		Message: message,
		Cause:   cause,
	}
}

// NewProcessingError creates a new ProcessingError
func NewProcessingError(message string, cause error) *ProcessingError {
	return &ProcessingError{
		Message: message,
		Cause:   cause,
	}
}
