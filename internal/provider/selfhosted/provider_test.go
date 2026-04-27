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
	// Selfhosted's models == voices upstream. ListModels deliberately returns
	// (nil, nil) so the UI's Model dropdown stays empty for selfhosted users
	// (the Voice dropdown already exposes the same entities). The contract is
	// that no upstream HTTP call is made — assert that by failing if the test
	// server is contacted.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("ListModels should not make an upstream HTTP call (got %s %s)", r.Method, r.URL.Path)
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
		t.Errorf("expected ListModels to return nil, got %+v", models)
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

