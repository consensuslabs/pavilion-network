package tempfile

// TempFileManager defines the interface for temporary file operations
type TempFileManager interface {
	// CreateTempDir creates a new temporary directory and returns its path
	CreateTempDir() (string, error)

	// CleanupDir removes a temporary directory and its contents
	CleanupDir(dirPath string) error

	// CleanupAll removes all managed temporary directories
	CleanupAll() error

	// IsManaged checks if a directory is managed by this manager
	IsManaged(dirPath string) bool

	// GetActiveDirs returns a list of all active temporary directories
	GetActiveDirs() []string
}
