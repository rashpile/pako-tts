package mocks

import (
	"bytes"
	"context"
	"io"

	"github.com/pako-tts/server/internal/domain"
)

// MockStorage is a mock implementation of domain.AudioStorage for testing.
type MockStorage struct {
	StoreFunc    func(ctx context.Context, jobID string, audio []byte, format string) (string, error)
	RetrieveFunc func(ctx context.Context, jobID string) (io.ReadCloser, string, error)
	DeleteFunc   func(ctx context.Context, jobID string) error
	ExistsFunc   func(ctx context.Context, jobID string) bool
	GetPathFunc  func(ctx context.Context, jobID string) string
	StoredFiles  map[string][]byte
	StoreError   error
	RetrieveError error
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		StoredFiles: make(map[string][]byte),
	}
}

func (m *MockStorage) Store(ctx context.Context, jobID string, audio []byte, format string) (string, error) {
	if m.StoreFunc != nil {
		return m.StoreFunc(ctx, jobID, audio, format)
	}
	if m.StoreError != nil {
		return "", m.StoreError
	}
	path := "/storage/" + jobID + "." + format
	m.StoredFiles[jobID] = audio
	return path, nil
}

func (m *MockStorage) Retrieve(ctx context.Context, jobID string) (io.ReadCloser, string, error) {
	if m.RetrieveFunc != nil {
		return m.RetrieveFunc(ctx, jobID)
	}
	if m.RetrieveError != nil {
		return nil, "", m.RetrieveError
	}
	data, ok := m.StoredFiles[jobID]
	if !ok {
		return nil, "", domain.ErrJobNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), "audio/mpeg", nil
}

func (m *MockStorage) Delete(ctx context.Context, jobID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, jobID)
	}
	delete(m.StoredFiles, jobID)
	return nil
}

func (m *MockStorage) Exists(ctx context.Context, jobID string) bool {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(ctx, jobID)
	}
	_, ok := m.StoredFiles[jobID]
	return ok
}

func (m *MockStorage) GetPath(ctx context.Context, jobID string) string {
	if m.GetPathFunc != nil {
		return m.GetPathFunc(ctx, jobID)
	}
	if _, ok := m.StoredFiles[jobID]; ok {
		return "/storage/" + jobID + ".mp3"
	}
	return ""
}
