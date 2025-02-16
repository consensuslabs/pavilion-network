package http

// Response represents a standard API response structure
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents the error structure in responses
type Error struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Field   string `json:"field,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Response
	Pagination *Pagination `json:"pagination,omitempty"`
}

// Pagination contains pagination information
type Pagination struct {
	CurrentPage  int   `json:"currentPage"`
	TotalPages   int   `json:"totalPages"`
	TotalRecords int64 `json:"totalRecords"`
	PageSize     int   `json:"pageSize"`
}
