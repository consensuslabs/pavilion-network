# Video API Documentation

## Overview

The Video API is a core component of the Pavilion Network platform, providing a comprehensive set of endpoints for video management. This document outlines the current implementation, architecture, and future plans for the Video API.

The Video API enables users to:
- Upload videos to the platform
- Retrieve video details and status
- List videos with pagination
- Update video metadata
- Delete videos (soft delete)

## Technology Stack

The Video API is built using the following technologies:
- **Backend**: Go with Gin web framework (version 1.10+)
- **Database**: CockroachDB (SQL database, version 23+)
- **Storage**: Hybrid approach with Amazon S3 for performance and IPFS for decentralization
- **Video Processing**: FFmpeg (version 5+) for transcoding and thumbnail generation
- **Authentication**: JWT-based authentication

## Current Implementation

### API Endpoints

#### 1. POST /video/upload
- **Authentication**: Required (BearerAuth)
- **Input**: Multipart form data
  - `video`: File (supported formats: .mp4, .mov)
  - `title`: String (3-100 characters)
  - `description`: String (max 1000 characters, optional)
- **Processing**: Synchronous upload with background processing for transcoding
- **Storage**: Dual storage in IPFS and S3 (using path format `videos/{video_id}/[original|720p|480p|360p].mp4`)
- **Response**: 
  ```json
  {
    "data": {
      "id": "uuid",
      "title": "string",
      "description": "string",
      "file_id": "string",
      "status": "string"
    },
    "message": "Video uploaded successfully"
  }
  ```

#### 2. GET /videos
- **Authentication**: Required (BearerAuth)
- **Input**: Query parameters
  - `page`: Integer (default: 1)
  - `limit`: Integer (default: 10, max: 100)
- **Response**:
  ```json
  {
    "data": {
      "videos": [
        {
          "id": "uuid",
          "title": "string",
          "description": "string",
          "file_id": "string",
          "ipfs_cid": "string",
          "created_at": "timestamp",
          "updated_at": "timestamp"
        }
      ],
      "pagination": {
        "total": "integer",
        "page": "integer",
        "limit": "integer",
        "total_pages": "integer"
      }
    },
    "message": "Videos retrieved successfully"
  }
  ```

#### 3. GET /video/:id
- **Authentication**: Required (BearerAuth)
- **Input**: Path parameter
  - `id`: UUID of the video
- **Response**:
  ```json
  {
    "data": {
      "id": "uuid",
      "title": "string",
      "description": "string",
      "file_id": "string",
      "ipfs_cid": "string",
      "created_at": "timestamp",
      "updated_at": "timestamp",
      "upload": {
        "status": "string",
        "start_time": "timestamp",
        "end_time": "timestamp"
      },
      "transcodes": [
        {
          "format": "string",
          "segments": [
            {
              "resolution": "string",
              "path": "string"
            }
          ]
        }
      ]
    },
    "message": "Video details retrieved successfully"
  }
  ```

#### 4. GET /video/:id/status
- **Authentication**: Required (BearerAuth)
- **Input**: Path parameter
  - `id`: UUID of the video
- **Response**:
  ```json
  {
    "data": {
      "status": "string",
      "progress": "float",
      "start_time": "timestamp",
      "end_time": "timestamp"
    },
    "message": "Video status retrieved successfully"
  }
  ```

#### 5. PATCH /video/:id
- **Authentication**: Required (BearerAuth)
- **Input**: JSON body
  ```json
  {
    "title": "string",
    "description": "string"
  }
  ```
- **Response**:
  ```json
  {
    "data": {
      "id": "uuid",
      "title": "string",
      "description": "string"
    },
    "message": "Video updated successfully"
  }
  ```

#### 6. DELETE /video/:id
- **Authentication**: Required (BearerAuth)
- **Input**: Path parameter
  - `id`: UUID of the video
- **Processing**: Soft delete (sets DeletedAt timestamp)
- **Response**:
  ```json
  {
    "message": "Video deleted successfully"
  }
  ```

### Database Schema

The Video API uses the following database tables:

#### videos
- `id` (UUID, primary key)
- `file_id` (string, unique)
- `title` (string)
- `description` (string)
- `storage_path` (string)
- `ipfs_cid` (string)
- `checksum` (string)
- `file_size` (int64)
- `created_at` (timestamp)
- `updated_at` (timestamp)
- `deleted_at` (timestamp, nullable)

#### video_uploads
- `id` (UUID, primary key)
- `video_id` (UUID, foreign key)
- `start_time` (timestamp)
- `end_time` (timestamp, nullable)
- `status` (enum: pending, processing, completed, failed)
- `created_at` (timestamp)
- `updated_at` (timestamp)

#### transcodes
- `id` (UUID, primary key)
- `video_id` (UUID, foreign key)
- `format` (string: mp4, hls)
- `created_at` (timestamp)
- `updated_at` (timestamp)

