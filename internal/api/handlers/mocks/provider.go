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
	ListModelsFunc    func(ctx context.Context) ([]domain.Model, error)
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

func (m *MockProvider) ListModels(ctx context.Context) ([]domain.Model, error) {
	if m.ListModelsFunc != nil {
		return m.ListModelsFunc(ctx)
	}
	return []domain.Model{
		{
			ModelID:   "model1",
			Name:      "Test Model 1",
			Provider:  m.NameValue,
			Languages: []string{"en"},
		},
		{
			ModelID:   "model2",
			Name:      "Test Model 2",
			Provider:  m.NameValue,
			Languages: []string{"en", "es"},
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

// MockProviderRegistry is a mock implementation of domain.ProviderRegistry for testing.
type MockProviderRegistry struct {
	Providers       map[string]domain.TTSProvider
	DefaultProvider domain.TTSProvider
	DefaultNameVal  string
}

// NewMockProviderRegistry creates a new mock registry with a single default provider.
func NewMockProviderRegistry(provider domain.TTSProvider) *MockProviderRegistry {
	return &MockProviderRegistry{
		Providers: map[string]domain.TTSProvider{
			provider.Name(): provider,
		},
		DefaultProvider: provider,
		DefaultNameVal:  provider.Name(),
	}
}

func (r *MockProviderRegistry) Get(name string) (domain.TTSProvider, error) {
	if p, ok := r.Providers[name]; ok {
		return p, nil
	}
	return nil, domain.ErrProviderNotFound.WithMessage("Provider '" + name + "' not found")
}

func (r *MockProviderRegistry) Default() domain.TTSProvider {
	return r.DefaultProvider
}

func (r *MockProviderRegistry) List() []domain.TTSProvider {
	result := make([]domain.TTSProvider, 0, len(r.Providers))
	for _, p := range r.Providers {
		result = append(result, p)
	}
	return result
}

func (r *MockProviderRegistry) ListInfo(ctx context.Context) []domain.ProviderInfo {
	result := make([]domain.ProviderInfo, 0, len(r.Providers))
	for _, p := range r.Providers {
		result = append(result, domain.ProviderInfo{
			Name:          p.Name(),
			Type:          "MockProvider",
			MaxConcurrent: p.MaxConcurrent(),
			IsDefault:     p.Name() == r.DefaultNameVal,
			IsAvailable:   p.IsAvailable(ctx),
		})
	}
	return result
}

func (r *MockProviderRegistry) DefaultName() string {
	return r.DefaultNameVal
}
