package selfhosted

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pako-tts/server/internal/domain"
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

func TestProvider_Synthesize_HonorsExplicitModelID(t *testing.T) {
	var captured SynthesisRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/tts":
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &captured); err != nil {
				t.Fatalf("decode tts body: %v", err)
			}
			w.Header().Set("Content-Type", "audio/wav")
			_, _ = w.Write([]byte("audio"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	p, err := NewProviderFromConfig(config.ProviderConfig{
		Name:    "local",
		BaseURL: srv.URL,
	}, true)
	if err != nil {
		t.Fatalf("NewProviderFromConfig: %v", err)
	}

	cases := []struct {
		name      string
		req       *domain.SynthesisRequest
		wantModel string
	}{
		{
			name:      "explicit model_id wins",
			req:       &domain.SynthesisRequest{Text: "hi", VoiceID: "21m00Tcm4TlvDq8ikWAM_long_id", ModelID: "my-model"},
			wantModel: "my-model",
		},
		{
			name:      "no model_id falls back to short voice_id heuristic",
			req:       &domain.SynthesisRequest{Text: "hi", VoiceID: "short-voice"},
			wantModel: "short-voice",
		},
		{
			name:      "no model_id and long voice_id leaves model empty",
			req:       &domain.SynthesisRequest{Text: "hi", VoiceID: "21m00Tcm4TlvDq8ikWAM"},
			wantModel: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			captured = SynthesisRequest{}
			if _, err := p.Synthesize(context.Background(), tc.req); err != nil {
				t.Fatalf("Synthesize: %v", err)
			}
			if captured.ModelID != tc.wantModel {
				t.Errorf("captured ModelID = %q, want %q", captured.ModelID, tc.wantModel)
			}
		})
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
