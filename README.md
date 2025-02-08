# Pavilion Network

Pavilion Network is a decentralized peer-to-peer social media platform focused on video content. It leverages IPFS for decentralized storage, uses video transcoding via FFmpeg, and sets a foundation for integrating consensus algorithms (Raft) as well as Ethereum blockchain anchoring for rewards.

## Features
- **Decentralized Storage:** Uses IPFS to store and retrieve video content.
- **Video Transcoding:** Transcodes videos into multiple formats (HLS and MP4) using FFmpeg.
- **Video Upload Validation:** Configurable validation for video uploads (size, format, metadata).
- **Request Tracking:** Unique request IDs and structured logging for better debugging.
- **Peer-to-Peer Connectivity:** Establishes p2p communications with libp2p.
- **Database & Caching:** Utilizes PostgreSQL (via GORM) for persistence and Redis for caching.
- **Configuration Management:** YAML-based configuration with environment support.
- **Error Handling:** Centralized error handling with custom error types and consistent messaging.
- **Future Integrations:** Planning to add Raft consensus and Ethereum-based transactions.

## Requirements
- Go (1.16 or later)
- PostgreSQL
- Redis
- FFmpeg
- IPFS node or API access
- Git

## Setup & Installation
1. Clone the repository:
   ```
   git clone <repository_url>
   cd pavilion-network
   ```
2. Install backend dependencies:
   ```
   cd backend
   go mod download
   ```
3. Configure your environment:
   - Copy `config.yaml.example` to `config.yaml` (if not exists)
   - Update the configuration values in `config.yaml`:
     ```yaml
     # Database and Redis settings...
     
     video:
       maxSize: 104857600  # 100MB in bytes
       minTitleLength: 3
       maxTitleLength: 100
       maxDescLength: 500
       allowedFormats:
         - ".mp4"
         - ".mov"
         - ".avi"
         - ".webm"
     ```
4. Ensure FFmpeg is installed and accessible in your system
5. Run the application:
   ```
   go run main.go
   ```
**Note:**  Most Go commands related to building and managing dependencies should be run from within the `backend` directory.

## Development Process
- Follow the guidelines in the `instructions.md` file to understand feature planning and implementation.
- Test each feature after implementation.
- Update documentation as new features are added.

## Future Enhancements
- Integration of Raft consensus algorithm for distributed state management.
- Anchoring transactions on the Ethereum blockchain for reward mechanisms.
- Expanded frontend functionalities.

## License
See [LICENSE](LICENSE) for more details.

## Development Environment Setup

- **Docker Compose:**
  - Navigate to the `backend/docker` directory.
  - Run `docker compose up` to start the required containers.
  - On Windows, ensure that Docker Desktop is set to use Linux containers.

- **FFmpeg Installation:**
  - Download FFmpeg from [ffmpeg.org](https://ffmpeg.org/download.html).
  - Install FFmpeg and add the executable to your system PATH.
  - Verify the installation by running `ffmpeg -version` in your terminal.

## API Documentation

### Error Handling
The application uses a centralized error handling system with custom error types:

- **ValidationError**: For input validation errors
  ```json
  {
    "success": false,
    "error": {
      "code": "ERR_VALIDATION",
      "field": "file",
      "message": "File size exceeds maximum allowed size"
    }
  }
  ```

- **StorageError**: For storage-related errors (file system, database)
  ```json
  {
    "success": false,
    "error": {
      "code": "ERR_STORAGE",
      "message": "Failed to store file",
      "cause": "underlying error message"
    }
  }
  ```

- **ProcessingError**: For video processing errors
  ```json
  {
    "success": false,
    "error": {
      "code": "ERR_PROCESSING",
      "message": "Failed to process video",
      "cause": "underlying error message"
    }
  }
  ```

### Video Upload