package auth_test

import (
	"testing"
	"time"

	"github.com/consensuslabs/pavilion-network/backend/internal/auth"
	"github.com/consensuslabs/pavilion-network/backend/testhelper"
	"github.com/google/uuid"
)

// Define DummyTokenService in this test package to satisfy auth.TokenService

type DummyTokenService struct{}

func (d DummyTokenService) GenerateAccessToken(user *auth.User) (string, error) {
	return "dummy_access_token", nil
}

func (d DummyTokenService) GenerateRefreshToken(user *auth.User) (string, error) {
	return "dummy_refresh_token", nil
}

func (d DummyTokenService) ValidateAccessToken(token string) (*auth.TokenClaims, error) {
	return &auth.TokenClaims{}, nil
}

func (d DummyTokenService) ValidateRefreshToken(token string) (*auth.TokenClaims, error) {
	return &auth.TokenClaims{}, nil
}

func TestRegisterAndLogin(t *testing.T) {
	db := testhelper.SetupTestDB(t)
	dummyTS := DummyTokenService{}
	// Use auth.NewService from the imported auth package
	authService := auth.NewService(db, dummyTS)

	// Test Registration
	regReq := auth.RegisterRequest{
		Username: "johndoe",
		Email:    "user@example.com",
		Password: "Pass123",
		Name:     "John Doe",
	}

	user, err := authService.Register(regReq)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if user.Username != regReq.Username {
		t.Errorf("expected username %s, got %s", regReq.Username, user.Username)
	}

	if user.EmailVerified != false {
		t.Errorf("expected EmailVerified false, got true")
	}

	// Test duplicate registration
	_, err = authService.Register(regReq)
	if err == nil {
		t.Errorf("expected error on duplicate registration, got nil")
	}

	// Update the user to mark email as verified
	user.EmailVerified = true
	user.LastLoginAt = time.Now()
	if err := db.Save(user).Error; err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	// Test Login with username
	loginResp, err := authService.Login("johndoe", "Pass123")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if loginResp.AccessToken != "dummy_access_token" {
		t.Errorf("expected dummy_access_token, got %s", loginResp.AccessToken)
	}

	if loginResp.RefreshToken != "dummy_refresh_token" {
		t.Errorf("expected dummy_refresh_token, got %s", loginResp.RefreshToken)
	}

	// Test Login with email
	loginResp, err = authService.Login("user@example.com", "Pass123")
	if err != nil {
		t.Fatalf("Login failed with email: %v", err)
	}

	// Test login with invalid password
	_, err = authService.Login("johndoe", "WrongPass")
	if err == nil {
		t.Errorf("expected error on invalid password, got nil")
	}
}

func TestLogout(t *testing.T) {
	db := testhelper.SetupTestDB(t)
	dummyTS := DummyTokenService{}
	authService := auth.NewService(db, dummyTS)

	dummyUserID := uuid.New()
	err := authService.Logout(dummyUserID, "dummy_refresh_token")
	if err == nil || err.Error() != "not implemented" {
		t.Errorf("expected 'not implemented' error, got %v", err)
	}
}

func TestRefreshToken(t *testing.T) {
	db := testhelper.SetupTestDB(t)
	dummyTS := DummyTokenService{}
	authService := auth.NewService(db, dummyTS)

	_, err := authService.RefreshToken("dummy_refresh_token")
	if err == nil || err.Error() != "not implemented" {
		t.Errorf("expected 'not implemented' error, got %v", err)
	}
}

func TestValidateToken(t *testing.T) {
	db := testhelper.SetupTestDB(t)
	dummyTS := DummyTokenService{}
	authService := auth.NewService(db, dummyTS)

	_, err := authService.ValidateToken("dummy_token")
	if err == nil || err.Error() != "not implemented" {
		t.Errorf("expected 'not implemented' error, got %v", err)
	}
}
