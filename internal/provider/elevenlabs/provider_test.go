package elevenlabs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewProvider(t *testing.T) {
	provider := NewProvider("test-api-key", true)

	if provider == nil {
		t.Fatal("Expected non-nil provider")
	}
	if provider.client == nil {
		t.Error("Expected client to be initialized")
	}
	if !provider.isDefault {
		t.Error("Expected isDefault to be true")
	}
}

func TestProvider_Name(t *testing.T) {
	provider := NewProvider("test-api-key", true)

	name := provider.Name()

	if name != "elevenlabs" {
		t.Errorf("Expected name 'elevenlabs', got %s", name)
	}
}

func TestProvider_MaxConcurrent(t *testing.T) {
	provider := NewProvider("test-api-key", true)

	maxConcurrent := provider.MaxConcurrent()

	if maxConcurrent != 4 {
		t.Errorf("Expected maxConcurrent 4, got %d", maxConcurrent)
	}
}

func TestProvider_ActiveJobs(t *testing.T) {
	provider := NewProvider("test-api-key", true)

	activeJobs := provider.ActiveJobs()

	if activeJobs != 0 {
		t.Errorf("Expected activeJobs 0, got %d", activeJobs)
	}
}

func TestGetFloatValue(t *testing.T) {
	tests := []struct {
		name        string
		ptr         *float64
		defaultVal  float64
		expected    float64
	}{
		{"nil pointer", nil, 0.5, 0.5},
		{"non-nil pointer", ptrFloat(0.8), 0.5, 0.8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFloatValue(tt.ptr, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestGetBoolValue(t *testing.T) {
	tests := []struct {
		name       string
		ptr        *bool
		defaultVal bool
		expected   bool
	}{
		{"nil pointer true default", nil, true, true},
		{"nil pointer false default", nil, false, false},
		{"non-nil pointer true", ptrBool(true), false, true},
		{"non-nil pointer false", ptrBool(false), true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBoolValue(tt.ptr, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func ptrFloat(f float64) *float64 {
	return &f
}

func ptrBool(b bool) *bool {
	return &b
}

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := &Client{
		apiKey:  "test-key",
		baseURL: srv.URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
	return c, srv
}

func TestProvider_ListModels_Success(t *testing.T) {
	body := `[
		{"model_id":"eleven_multilingual_v2","name":"Multilingual v2","description":"desc","can_do_text_to_speech":true,"languages":[{"language_id":"en","name":"English"},{"language_id":"es","name":"Spanish"}]},
		{"model_id":"eleven_flash_v2_5","name":"Flash v2.5","description":"fast","can_do_text_to_speech":true,"languages":[{"language_id":"en","name":"English"}]}
	]`
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("expected /models, got %s", r.URL.Path)
		}
		if r.Header.Get("xi-api-key") != "test-key" {
			t.Errorf("missing/invalid xi-api-key header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
	defer srv.Close()

	p := &Provider{client: client}
	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].ModelID != "eleven_multilingual_v2" {
		t.Errorf("expected first model_id eleven_multilingual_v2, got %s", models[0].ModelID)
	}
	if models[0].Provider != "elevenlabs" {
		t.Errorf("expected provider 'elevenlabs', got %s", models[0].Provider)
	}
	if models[0].Description != "desc" {
		t.Errorf("expected description 'desc', got %s", models[0].Description)
	}
	if len(models[0].Languages) != 2 || models[0].Languages[0] != "en" || models[0].Languages[1] != "es" {
		t.Errorf("expected languages [en es], got %v", models[0].Languages)
	}
}

func TestProvider_ListModels_FiltersNonTTS(t *testing.T) {
	body := `[
		{"model_id":"eleven_multilingual_v2","name":"Multilingual","can_do_text_to_speech":true,"languages":[{"language_id":"en","name":"English"}]},
		{"model_id":"eleven_english_sts_v2","name":"STS","can_do_text_to_speech":false,"languages":[{"language_id":"en","name":"English"}]}
	]`
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
	defer srv.Close()

	p := &Provider{client: client}
	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model after filtering, got %d", len(models))
	}
	if models[0].ModelID != "eleven_multilingual_v2" {
		t.Errorf("expected eleven_multilingual_v2, got %s", models[0].ModelID)
	}
}

func TestProvider_ListModels_UpstreamError(t *testing.T) {
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"unauthorized"}`, http.StatusUnauthorized)
	})
	defer srv.Close()

	p := &Provider{client: client}
	models, err := p.ListModels(context.Background())
	if err == nil {
		t.Fatalf("expected error, got models=%v", models)
	}
	if !strings.Contains(err.Error(), "ElevenLabs API error") {
		t.Errorf("expected wrapped error, got %v", err)
	}
}
