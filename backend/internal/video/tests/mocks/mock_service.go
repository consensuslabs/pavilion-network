package mocks

import (
	"io"
	"mime/multipart"

	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockVideoService is a mock implementation of VideoService
type MockVideoService struct {
	mock.Mock
}

func (m *MockVideoService) InitializeUpload(title, description string, size int64) (*video.VideoUpload, error) {
	args := m.Called(title, description, size)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*video.VideoUpload), args.Error(1)
}

func (m *MockVideoService) ProcessUpload(upload *video.VideoUpload, file multipart.File, header *multipart.FileHeader) error {
	args := m.Called(upload, file, header)
	return args.Error(0)
}

func (m *MockVideoService) GetVideo(videoID uuid.UUID) (*video.Video, error) {
	args := m.Called(videoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*video.Video), args.Error(1)
}

func (m *MockVideoService) ListVideos(page, limit int) ([]video.Video, error) {
	args := m.Called(page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]video.Video), args.Error(1)
}

func (m *MockVideoService) DeleteVideo(videoID uuid.UUID) error {
	args := m.Called(videoID)
	return args.Error(0)
}

func (m *MockVideoService) UpdateVideo(videoID uuid.UUID, title, description string) error {
	args := m.Called(videoID, title, description)
	return args.Error(0)
}

func (m *MockVideoService) GetVideoUpload(videoID uuid.UUID) (*video.VideoUpload, error) {
	args := m.Called(videoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*video.VideoUpload), args.Error(1)
}

func (m *MockVideoService) CreateVideo(userId uuid.UUID, title string, description string) (*video.Video, error) {
	args := m.Called(userId, title, description)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*video.Video), args.Error(1)
}

// MockIPFSService is a mock implementation of IPFSService
type MockIPFSService struct {
	mock.Mock
}

func (m *MockIPFSService) UploadFileStream(reader io.Reader) (string, error) {
	args := m.Called(reader)
	return args.String(0), args.Error(1)
}

func (m *MockIPFSService) DownloadFile(cid string) (string, error) {
	args := m.Called(cid)
	return args.String(0), args.Error(1)
}

