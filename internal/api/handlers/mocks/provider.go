package mocks

import (
	"bytes"
	"context"

	"github.com/pako-tts/server/internal/domain"
)

// MockProvider is a mock implementation of domain.TTSProvider for testing.
type MockProvider struct {
	NameValue         string
	AvailableValue    bool
	MaxConcurrentVal  int
	ActiveJobsVal     int
	SynthesizeFunc    func(ctx context.Context, req *domain.SynthesisRequest) (*domain.SynthesisResult, error)
	ListVoicesFunc    func(ctx context.Context) ([]domain.Voice, error)
	SynthesizeError   error
	SynthesizeResult  *domain.SynthesisResult
}

func (m *MockProvider) Name() string {
	return m.NameValue
}

func (m *MockProvider) Synthesize(ctx context.Context, req *domain.SynthesisRequest) (*domain.SynthesisResult, error) {
	if m.SynthesizeFunc != nil {
		return m.SynthesizeFunc(ctx, req)
	}
	if m.SynthesizeError != nil {
		return nil, m.SynthesizeError
	}
	if m.SynthesizeResult != nil {
		return m.SynthesizeResult, nil
	}
	// Default mock result
	return &domain.SynthesisResult{
		Audio:       bytes.NewReader([]byte("mock audio data")),
		ContentType: "audio/mpeg",
		SizeBytes:   15,
	}, nil
}

func (m *MockProvider) ListVoices(ctx context.Context) ([]domain.Voice, error) {
	if m.ListVoicesFunc != nil {
		return m.ListVoicesFunc(ctx)
	}
	return []domain.Voice{
		{
			VoiceID:  "voice1",
			Name:     "Test Voice 1",
			Provider: m.NameValue,
			Language: "en",
			Gender:   "female",
		},
		{
			VoiceID:  "voice2",
			Name:     "Test Voice 2",
			Provider: m.NameValue,
			Language: "en",
			Gender:   "male",
		},
	}, nil
}

func (m *MockProvider) IsAvailable(ctx context.Context) bool {
	return m.AvailableValue
}

func (m *MockProvider) MaxConcurrent() int {
	if m.MaxConcurrentVal == 0 {
		return 4
	}
	return m.MaxConcurrentVal
}

func (m *MockProvider) ActiveJobs() int {
	return m.ActiveJobsVal
}

func (m *MockProvider) Info(ctx context.Context) domain.ProviderInfo {
	return domain.ProviderInfo{
		Name:          m.NameValue,
		Type:          "MockProvider",
		MaxConcurrent: m.MaxConcurrent(),
		IsDefault:     true,
		IsAvailable:   m.AvailableValue,
	}
}

func (m *MockProvider) Status(ctx context.Context) domain.ProviderStatus {
	return domain.ProviderStatus{
		Name:          m.NameValue,
		Available:     m.AvailableValue,
		ActiveJobs:    m.ActiveJobsVal,
		MaxConcurrent: m.MaxConcurrent(),
	}
}
