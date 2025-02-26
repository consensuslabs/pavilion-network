package mocks

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockStorageService is a mock implementation of storage service
type MockStorageService struct {
	mock.Mock
}

func (m *MockStorageService) UploadVideo(ctx context.Context, videoID uuid.UUID, resolution string, file io.Reader) (string, error) {
	args := m.Called(ctx, videoID, resolution, file)
	return args.String(0), args.Error(1)
}

func (m *MockStorageService) DeleteVideo(ctx context.Context, videoID uuid.UUID) error {
	args := m.Called(ctx, videoID)
	return args.Error(0)
}

func (m *MockStorageService) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStorageService) GetVideoURL(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}
