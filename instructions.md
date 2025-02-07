# Pavilion Network Development Instructions

## 1. Project Overview
Pavilion Network is a decentralized peer-to-peer social media network that enables robust video content sharing. The platform leverages IPFS for storing and retrieving video content, utilizes FFmpeg for video transcoding, and uses libp2p for p2p connectivity. Future plans include integrating the Raft consensus algorithm for distributed state management and anchoring transactions on the Ethereum blockchain for rewards and monetary-related features.

## 2. Core Functionalities

### 2.1 Video Transcoding and IPFS Integration
- Upload video files and generate a unique file ID.
- Transcode videos into multiple formats (HLS and MP4) using FFmpeg.
- Generate and adjust HLS manifests by scanning for TS segments.
- Upload video segments to IPFS and retrieve their CIDs for decentralized access.

### 2.2 Database and Caching
- Use PostgreSQL (via GORM) to persist user, video, and transcode metadata.
- Utilize Redis for caching to improve performance and reliability.

### 2.3 Peer-to-Peer Networking
- Establish p2p communication channels using libp2p.
- Handle node discovery and secure messaging between peers.

### 2.4 Future Integrations (Planned)
- Implement the Raft consensus algorithm for distributed state consistency.
- Integrate Ethereum blockchain anchoring for transactions and reward mechanisms.

## 3. Documentation and Libraries

### Libraries/Frameworks Used
- Gin: HTTP server framework for API endpoints.
- GORM: ORM for PostgreSQL interactions.
- go-redis: Redis client for caching.
- libp2p: Peer-to-peer networking framework.
- FFmpeg: External tool for video transcoding (invoked via exec commands).
- IPFS Go library: Communicate with IPFS for file storage and retrieval.

### Documentation
- README.md: Provides project overview, features, setup instructions, and future roadmap.
- instructions.md: Outlines the development process, core functionalities, libraries, and file structure.
- Inline comments: Offer context and guidance within the codebase.

## 4. Current File Structure
- /backend
  - main.go: Application entry point, API routes, models, and transcoding logic.
  - p2p.go: Functions for establishing and managing p2p connections.
  - database.go: Database connection and migration logic.
  - cache.go: Cache (Redis) initialization and usage.
  - docker/: Docker-related configurations and files.
- /frontend
  - public/: Contains static assets and frontend resources.
- .git, LICENSE, README.md, instructions.md

## 5. Development Process
- Understand requirements completely before starting a feature.
- Plan features in small, focused increments (keep files <200 lines).
- Write clean, simple, and reliable code with clear, consistent naming conventions.
- Test after every meaningful change to ensure reliability.
- Update documentation (README.md and instructions.md) after each feature is implemented, tested, and verified.
- Prioritize core functionality before optimization.

## 6. Testing and Deployment
- Run local tests for each implemented feature (video transcoding, p2p connectivity, IPFS integration).
- Ensure unit tests and integration tests cover critical components.
- Follow deployment guidelines to replicate a production-like environment during testing.
- **Important:** Most Go commands (build, run, get, mod tidy) should be executed from within the `/backend` directory, as this is where the Go module is defined. 