# Specification Document for Video API Endpoints (MVP)

## Overview
This specification document defines the requirements and technical details for the video API endpoints of the Minimum Viable Product (MVP) for a decentralized media-oriented social media platform. The MVP focuses on simplified video upload, transcoding, storage, and management using Go (version 1.24), Gin (version 1.10), CockroachDB, FFmpeg, and Amazon S3. Videos are uploaded synchronously, transcoded to MP4 with H.264/AAC (720p, 480p, 360p), stored in Amazon S3 (with plans to migrate to Akamai Connected Cloud Object Storage later), and managed via RESTful APIs documented in Swagger (OpenAPI). No IPFS pinning or authentication is included for the MVP, with plans to add HLS transcoding and IPFS later.

## Current Date
- February 23, 2025

## Objectives
- Provide a simple, synchronous RESTful API for video upload, transcoding, storage, and management.
- Transcode videos to MP4 using H.264/AAC with resolutions 720p, 480p, and 360p for broad platform compatibility and simplicity in the MVP.
- Store videos in Amazon S3 for performance (with plans to migrate to Akamai later).
- Document APIs in Swagger for testing, with no frontend or authentication for the MVP.
- Ensure low complexity and quick deployment, supporting low traffic (10-50 concurrent users, 500MB max video size).

## API Specifications

### Web Server Endpoints
1. **POST /upload/video**:
   - **Input**: Multipart form data with `video` (file, max 500MB, e.g., `.mp4`, `.mov`).
   - **Output**: JSON `{ video_id: string, file_path: string, transcodes: array, status: string }`
     - `video_id`: UUID of the video.
     - `file_path`: Path to the video in Amazon S3 (e.g., `videos/{video_id}/original.mp4`).
     - `transcodes`: Array of objects `{ id: string, file_path: string, resolution: string }` for each transcoded MP4 (720p, 480p, 360p).
     - `status`: String indicating the result (e.g., `completed`, `failed`).
   - **Description**: Uploads a video locally, transcodes it to H.264/AAC MP4 (720p, 480p, 360p), stores the original and transcoded files in Amazon S3, updates CockroachDB, deletes local files, and returns results synchronously.
   - **Response Codes**:
     - `200 OK`: Successful upload and transcoding with results.
     - `400 Bad Request`: Invalid video file or size exceeds 500MB.
     - `500 Internal Server Error`: Transcoding, storage, or database errors.

2. **GET /videos**:
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
   - **Path Parameter**: `:id` (UUID of the video).
   - **Output**: JSON `{ id: string, file_id: string, title: string, description: string, file_path: string, upload_status: string, file_size: number, created_at: string, updated_at: string, transcodes: array }`
     - `transcodes`: Array of objects `{ id: string, file_path: string, resolution: string }` for each transcoded MP4 (720p, 480p, 360p).
   - **Description**: Retrieves detailed metadata for a specific video from CockroachDB.
   - **Response Codes**:
     - `200 OK`: Successful retrieval of video details.
     - `404 Not Found`: Video not found.
     - `500 Internal Server Error`: Database errors.

4. **GET /video/:id/status**:
   - **Path Parameter**: `:id` (UUID of the video).
   - **Output**: JSON `{ status: string }` (e.g., `completed`, `failed`, `pending`).
   - **Description**: Returns the current upload status of a specific video from CockroachDB.
   - **Response Codes**:
     - `200 OK`: Successful status retrieval.
     - `404 Not Found`: Video not found.
     - `500 Internal Server Error`: Database errors.

5. **DELETE /video/:id**:
   - **Path Parameter**: `:id` (UUID of the video).
   - **Output**: JSON `{ message: string, status: string }` (e.g., `{ message: "Video deleted", status: "success" }`).
   - **Description**: Deletes a video’s metadata from CockroachDB (hard delete for MVP, no file deletion from S3 for simplicity).
   - **Response Codes**:
     - `200 OK`: Successful deletion.
     - `404 Not Found`: Video not found.
     - `500 Internal Server Error`: Database errors.

6. **PATCH /video/:id**:
   - **Path Parameter**: `:id` (UUID of the video).
   - **Input**: JSON `{ title?: string, description?: string }` (at least one field required, max 100 chars for `title`, 1000 chars for `description`).
   - **Output**: JSON `{ message: string, status: string, video: object }` (e.g., `{ message: "Video updated", status: "success", video: { id: string, title: string, description: string } }`).
   - **Description**: Partially updates a video’s `title` and/or `description` in CockroachDB.
   - **Response Codes**:
     - `200 OK`: Successful update.
     - `400 Bad Request`: Invalid or missing input.
     - `404 Not Found`: Video not found.
     - `500 Internal Server Error`: Database errors.

## Technology Stack Details
- **Go**: Version 1.24+ for backend services.
- **Gin**: Version 1.10+ for RESTful API.
- **CockroachDB**: Version 23+ for SQL-based storage, ensuring ACID compliance.
- **FFmpeg**: Version 5+ for video transcoding (H.264/AAC for MP4).
- **Amazon S3**: Accessed via AWS SDK for Go for uploads and storage (with plans to migrate to Akamai Connected Cloud Object Storage later).
- **Custom Logger**: Use your existing logging implementation instead of `fmt` or `log`.

## Constraints
- Videos limited to 500MB per upload for the MVP.
- Transcoding supports only H.264/AAC for MP4, with resolutions 720p, 480p, and 360p.
- Amazon S3 uses local processing for uploads, with synchronous storage.
- No IPFS pinning or HLS transcoding for the MVP.
- No authentication for MVP testing; open access for simplicity.

## Assumptions
- Users have reliable internet for uploads and API calls.
- Amazon S3 is accessible with low latency from the web server.
- Sufficient CPU, RAM, and disk space are available on the web server for transcoding and temporary storage.

## Risks and Mitigations
- **Risk**: Synchronous transcoding delays impact API performance.
  - **Mitigation**: Limit video size to 500MB, document the need for asynchronous processing (e.g., Pulsar/workers) post-MVP.
- **Risk**: Local storage fills up or risks security breaches.
  - **Mitigation**: Use a secure temporary directory (`/tmp/videos`) with restricted permissions, limit video size, and ensure cleanup.
- **Risk**: High traffic exceeds synchronous server capacity.
  - **Mitigation**: Assume low traffic (10-50 users) for MVP, scale to asynchronous processing later.

## Future Considerations
- Add asynchronous processing with Apache Pulsar and workers for scalability.
- Implement authentication/authorization (e.g., JWT, API keys) for secure access.
- Migrate storage from Amazon S3 to Akamai Connected Cloud Object Storage for cost and performance benefits.
- Add HLS transcoding and IPFS pinning for decentralization and advanced streaming.
- Add WebM transcoding for open-source browser compatibility.
- Implement DRM (e.g., FairPlay, Widevine) for protected content.

## Acceptance Criteria
- Users can upload videos up to 500MB, receiving immediate results after transcoding and storage, playable on Windows, macOS, iOS, and Android via MP4 (H.264/AAC).
- Videos are transcoded to MP4 with 720p, 480p, and 360p resolutions, stored in Amazon S3, with metadata in CockroachDB.
- The API supports listing, retrieving, updating (title/description), and deleting videos, returning within 30 seconds for 500MB videos.
- Swagger documentation is complete and testable, with no authentication for MVP.
- The system handles 10-50 concurrent users with < 1% failure rate, storing files securely and cleaning up local storage after processing.