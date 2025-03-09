# Pavilion Network API Documentation

## Overview

Pavilion Network is a decentralized video platform that provides a robust API for video management, user authentication, and health monitoring. This document provides detailed information about the available API endpoints, authentication methods, and response formats.

## Base URL

```
http://localhost:8080
```

## Authentication

The API uses JWT (JSON Web Token) for authentication. Protected endpoints require a valid Bearer token in the Authorization header.

### Bearer Token Format
```
Authorization: Bearer <your_jwt_token>
```

## Common Response Format

All API endpoints follow a standard response format:

```json
{
  "success": true,
  "data": {},
  "error": null,
  "message": "Operation completed successfully"
}
```

### Error Response Format
```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "ERROR_CODE",
    "message": "Error description",
    "field": "field_name"
  }
}
```

## Endpoints

### Notifications

#### GET /api/v1/notifications/
Retrieve paginated list of notifications for the authenticated user.

**Query Parameters:**
- `limit`: Number of notifications to return (default: 10)
- `page`: Page number for pagination (default: 1)

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "userId": "550e8400-e29b-41d4-a716-446655440001",
      "type": "VIDEO_UPLOADED",
      "content": "Your video 'My awesome video' has been uploaded successfully",
      "metadata": {
        "videoId": "550e8400-e29b-41d4-a716-446655440002",
        "title": "My awesome video",
        "duration": "2m30s"
      },
      "createdAt": "2025-03-05T21:26:06Z"
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "userId": "550e8400-e29b-41d4-a716-446655440001",
      "type": "COMMENT_CREATED",
      "content": "Someone commented on your video",
      "metadata": {
        "commentId": "770e8400-e29b-41d4-a716-446655440000",
        "videoId": "550e8400-e29b-41d4-a716-446655440002",
        "contentPreview": "Great video!"
      },
      "readAt": "2025-03-05T21:30:00Z",
      "createdAt": "2025-03-05T21:20:00Z"
    }
  ],
  "message": "Notifications retrieved successfully"
}
```

#### GET /api/v1/notifications/unread-count
Get count of unread notifications for the authenticated user.

**Response:**
```json
{
  "success": true,
  "data": {
    "count": 5
  },
  "message": "Unread count retrieved successfully"
}
```

#### PUT /api/v1/notifications/:id/read
Mark a specific notification as read.

**Path Parameters:**
- `id`: Notification ID (UUID)

**Response:**
```json
{
  "success": true,
  "message": "Notification marked as read"
}
```

**Error Responses:**
- `NOT_FOUND`: Notification not found
- `INVALID_ID`: Invalid notification ID format

#### PUT /api/v1/notifications/read-all
Mark all notifications as read for the authenticated user.

**Response:**
```json
{
  "success": true,
  "message": "All notifications marked as read"
}
```

### Authentication

#### POST /auth/login
Authenticate user and obtain JWT tokens.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "your_password"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

#### POST /auth/register
Register a new user account.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "your_password",
  "name": "User Name"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "user_id": "123",
    "email": "user@example.com"
  }
}
```

#### POST /auth/refresh
Refresh an expired access token using a valid refresh token.

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

#### POST /auth/logout
Invalidate the current session. Requires authentication.

**Response:**
```json
{
  "success": true,
  "message": "Successfully logged out"
}
```

### Video Management

#### POST /video/upload
Upload a new video file. Requires authentication.

**Request Body (multipart/form-data):**
- `video`: Video file (Supported formats: .mp4, .mov, .avi, .webm)
- `title`: Video title (3-100 characters)
- `description`: Video description (max 500 characters)

**Response:**
```json
{
  "success": true,
  "data": {
    "fileId": "video123",
    "title": "My Video",
    "description": "Video description",
    "status": "pending"
  },
  "message": "Upload initiated successfully"
}
```

**Error Responses:**
- `ERR_NO_FILE`: No video file received
- `ERR_VALIDATION`: File size or format validation failed
- `UPLOAD_FAILED`: Failed to initialize upload

#### GET /video/watch
Stream a video by IPFS CID or file path.

**Query Parameters:**
- `cid`: IPFS Content ID (optional)
- `file`: Video file path (optional)

