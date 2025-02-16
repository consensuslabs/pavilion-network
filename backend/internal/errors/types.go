package errors

// ValidationError represents a validation error with a field and message
type ValidationError struct {
	Field   string
	Message string
}

// StorageError represents an error during storage operations
type StorageError struct {
	Message string
	Cause   error
}

// ProcessingError represents an error during video processing
type ProcessingError struct {
	Message string
	Cause   error
}
