package http

// APIResponse represents the standard API response format
// @Description Standard API response format
type APIResponse struct {
	// Indicates if the request was successful
	Success bool `json:"success" example:"true"`
	// Optional data returned by the API
	Data interface{} `json:"data,omitempty"`
	// Error information, if any
	Error *APIError `json:"error,omitempty"`
	// Optional message describing the response
	Message string `json:"message,omitempty" example:"Operation completed successfully"`
}

// APIError represents the error response structure
// @Description Error response structure
type APIError struct {
	// Error code identifying the type of error
	Code string `json:"code" example:"VALIDATION_ERROR"`
	// Human-readable error message
	Message string `json:"message" example:"Invalid input parameters"`
	// Optional field name for validation errors
	Field string `json:"field,omitempty" example:"email"`
}

// PaginationResponse represents a paginated response
// @Description Paginated response structure
type PaginationResponse struct {
	// Current page number
	Page int `json:"page" example:"1"`
	// Number of items per page
	PerPage int `json:"per_page" example:"10"`
	// Total number of items
	Total int64 `json:"total" example:"100"`
	// Total number of pages
	TotalPages int `json:"total_pages" example:"10"`
}

// ValidationError represents a field validation error
// @Description Validation error structure
type ValidationError struct {
	// Field that failed validation
	Field string `json:"field" example:"email"`
	// Validation error message
	Message string `json:"message" example:"Email is required"`
	// Validation rule that failed
	Rule string `json:"rule" example:"required"`
}

// HealthResponse represents the health check response
// @Description Health check response structure
type HealthResponse struct {
	// Status of the service
	Status string `json:"status" example:"healthy"`
	// Version of the service
	Version string `json:"version" example:"1.0.0"`
	// Uptime in seconds
	Uptime int64 `json:"uptime" example:"3600"`
}
