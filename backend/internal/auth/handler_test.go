package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/auth"
	httpHandler "github.com/consensuslabs/pavilion-network/backend/internal/http"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/testhelper"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Response represents the API response structure
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents the error structure in responses
type Error struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Field   string `json:"field,omitempty"`
}

func setupTestRouter(t *testing.T) (*gin.Engine, *auth.Service, *gorm.DB) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Setup test DB
	db := testhelper.SetupTestDB(t)

	// Create auth config
	config := &auth.Config{
		JWT: struct {
			Secret          string
			AccessTokenTTL  time.Duration
			RefreshTokenTTL time.Duration
		}{
			Secret:          "test-secret-key",
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: time.Hour * 24 * 7,
		},
	}

	// Create services
	jwtService := auth.NewJWTService(config)
	refreshTokenRepo := auth.NewRefreshTokenRepository(db)
	authService := auth.NewService(db, jwtService, refreshTokenRepo, config)

	// Create router
	router := gin.New()
	router.Use(gin.Recovery())

	// Create response handler
	logger, _ := logger.NewLogger(&logger.Config{
		Level:       logger.Level("debug"),
		Format:      "console",
		Output:      "stdout",
		Development: true,
		File: struct {
			Enabled bool   `mapstructure:"enabled" yaml:"enabled"`
			Path    string `mapstructure:"path" yaml:"path"`
			Rotate  bool   `mapstructure:"rotate" yaml:"rotate"`
			MaxSize string `mapstructure:"maxSize" yaml:"maxSize"`
			MaxAge  string `mapstructure:"maxAge" yaml:"maxAge"`
		}{
			Enabled: false,
			Path:    "/var/log/pavilion",
			Rotate:  false,
			MaxSize: "100MB",
			MaxAge:  "7d",
		},
		Sampling: struct {
			Initial    int `mapstructure:"initial" yaml:"initial"`
			Thereafter int `mapstructure:"thereafter" yaml:"thereafter"`
		}{
			Initial:    100,
			Thereafter: 100,
		},
	})
	responseHandler := httpHandler.NewResponseHandler(logger)

	// Create auth handler and register routes
	authHandler := auth.NewHandler(authService, responseHandler)

	// Register all routes
	authHandler.RegisterRoutes(router)

	return router, authService, db
}

func TestRegisterAPI(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	// Test case 1: Successful registration
	t.Run("Successful Registration", func(t *testing.T) {
		reqBody := auth.RegisterRequest{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "Pass123!",
			Name:     "Test User",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if !response.Success {
			t.Error("Expected successful registration")
		}
	})

	// Test case 2: Invalid request (missing required fields)
	t.Run("Invalid Registration Request", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "testuser",
			// Missing required fields
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	// Test case 3: Duplicate registration
	t.Run("Duplicate Registration", func(t *testing.T) {
		reqBody := auth.RegisterRequest{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "Pass123!",
			Name:     "Test User",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestLoginAPI(t *testing.T) {
	router, authService, db := setupTestRouter(t)

	// Create a test user first
	user, err := authService.Register(auth.RegisterRequest{
		Username: "logintest",
		Email:    "login@example.com",
		Password: "Pass123!",
		Name:     "Login Test",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Mark email as verified
	user.EmailVerified = true
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Test case 1: Successful login
	t.Run("Successful Login", func(t *testing.T) {
		reqBody := auth.LoginRequest{
			Email:    "login@example.com",
			Password: "Pass123!",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if !response.Success {
			t.Error("Expected successful login")
		}

		// Verify response contains tokens
		data, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Error("Expected response data to be a map")
		}

		if data["accessToken"] == "" {
			t.Error("Expected access token in response")
		}

		if data["refreshToken"] == "" {
			t.Error("Expected refresh token in response")
		}
	})

	// Test case 2: Invalid credentials
	t.Run("Invalid Credentials", func(t *testing.T) {
		reqBody := auth.LoginRequest{
			Email:    "login@example.com",
			Password: "WrongPass123!",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}

func TestRefreshTokenAPI(t *testing.T) {
	router, authService, db := setupTestRouter(t)

	// Create and login a test user first
	user, err := authService.Register(auth.RegisterRequest{
		Username: "refreshtest",
		Email:    "refresh@example.com",
		Password: "Pass123!",
		Name:     "Refresh Test",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Mark email as verified
	user.EmailVerified = true
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Login to get tokens
	loginResp, err := authService.Login("refresh@example.com", "Pass123!")
	if err != nil {
		t.Fatalf("Failed to login test user: %v", err)
	}

	// Test case 1: Successful token refresh
	t.Run("Successful Token Refresh", func(t *testing.T) {
		reqBody := map[string]string{
			"refreshToken": loginResp.RefreshToken,
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if !response.Success {
			t.Error("Expected successful token refresh")
		}

		// Verify response contains new tokens
		data, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Error("Expected response data to be a map")
		}

		if data["accessToken"] == "" {
			t.Error("Expected new access token in response")
		}

		if data["refreshToken"] == "" {
			t.Error("Expected new refresh token in response")
		}
	})

	// Test case 2: Invalid refresh token
	t.Run("Invalid Refresh Token", func(t *testing.T) {
		reqBody := map[string]string{
			"refreshToken": "invalid-token",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}

func TestLogoutAPI(t *testing.T) {
	router, authService, db := setupTestRouter(t)

	// Create and login a test user first
	user, err := authService.Register(auth.RegisterRequest{
		Username: "logouttest",
		Email:    "logout@example.com",
		Password: "Pass123!",
		Name:     "Logout Test",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Mark email as verified
	user.EmailVerified = true
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Login to get tokens
	loginResp, err := authService.Login("logout@example.com", "Pass123!")
	if err != nil {
		t.Fatalf("Failed to login test user: %v", err)
	}

	// Test case 1: Successful logout
	t.Run("Successful Logout", func(t *testing.T) {
		reqBody := map[string]string{
			"refreshToken": loginResp.RefreshToken,
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/logout", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}

		var response Response
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if !response.Success {
			t.Error("Expected successful logout")
		}

		// Verify the refresh token is actually revoked
		_, err = authService.RefreshToken(loginResp.RefreshToken)
		if err == nil {
			t.Error("Expected error when using revoked refresh token")
		}
	})

	// Test case 2: Unauthorized logout (missing token)
	t.Run("Unauthorized Logout", func(t *testing.T) {
		reqBody := map[string]string{
			"refreshToken": loginResp.RefreshToken,
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/logout", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		// Intentionally not setting Authorization header
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	// Test case 3: Invalid refresh token
	t.Run("Invalid Refresh Token", func(t *testing.T) {
		reqBody := map[string]string{
			"refreshToken": "invalid-token",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/logout", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+loginResp.AccessToken)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}
