# Pavilion Network

Pavilion Network is a decentralized peer-to-peer social media platform focused on video content. It leverages IPFS for decentralized storage, uses video transcoding via FFmpeg, and sets a foundation for integrating consensus algorithms (Raft) as well as Ethereum blockchain anchoring for rewards.

## Features
- **Decentralized Storage:** Uses IPFS to store and retrieve video content.
- **Video Transcoding:** Transcodes videos into multiple formats (HLS and MP4) using FFmpeg.
- **Peer-to-Peer Connectivity:** Establishes p2p communications with libp2p.
- **Database & Caching:** Utilizes PostgreSQL (via GORM) for persistence and Redis for caching.
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
3. Configure your PostgreSQL and Redis connections in the code (see backend/main.go).
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