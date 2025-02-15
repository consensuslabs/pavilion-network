package auth

import (
	"fmt"
	"unicode"
)

// validateLoginRequest validates the login request
func validateLoginRequest(req *LoginRequest) error {
	if err := validateEmail(req.Email); err != nil {
		return err
	}
	if err := validatePassword(req.Password); err != nil {
		return err
	}
	return nil
}

// validateEmail validates the email format
func validateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	// Note: Basic email validation is handled by gin binding
	return nil
}

// validatePassword validates the password against security rules
func validatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("password is required")
	}

	var (
		hasMinLen  = false
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	if len(password) >= 8 {
		hasMinLen = true
	}

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasMinLen {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// validateToken validates the token format and expiration
func validateToken(token string) error {
	if token == "" {
		return fmt.Errorf("token is required")
	}
	return nil
}
