package mocks

import (
	"context"

	"github.com/consensuslabs/pavilion-network/backend/internal/video/ffmpeg"
	"github.com/stretchr/testify/mock"
)

// MockFFmpegService is a mock implementation of ffmpeg.Service
type MockFFmpegService struct {
	mock.Mock
}

func (m *MockFFmpegService) GetMetadata(ctx context.Context, filePath string) (*ffmpeg.VideoMetadata, error) {
	args := m.Called(ctx, filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ffmpeg.VideoMetadata), args.Error(1)
}

func (m *MockFFmpegService) Transcode(ctx context.Context, inputPath, outputPath, resolution string) error {
	args := m.Called(ctx, inputPath, outputPath, resolution)
	return args.Error(0)
}

// MockTempFileManager is a mock implementation of tempfile.TempFileManager
type MockTempFileManager struct {
	mock.Mock
}

func (m *MockTempFileManager) CreateTempDir() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockTempFileManager) CleanupDir(dir string) error {
	args := m.Called(dir)
	return args.Error(0)
}

func (m *MockTempFileManager) CleanupAll() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockTempFileManager) GetActiveDirs() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockTempFileManager) IsManaged(dirPath string) bool {
	args := m.Called(dirPath)
	return args.Bool(0)
}
