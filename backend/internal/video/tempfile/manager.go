package tempfile

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/google/uuid"
)

// Manager handles temporary file operations
type Manager struct {
	baseDir     string
	activeDirs  map[string]bool
	logger      logger.Logger
	mu          sync.RWMutex
	permissions os.FileMode
}

// Config represents the configuration for the temporary file manager
type Config struct {
	BaseDir     string      // Base directory for temporary files
	Permissions os.FileMode // File permissions for created directories
}

// NewManager creates a new temporary file manager
func NewManager(config *Config, logger logger.Logger) (*Manager, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(config.BaseDir, config.Permissions); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &Manager{
		baseDir:     config.BaseDir,
		activeDirs:  make(map[string]bool),
		logger:      logger,
		permissions: config.Permissions,
	}, nil
}

// CreateTempDir creates a new temporary directory and returns its path
func (m *Manager) CreateTempDir() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate unique directory name
	dirName := uuid.New().String()
	dirPath := filepath.Join(m.baseDir, dirName)

	// Create the directory
	if err := os.MkdirAll(dirPath, m.permissions); err != nil {
		m.logger.LogError(err, fmt.Sprintf("Failed to create temporary directory: path=%s", dirPath))
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Track the directory
	m.activeDirs[dirPath] = true

	m.logger.LogInfo("Created temporary directory", map[string]interface{}{
		"path": dirPath,
	})

	return dirPath, nil
}

// CleanupDir removes a temporary directory and its contents
func (m *Manager) CleanupDir(dirPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if this is a managed directory
	if !m.activeDirs[dirPath] {
		return fmt.Errorf("not a managed temporary directory: %s", dirPath)
	}

	// Remove the directory and its contents
	if err := os.RemoveAll(dirPath); err != nil {
		m.logger.LogError(err, fmt.Sprintf("Failed to cleanup temporary directory: path=%s", dirPath))
		return fmt.Errorf("failed to cleanup temporary directory: %w", err)
	}

	// Remove from tracking
	delete(m.activeDirs, dirPath)

	m.logger.LogInfo("Cleaned up temporary directory", map[string]interface{}{
		"path": dirPath,
	})

	return nil
}

// CleanupAll removes all managed temporary directories
func (m *Manager) CleanupAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for dirPath := range m.activeDirs {
		if err := os.RemoveAll(dirPath); err != nil {
			m.logger.LogError(err, fmt.Sprintf("Failed to cleanup temporary directory: path=%s", dirPath))
			lastErr = err
		} else {
			delete(m.activeDirs, dirPath)
		}
	}

	if lastErr != nil {
		return fmt.Errorf("failed to cleanup all temporary directories: %w", lastErr)
	}

	m.logger.LogInfo("Cleaned up all temporary directories", nil)
	return nil
}

// IsManaged checks if a directory is managed by this manager
func (m *Manager) IsManaged(dirPath string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeDirs[dirPath]
}

// GetActiveDirs returns a list of all active temporary directories
func (m *Manager) GetActiveDirs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dirs := make([]string, 0, len(m.activeDirs))
	for dir := range m.activeDirs {
		dirs = append(dirs, dir)
	}
	return dirs
}
