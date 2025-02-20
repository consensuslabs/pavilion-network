package auth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/auth"
	"github.com/consensuslabs/pavilion-network/backend/testhelper"
	"github.com/google/uuid"
)

func TestRegisterAndLogin(t *testing.T) {
	fmt.Printf("\n=== Starting TestRegisterAndLogin ===\n")

	db := testhelper.SetupTestDB(t)

	config := &auth.Config{
		JWT: struct {
			Secret          string
			AccessTokenTTL  time.Duration
			RefreshTokenTTL time.Duration
		}{
			Secret:          "test-secret-" + uuid.New().String(),
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: time.Hour * 24 * 7,
		},
	}

	fmt.Printf("Test config initialized with secret: %s\n", config.JWT.Secret)

	// Use real JWT service instead of dummy
	jwtService := auth.NewJWTService(config)
	refreshTokenRepo := auth.NewRefreshTokenRepository(db)
	authService := auth.NewService(db, jwtService, refreshTokenRepo, config)

	// Test Registration
	regReq := auth.RegisterRequest{
		Username: "testuser1",
		Email:    "test1@example.com",
		Password: "Pass123!",
		Name:     "Test User 1",
	}

	fmt.Printf("Attempting to register user: %s (%s)\n", regReq.Username, regReq.Email)

	user, err := authService.Register(regReq)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	fmt.Printf("User registered successfully with ID: %s\n", user.ID)

	if user.Username != regReq.Username {
		t.Errorf("expected username %s, got %s", regReq.Username, user.Username)
	}

	if user.EmailVerified {
		t.Errorf("expected EmailVerified false, got true")
	}

	// Verify user was created with correct data
	if user.Email != regReq.Email {
		t.Errorf("expected email %s, got %s", regReq.Email, user.Email)
	}

	// Test duplicate registration
	_, err = authService.Register(regReq)
	if err == nil {
		t.Errorf("expected error on duplicate registration, got nil")
	}

	// Update the user to mark email as verified
	user.EmailVerified = true
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	fmt.Printf("User email marked as verified\n")

	// Test Login with username
	fmt.Printf("Attempting login with username: %s\n", regReq.Username)
	loginResp, err := authService.Login(regReq.Username, regReq.Password)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	fmt.Printf("Login successful, received tokens - Access: %s, Refresh: %s\n",
		loginResp.AccessToken[:10]+"...",
		loginResp.RefreshToken[:10]+"...")

	// Verify we got real JWT tokens, not dummy ones
	if loginResp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}

	if loginResp.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}

	// Test Login with email
	fmt.Printf("Attempting login with email: %s\n", regReq.Email)
	loginResp, err = authService.Login(regReq.Email, regReq.Password)
	if err != nil {
		t.Fatalf("Login failed with email: %v", err)
	}

	fmt.Printf("Login successful with email\n")

	// Test login with invalid password
	_, err = authService.Login(regReq.Username, "WrongPass")
	if err == nil {
		t.Errorf("expected error on invalid password, got nil")
	}

	fmt.Printf("=== TestRegisterAndLogin Completed ===\n")
}

func TestLogout(t *testing.T) {
	db := testhelper.SetupTestDB(t)

	config := &auth.Config{
		JWT: struct {
			Secret          string
			AccessTokenTTL  time.Duration
			RefreshTokenTTL time.Duration
		}{
			Secret:          "test-secret-" + uuid.New().String(),
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: time.Hour * 24 * 7,
		},
	}

	// Use real JWT service instead of dummy
	jwtService := auth.NewJWTService(config)
	refreshTokenRepo := auth.NewRefreshTokenRepository(db)
	authService := auth.NewService(db, jwtService, refreshTokenRepo, config)

	// Create a test user with unique credentials
	regReq := auth.RegisterRequest{
		Username: "testuser2",
		Email:    "test2@example.com",
		Password: "Pass123!",
		Name:     "Test User 2",
	}

	user, err := authService.Register(regReq)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Verify user was created with correct data
	if user.ID.String() == "" {
		t.Error("expected non-empty user ID")
	}

	// Mark email as verified
	user.EmailVerified = true
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Login to get a real refresh token
	loginResp, err := authService.Login(regReq.Username, regReq.Password)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// Test logout with the real refresh token
	err = authService.Logout(user.ID, loginResp.RefreshToken)
	if err != nil {
		t.Errorf("expected successful logout, got error: %v", err)
	}

	// Verify token was actually revoked
	_, err = refreshTokenRepo.GetByToken(loginResp.RefreshToken)
	if err == nil {
		t.Error("expected error getting revoked token")
	}
}

