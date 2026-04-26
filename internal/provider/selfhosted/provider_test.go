package selfhosted

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pako-tts/server/pkg/config"
)

func TestProvider_ListModels_ReturnsNil(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"id":"m1","name":"Model 1","is_available":true}]}`))
	}))
	defer srv.Close()

	p, err := NewProviderFromConfig(config.ProviderConfig{
		Name:    "local",
		BaseURL: srv.URL,
	}, true)
	if err != nil {
		t.Fatalf("unexpected error from NewProviderFromConfig: %v", err)
	}

	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if models != nil {
		t.Errorf("expected nil models slice, got %v", models)
	}
	if called {
		t.Errorf("ListModels must not call the upstream — selfhosted has no separate model concept")
	}
}

func TestProvider_ListVoices_StillWorksAfterListModelsAdded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models" {
			t.Errorf("expected /api/v1/models, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"id":"voice-a","name":"Voice A","is_available":true,"languages":["en"]}]}`))
	}))
	defer srv.Close()

	p, err := NewProviderFromConfig(config.ProviderConfig{
		Name:    "local",
		BaseURL: srv.URL,
	}, true)
	if err != nil {
		t.Fatalf("unexpected error from NewProviderFromConfig: %v", err)
	}

	voices, err := p.ListVoices(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(voices) != 1 {
		t.Fatalf("expected 1 voice, got %d", len(voices))
	}
	if voices[0].VoiceID != "voice-a" {
		t.Errorf("expected voice id 'voice-a', got %s", voices[0].VoiceID)
	}
}
