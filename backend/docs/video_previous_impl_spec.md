# Previous Video Implementation Specification

## Overview
This document outlines the previous implementation of the video API endpoints. The implementation used asynchronous processing with separate endpoints for upload and transcoding, featuring detailed progress tracking and complex status monitoring.

## Authentication
All video upload and transcode endpoints required authentication using JWT tokens (BearerAuth).

## API Endpoints

### 1. POST /video/upload
- **Authentication**: Required (BearerAuth)
- **Input**: Multipart form data
  - `video`: File (supported formats: .mp4, .mov, .avi, .webm)
  - `title`: String (3-100 characters)
  - `description`: String (max 500 characters)
- **Processing**: Asynchronous (background processing)
- **Storage**: Dual storage in IPFS and S3
- **Response**: 
  ```json
  {
    "fileId": "string",
    "title": "string",
    "description": "string",
    "status": "string"
  }
  ```

### 2. POST /video/transcode
- **Authentication**: Required (BearerAuth)
- **Input**: JSON
  ```json
  {
    "cid": "string"  // IPFS Content ID
  }
  ```
- **Processing**: Separate transcoding process
- **Response**: Transcode result details

### 3. GET /video/status/{fileId}
- **Authentication**: Not required
- **Detailed Progress Tracking**:
  ```json
  {
    "fileId": "string",
    "title": "string",
    "status": "string",
    "currentPhase": "string",
    "totalSize": "number",
    "totalProgress": "number",
    "ipfsProgress": {
      "bytesUploaded": "number",
      "percentage": "number",
      "startTime": "string",
      "endTime": "string",
      "duration": "string"
    },
    "s3Progress": {
      "bytesUploaded": "number",
      "percentage": "number",
      "startTime": "string",
      "endTime": "string",
      "duration": "string"
    },
    "errorMessage": "string",
    "completedAt": "string",
    "estimatedDuration": "string"
  }
  ```

### 4. GET /video/list
- **Authentication**: Not required
- **Response**: List of videos with detailed status information
- **No Pagination Support**

### 5. GET /video/watch
- **Authentication**: Not required
- **Query Parameters**:
  - `cid`: IPFS Content ID
  - `file`: File path
- **Response**: Video stream or redirect to IPFS gateway

## Status Types
```go
const (
    StatusPending       = "pending"
    StatusIPFSUploading = "ipfs_uploading"
    StatusIPFSCompleted = "ipfs_completed"
    StatusIPFSFailed    = "ipfs_failed"
    StatusS3Uploading   = "s3_uploading"
    StatusS3Failed      = "s3_failed"
    StatusCompleted     = "completed"
    StatusFailed        = "failed"
)
```

## Models

### Video Model
```go
type Video struct {
    ID           int64
    FileId       string
    Title        string
    Description  string
    FilePath     string
    IPFSCID      string
    Checksum     string
    UploadStatus string
    FileSize     int64
    CreatedAt    time.Time
    UpdatedAt    time.Time
    Transcodes   []Transcode
}
```

### VideoUpload Model
```go
type VideoUpload struct {
    ID                uint
    TempFileId        string
    Title             string
    Description       string
    FileSize          int64
    IPFSBytesUploaded int64
    S3BytesUploaded   int64
    UploadStatus      string
    CurrentPhase      string
    IPFSCID           string
    S3URL             string
    ErrorMessage      string
    IPFSStartTime     *time.Time
    IPFSEndTime       *time.Time
    S3StartTime       *time.Time
    S3EndTime         *time.Time
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
```

## Implementation Details

### Upload Process
1. Receive file upload request
2. Validate file and metadata
3. Create initial upload record
4. Start background processing:
   - Upload to IPFS
   - Track IPFS upload progress
   - Upload to S3
   - Track S3 upload progress
5. Return immediate response with file ID

### Progress Tracking
- Detailed progress tracking for both IPFS and S3 uploads
- Percentage completion calculation
- Duration estimation
- Phase tracking (IPFS/S3)

### Storage
- Dual storage in IPFS and S3
- No specific path structure in S3
- IPFS uploads with full pinning 