func TestRefreshToken(t *testing.T) {
	db := testhelper.SetupTestDB(t)

	config := &auth.Config{
		JWT: struct {
			Secret          string
			AccessTokenTTL  time.Duration
			RefreshTokenTTL time.Duration
		}{
			Secret:          "test-secret-" + uuid.New().String(),
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: time.Hour * 24 * 7,
		},
	}

	// Use real JWT service instead of dummy
	jwtService := auth.NewJWTService(config)
	refreshTokenRepo := auth.NewRefreshTokenRepository(db)
	authService := auth.NewService(db, jwtService, refreshTokenRepo, config)

	// Create a test user with unique credentials
	regReq := auth.RegisterRequest{
		Username: "testuser3",
		Email:    "test3@example.com",
		Password: "Pass123!",
		Name:     "Test User 3",
	}

	user, err := authService.Register(regReq)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Mark email as verified
	user.EmailVerified = true
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Login to get initial tokens
	loginResp, err := authService.Login(regReq.Username, regReq.Password)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// Store the original tokens for comparison
	originalAccessToken := loginResp.AccessToken
	originalRefreshToken := loginResp.RefreshToken

	// Verify initial refresh token is stored
	storedToken, err := refreshTokenRepo.GetByToken(originalRefreshToken)
	if err != nil {
		t.Fatalf("Failed to get stored refresh token: %v", err)
	}
	if storedToken == nil {
		t.Fatal("Initial refresh token not found in database")
	}

	// Test refresh token
	refreshResp, err := authService.RefreshToken(originalRefreshToken)
	if err != nil {
		t.Errorf("Token refresh failed: %v", err)
	}

	// Verify response
	if refreshResp.AccessToken == "" {
		t.Error("Expected non-empty access token after refresh")
	}
	if refreshResp.AccessToken == originalAccessToken {
		t.Error("Expected new access token to be different from original")
	}
	if refreshResp.RefreshToken != originalRefreshToken {
		t.Error("Expected refresh token to remain the same")
	}

	// Verify the original refresh token is still valid and in database
	storedToken, err = refreshTokenRepo.GetByToken(originalRefreshToken)
	if err != nil {
		t.Fatalf("Failed to get refresh token after refresh: %v", err)
	}
	if storedToken == nil {
		t.Fatal("Refresh token should still be in database")
	}
	if storedToken.RevokedAt != nil {
		t.Error("Refresh token should not be revoked")
	}

	// Count refresh tokens for this user - should only be one
	var tokenCount int64
	if err := db.Model(&auth.RefreshToken{}).Where("user_id = ?", user.ID).Count(&tokenCount).Error; err != nil {
		t.Fatalf("Failed to count refresh tokens: %v", err)
	}
	if tokenCount != 1 {
		t.Errorf("Expected exactly 1 refresh token, got %d", tokenCount)
	}

	// Test error cases
	t.Run("Invalid refresh token", func(t *testing.T) {
		_, err := authService.RefreshToken("invalid-token")
		if err == nil {
			t.Error("Expected error with invalid refresh token")
		}
	})

	t.Run("Expired refresh token", func(t *testing.T) {
		// Manually expire the token in the database
		if err := db.Model(&auth.RefreshToken{}).
			Where("token = ?", originalRefreshToken).
			Update("expires_at", time.Now().Add(-time.Hour)).Error; err != nil {
			t.Fatalf("Failed to expire token: %v", err)
		}

		_, err := authService.RefreshToken(originalRefreshToken)
		if err == nil {
			t.Error("Expected error with expired refresh token")
		}
	})
}

func TestValidateToken(t *testing.T) {
	db := testhelper.SetupTestDB(t)

	config := &auth.Config{
		JWT: struct {
			Secret          string
			AccessTokenTTL  time.Duration
			RefreshTokenTTL time.Duration
		}{
			Secret:          "test-secret-" + uuid.New().String(),
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: time.Hour * 24 * 7,
		},
	}

	// Use real JWT service
	jwtService := auth.NewJWTService(config)
	refreshTokenRepo := auth.NewRefreshTokenRepository(db)
	authService := auth.NewService(db, jwtService, refreshTokenRepo, config)

	// Create a test user with unique credentials
	regReq := auth.RegisterRequest{
		Username: "testuser4",
		Email:    "test4@example.com",
		Password: "Pass123!",
		Name:     "Test User 4",
	}

	user, err := authService.Register(regReq)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Verify user was created with correct data
	if user.ID.String() == "" {
		t.Error("expected non-empty user ID")
	}

	// Mark email as verified
	user.EmailVerified = true
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Login to get a real token
	loginResp, err := authService.Login(regReq.Username, regReq.Password)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// Test token validation
	claims, err := authService.ValidateToken(loginResp.AccessToken)
	if err != nil {
		t.Errorf("expected successful token validation, got error: %v", err)
	}

	if claims == nil {
		t.Error("expected non-nil claims")
	}

	if claims.Email != regReq.Email {
		t.Errorf("expected email %s, got %s", regReq.Email, claims.Email)
	}

	if claims.UserID != user.ID.String() {
		t.Errorf("expected user ID %s, got %s", user.ID.String(), claims.UserID)
	}
}
