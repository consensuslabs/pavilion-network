package main

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// Standardized response helpers
func (a *App) successResponse(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"message": message,
	})
}

func (a *App) errorResponse(c *gin.Context, status int, code, message string, err error) {
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

// handleHealthCheck handles the health check endpoint
func (a *App) handleHealthCheck(c *gin.Context) {
	a.successResponse(c, nil, "Health check successful")
}

// handleLogin handles the OAuth login endpoint
func (a *App) handleLogin(c *gin.Context) {
	user, err := a.auth.Login("test@example.com")
	if err != nil {
		a.errorResponse(c, http.StatusInternalServerError, "ERR_DB", "Failed to save user", err)
		return
	}
	a.successResponse(c, gin.H{"user": user}, "OAuth login stub - token: dummy-token")
}

// handleVideoUpload handles the video upload endpoint
func (a *App) handleVideoUpload(c *gin.Context) {
	file, err := c.FormFile("video")
	if err != nil {
		a.errorResponse(c, http.StatusBadRequest, "ERR_NO_FILE", "No video file received", err)
		return
	}

	uploadedFile, err := file.Open()
	if err != nil {
		a.errorResponse(c, http.StatusInternalServerError, "ERR_FILE_OPEN", "Failed to open uploaded file", err)
		return
	}
	defer uploadedFile.Close()

	title := c.PostForm("title")
	description := c.PostForm("description")

	video, err := a.video.UploadVideo(uploadedFile, file.Filename, title, description)
	if err != nil {
		a.errorResponse(c, http.StatusInternalServerError, "ERR_UPLOAD", "Failed to process video upload", err)
		return
	}

	a.successResponse(c, gin.H{
		"video":    video,
		"filePath": video.FilePath,
		"ipfsCid":  video.IPFSCID,
	}, "Video uploaded and stored in IPFS successfully")
}

// handleVideoWatch handles the video watch endpoint
func (a *App) handleVideoWatch(c *gin.Context) {
	cid := c.Query("cid")
	file := c.Query("file")

	if cid != "" {
		ipfsURL := a.ipfs.GetGatewayURL(cid)
		c.Redirect(http.StatusTemporaryRedirect, ipfsURL)
		return
	}

	if file != "" {
		c.File(filepath.Join(a.Config.Storage.UploadDir, file))
		return
	}

	a.errorResponse(c, http.StatusBadRequest, "ERR_NO_PARAM", "No 'cid' or 'file' parameter provided", nil)
}

// handleVideoList handles the video list endpoint
func (a *App) handleVideoList(c *gin.Context) {
	videos, err := a.video.GetVideoList()
	if err != nil {
		a.errorResponse(c, http.StatusInternalServerError, "ERR_DB", "Failed to list videos", err)
		return
	}
	a.successResponse(c, gin.H{"videos": videos}, "Videos retrieved successfully")
}
