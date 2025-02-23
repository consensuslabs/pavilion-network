# Implementation Plan for Video Upload, Transcoding, and Storage in Decentralized Media Platform

## Overview
This implementation plan outlines the development and deployment strategy for a decentralized media-oriented social media platform using Go, Gin, CockroachDB, Apache Pulsar, FFmpeg, Akamai Connected Cloud Object Storage (Linode), and IPFS. The goal is to handle video uploads asynchronously, transcode videos into MP4 and HLS formats with multiple resolutions, store files in both Akamai Object Storage and IPFS, and provide real-time status updates to users via WebSockets. The plan ensures scalability, performance, and decentralization while minimizing traffic and resource consumption.

## Stack
- **Backend**: Go, Gin (web server), Apache Pulsar (message queue), FFmpeg (transcoding)
- **Database**: CockroachDB (SQL for video metadata, transcodes, and segments)
- **Storage**: Akamai Connected Cloud Object Storage (Linode) for performance, IPFS for decentralization
- **Frontend**: Vanilla JavaScript for testing UI
- **Current Date**: February 22, 2025

## Phases

### Phase 1: Setup and Configuration
#### Objectives
- Set up development environment, install dependencies, and configure services.
- Ensure compatibility with Akamai Object Storage and IPFS.

#### Tasks
1. **Install Dependencies**:
   - Install Go, Gin, Apache Pulsar, FFmpeg, and CockroachDB locally.
   - Install IPFS (`ipfs-go`) and configure a local or remote IPFS node.
   - Install Akamai Object Storage SDK (e.g., MinIO Go client) and configure credentials.

2. **Configure Services**:
   - Set up CockroachDB with tables (`videos`, `transcodes`, `transcode_segments`).
   - Configure Apache Pulsar with topics (`video-transcode-queue`, `video-status-updates`).
   - Configure Akamai Object Storage with buckets (e.g., `media-videos`) and generate access/secret keys.
   - Ensure IPFS node is running (local or via a service like Pinata).

3. **Test Connectivity**:
   - Verify connections to CockroachDB, Pulsar, Akamai Object Storage, and IPFS using simple Go scripts.

#### Deliverables
- Working development environment with all services configured.
- Basic scripts to test connectivity and storage operations.

---

### Phase 2: Backend Development
#### Objectives
- Implement the web server (Gin) for video uploads, progress tracking, and WebSocket status updates.
- Develop the worker service for transcoding and storage management.

#### Tasks
1. **Web Server (Gin)**:
   - Create `POST /upload/video` to generate pre-signed URLs for Akamai Object Storage uploads, save metadata to CockroachDB, and publish `video_id` to Pulsar.
   - Implement `GET /video/:id/progress` for polling (temporary) and `GET /ws` for WebSocket status updates.
   - Use `github.com/minio/minio-go` for Akamai Object Storage interactions.

   ```go
   // Example snippet for pre-signed URL generation
   func getPresignedURLHandler(c *gin.Context) {
       var req struct { Filename string `json:"filename"`; FileSize int64 `json:"fileSize"` }
       if err := c.BindJSON(&req); err != nil { c.JSON(400, gin.H{"error": "Invalid request"}) return }
       videoID := uuid.New().String()
       s3Path := fmt.Sprintf("videos/%s/%s", videoID, req.Filename)
       ctx := context.Background()
       client, _ := minio.New("s3.us-east-1.linodeobjects.com", "access-key", "secret-key", true)
       presignedURL, _ := client.PresignPutObject(ctx, "media-videos", s3Path, time.Hour)
       _, _ = db.Exec(`INSERT INTO videos (id, file_id, file_path, upload_status, file_size, created_at) VALUES ($1, gen_random_uuid(), $2, 'pending', $3, NOW())`, videoID, s3Path, req.FileSize)
       publishToPulsar(videoID, nil)
       c.JSON(200, gin.H{"video_id": videoID, "presigned_url": presignedURL, "s3_path": s3Path})
   }
   ```

2. **Worker Service**:
   - Develop a standalone Go service consuming from `video-transcode-queue` using Pulsar.
   - Implement FFmpeg transcoding for MP4 (480p, 720p, 1080p) and HLS (same resolutions, with segments).
   - Store transcoded files in Akamai Object Storage and pin to IPFS asynchronously.
   - Update CockroachDB with `upload_status`, `file_path`, and `file_cid`.

   ```go
   // Example snippet for worker
   func processVideo(videoID string) {
       var filePath string
       db.QueryRow(`SELECT file_path FROM videos WHERE id = $1`, videoID).Scan(&filePath)
       localFile, _ := downloadFromAkamaiObjectStorage("media-videos", filePath, "temp.mp4")
       go pinToIPFS(localFile) // Pin original to IPFS
       transcodeVideo(videoID, localFile)
       updateStatus(videoID, "completed")
   }
   ```

3. **WebSocket Integration**:
   - Use `gorilla/websocket` to broadcast status updates from Pulsar (`video-status-updates`) to connected clients.

#### Deliverables
- Functional web server with upload, progress, and WebSocket endpoints.
- Operational worker service for transcoding and storage.
- Unit tests for core functions (e.g., upload, transcoding, storage).

---

### Phase 3: Frontend Development
#### Objectives
- Build a simple vanilla JavaScript UI for testing video uploads and real-time status updates.

#### Deliverables
- Functional frontend for testing uploads, progress tracking, and status updates.

---

### Phase 4: Testing and Optimization
#### Objectives
- Ensure system reliability, performance, and scalability.
- Optimize for traffic, latency, and resource usage.

#### Deliverables
- Comprehensive test report with performance metrics.
- Optimized system for production readiness.

---

### Phase 5: Deployment and Monitoring
#### Objectives
- Deploy the platform to production and set up monitoring.

#### Deliverables
- Production-ready platform.
- Monitoring dashboard and documentation.

---

## Database Tables

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
