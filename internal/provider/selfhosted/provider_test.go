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

func TestProvider_ListModels_PreservesAllLanguages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models" {
			t.Errorf("expected /api/v1/models, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[` +
			`{"id":"m1","name":"Model 1","is_available":true,"languages":["en","es","fr"]},` +
			`{"id":"m2","name":"Model 2","is_available":true,"languages":["de","en"]},` +
			`{"id":"m3","name":"Hidden","is_available":false,"languages":["pt"]}` +
			`]}`))
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
	if len(models) != 2 {
		t.Fatalf("expected 2 available models (unavailable filtered), got %d", len(models))
	}

	// Verify model fields and that ALL languages are preserved (not flattened
	// to the primary language as ListVoices does).
	if models[0].ModelID != "m1" || models[0].Name != "Model 1" || models[0].Provider != "local" {
		t.Errorf("model[0] mismatch: %+v", models[0])
	}
	if got, want := models[0].Languages, []string{"en", "es", "fr"}; !equalStringSlices(got, want) {
		t.Errorf("model[0].Languages = %v, want %v (all languages, not just primary)", got, want)
	}
	if models[1].ModelID != "m2" || models[1].Name != "Model 2" {
		t.Errorf("model[1] mismatch: %+v", models[1])
	}
	if got, want := models[1].Languages, []string{"de", "en"}; !equalStringSlices(got, want) {
		t.Errorf("model[1].Languages = %v, want %v", got, want)
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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

func TestProvider_Synthesize_ForwardsLanguageCode(t *testing.T) {
	var rawBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/tts":
			b, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read tts body: %v", err)
			}
			rawBody = b
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

	req := &domain.SynthesisRequest{
		Text:         "hi",
		VoiceID:      "short-voice",
		LanguageCode: "en",
	}
	if _, err := p.Synthesize(context.Background(), req); err != nil {
		t.Fatalf("Synthesize: %v", err)
	}

	// Capture the raw outbound bytes and assert the language key is present
	// with the expected value. We decode into map[string]any (rather than
	// selfhosted.SynthesisRequest) so the assertion checks the wire format
	// the upstream local TTS API actually sees.
	var asMap map[string]any
	if err := json.Unmarshal(rawBody, &asMap); err != nil {
		t.Fatalf("decode raw body: %v", err)
	}
	got, ok := asMap["language"]
	if !ok {
		t.Fatalf("selfhosted upstream body missing language key: %s", string(rawBody))
	}
	if got != "en" {
		t.Errorf("selfhosted upstream body language = %v, want %q", got, "en")
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
