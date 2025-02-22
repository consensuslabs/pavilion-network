# Swagger Implementation Plan for Pavilion Network API

## Overview

This document outlines the plan for implementing Swagger/OpenAPI documentation for the Pavilion Network API. The implementation will use the `swag` package and follow a systematic approach to document all API endpoints.

## Current API Structure

### 1. Auth Endpoints (`/auth/*`)
- POST `/auth/login` - User login
- POST `/auth/register` - User registration
- POST `/auth/refresh` - Token refresh
- POST `/auth/logout` - User logout (protected)

### 2. Video Endpoints
- POST `/video/upload` - Upload video (protected)
- GET `/video/watch` - Watch video
- GET `/video/list` - List videos
- GET `/video/status/:fileId` - Get video status
- POST `/video/transcode` - Transcode video (protected)

### 3. Health Endpoint
- GET `/health` - Health check

## Implementation Steps

### Phase 1: Base Configuration Setup

1. **Update Main Swagger Configuration**
   ```go
   // @title           Pavilion Network API
   // @version         1.0
   // @description     API Server for Pavilion Network Application
   // @termsOfService  http://swagger.io/terms/
   
   // @contact.name   API Support
   // @contact.url    http://www.swagger.io/support
   // @contact.email  support@swagger.io
   
   // @license.name  Apache 2.0
   // @license.url   http://www.apache.org/licenses/LICENSE-2.0.html
   
   // @host      localhost:8080
   // @BasePath  /api/v1
   ```

2. **Security Definitions**
   ```go
   // @securityDefinitions.apikey BearerAuth
   // @in header
   // @name Authorization
   ```

### Phase 2: Common Models Documentation

1. **Response Models**
   ```go
   // Response represents the standard API response format
   type Response struct {
       Success bool        `json:"success"`
       Data    interface{} `json:"data,omitempty"`
       Error   *Error      `json:"error,omitempty"`
   }
   
   // Error represents the error response structure
   type Error struct {
       Code    string `json:"code"`
       Message string `json:"message"`
       Field   string `json:"field,omitempty"`
   }
   ```

2. **Auth Models**
   - LoginRequest
   - RegisterRequest
   - TokenResponse
   - UserResponse

3. **Video Models**
   - UploadRequest
   - VideoResponse
   - TranscodeRequest
   - StatusResponse

### Phase 3: API Documentation Implementation

1. **Auth Endpoints**
   - Document authentication flow
   - Add request/response examples
   - Include validation rules
   - Document error scenarios

2. **Video Endpoints**
   - Document file upload requirements
   - Add streaming information
   - Include format specifications
   - Document transcoding options

3. **Health Endpoint**
   - Document health check response
   - Include system status information

### Phase 4: Testing and Validation

1. **Swagger Generation**
   ```bash
   swag init -g main.go -o ./docs/api
   ```

2. **Validation Steps**
   - Verify all endpoints are documented
   - Check request/response examples
   - Validate security definitions
   - Test API explorer functionality

### Phase 5: Integration and UI Setup

1. **Swagger UI Integration**
   ```go
   router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
   ```

2. **Configuration**
   - Set up development/production URLs
   - Configure CORS for Swagger UI
   - Set up authentication for docs

## Implementation Order

1. **Stage 1: Foundation**
   - [ ] Update base Swagger configuration
   - [ ] Set up security definitions
   - [ ] Document common response formats
   - [ ] Configure Swagger UI

2. **Stage 2: Auth API**
   - [ ] Document login endpoint
   - [ ] Document registration endpoint
   - [ ] Document token refresh
   - [ ] Document logout endpoint

3. **Stage 3: Video API**
   - [ ] Document upload endpoint
   - [ ] Document watch endpoint
   - [ ] Document list endpoint
   - [ ] Document status endpoint
   - [ ] Document transcode endpoint

4. **Stage 4: Health API**
   - [ ] Document health check endpoint

5. **Stage 5: Testing & Refinement**
   - [ ] Generate and validate documentation
   - [ ] Test all endpoints in Swagger UI
   - [ ] Review and refine documentation
   - [ ] Update examples and descriptions

## Example Endpoint Documentation

```go
// @Summary Login user
// @Description Authenticate user and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} Response{data=LoginResponse} "Login successful"
// @Failure 400 {object} Response{error=Error} "Invalid request"
// @Failure 401 {object} Response{error=Error} "Authentication failed"
// @Router /auth/login [post]
```

## Best Practices

1. **Documentation Quality**
   - Use clear, concise descriptions
   - Include request/response examples
   - Document all possible responses
   - Add validation rules and constraints

2. **Security**
   - Clearly mark protected endpoints
   - Document authentication methods
   - Include token requirements
   - Specify permission levels

3. **Maintenance**
   - Keep documentation in sync with code
   - Update examples regularly
   - Version API documentation
   - Review and update periodically

## Next Steps

1. Begin with Stage 1 implementation
2. Review and update base configuration
3. Implement common models
4. Proceed with Auth API documentation
5. Continue with remaining stages

## Success Criteria

- All endpoints are properly documented
- Swagger UI is accessible and functional
- Authentication works in Swagger UI
- Documentation is clear and accurate
- Examples are provided for all endpoints
- Error responses are well documented 