#### transcode_segments
- `id` (UUID, primary key)
- `transcode_id` (UUID, foreign key)
- `resolution` (string: 480p, 720p, 1080p)
- `path` (string)
- `created_at` (timestamp)
- `updated_at` (timestamp)

### Architecture

The Video API follows a clean architecture pattern with the following components:

1. **Handler Layer**: Handles HTTP requests and responses
   - `VideoHandler`: Processes HTTP requests, validates input, and calls the service layer

2. **Service Layer**: Contains business logic
   - `VideoService`: Implements video operations like upload, retrieval, and deletion
   - `IPFSService`: Handles IPFS interactions
   - `FFmpegService`: Manages video transcoding and processing

3. **Data Layer**: Manages data persistence
   - Uses GORM as the ORM for database operations
   - Models: Video, VideoUpload, Transcode, TranscodeSegment

4. **Infrastructure Layer**: Provides supporting functionality
   - `Logger`: Structured logging
   - `ResponseHandler`: Standardized HTTP responses
   - `TempFileManager`: Manages temporary files during processing

## Testing

The Video API has a comprehensive test suite organized as follows:

```
internal/video/tests/
├── mocks/              # Mock implementations for testing
├── helpers/            # Test helper functions
├── unit/               # Unit tests for specific functionality
├── integration/        # Integration tests
└── e2e/                # End-to-end tests
```

For detailed information on testing, refer to the [Video Testing Guide](video_testing.md).

## Risks and Mitigations

1. **Synchronous Transcoding Delays**
   - **Risk**: Transcoding large videos synchronously can lead to API timeouts and poor user experience.
   - **Mitigation**: Limit video size to 500MB for the current implementation, optimize the transcoding process, and use a secure temporary directory with cleanup procedures.

2. **Storage Management**
   - **Risk**: Local storage fills up or risks security breaches during processing.
   - **Mitigation**: Use a secure temporary directory (`/tmp/videos`) with restricted permissions, implement proper cleanup after processing, and enforce file size limits.

3. **High Traffic Handling**
   - **Risk**: High concurrent traffic exceeds server capacity.
   - **Mitigation**: The current implementation is designed for low to moderate traffic (10-50 concurrent users). Rate limiting is implemented to prevent abuse.

4. **IPFS Reliability**
   - **Risk**: IPFS upload failures delay API responses or cause inconsistencies.
   - **Mitigation**: Handle IPFS upload failures gracefully, log errors, and prioritize S3 storage for reliability. IPFS content is not pinned in the current implementation to simplify the process.

## Future Plans

### 1. Asynchronous Upload with WebSockets (Q3 2025)

The next major enhancement will be implementing fully asynchronous video uploads with real-time progress tracking via WebSockets:

- Generate pre-signed URLs for direct uploads to object storage
- Implement WebSocket connections for real-time status updates
- Add support for resumable uploads
- Enhance the progress tracking system

### 2. Enhanced Transcoding Pipeline (Q4 2025)

Improvements to the transcoding pipeline:

- Implement a worker service using Apache Pulsar for message queuing
- Support for adaptive bitrate streaming (HLS)
- Multiple resolution options (480p, 720p, 1080p)
- Parallel transcoding for faster processing
- Enhanced error handling and retry mechanisms

### 3. Content Delivery Network Integration (Q1 2026)

To improve video delivery performance:

- Integration with Akamai Connected Cloud for edge caching
- Geographic distribution of content
- Bandwidth optimization
- Analytics for content delivery performance

### 4. Enhanced Metadata and Search (Q2 2026)

Improvements to video metadata and search capabilities:

- Automatic metadata extraction (duration, codec, resolution)
- Content-based tagging using machine learning
- Full-text search for video titles and descriptions
- Category and tag-based filtering

### 5. Advanced Media Features (Q3 2026)

- Add WebM transcoding for open-source browser compatibility
- Implement DRM (e.g., FairPlay, Widevine) for protected content
- Add IPFS pinning for improved decentralization
- Support for live streaming

## Constraints and Assumptions

1. **File Size Limits**: Maximum video file size is 500MB for the current implementation
2. **Supported Formats**: Currently limited to .mp4 and .mov formats
3. **Transcoding**: Videos are transcoded to MP4 with H.264/AAC codec at 720p, 480p, and 360p resolutions
4. **Storage Path**: S3 storage follows a standardized path structure: `videos/{video_id}/[original|720p|480p|360p].mp4`
5. **Authentication**: All endpoints require valid JWT authentication
6. **Soft Delete**: Videos are soft-deleted (marked as deleted but not removed from storage)
7. **IPFS Availability**: Assumes IPFS node is available and operational, but content is not pinned in the current implementation

## Conclusion

The Video API provides a robust foundation for video management in the Pavilion Network platform. The current implementation supports basic video operations, with plans for significant enhancements to support asynchronous uploads, improved transcoding, and better content delivery in future releases. 