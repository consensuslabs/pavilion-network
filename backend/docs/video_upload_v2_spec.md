# Specification Document for Decentralized Media-Oriented Social Media Platform

## Overview
This specification document defines the requirements, architecture, and technical details for developing a decentralized media-oriented social media platform. The platform enables users to upload, transcode, store, and stream videos, leveraging a hybrid storage approach with Akamai Connected Cloud Object Storage (Linode) for performance and IPFS for decentralization. The system uses Go, Gin, CockroachDB, Apache Pulsar, FFmpeg, and vanilla JavaScript for a testing UI, ensuring scalability, performance, and real-time updates via WebSockets.

## Current Date
- February 22, 2025

## Objectives
- Enable asynchronous video uploads with real-time progress tracking.
- Transcode videos into MP4 and HLS formats with multiple resolutions (480p, 720p, 1080p) for broad platform compatibility.
- Store videos in Akamai Object Storage for performance and IPFS for decentralization.
- Provide a simple testing UI using vanilla JavaScript with WebSocket-based status updates.
- Ensure scalability, low latency, and minimal traffic for a decentralized media platform.

## System Architecture
### Components
1. **Web Server (Gin)**:
   - Handles HTTP requests for video uploads, progress polling, and WebSocket connections.
   - Generates pre-signed URLs for direct uploads to Akamai Object Storage.
   - Publishes video metadata to Apache Pulsar for asynchronous processing.

2. **Worker Service**:
   - A standalone Go service consuming from Apache Pulsar.
   - Performs video transcoding using FFmpeg and manages storage in Akamai Object Storage and IPFS.
   - Updates CockroachDB with video status and metadata.

3. **Database (CockroachDB)**:
   - Stores video metadata (`videos`), transcoding information (`transcodes`), and HLS segments (`transcode_segments`).
   - Ensures transactional consistency for video processing states.

4. **Message Queue (Apache Pulsar)**:
   - Manages asynchronous task distribution for video transcoding and status updates.
   - Uses topics like `video-transcode-queue` and `video-status-updates`.

5. **Storage**:
   - **Akamai Connected Cloud Object Storage (Linode)**: Primary storage for performance, using S3-compatible APIs for uploads, downloads, and transcoded outputs.
   - **IPFS**: Decentralized storage for long-term archival, pinned asynchronously in workers.

6. **Frontend (Vanilla JavaScript)**:
   - Simple UI for testing video uploads, progress tracking, and status updates via WebSockets or polling.

### Data Flow
1. User uploads a video via the frontend, which uses a pre-signed URL to send it directly to Akamai Object Storage.
2. The web server saves metadata to CockroachDB, publishes the `video_id` to Pulsar, and returns a task ID.
3. A worker consumes the `video_id`, downloads the video from Akamai Object Storage, transcodes it, uploads outputs to Akamai Object Storage, and pins files to IPFS.
4. The worker updates CockroachDB with progress and status, which the web server broadcasts via WebSocket to the frontend.

## Requirements
### Functional Requirements
1. **Video Upload**:
   - Support video uploads (e.g., `.mp4`, `.mov`) up to 10GB, directly to Akamai Object Storage using pre-signed URLs.
   - Display “Uploading...” with real-time progress and estimated completion time.

2. **Video Transcoding**:
   - Transcode videos into MP4 (H.264/AAC, 480p, 720p, 1080p) and HLS (H.264/AAC, same resolutions, with 10-second segments).
   - Ensure compatibility with Windows, macOS, iOS, and Android.

3. **Storage**:
   - Store original and transcoded videos in Akamai Object Storage for performance.
   - Pin files to IPFS asynchronously for decentralization, updating `ipfs_cid` in CockroachDB.

4. **Status Updates**:
   - Provide real-time status updates via WebSockets (e.g., “Uploading”, “Transcoding”, “Completed”, “Failed”).
   - Support polling (`GET /video/:id/progress`) as a fallback for testing.

5. **Scalability**:
   - Handle concurrent uploads and transcodings using Apache Pulsar and multiple workers.
   - Ensure single-worker processing per `video_id` using Pulsar’s key-based routing.

### Non-Functional Requirements
1. **Performance**:
   - Upload latency < 5 seconds for small videos, transcoding latency < 2x video duration for 1080p.
   - Storage access latency < 1 second for Akamai Object Storage, IPFS retrieval < 5 seconds via gateways.

2. **Scalability**:
   - Support 1,000 concurrent users uploading 100MB videos, scaling workers and Pulsar as needed.
   - Handle 10,000 daily transcoding requests with < 1% failure rate.

3. **Reliability**:
   - Ensure 99.9% uptime for web servers, workers, and storage.
   - Implement retries and deduplication for Pulsar messages and IPFS pinning.

4. **Security**:
   - Use pre-signed URLs for secure Akamai Object Storage access, expiring in 1 hour.
   - Encrypt data in transit (TLS) and at rest (Akamai Object Storage encryption).

5. **Cost Efficiency**:
   - Minimize egress costs using Akamai CDN and Object Storage pricing.
   - Store original videos temporarily (30 days) on Akamai Object Storage, then archive or delete unless requested.

## Database Schema
### `videos`
```sql
CREATE TABLE videos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID,
    title STRING,
    description STRING,
    file_path STRING,
    ipfs_cid STRING,
    checksum STRING,
    upload_status STRING NOT NULL DEFAULT 'pending',
    file_size BIGINT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### `transcodes`
```sql
CREATE TABLE transcodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID NOT NULL,
    file_path STRING,
    file_cid STRING,
    format STRING NOT NULL,
    resolution STRING NOT NULL,
    storage_type STRING NOT NULL DEFAULT 's3',
    type STRING NOT NULL DEFAULT 'transcode',
    created_at TIMESTAMP DEFAULT NOW()
);
```

### `transcode_segments`
```sql
CREATE TABLE transcode_segments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transcode_id UUID NOT NULL,
    file_path STRING,
    file_cid STRING,
    storage_type STRING NOT NULL DEFAULT 's3',
    sequence BIGINT NOT NULL,
    duration DECIMAL NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Risks and Mitigations
- **Risk: Akamai Object Storage latency impacts transcoding performance.**
  - **Mitigation:** Deploy workers in regions close to Akamai storage.
- **Risk: IPFS pinning delays.**
  - **Mitigation:** Use dedicated pinning services with retry mechanisms.
- **Risk: High traffic overloads.**
  - **Mitigation:** Scale workers dynamically with Kubernetes and optimize CDN usage.

## Future Considerations
- Add WebM transcoding for open-source browser compatibility.
- Implement DRM (e.g., FairPlay, Widevine) for protected content.
- Explore edge transcoding with Akamai to reduce worker load.
- Add user authentication and authorization for secure video access.
