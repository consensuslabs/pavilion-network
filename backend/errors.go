package main

import "fmt"

// Error message constants
const (
	ErrMsgFileNotFound = "File not found"
	ErrMsgFileSize     = "File size exceeds maximum allowed size"
	ErrMsgFileType     = "File type not allowed"
	ErrMsgTitleLength  = "Title length must be between min and max length"
	ErrMsgDescLength   = "Description length exceeds maximum allowed length"
)

// ValidationError represents a validation error with a field and message
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// StorageError represents an error during storage operations
type StorageError struct {
	Message string
	Cause   error
}

func (e *StorageError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// ProcessingError represents an error during video processing
type ProcessingError struct {
	Message string
	Cause   error
}

func (e *ProcessingError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}