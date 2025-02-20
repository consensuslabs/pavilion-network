# Authentication System Implementation Plan (Revised)

## Table of Contents
1. [Overview](#overview)
2. [Core Components](#core-components)
   - [User Model](#user-model)
   - [Password Complexity Requirements](#password-complexity-requirements)
3. [API Endpoints](#api-endpoints)
   - [User Registration](#user-registration)
   - [Email Verification / Confirmation](#email-verification--confirmation)
   - [User Login](#user-login)
   - [Token Refresh](#token-refresh)
   - [Logout](#logout)
   - [Password Reset Request & Reset](#password-reset)
   - [OAuth Login](#oauth-login)
4. [JWT Token System](#jwt-token-system)
5. [Security Measures](#security-measures)
   - [Password Hashing](#password-hashing)
   - [Rate Limiting & Middleware](#rate-limiting-and-middleware)
   - [Token Revocation and Session Management](#token-revocation-and-session-management)
6. [Database Schema](#database-schema)
7. [Login Flow](#login-flow)
8. [Implementation Steps](#implementation-steps)
9. [Testing Strategy](#testing-strategy)
10. [Monitoring and Logging](#monitoring-and-logging)
11. [Error Handling](#error-handling)
12. [Future Enhancements](#future-enhancements)
13. [Additional Future Considerations](#additional-future-considerations)

---

## Overview

This document defines the implementation plan for a robust, secure, and extendable authentication system. It covers user management, secure session and token management, email confirmation, OAuth login flows, detailed testing, and guidelines for future enhancements while following industry best practices.

---

## Core Components

### User Model

```go
// User represents a system user with embedded refresh tokens.
type User struct {
    ID            uuid.UUID    `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Username      string       `gorm:"unique;not null"`
    Email         string       `gorm:"unique;not null"`
    Password      string       `gorm:"not null"` // Stored as bcrypt hash
    Name          string
    EmailVerified bool         `gorm:"default:false"` // Indicates if the user's email has been confirmed.
    LastLoginAt   time.Time
    CreatedAt     time.Time
    UpdatedAt     time.Time
    RefreshTokens []RefreshToken
}
```

### Password Complexity Requirements

- **Minimum length:** 6 characters  
- **Maximum length:** 72 characters (due to bcrypt constraints)  
- **Content:** Must include at least one letter (upper or lower case) and one number  
- **Future Enhancements:** Consider integrating checks for special characters and support for passphrases.

---

## API Endpoints

### User Registration

```
POST /auth/register
```

**Request:**
```json
{
    "username": "johndoe",
    "email": "user@example.com",
    "password": "Pass123",
    "name": "John Doe"
}
```

**Response:**
```json
{
    "success": true,
    "data": {
        "user": {
            "id": "550e8400-e29b-41d4-a716-446655440000",
            "username": "johndoe",
            "email": "user@example.com",
            "name": "John Doe"
        }
    },
    "message": "Registration successful. Please verify your email to activate your account."
}
```

> **Note:** After successful registration, the system will generate an email verification token and dispatch an email with a confirmation link.

### Email Verification / Confirmation

#### Verify Email

```
GET /auth/verify?token=<verification_token>
```

- **Description:**  
  Users click the link in their email to verify their account.
- **Response:**
  ```json
  {
      "success": true,
      "message": "Email successfully verified."
  }
  ```

#### Resend Verification Email

```
POST /auth/verify/resend
```

**Request:**
```json
{
    "email": "user@example.com"
}
```

**Response:**
```json
{
    "success": true,
    "message": "Verification email resent. Please check your inbox."
}
```

### User Login

```
POST /auth/login
```

**Request:**
```json
{
    "identifier": "johndoe", // Can be username or email
    "password": "Pass123"
}
```

**Response:**
```json
{
    "success": true,
    "data": {
        "user": {
            "id": "550e8400-e29b-41d4-a716-446655440000",
            "username": "johndoe",
            "email": "user@example.com",
            "name": "John Doe",
            "emailVerified": true
        },
        "tokens": {
            "accessToken": "eyJ...",
            "refreshToken": "eyJ...",
            "expiresIn": 3600,
            "tokenType": "Bearer"
        }
    },
    "message": "Login successful."
}
```

> **Important:** Login should only be permitted for users whose email is verified. If the email is not confirmed, return an appropriate error message.

### Token Refresh

```
POST /auth/refresh
```

**Request:**
```json
{
    "refreshToken": "eyJ..."
}
```

**Response:**
```json
{
    "success": true,
    "data": {
        "tokens": {
            "accessToken": "eyJ...",
            "refreshToken": "eyJ...",
            "expiresIn": 3600
        }
    }
}
```

### Logout

```
POST /auth/logout
```

**Request:**
```json
{
    "refreshToken": "eyJ..."
}
```

**Response:**
```json
{
    "success": true,
    "message": "Logged out successfully."
}
```

### Password Reset

#### Request Reset

```
POST /auth/password/reset-request
```

**Request:**
```json
{
    "email": "user@example.com"
}
```

**Response:**
```json
{
    "success": true,
    "message": "Password reset instructions sent to email."
}
```

#### Perform Reset

```
POST /auth/password/reset
```

**Request:**
```json
{
    "token": "reset_token",
    "newPassword": "NewSecurePass123!"
}
```

**Response:**
```json
{
    "success": true,
    "message": "Password reset successful."
}
```

### OAuth Login

This section describes the OAuth-based login flow for third-party providers: Google, Apple, and Ethereum. The OAuth integration allows users to authenticate using their existing accounts (or wallet for Ethereum) without creating another password. Below are the details for each provider.

#### 1. Google OAuth

**Initiate Google OAuth Flow**  
```
GET /auth/oauth/google
```
- **Description:**  
  When a client accesses this endpoint, the system redirects the user to Google's OAuth 2.0 authorization URL.
- **Required Parameters:**  
  - `client_id`: Your Google OAuth Client ID.
  - `redirect_uri`: E.g., `https://yourapp.com/auth/oauth/google/callback`
  - `scope`: Recommended scopes are `openid`, `email`, and `profile`.
  - `response_type`: Typically `code`.
  - `state`: A random string to prevent CSRF.

**Google OAuth Callback**  
```
GET /auth/oauth/google/callback
```
- **Flow:**  
  The authorization code returned by Google is exchanged for an access token and an ID token. The ID token contains user information (email, name, subject id).  
- **Actions:**  
  - Validate the authorization code and exchange it with Google's OAuth endpoint for tokens.
  - Verify the ID token (using Google's public keys) and extract user information.
  - Check if the user's email is already registered. If not, create a new user account.
  - Issue local access and refresh tokens and update the session details.
- **Response:**  
  Returns a JSON object with user data and the generated tokens.

#### 2. Apple OAuth

**Initiate Apple OAuth Flow**  
```
GET /auth/oauth/apple
```
- **Description:**  
  When a user accesses this endpoint, the system redirects the client to Apple's Sign In with Apple portal.
- **Required Parameters:**  
  - `client_id`: Your Apple Services ID.
  - `redirect_uri`: E.g., `https://yourapp.com/auth/oauth/apple/callback`
  - `response_type`: Typically `code id_token`.
  - `scope`: Typically `name` and `email`.
  - `state`: A random, secure string.
  - **Note:** Apple requires a client secret in JWT format generated using your team identifier, key identifier, and a private key.

**Apple OAuth Callback**  
```
GET /auth/oauth/apple/callback
```
- **Flow:**  
  - The callback endpoint receives an authorization code and (optionally) an ID token.
  - Validate and decode the ID token to retrieve user data.
  - Apple may only provide the user email on the very first authentication attempt.
  - Follow similar steps as with Google OAuth:
    - Validate token authenticity using Apple's public keys.
    - Create or update the user account accordingly.
    - Issue local access and refresh tokens, then begin the session.
- **Response:**  
  Returns a JSON object with the user's data and local tokens.

#### 3. Ethereum OAuth (Sign-In with Ethereum)

**Initiate Ethereum OAuth Flow**  
```
GET /auth/oauth/ethereum
```
- **Description:**  
  Unlike standard OAuth providers, Ethereum login is typically performed via a "Sign-In with Ethereum" mechanism which leverages wallet signature challenges.
- **Flow:**  
  - Generate and send a unique challenge (nonce) to the client.
  - The client (using a browser wallet like MetaMask) signs the challenge with their private key.
  - The client then sends the signature along with their Ethereum address to the server for verification.

**Challenge and Verification Endpoints:**

- **Challenge Request:**  
  ```
  GET /auth/oauth/ethereum/challenge
  ```
  - **Response:** Returns a JSON object with a unique nonce to be signed.

- **Challenge Verification:**  
  ```
  POST /auth/oauth/ethereum/verify
  ```
  - **Request Payload:**
    ```json
    {
       "address": "0x1234567890abcdef...",
       "signature": "0xabc...",
       "nonce": "randomly_generated_nonce"
    }
    ```
  - **Actions:**  
    - Verify the signature using the provided Ethereum address and nonce.
    - If verification succeeds, check if the Ethereum address is linked to an existing account; if not, optionally create an account.
    - Issue local access and refresh tokens.
- **Response:**  
  Returns a JSON object with user information, local tokens, and session details once the signature is verified.

---

## JWT Token System

Define a dedicated token claim structure:

```go
// TokenClaims contains the payload for JWTs used in the system.
type TokenClaims struct {
    UserID    uuid.UUID `json:"userId"`
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    TokenType string    `json:"tokenType"` // "access" or "refresh"
    jwt.RegisteredClaims
}
```

---

## Security Measures

### Password Hashing

Use bcrypt with a cost factor of 12:

```go
func hashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
    return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
```

### Rate Limiting and Middleware

Implement rate limiting with Redis (or a similar system) for:
- **Login attempts:** 5 per minute per IP  
- **Password resets and verification requests:** 3 per hour per email  

```go
func rateLimiter(key string, limit int, window time.Duration) bool {
    current, err := redisClient.Incr(key).Result()
    if err != nil {
        return false
    }
    if current == 1 {
        redisClient.Expire(key, window)
    }
    return current <= limit
}
```

*Integrate this as middleware to protect sensitive endpoints.*

### Token Revocation and Session Management

- **Access tokens** expire after 1 hour.
- **Refresh tokens** are single-use.
- **Logout** invalidates all tokens for the session.
- Store tokens securely, e.g., via HTTP-only cookies for web clients or secure storage when using APIs.

---

## Database Schema

### Users Table

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username STRING NOT NULL UNIQUE,
    email STRING NOT NULL UNIQUE,
    password STRING NOT NULL,
    name STRING,
    email_verified BOOL DEFAULT false,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
```

### Refresh Tokens Table

```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    token STRING NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### Password Reset Tokens Table

```sql
CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    token STRING NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id)
);
```

*Note: Adjust SQL dialect and syntax as needed (e.g., for CockroachDB specifics).*

---

## Login Flow

1. **Input Validation:**  
   - Validate the identifier (username or email), ensuring proper formats.
2. **User Lookup:**  
   - Fetch the user record using the provided identifier.
3. **Email Verification Check:**  
   - If the user's email is not verified, return an error prompting the user to confirm their email.
4. **Password Verification:**  
   - Check the input password against the stored bcrypt hash.
5. **Token Generation:**  
   - Generate new access and refresh tokens.
6. **Timestamp Update:**  
   - Update the user's `last_login_at` timestamp.
7. **Response:**  
   - Return the updated user data and generated tokens.

```go
func (s *Service) Login(identifier, password string) (*LoginResponse, error) {
    var user User
    
    err := s.db.Where("email = ? OR username = ?", identifier, identifier).First(&user).Error
    if err != nil {
        return nil, ErrInvalidCredentials
    }

    if !user.EmailVerified {
        return nil, ErrEmailNotVerified
    }

    if !checkPasswordHash(password, user.Password) {
        return nil, ErrInvalidCredentials
    }

    // Generate tokens, update login timestamp, and return response
    // ...
}
```

---

## Implementation Steps

1. **Database Setup:**  
   - Write migration scripts to create users, refresh tokens, password reset tokens, and any additional tables required for email verification.
   - Apply indexes and foreign key constraints.
2. **Core Authentication Modules:**  
   - Develop password hashing utilities, JWT token generation, verification, refresh token rotation, and email verification logic.
3. **API Endpoints:**  
   - Implement registration, email verification, login, token refresh, logout, password reset, and OAuth endpoints.
   - Ensure endpoints are protected by input and rate-limit validation middleware.
4. **Middleware Integration:**  
   - Add authentication, rate-limiting, error handling, and request validation middleware.
5. **Email Service:**  
   - Configure email templates and integration with an email service for sending verification and password reset emails.
6. **OAuth Integration:**  
   - Implement OAuth flows for Google, Apple, and Ethereum.  
   - Ensure proper provider configuration, scope management, and callback handling.
7. **Configuration:**  
   - Centralize configuration (e.g., secret keys, expiry durations) in environment variables or a configuration file.

---

## Testing Strategy

1. **Unit Tests:**
   - Validate password hashing and comparison functions.
   - Test token generation and verification.
   - Test individual middleware components including rate limiting.
   - Test email verification token generation and validation.
   - Ensure OAuth handlers correctly parse provider responses.

2. **Integration Tests:**
   - Simulate full flows: registration, email verification, login, token refresh, logout, and password reset.
   - Test OAuth flows for Google, Apple, and Ethereum, including token exchange and callback handling.
   - Verify that non-verified accounts cannot log in.

3. **Security Tests:**
   - Validate password strength and ensure proper hashing.
   - Test token handling to avoid vulnerabilities such as token reuse.
   - Perform tests for SQL injection, XSS, and other common vulnerabilities.
   - Ensure appropriate authorization for OAuth endpoints.

4. **Performance Tests:**
   - Ensure that authentication endpoints (including email verification and OAuth redirects) handle load gracefully.
   - Test rate-limiting middleware performance under simulated high traffic.

5. **End-to-End Tests:**
   - Simulate user registration through email verification.
   - Simulate third-party logins via OAuth (Google, Apple, Ethereum).
   - Validate error states (e.g., expired tokens, invalid verification links).

---

## Monitoring and Logging

1. **Logged Events:**
   - Successful and failed login attempts.
   - Email verification requests and completions.
   - Password reset requests and completions.
   - OAuth login events and potential errors.
   - Rate limit triggers and account lockouts.

2. **Metrics to Track:**
   - Authentication success rates.
   - Average response times and system latency.
   - Token usage and expiration metrics.
   - Email delivery and verification metrics.

---

## Error Handling

**Standard Error Response Format:**
```json
{
    "success": false,
    "error": {
        "code": "ERROR_CODE",
        "message": "User-friendly message",
        "details": "Optional technical details"
    }
}
```

**Sample Error Codes:**
- `AUTH001`: Invalid credentials.
- `AUTH002`: Account locked.
- `AUTH003`: Rate limit exceeded.
- `AUTH004`: Invalid token or session.
- `AUTH005`: Token expired.
- `AUTH006`: Invalid password format.
- `AUTH007`: Email already registered.
- `AUTH008`: Invalid reset token.
- `AUTH009`: Email not verified.

---

## Future Enhancements

1. **Multi-Factor Authentication (MFA):**
   - SMS-based or authenticator app verification.
2. **OAuth Integration Enhancements:**
   - Expand support to include additional providers.
3. **Advanced Session Management:**
   - Implement device fingerprinting and per-device session control.
4. **Extended Security Features:**
   - Implement account lockout policies after multiple failed attempts.
   - Enhance logging and anomaly detection for suspicious activity.
5. **Role-Based Access Control (RBAC):**
   - Incorporate user permissions and role management for secure resource access.

---

## Additional Future Considerations

*The following items are currently outside the scope of the initial implementation but should be considered for future enhancements:*

- **Enhanced Token Lifecycle Management:**
  - Rotate refresh tokens on every use.
  - Define strict TTLs for access and refresh tokens.
  - Revoke all tokens on key events (e.g., password change, email re-verification).

- **Enhanced Secret & Configuration Management:**
  - Utilize secret management tools or environment variables for securing all sensitive credentials (e.g., OAuth secrets, JWT keys).
  - Maintain provider-specific secrets securely (e.g., Apple's JWT-based client secret).

- **Advanced Security Mechanisms:**
  - Implement robust CSRF protection, including state verification in OAuth flows.
  - Introduce nonce or anti-replay tokens for sensitive endpoints (e.g., password resets).

- **Client-Side Security Enhancements:**
  - Integrate PKCE (Proof Key for Code Exchange) for OAuth flows in mobile and public clients.
  - Recommend best practices for secure token storage on clients (e.g., secure, HTTP-only cookies for web apps).

- **Comprehensive Logging and Monitoring Enhancements:**
  - Develop audit trails for critical actions (login attempts, token refreshes, etc.) to aid in anomaly detection.
  - Set up alerts for abnormal behaviors such as failed logins or repeated token misuse.

- **Fallbacks and Error Handling Improvements:**
  - Plan for graceful degradation in case of OAuth provider downtime.
  - Provide clear error messages and retry mechanisms if third-party authentication fails.

- **Compliance and Documentation:**
  - Ensure compliance with relevant privacy and data protection regulations (e.g., GDPR, HIPAA).
  - Expand the documentation with detailed API usage examples, sequence diagrams, and regulatory compliance notes.

- **User Account Management Process Improvements:**
  - Define processes for account deletion/deactivation and data retention.
  - Establish guidelines for handling stale or inactive accounts.

---

By keeping these additional future considerations in mind, you can build on the current implementation to further enhance the security, scalability, and usability of your authentication system as the project matures. 