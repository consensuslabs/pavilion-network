package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	shell "github.com/ipfs/go-ipfs-api"
	"gorm.io/gorm"
)

// TranscodeService handles video transcoding operations
type TranscodeService struct {
	db     *gorm.DB
	ipfs   *IPFSService
	config *Config
}

// NewTranscodeService creates a new transcoding service instance
func NewTranscodeService(db *gorm.DB, ipfs *IPFSService, config *Config) *TranscodeService {
	return &TranscodeService{
		db:     db,
		ipfs:   ipfs,
		config: config,
	}
}

// transcodeToHLS uses FFmpeg to transcode a video to an HLS output
func (s *TranscodeService) transcodeToHLS(inputFile, outputManifest, scaleHeight string) error {
	cmd := exec.Command(
		s.config.Ffmpeg.Path,
		"-i", inputFile,
		"-vf", "scale=-2:"+scaleHeight,
		"-c:v", s.config.Ffmpeg.VideoCodec,
		"-c:a", s.config.Ffmpeg.AudioCodec,
		"-preset", s.config.Ffmpeg.Preset,
		"-hls_time", string(s.config.Ffmpeg.HLSTime),
		"-hls_playlist_type", s.config.Ffmpeg.HLSPlaylistType,
		outputManifest,
	)
	return cmd.Run()
}

// transcodeToMP4 transcodes the original video to a smaller MP4 output
func (s *TranscodeService) transcodeToMP4(inputFile, outputFile, scaleHeight string) error {
	cmd := exec.Command(
		s.config.Ffmpeg.Path,
		"-i", inputFile,
		"-vf", "scale=-2:"+scaleHeight,
		"-c:v", s.config.Ffmpeg.VideoCodec,
		"-c:a", s.config.Ffmpeg.AudioCodec,
		"-preset", s.config.Ffmpeg.Preset,
		outputFile,
	)
	return cmd.Run()
}

// uploadSegmentsAndAdjustManifestForTarget scans for TS segments and uploads them to IPFS
func (s *TranscodeService) uploadSegmentsAndAdjustManifestForTarget(manifestPath, targetLabel string) (map[string]string, error) {
	dir := filepath.Dir(manifestPath)
	manifestBase := strings.TrimSuffix(filepath.Base(manifestPath), filepath.Ext(manifestPath))
	pattern := filepath.Join(dir, manifestBase+"_*"+".ts")
	segmentFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	segmentCIDs := make(map[string]string)
	for _, segFile := range segmentFiles {
		cid, err := s.ipfs.UploadFile(segFile)
		if err != nil {
			return nil, err
		}
		filename := filepath.Base(segFile)
		segmentCIDs[filename] = cid
		log.Printf("Uploaded segment %s, CID: %s", filename, cid)
	}

	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") && strings.HasSuffix(line, ".ts") {
			if segCID, ok := segmentCIDs[line]; ok {
				lines[i] = s.ipfs.GetGatewayURL(segCID)
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

func downloadFileFromIPFS(cid string) (string, error) {
	sh := shell.NewShell("localhost:5001")
	r, err := sh.Cat(cid)
	if err != nil {
		return "", err
	}
	defer r.Close()
	tempFile := "temp_" + uuid.New().String() + ".mp4"
	outFile, err := os.Create(tempFile)
	if err != nil {
		return "", err
	}
	defer outFile.Close()
	_, err = io.Copy(outFile, r)
	if err != nil {
		return "", err
	}
	return tempFile, nil
}

// transcodeByCID downloads a video from IPFS, transcodes it, and uploads the results
func (s *TranscodeService) transcodeByCID(cid string, videoID uint, target TranscodeTarget) error {
	inputFile, err := s.ipfs.DownloadFile(cid)
	if err != nil {
		return err
	}
	manifestPath := strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + "_" + target.Label + ".m3u8"
	err = s.transcodeToHLS(inputFile, manifestPath, target.Resolution)
	if err != nil {
		return err
	}
	segmentsMap, err := s.uploadSegmentsAndAdjustManifestForTarget(manifestPath, target.Label)
	if err != nil {
		return err
	}
	manifestCID, err := s.ipfs.UploadFile(manifestPath)
	if err != nil {
		return err
	}
	newTranscode := Transcode{
		VideoID:    videoID,
		FilePath:   manifestPath,
		FileCID:    manifestCID,
		Type:       "hlsManifest",
		Resolution: target.Resolution,
		CreatedAt:  time.Now(),
	}
	if err := s.db.Create(&newTranscode).Error; err != nil {
		return err
	}
	seq := 1
	for segmentPath, segCID := range segmentsMap {
		newSegment := TranscodeSegment{
			TranscodeID: newTranscode.ID,
			FilePath:    segmentPath,
			FileCID:     segCID,
			Sequence:    seq,
			CreatedAt:   time.Now(),
		}
		if err := s.db.Create(&newSegment).Error; err != nil {
			return err
		}
		seq++
	}
	return nil
}
