package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	shell "github.com/ipfs/go-ipfs-api"
	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Global variables for the database, cache, and context.
var (
	DB    *gorm.DB
	Cache *redis.Client
	ctx   = context.Background()
)

// User model definition.
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	Email     string    `gorm:"unique" json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Video model definition.
// IPFSCID stores the IPFS CID for the video.
type Video struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	FilePath    string    `json:"file_path"`
	IPFSCID     string    `json:"ipfs_cid"`
	CreatedAt   time.Time `json:"created_at"`
}

// TranscodeTarget defines parameters for each output.
type TranscodeTarget struct {
	Label      string // e.g., "720p_mp4"
	Resolution string // e.g., "720"
	OutputExt  string // e.g., "m3u8"
}

// ConnectDatabase initializes the PostgreSQL connection using GORM.
func ConnectDatabase() {
	dsn := "host=localhost user=youruser password=yourpassword dbname=pavilion_db port=5432 sslmode=disable TimeZone=UTC"
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// Auto-migrate models.
	err = DB.AutoMigrate(&User{}, &Video{})
	if err != nil {
		log.Fatalf("Auto migration failed: %v", err)
	}
	log.Println("Database connected and migrated successfully.")
}

// ConnectRedis initializes the Redis connection.
func ConnectRedis() {
	Cache = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	pong, err := Cache.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Printf("Redis connected: %s", pong)
}

// transcodeToHLS uses FFmpeg to transcode a video to HLS output.
// It scales the video to the target resolution, uses H.264 for video encoding, and copies the audio.
func transcodeToHLS(inputFile, outputManifest, scaleHeight string) error {
	cmd := exec.Command("ffmpeg", "-i", inputFile, "-vf", "scale=-2:"+scaleHeight, "-c:v", "libx264", "-c:a", "copy", "-preset", "fast", "-hls_time", "10", "-hls_playlist_type", "vod", outputManifest)
	return cmd.Run()
}

// startP2PHost starts a libp2p host.
func startP2PHost() (host.Host, error) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		return nil, err
	}
	return h, nil
}

// printHostInfo logs the libp2p host's ID and addresses.
func printHostInfo(h host.Host) {
	log.Printf("P2P Host ID: %s", h.ID().String())
	for _, addr := range h.Addrs() {
		log.Printf("Address: %s", addr.String())
	}
}

// uploadVideoToIPFS uploads a file to IPFS and returns its CID.
func uploadVideoToIPFS(filePath string) (string, error) {
	sh := shell.NewShell("localhost:5001")
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	cid, err := sh.Add(file)
	if err != nil {
		return "", err
	}
	return cid, nil
}

func main() {
	// Connect to PostgreSQL and Redis.
	ConnectDatabase()
	ConnectRedis()

	// Start the P2P host.
	p2pHost, err := startP2PHost()
	if err != nil {
		log.Fatalf("Failed to start P2P host: %v", err)
	}
	printHostInfo(p2pHost)

	router := gin.Default()
	// Serve static files.
	router.Static("/public", "../frontend/public")
	router.Static("/uploads", "./uploads")

	// Health-check endpoint.
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// OAuth login stub endpoint.
	router.POST("/auth/login", func(c *gin.Context) {
		user := User{
			Name:  "Test User",
			Email: "test@example.com",
		}
		if err := DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "OAuth login stub - token: dummy-token",
			"user":    user,
		})
	})

	// Video upload endpoint.
	router.POST("/video/upload", func(c *gin.Context) {
		file, err := c.FormFile("video")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No video file received"})
			return
		}
		uploadDir := "./uploads"
		if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
			if err := os.Mkdir(uploadDir, 0755); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create uploads directory"})
				return
			}
		}
		destination := uploadDir + "/" + file.Filename
		if err := c.SaveUploadedFile(file, destination); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		cid, err := uploadVideoToIPFS(destination)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload to IPFS: " + err.Error()})
			return
		}
		video := Video{
			Title:       file.Filename,
			Description: "Uploaded video",
			FilePath:    destination,
			IPFSCID:     cid,
			CreatedAt:   time.Now(),
		}
		if err := DB.Create(&video).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save video record: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":   "Video uploaded and stored in IPFS successfully",
			"file_path": destination,
			"ipfs_cid":  cid,
			"video":     video,
		})
	})

	// Transcoding endpoint: Transcode video file to HLS outputs in multiple resolutions.
	router.POST("/video/transcode", func(c *gin.Context) {
		// Expected JSON payload: {"input_file": "<local file path>"}
		var payload struct {
			InputFile string `json:"input_file"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload: " + err.Error()})
			return
		}
		if payload.InputFile == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "input_file is required"})
			return
		}
		if _, err := os.Stat(payload.InputFile); os.IsNotExist(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Input file does not exist"})
			return
		}

		// Extract base name for output naming.
		baseName := strings.TrimSuffix(filepath.Base(payload.InputFile), filepath.Ext(payload.InputFile))

		// Define transcoding targets: HLS outputs using H.264.
		targets := []TranscodeTarget{
			{"720p_mp4", "720", "m3u8"},
			{"480p_mp4", "480", "m3u8"},
			{"360p_mp4", "360", "m3u8"},
		}

		var transcodedVideos []Video

		// Transcode for each target.
		for _, target := range targets {
			// Construct output manifest file name.
			outputManifest := "./uploads/" + baseName + "_" + target.Label + "." + target.OutputExt
			// Transcode using FFmpeg: HLS output with H.264 and copying audio.
			err := transcodeToHLS(payload.InputFile, outputManifest, target.Resolution)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Transcoding failed for " + target.Label + ": " + err.Error()})
				return
			}
			// Upload the generated HLS manifest (and segments) to IPFS.
			newCID, err := uploadVideoToIPFS(outputManifest)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload transcoded video (" + target.Label + ") to IPFS: " + err.Error()})
				return
			}
			// Create a new video record.
			video := Video{
				Title:       baseName + "_" + target.Label + "." + target.OutputExt,
				Description: "Transcoded (" + target.Label + ") video from " + payload.InputFile,
				FilePath:    outputManifest,
				IPFSCID:     newCID,
				CreatedAt:   time.Now(),
			}
			if err := DB.Create(&video).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transcoded video record (" + target.Label + "): " + err.Error()})
				return
			}
			transcodedVideos = append(transcodedVideos, video)
		}

		// Return details of all transcoded videos.
		c.JSON(http.StatusOK, gin.H{
			"message": "Video transcoded to HLS and uploaded to IPFS successfully",
			"videos":  transcodedVideos,
		})
	})

	// Social interaction stub endpoint.
	router.POST("/social/action", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Social action recorded"})
	})

	// Blockchain anchoring stub endpoint.
	router.POST("/blockchain/anchor", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Blockchain anchoring stub"})
	})

	// Redis test endpoint.
	router.GET("/cache/test", func(c *gin.Context) {
		err := Cache.Set(ctx, "testKey", "Hello, Redis!", 10*time.Minute).Err()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		val, err := Cache.Get(ctx, "testKey").Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Cache test successful",
			"value":   val,
		})
	})

	// List videos endpoint: returns video records.
	router.GET("/video/list", func(c *gin.Context) {
		var videos []Video
		if err := DB.Find(&videos).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"videos": videos})
	})

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}
}
