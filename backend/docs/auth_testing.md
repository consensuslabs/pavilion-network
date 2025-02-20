# Authentication Testing Documentation

## Overview

This document outlines the testing strategy and implementation details for the authentication system in the Pavilion Network backend. The authentication tests cover user registration, login, token management, and session handling.

## Test Environment Setup

### Configuration

The authentication tests use a dedicated test environment with:

- Test database: `pavilion_test`
- Test JWT secret: `test-secret-key`
- Shorter token expiration times for testing
- Auto-migrations enabled

### Required Test Files

1. `config_test.yaml`:
   ```yaml
   auth:
     jwt:
       secret: "test-secret-key"
       accessTokenTTL: 15m
       refreshTokenTTL: 168h
   ```

2. `.env.test`:
   ```env
   ENV=test
   AUTO_MIGRATE=true
   FORCE_MIGRATION=true
   ```

## Test Cases

### 1. User Registration and Login (`TestRegisterAndLogin`)

Tests the complete flow of user registration and login:
- User registration with valid credentials
- Email verification status
- Login with registered credentials
- Access token generation
- Refresh token storage

```go
func TestRegisterAndLogin(t *testing.T) {
    // Setup test database
    db := testhelper.SetupTestDB(t)
    
    // Test registration
    user, err := service.Register(RegisterRequest{...})
    
    // Test login
    response, err := service.Login(email, password)
    
    // Verify tokens
    validateTokens(response.AccessToken, response.RefreshToken)
}
```

### 2. Logout Functionality (`TestLogout`)

Tests the logout process:
- Token validation before logout
- Refresh token revocation
- Multiple device logout
- Invalid token handling

Key test scenarios:
- Successful logout with valid tokens
- Logout with invalid tokens
- Logout with already revoked tokens
- Multiple concurrent logout attempts

### 3. Token Refresh (`TestRefreshToken`)

Tests the token refresh mechanism:
- Refresh token validation
- Access token generation
- Token expiration handling
- Invalid token detection

Test cases include:
- Valid refresh token usage
- Expired refresh token handling
- Invalid refresh token detection
- Multiple refresh attempts

### 4. Token Validation (`TestValidateToken`)

Tests token validation logic:
- Access token validation
- Refresh token validation
- Token expiration checks
- Invalid token detection

## Test Utilities

### JWT Test Helper

The test suite includes utilities for JWT token handling:

```go
func generateTestToken(userID string, expiry time.Duration) string
func validateTestToken(token string) (*TokenClaims, error)
```

## Best Practices

1. **Test Data Isolation**
   - Use unique test users for each test
   - Test data is automatically managed by the test environment
   - Don't rely on test order

2. **Token Management**
   - Use short-lived tokens in tests
   - Test both valid and invalid scenarios
   - Verify token claims thoroughly

3. **Error Handling**
   - Test all error conditions
   - Verify error messages
   - Check error types

4. **Database State**
   - Verify database state after operations
   - Check refresh token storage
   - Validate user status updates

## Common Test Scenarios

### 1. Registration Validation

```go
// Test username validation
t.Run("Invalid Username", func(t *testing.T) {...})

// Test email validation
t.Run("Invalid Email", func(t *testing.T) {...})

// Test password requirements
t.Run("Weak Password", func(t *testing.T) {...})
```

### 2. Login Security

```go
// Test brute force prevention
t.Run("Multiple Failed Attempts", func(t *testing.T) {...})

// Test account locking
t.Run("Locked Account", func(t *testing.T) {...})
```

### 3. Token Lifecycle

```go
// Test token expiration
t.Run("Expired Tokens", func(t *testing.T) {...})

// Test token revocation
t.Run("Revoked Tokens", func(t *testing.T) {...})
```

## Running Auth Tests

To run only authentication tests:

```bash
ENV=test go test ./internal/auth -v
```

To run a specific test:

```bash
ENV=test go test ./internal/auth -v -run TestRegisterAndLogin
```

## Troubleshooting

Common issues and solutions:

1. **Token Validation Failures**
   - Check JWT secret configuration
   - Verify token expiration times
   - Ensure correct token format

2. **Database Errors**
   - Verify test database connection
   - Check migration status
   - Ensure test environment is properly configured

3. **Concurrent Test Issues**
   - Use unique test data
   - Avoid shared state
   - Ensure proper test isolation

## Future Improvements

Planned enhancements to auth testing:

1. Add property-based testing for token validation
2. Implement fuzzing tests for input validation
3. Add performance testing for token operations
4. Enhance concurrent testing scenarios
5. Add integration tests with external auth providers 