**Response:**
- Success: Video stream (video/mp4 or application/x-mpegURL)
- Error: 
```json
{
  "success": false,
  "error": {
    "code": "ERR_NO_PARAM",
    "message": "No 'cid' or 'file' parameter provided"
  }
}
```

#### GET /video/list
List all available videos.

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "fileId": "video123",
      "title": "My Video",
      "status": "completed",
      "currentPhase": "S3",
      "totalSize": 1048576,
      "totalProgress": 100,
      "ipfsProgress": {
        "bytesUploaded": 1048576,
        "percentage": 100,
        "startTime": "2024-02-22T14:20:18Z",
        "endTime": "2024-02-22T14:21:18Z",
        "duration": "1m0s"
      },
      "s3Progress": {
        "bytesUploaded": 1048576,
        "percentage": 100,
        "startTime": "2024-02-22T14:21:18Z",
        "endTime": "2024-02-22T14:22:18Z",
        "duration": "1m0s"
      },
      "completedAt": "2024-02-22T14:22:18Z"
    }
  ],
  "message": "Video list retrieved successfully"
}
```

#### GET /video/status/{fileId}
Get the current status of a video upload.

**Path Parameters:**
- `fileId`: Video file ID (required)

**Response:**
```json
{
  "success": true,
  "data": {
    "fileId": "video123",
    "title": "My Video",
    "status": "ipfs_uploading",
    "currentPhase": "IPFS",
    "totalSize": 1048576,
    "totalProgress": 45.5,
    "ipfsProgress": {
      "bytesUploaded": 476928,
      "percentage": 45.5,
      "startTime": "2024-02-22T14:20:18Z",
      "duration": "30s"
    },
    "estimatedDuration": "1m5s"
  },
  "message": "Video status retrieved successfully"
}
```

**Error Responses:**
- `INVALID_REQUEST`: File ID is required
- `NOT_FOUND`: Video upload not found

#### POST /video/transcode
Initiate video transcoding. Requires authentication.

**Request Body:**
```json
{
  "cid": "QmX..."
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "transcodes": [
      {
        "id": "trans123",
        "videoId": "video123",
        "filePath": "/path/to/transcoded/video",
        "fileCid": "QmY...",
        "format": "hls",
        "resolution": "720",
        "storageType": "ipfs",
        "type": "manifest",
        "createdAt": "2024-02-22T14:25:18Z"
      }
    ],
    "transcodeSegments": [
      {
        "id": "seg123",
        "transcodeId": "trans123",
        "filePath": "/path/to/segment",
        "fileCid": "QmZ...",
        "storageType": "ipfs",
        "sequence": 1,
        "duration": 10.0,
        "createdAt": "2024-02-22T14:25:28Z"
      }
    ]
  },
  "message": "Transcoding initiated successfully"
}
```

**Error Responses:**
- `ERR_INVALID_JSON`: Invalid JSON input
- `ERR_MISSING_CID`: Missing CID in request body

## Error Codes

Common error codes you might encounter:

- `VALIDATION_ERROR`: Invalid input parameters
- `AUTHENTICATION_ERROR`: Authentication failed
- `AUTHORIZATION_ERROR`: Insufficient permissions
- `NOT_FOUND`: Resource not found
- `INTERNAL_ERROR`: Internal server error
- `ERR_NO_FILE`: No file provided in upload request
- `ERR_VALIDATION`: File validation failed
- `UPLOAD_FAILED`: Upload initialization failed
- `ERR_NO_PARAM`: Missing required parameters
- `ERR_INVALID_JSON`: Invalid JSON payload
- `ERR_MISSING_CID`: Missing IPFS CID

## File Upload Limits

- Maximum video file size: 100MB
- Supported formats: .mp4, .mov, .avi, .webm
- Title length: 3-100 characters
- Description length: max 500 characters

## Development and Testing

For development and testing purposes:

1. Use the test environment:
   ```bash
   ENV=test go run .
   ```

2. Access Swagger UI:
   ```
   http://localhost:8080/swagger/index.html
   ```

## Support

For API support and questions, contact:
- Email: support@swagger.io
- Support URL: http://www.swagger.io/support 