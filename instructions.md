# Pavilion Network Development Instructions

## 1. Project Overview
Pavilion Network is a decentralized peer-to-peer social media network that enables robust video content sharing. The platform leverages IPFS for storing and retrieving video content, utilizes FFmpeg for video transcoding, and uses libp2p for p2p connectivity. Future plans include integrating the Raft consensus algorithm for distributed state management and anchoring transactions on the Ethereum blockchain for rewards and monetary-related features.

## 2. Core Functionalities

### 2.1 Video Transcoding and IPFS Integration
- Upload video files and generate a unique file ID.
- Validate video uploads (size, format, metadata)
- Transcode videos into multiple formats (HLS and MP4) using FFmpeg.
- Generate and adjust HLS manifests by scanning for TS segments.
- Upload video segments to IPFS and retrieve their CIDs for decentralized access.

### 2.2 Database and Caching
- Use PostgreSQL (via GORM) to persist user, video, and transcode metadata.
- Utilize Redis for caching to improve performance and reliability.

### 2.3 Peer-to-Peer Networking
- Establish p2p communication channels using libp2p.
- Handle node discovery and secure messaging between peers.

### 2.4 Request Tracking and Error Handling
- Unique request ID generation for all API requests
- Consistent error response format with request tracking
- Structured logging with request context
- Centralized error handling system:
  - Custom error types for different scenarios (validation, storage, processing)
  - Consistent error message format
  - Error cause tracking for debugging
  - Field-level validation error reporting

### 2.5 Configuration Management
- Centralized configuration via YAML
- Environment-specific settings support
- Configurable video upload parameters:
  - File size limits
  - Allowed formats
  - Title and description constraints

### 2.6 Future Integrations (Planned)
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
  - main.go: Application entry point and core setup
  - services.go: Core business logic for video handling
  - handlers.go: HTTP request handlers
  - routes.go: API route definitions
  - errors.go: Centralized error types and messages
  - p2p.go: Functions for establishing and managing p2p connections
  - database.go: Database connection and migration logic
  - cache.go: Cache (Redis) initialization and usage
  - config.go: Configuration structures and loading
  - models.go: Database models and relationships
  - docker/: Docker-related configurations and files
- /frontend
  - public/: Contains static assets and frontend resources
    - js/: JavaScript files
    - css/: Stylesheets
    - index.html: Main frontend entry point
- .git, LICENSE, README.md, instructions.md

## 5. Development Cycle

The development process follows a strict cycle to ensure high-quality, well-tested, and properly documented code:

### 5.1 Implementation Phase
- Write clean, maintainable code
- Follow Go best practices and project conventions
- Keep functions focused and modular
- Add proper error handling
- Consider edge cases
- Implement configuration where appropriate

### 5.2 Testing Phase
- Build and verify compilation
- Test new functionality
- Verify error handling
- Check edge cases
- Ensure configuration works as expected
- Validate integration with existing features

### 5.3 Documentation Phase
- Update relevant documentation files
- Document new configuration options
- Add inline code comments where needed
- Update API documentation if endpoints change
- Document any new dependencies or requirements
- Add examples for new features

This cycle repeats for each feature or enhancement, ensuring that:
1. Code is properly implemented and follows best practices
2. All features are thoroughly tested
3. Documentation stays up-to-date and comprehensive

By following this cycle, we maintain high code quality, reduce technical debt, and make the codebase more maintainable for all developers.

## 6. Development Process
- Understand requirements completely before starting a feature.
- Plan features in small, focused increments (keep files <200 lines).
- Write clean, simple, and reliable code with clear, consistent naming conventions.
- Test after every meaningful change to ensure reliability.
- Update documentation (README.md and instructions.md) after each feature is implemented, tested, and verified.
- Prioritize core functionality before optimization.

## 7. Testing and Deployment
- Run local tests for each implemented feature (video transcoding, p2p connectivity, IPFS integration).
- Ensure unit tests and integration tests cover critical components.
- Follow deployment guidelines to replicate a production-like environment during testing.
- **Important:** Most Go commands (build, run, get, mod tidy) should be executed from within the `/backend` directory, as this is where the Go module is defined. 