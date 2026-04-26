package memory

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/pako-tts/server/internal/domain"
)

// fakeProvider is a minimal in-package stub of domain.TTSProvider for worker tests.
type fakeProvider struct {
	mu       sync.Mutex
	captured *domain.SynthesisRequest
	done     chan struct{}
}

func newFakeProvider() *fakeProvider {
	return &fakeProvider{done: make(chan struct{}, 1)}
}

func (p *fakeProvider) Name() string { return "fake-provider" }
func (p *fakeProvider) Synthesize(ctx context.Context, req *domain.SynthesisRequest) (*domain.SynthesisResult, error) {
	p.mu.Lock()
	captured := *req
	p.captured = &captured
	p.mu.Unlock()
	select {
	case p.done <- struct{}{}:
	default:
	}
	return &domain.SynthesisResult{
		Audio:       bytes.NewReader([]byte("audio")),
		ContentType: "audio/mpeg",
		SizeBytes:   5,
	}, nil
}
func (p *fakeProvider) ListVoices(ctx context.Context) ([]domain.Voice, error) { return nil, nil }
func (p *fakeProvider) ListModels(ctx context.Context) ([]domain.Model, error) { return nil, nil }
func (p *fakeProvider) IsAvailable(ctx context.Context) bool                   { return true }
func (p *fakeProvider) MaxConcurrent() int                                     { return 1 }
func (p *fakeProvider) ActiveJobs() int                                        { return 0 }
func (p *fakeProvider) Status(ctx context.Context) domain.ProviderStatus {
	return domain.ProviderStatus{Name: p.Name(), Available: true, MaxConcurrent: 1}
}

func (p *fakeProvider) capturedRequest() *domain.SynthesisRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.captured
}

// fakeRegistry is an in-package stub of domain.ProviderRegistry.
type fakeRegistry struct {
	provider domain.TTSProvider
}

func (r *fakeRegistry) Get(name string) (domain.TTSProvider, error) {
	if r.provider != nil && r.provider.Name() == name {
		return r.provider, nil
	}
	return nil, domain.ErrProviderNotFound
}
func (r *fakeRegistry) Default() domain.TTSProvider                       { return r.provider }
func (r *fakeRegistry) List() []domain.TTSProvider                        { return []domain.TTSProvider{r.provider} }
func (r *fakeRegistry) DefaultName() string                               { return r.provider.Name() }
func (r *fakeRegistry) ListInfo(ctx context.Context) []domain.ProviderInfo { return nil }

// fakeStorage is an in-package stub of domain.AudioStorage.
type fakeStorage struct{}

func (s *fakeStorage) Store(ctx context.Context, jobID string, audio []byte, format string) (string, error) {
	return "/tmp/" + jobID + "." + format, nil
}
func (s *fakeStorage) Retrieve(ctx context.Context, jobID string) (io.ReadCloser, string, error) {
	return io.NopCloser(bytes.NewReader(nil)), "audio/mpeg", nil
}
func (s *fakeStorage) Delete(ctx context.Context, jobID string) error { return nil }
func (s *fakeStorage) Exists(ctx context.Context, jobID string) bool  { return true }
func (s *fakeStorage) GetPath(ctx context.Context, jobID string) string {
	return "/tmp/" + jobID
}

func TestWorker_PropagatesJobModelIDToSynthesisRequest(t *testing.T) {
	logger := zap.NewNop()
	queue := NewQueue(10)
	provider := newFakeProvider()
	registry := &fakeRegistry{provider: provider}
	storage := &fakeStorage{}

	worker := NewWorker(queue, registry, storage, logger, 24)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	worker.Start(ctx, 1)
	defer worker.Stop()

	job := domain.NewJob("hello", "voice1", "eleven_v3", "", "fake-provider", "mp3", nil)
	if err := queue.Enqueue(ctx, job); err != nil {
		t.Fatalf("failed to enqueue job: %v", err)
	}

	// Wait for the worker to process the job (or timeout).
	select {
	case <-provider.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for worker to call Synthesize")
	}

	captured := provider.capturedRequest()
	if captured == nil {
		t.Fatal("expected provider.Synthesize to be called")
	}
	if captured.ModelID != "eleven_v3" {
		t.Errorf("expected SynthesisRequest.ModelID %q, got %q", "eleven_v3", captured.ModelID)
	}
	if captured.VoiceID != "voice1" {
		t.Errorf("expected SynthesisRequest.VoiceID %q, got %q", "voice1", captured.VoiceID)
	}
}

func TestWorker_PropagatesJobLanguageCodeToSynthesisRequest(t *testing.T) {
	logger := zap.NewNop()
	queue := NewQueue(10)
	provider := newFakeProvider()
	registry := &fakeRegistry{provider: provider}
	storage := &fakeStorage{}

	worker := NewWorker(queue, registry, storage, logger, 24)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	worker.Start(ctx, 1)
	defer worker.Stop()

	job := domain.NewJob("hola", "voice1", "eleven_v3", "es", "fake-provider", "mp3", nil)
	if err := queue.Enqueue(ctx, job); err != nil {
		t.Fatalf("failed to enqueue job: %v", err)
	}

	// Wait for the worker to process the job (or timeout).
	select {
	case <-provider.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for worker to call Synthesize")
	}

	captured := provider.capturedRequest()
	if captured == nil {
		t.Fatal("expected provider.Synthesize to be called")
	}
	if captured.LanguageCode != "es" {
		t.Errorf("expected SynthesisRequest.LanguageCode %q, got %q", "es", captured.LanguageCode)
	}
}
