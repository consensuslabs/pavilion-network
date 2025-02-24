# Specification Document for Video API Endpoints (MVP)

## Overview
This specification document defines the requirements and technical details for the video API endpoints of the Minimum Viable Product (MVP) for a decentralized media-oriented social media platform. The MVP focuses on simplified video upload, transcoding, storage, and management using Go (version 1.24), Gin (version 1.10), CockroachDB, FFmpeg, Amazon S3, and IPFS. Videos are uploaded synchronously, transcoded to MP4 with H.264/AAC (720p, 480p, 360p), stored in Amazon S3 (with plans to migrate to Akamai Connected Cloud Object Storage later) and uploaded to IPFS (but not pinned for the MVP), and managed via RESTful APIs documented in Swagger (OpenAPI).

## Current Date
- February 23, 2025

## Objectives
- Provide a simple, synchronous RESTful API for video upload, transcoding, storage, and management.
- Transcode videos to MP4 using H.264/AAC with resolutions 720p, 480p, and 360p for broad platform compatibility and simplicity in the MVP.
- Store videos in Amazon S3 for performance (with plans to migrate to Akamai later) and upload to IPFS (but not pin) for decentralization.
- Document APIs in Swagger for testing.
- Ensure low complexity and quick deployment, supporting low traffic (10-50 concurrent users, 500MB max video size).

## API Specifications

### Web Server Endpoints
1. **POST /upload/video**:
   - **Authentication**: Required (BearerAuth)
   - **Input**: Multipart form data with `video` (file, max 500MB, e.g., `.mp4`, `.mov`).
   - **Output**: JSON `{ video_id: string, file_path: string, ipfs_cid: string, transcodes: array, status: string }`
     - `video_id`: UUID of the video.
     - `file_path`: Path to the video in Amazon S3 (e.g., `videos/{video_id}/original.mp4`).
     - `ipfs_cid`: Content Identifier for the video in IPFS (uploaded but not pinned for the MVP).
     - `transcodes`: Array of objects `{ id: string, file_path: string, ipfs_cid: string, resolution: string }` for each transcoded MP4 (720p, 480p, 360p).
     - `status`: String indicating the result (e.g., `completed`, `failed`).
   - **Description**: Uploads a video, transcodes it to H.264/AAC MP4 (720p, 480p, 360p), stores the original and transcoded files in Amazon S3 (using path format `videos/{video_id}/[original|720p|480p|360p].mp4`), uploads them to IPFS (but does not pin for the MVP), updates CockroachDB, deletes local files, and returns results synchronously.
   - **Response Codes**:
     - `200 OK`: Successful upload, transcoding, and storage with results.
     - `400 Bad Request`: Invalid video file or size exceeds 500MB.
     - `401 Unauthorized`: Missing or invalid authentication token.
     - `500 Internal Server Error`: Transcoding, storage, IPFS upload, or database errors.

2. **GET /videos**:
   - **Authentication**: Not required
   - **Query Parameters**:
     - `limit?: number` (default 10, max 50): Number of videos to return.
     - `offset?: number` (default 0): Offset for pagination.
     - `status?: string` (e.g., `completed`, `pending`, `failed`): Filter by upload status.
   - **Output**: JSON `{ videos: array, total: number }`
     - `videos`: Array of objects `{ id: string, title: string, description: string, upload_status: string, created_at: string }`.
     - `total`: Total number of matching videos.
   - **Description**: Lists videos with optional pagination and status filtering from CockroachDB.
   - **Response Codes**:
     - `200 OK`: Successful list of videos.
     - `500 Internal Server Error`: Database errors.

3. **GET /video/:id**:
   - **Authentication**: Not required
   - **Path Parameter**: `:id` (UUID of the video).
   - **Output**: JSON `{ id: string, file_id: string, title: string, description: string, file_path: string, ipfs_cid: string, upload_status: string, file_size: number, created_at: string, updated_at: string, transcodes: array }`
     - `transcodes`: Array of objects `{ id: string, file_path: string, ipfs_cid: string, resolution: string }` for each transcoded MP4 (720p, 480p, 360p).
   - **Description**: Retrieves detailed metadata for a specific video from CockroachDB.
   - **Response Codes**:
     - `200 OK`: Successful retrieval of video details.
     - `404 Not Found`: Video not found.
     - `500 Internal Server Error`: Database errors.

4. **GET /video/:id/status**:
   - **Authentication**: Not required
   - **Path Parameter**: `:id` (UUID of the video).
   - **Output**: JSON `{ status: string }` (e.g., `completed`, `failed`, `pending`).
   - **Description**: Returns the current upload/processing status of a specific video.
   - **Response Codes**:
     - `200 OK`: Successful status retrieval.
     - `404 Not Found`: Video not found.
     - `500 Internal Server Error`: Database errors.

5. **DELETE /video/:id**:
   - **Authentication**: Required (BearerAuth)
   - **Path Parameter**: `:id` (UUID of the video).
   - **Output**: JSON `{ message: string, status: string }` (e.g., `{ message: "Video deleted", status: "success" }`).
   - **Description**: Deletes a video's metadata from CockroachDB (hard delete for MVP, no file deletion from S3 or IPFS for simplicity).
   - **Response Codes**:
     - `200 OK`: Successful deletion.
     - `401 Unauthorized`: Missing or invalid authentication token.
     - `404 Not Found`: Video not found.
     - `500 Internal Server Error`: Database errors.

