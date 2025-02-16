package http

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// StaticFileConfig represents configuration for static file serving
type StaticFileConfig struct {
	URLPath   string // URL path to serve files from
	FilePath  string // Physical path to the files
	IndexFile string // Name of the index file (e.g., "index.html")
}

// ServeStaticFiles configures static file serving for the router
func ServeStaticFiles(router *gin.Engine, configs []StaticFileConfig) error {
	for _, config := range configs {
		// Verify that the directory exists
		if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
			return fmt.Errorf("static file directory does not exist: %s", config.FilePath)
		}

		// Convert relative path to absolute
		absPath, err := filepath.Abs(config.FilePath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %v", config.FilePath, err)
		}

		// Create a file server handler
		fileServer := http.FileServer(http.Dir(absPath))

		// Create a handler function that serves files and handles index.html
		handler := func(c *gin.Context) {
			// Remove the URL prefix to get the file path
			path := strings.TrimPrefix(c.Request.URL.Path, config.URLPath)
			if path == "" || path == "/" {
				if config.IndexFile != "" {
					// Serve index file for root requests
					indexPath := filepath.Join(absPath, config.IndexFile)
					if _, err := os.Stat(indexPath); err == nil {
						c.File(indexPath)
						return
					}
				}
			}

			// Add HTML cache headers if needed
			if filepath.Ext(path) == ".html" {
				c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
				c.Header("Pragma", "no-cache")
				c.Header("Expires", "0")
			}

			// Serve the file using the standard file server
			fileServer.ServeHTTP(c.Writer, c.Request)
		}

		// Register the handler for all paths under the URL prefix
		router.GET(config.URLPath+"/*path", handler)
		if config.IndexFile != "" {
			router.GET(config.URLPath, handler)
		}
	}

	return nil
}
