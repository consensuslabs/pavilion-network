package video

// App represents the application context needed by video handlers
type App struct {
	Config          *Config
	Logger          Logger
	Video           VideoService
	IPFS            IPFSService
	ResponseHandler ResponseHandler
}

// Config represents the configuration for video handling
type Config struct {
	Storage struct {
		UploadDir string
	}
	Video struct {
		MaxSize        int64
		MinTitleLength int
		MaxTitleLength int
		MaxDescLength  int
		AllowedFormats []string
	}
}