6. **PATCH /video/:id**:
   - **Authentication**: Required (BearerAuth)
   - **Path Parameter**: `:id` (UUID of the video).
   - **Input**: JSON `{ title?: string, description?: string }` (at least one field required, max 100 chars for `title`, 1000 chars for `description`).
   - **Output**: JSON `{ message: string, status: string, video: object }` (e.g., `{ message: "Video updated", status: "success", video: { id: string, title: string, description: string } }`).
   - **Description**: Partially updates a video's `title` and/or `description` in CockroachDB.
   - **Response Codes**:
     - `200 OK`: Successful update.
     - `400 Bad Request`: Invalid or missing input.
     - `401 Unauthorized`: Missing or invalid authentication token.
     - `404 Not Found`: Video not found.
     - `500 Internal Server Error`: Database errors.


## Database Specifications
### Videos Table
-- Videos Table
-- Stores metadata about uploaded video files.
-- Relationship with Video Uploads is one-to-one.
CREATE TABLE videos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id VARCHAR(255) NOT NULL UNIQUE,  -- Unique identifier for the file, not PK
    title VARCHAR(255) NOT NULL,
    description TEXT,
    storage_path VARCHAR(255) NOT NULL,    -- Path to the file in S3 or cloud
    ipfs_cid VARCHAR(255),                 -- IPFS content identifier
    checksum VARCHAR(64),                  -- To be calculated and added
    file_size BIGINT NOT NULL,             -- Size in bytes
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

### Video Uploads Table
-- Tracks the upload process for each video (one-to-one with videos).
CREATE TABLE video_uploads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID NOT NULL UNIQUE,         -- Foreign key to videos.id, unique for 1:1
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    status ENUM('success', 'fail', 'pending') NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE
);

### Transcodes Table
-- Stores transcoding operations for videos.
CREATE TABLE transcodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID NOT NULL,                -- Foreign key to videos.id
    format ENUM('mp4', 'hls') NOT NULL,    -- Restricted to mp4 and hls
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE
);

### Transcode Segments Table
-- Stores segments of transcoding operations (e.g., HLS segments).
CREATE TABLE transcode_segments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transcode_id UUID NOT NULL,            -- Foreign key to transcodes.id
    storage_path VARCHAR(255) NOT NULL,    -- e.g., xxxx_hls_720, consistent with videos
    ipfs_cid VARCHAR(255),                 -- IPFS content identifier
    duration INT,                          -- Duration in seconds
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (transcode_id) REFERENCES transcodes(id) ON DELETE CASCADE
);

## Technology Stack Details
- **Go**: Version 1.24+ for backend services.
- **Gin**: Version 1.10+ for RESTful API.
- **CockroachDB**: Version 23+ for SQL-based storage, ensuring ACID compliance.
- **FFmpeg**: Version 5+ for video transcoding (H.264/AAC for MP4).
- **Amazon S3**: Accessed via AWS SDK for Go for uploads and storage (with plans to migrate to Akamai Connected Cloud Object Storage later).
- **IPFS**: Use `github.com/ipfs/go-ipfs-api` for uploading files to IPFS (but not pinning for the MVP).
- **Custom Logger**: Use your existing logging implementation instead of `fmt` or `log`.


## Constraints
- Videos limited to 500MB per upload for the MVP.
- Transcoding supports only H.264/AAC for MP4, with resolutions 720p, 480p, and 360p.
- Amazon S3 uses standardized path structure: `videos/{video_id}/[original|720p|480p|360p].mp4`
- IPFS upload is included but not pinned for the MVP; no HLS transcoding.
- Authentication required for upload, delete, and update operations.

## Assumptions
- Users have reliable internet for uploads and API calls.
- Amazon S3 and IPFS are accessible with low latency from the web server.
- Sufficient CPU, RAM, and disk space are available on the web server for transcoding and temporary storage.
- Authentication system is already in place and working.

## Risks and Mitigations
- **Risk**: Synchronous transcoding delays impact API performance.
  - **Mitigation**: Limit video size to 500MB, optimize transcoding process.
- **Risk**: Local storage fills up or risks security breaches.
  - **Mitigation**: Use a secure temporary directory (`/tmp/videos`) with restricted permissions, limit video size, and ensure cleanup.
- **Risk**: High traffic exceeds server capacity.
  - **Mitigation**: Assume low traffic (10-50 users) for MVP, implement rate limiting.
- **Risk**: IPFS upload failures delay API responses.
  - **Mitigation**: Handle IPFS upload failures gracefully, log errors with your custom logger, and prioritize Amazon S3 storage for the MVP.


## Future Considerations
- Add asynchronous processing with Apache Pulsar and workers for scalability.
- Migrate storage from Amazon S3 to Akamai Connected Cloud Object Storage for cost and performance benefits.
- Add HLS transcoding and IPFS pinning for decentralization and advanced streaming.
- Add WebM transcoding for open-source browser compatibility.
- Implement DRM (e.g., FairPlay, Widevine) for protected content.

## Acceptance Criteria
- Users can upload videos up to 500MB with proper authentication, receiving immediate results after transcoding and storage.
- Videos are transcoded to MP4 with 720p, 480p, and 360p resolutions, stored in Amazon S3 and uploaded to IPFS (but not pinned).
- The API supports listing, retrieving, updating (title/description), and deleting videos with proper authentication where required.
- S3 storage follows the standardized path structure.
- The system handles 10-50 concurrent users with < 1% failure rate, storing files securely and cleaning up local storage after processing.