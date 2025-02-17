package storage

// Config represents storage configuration
type Config struct {
	UploadDir string     `mapstructure:"uploadDir"`
	TempDir   string     `mapstructure:"tempDir"`
	IPFS      IPFSConfig `mapstructure:"ipfs"`
	S3        S3Config   `mapstructure:"s3"`
}

// IPFSConfig represents IPFS configuration settings
type IPFSConfig struct {
	APIAddress string `mapstructure:"apiAddress"`
	Gateway    string `mapstructure:"gateway"`
}

// S3Config represents S3 configuration settings
type S3Config struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"accessKeyId"`
	SecretAccessKey string `mapstructure:"secretAccessKey"`
	UseSSL          bool   `mapstructure:"useSSL"`
	Region          string `mapstructure:"region"`
	Bucket          string `mapstructure:"bucket"`
}
