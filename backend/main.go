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
	"github.com/google/uuid"
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
	CreatedAt time.Time `json:"createdAt"`
}

// Video model definition.
// fileId stores the unique identifier generated during upload.
// Transcodes represents the one-to-many relation to the Transcode table.
type Video struct {
	ID          uint        `gorm:"primaryKey" json:"id"`
	FileId      string      `json:"fileId"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	FilePath    string      `json:"filePath"`
	IPFSCID     string      `gorm:"column:ipfs_cid" json:"ipfsCid"`
	CreatedAt   time.Time   `json:"createdAt"`
	Transcodes  []Transcode `json:"transcodes"`
}

// Transcode model definition.
type Transcode struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	VideoID    uint      `json:"videoId"`
	FilePath   string    `json:"filePath"`
	FileCID    string    `gorm:"column:file_cid" json:"fileCid"`
	Type       string    `json:"type"`       // "hlsManifest", "hlsSegment", or "h264"
	Resolution string    `json:"resolution"` // e.g., "720", "480", "360", "240"
	CreatedAt  time.Time `json:"createdAt"`
}

// TranscodeTarget defines parameters for each transcoded output.
type TranscodeTarget struct {
	Label      string // e.g., "720pMp4", "480pMp4", "360pMp4", "240pMp4"
	Resolution string // target height (e.g., "720", "480", "360", "240")
	OutputExt  string // "m3u8" for HLS outputs, "mp4" for progressive MP4
}

// Standardized response helpers.
func successResponse(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"message": message,
	})
}

func errorResponse(c *gin.Context, status int, code, message string, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	c.JSON(status, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
			"details": errMsg,
		},
	})
}

// ConnectDatabase initializes the PostgreSQL connection using GORM.
func ConnectDatabase() {
	dsn := "host=localhost user=youruser password=yourpassword dbname=pavilion_db port=5432 sslmode=disable TimeZone=UTC"
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	if err = DB.AutoMigrate(&User{}, &Video{}, &Transcode{}); err != nil {
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

// transcodeToHLS uses FFmpeg to transcode a video to an HLS output.
func transcodeToHLS(inputFile, outputManifest, scaleHeight string) error {
	cmd := exec.Command("ffmpeg", "-i", inputFile, "-vf", "scale=-2:"+scaleHeight, "-c:v", "libx264", "-c:a", "copy", "-preset", "fast", "-hls_time", "10", "-hls_playlist_type", "vod", outputManifest)
	return cmd.Run()
}

// transcodeToMP4 transcodes the original video to a smaller MP4 output.
func transcodeToMP4(inputFile, outputFile, scaleHeight string) error {
	cmd := exec.Command("ffmpeg", "-i", inputFile, "-vf", "scale=-2:"+scaleHeight, "-c:v", "libx264", "-c:a", "aac", "-preset", "fast", outputFile)
	return cmd.Run()
}

// uploadSegmentsAndAdjustManifestForTarget scans for TS segments generated for a specific target,
// uploads them to IPFS, adjusts the manifest file (replacing TS filenames with absolute IPFS URLs),
// and returns a map of local TS filenames to their corresponding IPFS CID.
func uploadSegmentsAndAdjustManifestForTarget(manifestPath, baseIPFSGatewayURL, targetLabel string) (map[string]string, error) {
	dir := filepath.Dir(manifestPath)
	// Derive the base name from the manifest file (without extension).
	manifestBase := strings.TrimSuffix(filepath.Base(manifestPath), filepath.Ext(manifestPath))
	// Build a pattern to match only TS segments for this target.
	pattern := filepath.Join(dir, manifestBase+"_*"+".ts")
	segmentFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	segmentCIDs := make(map[string]string)
	for _, segFile := range segmentFiles {
		cid, err := uploadVideoToIPFS(segFile)
		if err != nil {
			return nil, err
		}
		filename := filepath.Base(segFile)
		segmentCIDs[filename] = cid
		log.Printf("Uploaded segment %s, CID: %s", filename, cid)
	}

	// Read and adjust manifest.
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		// For lines that are TS segment references, replace with absolute URL.
		if line != "" && !strings.HasPrefix(line, "#") && strings.HasSuffix(line, ".ts") {
			if segCID, ok := segmentCIDs[line]; ok {
				lines[i] = baseIPFSGatewayURL + "/" + segCID
			} else {
				log.Printf("Warning: No IPFS CID found for segment: %s", line)
			}
		}
	}
	newManifest := strings.Join(lines, "\n")
	if err := os.WriteFile(manifestPath, []byte(newManifest), 0644); err != nil {
		return nil, err
	}
	return segmentCIDs, nil
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
	ConnectDatabase()
	ConnectRedis()
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
		successResponse(c, nil, "Health check successful.")
	})

	// OAuth login stub endpoint.
	router.POST("/auth/login", func(c *gin.Context) {
		user := User{
			Name:  "Test User",
			Email: "test@example.com",
		}
		if err := DB.Create(&user).Error; err != nil {
			errorResponse(c, http.StatusInternalServerError, "ERR_DB", "Failed to save user", err)
			return
		}
		successResponse(c, gin.H{"user": user}, "OAuth login stub - token: dummy-token")
	})

	// Video upload endpoint.
	router.POST("/video/upload", func(c *gin.Context) {
		file, err := c.FormFile("video")
		if err != nil {
			errorResponse(c, http.StatusBadRequest, "ERR_NO_FILE", "No video file received", err)
			return
		}
		uploadDir := "./uploads"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			errorResponse(c, http.StatusInternalServerError, "ERR_MKDIR", "Failed to create uploads directory", err)
			return
		}
		// Generate a unique file ID.
		fileId := uuid.New().String()
		ext := filepath.Ext(file.Filename)
		uniqueName := fileId + ext
		destination := filepath.Join(uploadDir, uniqueName)
		if err := c.SaveUploadedFile(file, destination); err != nil {
			errorResponse(c, http.StatusInternalServerError, "ERR_SAVE_FILE", "Failed to save uploaded file", err)
			return
		}
		// Read additional form fields.
		inputTitle := c.PostForm("title")
		description := c.PostForm("description")
		// Use provided title if available; otherwise, use the fileId.
		title := fileId
		if inputTitle != "" {
			title = inputTitle
		}
		cid, err := uploadVideoToIPFS(destination)
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, "ERR_IPFS_UPLOAD", "Failed to upload file to IPFS", err)
			return
		}
		video := Video{
			FileId:      fileId,
			Title:       title,
			Description: description,
			FilePath:    destination,
			IPFSCID:     cid,
			CreatedAt:   time.Now(),
		}
		if err := DB.Create(&video).Error; err != nil {
			errorResponse(c, http.StatusInternalServerError, "ERR_DB", "Failed to save video record", err)
			return
		}
		successResponse(c, gin.H{"video": video, "filePath": destination, "ipfsCid": cid}, "Video uploaded and stored in IPFS successfully")
	})

	// Transcoding endpoint.
	// Accepts either an "inputFile" or a "cid" in the JSON payload.
	// Transcoded files will be named based on the video's fileId.
	router.POST("/video/transcode", func(c *gin.Context) {
		var payload struct {
			InputFile string `json:"inputFile"`
			CID       string `json:"cid"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			errorResponse(c, http.StatusBadRequest, "ERR_INVALID_PAYLOAD", "Invalid JSON payload", err)
			return
		}
		// Lookup video record if CID is provided.
		var sourceVideo Video
		if payload.CID != "" {
			if err := DB.Where("ipfs_cid = ?", payload.CID).First(&sourceVideo).Error; err != nil {
				errorResponse(c, http.StatusBadRequest, "ERR_NOT_FOUND", "No video record found for provided CID", err)
				return
			}
			payload.InputFile = sourceVideo.FilePath
		}
		if payload.InputFile == "" {
			errorResponse(c, http.StatusBadRequest, "ERR_NO_INPUT", "inputFile is required", nil)
			return
		}
		if _, err := os.Stat(payload.InputFile); os.IsNotExist(err) {
			errorResponse(c, http.StatusBadRequest, "ERR_FILE_NOT_FOUND", "Input file does not exist", err)
			return
		}
		// Use the video's fileId as the base name.
		baseName := ""
		if sourceVideo.FileId != "" {
			baseName = sourceVideo.FileId
		} else {
			baseName = strings.TrimSuffix(filepath.Base(payload.InputFile), filepath.Ext(payload.InputFile))
		}

		// Define HLS transcoding targets.
		hlsTargets := []TranscodeTarget{
			{"720pMp4", "720", "m3u8"},
			{"480pMp4", "480", "m3u8"},
			{"360pMp4", "360", "m3u8"},
		}
		// Define an additional target for a smaller MP4.
		mp4Target := TranscodeTarget{"240pMp4", "240", "mp4"}

		var transcodes []Transcode
		baseIPFSGatewayURL := "http://localhost:8081/ipfs"

		// Process HLS targets.
		for _, target := range hlsTargets {
			outputManifest := filepath.Join("./uploads", baseName+"_"+target.Label+"."+target.OutputExt)
			if err := transcodeToHLS(payload.InputFile, outputManifest, target.Resolution); err != nil {
				errorResponse(c, http.StatusInternalServerError, "ERR_HLS_TRANSCODE", "Transcoding failed for "+target.Label, err)
				return
			}
			// Adjust the manifest for IPFS: replace TS filenames with absolute IPFS URLs.
			segmentsMap, err := uploadSegmentsAndAdjustManifestForTarget(outputManifest, baseIPFSGatewayURL, target.Label)
			if err != nil {
				errorResponse(c, http.StatusInternalServerError, "ERR_MANIFEST_ADJUST", "Failed to adjust HLS manifest for "+target.Label, err)
				return
			}
			newCID, err := uploadVideoToIPFS(outputManifest)
			if err != nil {
				errorResponse(c, http.StatusInternalServerError, "ERR_IPFS_UPLOAD", "Failed to upload HLS manifest ("+target.Label+") to IPFS", err)
				return
			}
			// Create a record for the manifest file.
			manifestRecord := Transcode{
				VideoID:    sourceVideo.ID,
				FilePath:   outputManifest,
				FileCID:    newCID,
				Type:       "hlsManifest",
				Resolution: target.Resolution,
				CreatedAt:  time.Now(),
			}
			if err := DB.Create(&manifestRecord).Error; err != nil {
				errorResponse(c, http.StatusInternalServerError, "ERR_DB", "Failed to save HLS manifest record ("+target.Label+")", err)
				return
			}
			transcodes = append(transcodes, manifestRecord)

			// Create a record for each TS segment.
			for segFile, segCID := range segmentsMap {
				segmentPath := filepath.Join(filepath.Dir(outputManifest), segFile)
				segRecord := Transcode{
					VideoID:    sourceVideo.ID,
					FilePath:   segmentPath,
					FileCID:    segCID,
					Type:       "hlsSegment",
					Resolution: target.Resolution,
					CreatedAt:  time.Now(),
				}
				if err := DB.Create(&segRecord).Error; err != nil {
					errorResponse(c, http.StatusInternalServerError, "ERR_DB", "Failed to save HLS segment record ("+segFile+")", err)
					return
				}
				transcodes = append(transcodes, segRecord)
			}
		}

		// Process additional MP4 target.
		mp4Output := filepath.Join("./uploads", baseName+"_"+mp4Target.Label+"."+mp4Target.OutputExt)
		if err := transcodeToMP4(payload.InputFile, mp4Output, mp4Target.Resolution); err != nil {
			errorResponse(c, http.StatusInternalServerError, "ERR_MP4_TRANSCODE", "MP4 transcoding failed for "+mp4Target.Label, err)
			return
		}
		mp4CID, err := uploadVideoToIPFS(mp4Output)
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, "ERR_IPFS_UPLOAD", "Failed to upload MP4 output ("+mp4Target.Label+") to IPFS", err)
			return
		}
		mp4Record := Transcode{
			VideoID:    sourceVideo.ID,
			FilePath:   mp4Output,
			FileCID:    mp4CID,
			Type:       "h264",
			Resolution: mp4Target.Resolution,
			CreatedAt:  time.Now(),
		}
		if err := DB.Create(&mp4Record).Error; err != nil {
			errorResponse(c, http.StatusInternalServerError, "ERR_DB", "Failed to save MP4 transcode record ("+mp4Target.Label+")", err)
			return
		}
		transcodes = append(transcodes, mp4Record)

		// Reload the source video with associated transcodes.
		if err := DB.Preload("Transcodes").First(&sourceVideo, sourceVideo.ID).Error; err != nil {
			errorResponse(c, http.StatusInternalServerError, "ERR_DB", "Failed to reload video record", err)
			return
		}

		successResponse(c, gin.H{"video": sourceVideo}, "Video transcoded (HLS and MP4) and uploaded to IPFS successfully")
	})

	// Watch video endpoint.
	router.GET("/video/watch", func(c *gin.Context) {
		cid := c.Query("cid")
		file := c.Query("file")
		if cid != "" {
			ipfsURL := "http://localhost:8081/ipfs/" + cid
			c.Redirect(http.StatusTemporaryRedirect, ipfsURL)
		} else if file != "" {
			c.File(filepath.Join("./uploads", file))
		} else {
			errorResponse(c, http.StatusBadRequest, "ERR_NO_PARAM", "No 'cid' or 'file' parameter provided", nil)
		}
	})

	// Social action stub endpoint.
	router.POST("/social/action", func(c *gin.Context) {
		successResponse(c, nil, "Social action recorded")
	})

	// Blockchain anchoring stub endpoint.
	router.POST("/blockchain/anchor", func(c *gin.Context) {
		successResponse(c, nil, "Blockchain anchoring stub")
	})

	// Redis test endpoint.
	router.GET("/cache/test", func(c *gin.Context) {
		if err := Cache.Set(ctx, "testKey", "Hello, Redis!", 10*time.Minute).Err(); err != nil {
			errorResponse(c, http.StatusInternalServerError, "ERR_CACHE_SET", "Failed to set cache", err)
			return
		}
		val, err := Cache.Get(ctx, "testKey").Result()
		if err != nil {
			errorResponse(c, http.StatusInternalServerError, "ERR_CACHE_GET", "Failed to get cache", err)
			return
		}
		successResponse(c, gin.H{"value": val}, "Cache test successful")
	})

	// List videos endpoint: returns videos sorted by createdAt descending with associated transcodes.
	router.GET("/video/list", func(c *gin.Context) {
		var videos []Video
		if err := DB.Preload("Transcodes").Order("created_at desc").Find(&videos).Error; err != nil {
			errorResponse(c, http.StatusInternalServerError, "ERR_DB", "Failed to list videos", err)
			return
		}
		successResponse(c, gin.H{"videos": videos}, "Videos retrieved successfully")
	})

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}
}
