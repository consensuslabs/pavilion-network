# Authentication System Documentation

## Overview

This document outlines the current implementation of the authentication system in the Pavilion Network backend. For the complete implementation plan and future enhancements, see [Auth Implementation Plan](auth_implementation_plan.md). For testing details, refer to [Auth Testing Documentation](auth_testing.md).

## Current Implementation

### Core Components

#### 1. User Model
```go
type User struct {
    ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Username      string         `gorm:"unique;not null"`
    Email         string         `gorm:"unique;not null"`
    Password      string         `gorm:"not null"` // Stored as bcrypt hash
    Name          string         
    EmailVerified bool           `gorm:"default:false"`
    LastLoginAt   time.Time      
    CreatedAt     time.Time      
    UpdatedAt     time.Time      
    RefreshTokens []RefreshToken `gorm:"foreignKey:UserID"`
}
```

#### 2. Refresh Token Model
```go
type RefreshToken struct {
    ID        uuid.UUID  `gorm:"type:uuid;primary_key"`
    UserID    uuid.UUID  `gorm:"type:uuid;not null"`
    Token     string     `gorm:"unique;not null"`
    ExpiresAt time.Time  
    CreatedAt time.Time  
    RevokedAt *time.Time 
}
```

### JWT Implementation

The system uses a robust JWT (JSON Web Token) implementation with the following features:

1. **Token Types**:
   - Access Tokens (short-lived)
   - Refresh Tokens (long-lived)

2. **Token Claims Structure**:
```go
type TokenClaims struct {
    UserID string `json:"userId"`
    Email  string `json:"email"`
    jwt.RegisteredClaims
}
```

3. **Security Features**:
   - JTI (JWT ID) for unique token identification
   - Token expiration (TTL)
   - Token revocation support
   - Secure token storage in database

### Implemented Endpoints

1. **Login** (`POST /auth/login`):
   - Accepts email/username and password
   - Returns access and refresh tokens
   - Updates last login timestamp

2. **Logout** (`POST /auth/logout`):
   - Revokes refresh token
   - Requires authentication

3. **Token Refresh** (`POST /auth/refresh`):
   - Issues new access token
   - Rotates refresh token
   - Validates token expiration

4. **Register** (`POST /auth/register`):
   - Creates new user account
   - Validates input data
   - Hashes password securely

### Security Measures

1. **Password Security**:
   - Bcrypt hashing with cost factor 12
   - Password validation rules enforced

2. **Token Security**:
   - Unique JTI per token
   - Token revocation tracking
   - Secure token storage

3. **Authentication Middleware**:
   - Token validation
   - Route protection
   - Error handling

### Database Integration

1. **Tables**:
   - `users`
   - `refresh_tokens`

2. **Key Features**:
   - UUID primary keys
   - Unique constraints
   - Foreign key relationships
   - Timestamp tracking

### Error Handling

Standardized error responses:
```json
{
    "success": false,
    "error": {
        "code": "ERROR_CODE",
        "message": "User-friendly message"
    }
}
```

Common error codes:
- `AUTH001`: Invalid credentials
- `AUTH004`: Invalid token
- `AUTH005`: Token expired
- `AUTH009`: Email not verified

## Usage Examples

### Login Flow
```go
// 1. Create login request
loginReq := LoginRequest{
    Email:    "user@example.com",
    Password: "securepass"
}

// 2. Send request
POST /auth/login
Content-Type: application/json
{
    "email": "user@example.com",
    "password": "securepass"
}

// 3. Receive response
{
    "success": true,
    "data": {
        "user": {
            "id": "uuid",
            "email": "user@example.com",
            "username": "username"
        },
        "accessToken": "eyJ...",
        "refreshToken": "eyJ...",
        "expiresIn": 3600
    }
}
```

### Token Refresh Flow
```go
// 1. Send refresh request
POST /auth/refresh
Content-Type: application/json
{
    "refreshToken": "eyJ..."
}

// 2. Receive new tokens
{
    "success": true,
    "data": {
        "accessToken": "eyJ...",
        "refreshToken": "eyJ...",
        "expiresIn": 3600
    }
}
```

## Testing

For detailed testing information, including:
- Unit tests
- Integration tests
- Test configuration
- Common issues and solutions

Please refer to [Auth Testing Documentation](auth_testing.md).

## Future Enhancements

For planned improvements and future features, including:
- Multi-factor authentication
- OAuth integration
- Enhanced session management
- Role-based access control

Please refer to [Auth Implementation Plan](auth_implementation_plan.md#future-enhancements).

## References

- [JWT Specification](https://tools.ietf.org/html/rfc7519)
- [Auth Implementation Plan](auth_implementation_plan.md)
- [Auth Testing Documentation](auth_testing.md) 