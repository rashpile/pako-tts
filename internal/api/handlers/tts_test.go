package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pako-tts/server/internal/api/handlers/mocks"
	"github.com/pako-tts/server/internal/domain"
)

func TestSynthesizeTTS_PassesModelID(t *testing.T) {
	tests := []struct {
		name           string
		body           map[string]any
		wantModelID    string
		wantStatusCode int
	}{
		{
			name: "model_id is forwarded to SynthesisRequest when provided",
			body: map[string]any{
				"text":     "hello",
				"voice_id": "v1",
				"model_id": "eleven_v3",
			},
			wantModelID:    "eleven_v3",
			wantStatusCode: http.StatusOK,
		},
		{
			name: "model_id is empty when omitted from request body",
			body: map[string]any{
				"text":     "hello",
				"voice_id": "v1",
			},
			wantModelID:    "",
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testLogger()

			var captured *domain.SynthesisRequest
			mockProvider := &mocks.MockProvider{
				NameValue:      "test-provider",
				AvailableValue: true,
				SynthesizeFunc: func(ctx context.Context, req *domain.SynthesisRequest) (*domain.SynthesisResult, error) {
					captured = req
					return &domain.SynthesisResult{
						Audio:       bytes.NewReader([]byte("audio")),
						ContentType: "audio/mpeg",
						SizeBytes:   5,
					}, nil
				},
			}
			registry := mocks.NewMockProviderRegistry(mockProvider)

			handler := NewTTSHandler(registry, logger, 30*time.Second, 5000, "default-voice")

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.SynthesizeTTS(w, req)

			resp := w.Result()
			defer resp.Body.Close() //nolint:errcheck

			if resp.StatusCode != tt.wantStatusCode {
				t.Fatalf("expected status %d, got %d", tt.wantStatusCode, resp.StatusCode)
			}
			if captured == nil {
				t.Fatal("SynthesizeFunc was not called")
			}
			if captured.ModelID != tt.wantModelID {
				t.Errorf("expected SynthesisRequest.ModelID %q, got %q", tt.wantModelID, captured.ModelID)
			}
		})
	}
}

func TestSynthesizeTTS_PassesStyleInstructions(t *testing.T) {
	tests := []struct {
		name                    string
		body                    map[string]any
		wantStyleInstructions   string
		wantStatusCode          int
	}{
		{
			name: "style_instructions is forwarded to SynthesisRequest when provided",
			body: map[string]any{
				"text":     "hello",
				"voice_id": "v1",
				"voice_settings": map[string]any{
					"style_instructions": "warm and slow",
				},
			},
			wantStyleInstructions: "warm and slow",
			wantStatusCode:        http.StatusOK,
		},
		{
			name: "style_instructions is empty when voice_settings omitted",
			body: map[string]any{
				"text":     "hello",
				"voice_id": "v1",
			},
			wantStyleInstructions: "",
			wantStatusCode:        http.StatusOK,
		},
		{
			name: "style_instructions is empty when voice_settings present but field omitted",
			body: map[string]any{
				"text":     "hello",
				"voice_id": "v1",
				"voice_settings": map[string]any{},
			},
			wantStyleInstructions: "",
			wantStatusCode:        http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testLogger()

			var captured *domain.SynthesisRequest
			mockProvider := &mocks.MockProvider{
				NameValue:      "test-provider",
				AvailableValue: true,
				SynthesizeFunc: func(ctx context.Context, req *domain.SynthesisRequest) (*domain.SynthesisResult, error) {
					captured = req
					return &domain.SynthesisResult{
						Audio:       bytes.NewReader([]byte("audio")),
						ContentType: "audio/mpeg",
						SizeBytes:   5,
					}, nil
				},
			}
			registry := mocks.NewMockProviderRegistry(mockProvider)

			handler := NewTTSHandler(registry, logger, 30*time.Second, 5000, "default-voice")

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.SynthesizeTTS(w, req)

			resp := w.Result()
			defer resp.Body.Close() //nolint:errcheck

			if resp.StatusCode != tt.wantStatusCode {
				t.Fatalf("expected status %d, got %d", tt.wantStatusCode, resp.StatusCode)
			}
			if captured == nil {
				t.Fatal("SynthesizeFunc was not called")
			}
			gotStyleInstructions := ""
			if captured.Settings != nil {
				gotStyleInstructions = captured.Settings.StyleInstructions
			}
			if gotStyleInstructions != tt.wantStyleInstructions {
				t.Errorf("expected SynthesisRequest.Settings.StyleInstructions %q, got %q", tt.wantStyleInstructions, gotStyleInstructions)
			}
		})
	}
}

func TestSynthesizeTTS_PassesLanguageCode(t *testing.T) {
	tests := []struct {
		name             string
		body             map[string]any
		wantLanguageCode string
		wantStatusCode   int
	}{
		{
			name: "language_code is forwarded to SynthesisRequest when provided",
			body: map[string]any{
				"text":          "hello",
				"voice_id":      "v1",
				"language_code": "en",
			},
			wantLanguageCode: "en",
			wantStatusCode:   http.StatusOK,
		},
		{
			name: "language_code is empty when omitted from request body",
			body: map[string]any{
				"text":     "hello",
				"voice_id": "v1",
			},
			wantLanguageCode: "",
			wantStatusCode:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testLogger()

			var captured *domain.SynthesisRequest
			mockProvider := &mocks.MockProvider{
				NameValue:      "test-provider",
				AvailableValue: true,
				SynthesizeFunc: func(ctx context.Context, req *domain.SynthesisRequest) (*domain.SynthesisResult, error) {
					captured = req
					return &domain.SynthesisResult{
						Audio:       bytes.NewReader([]byte("audio")),
						ContentType: "audio/mpeg",
						SizeBytes:   5,
					}, nil
				},
			}
			registry := mocks.NewMockProviderRegistry(mockProvider)

			handler := NewTTSHandler(registry, logger, 30*time.Second, 5000, "default-voice")

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.SynthesizeTTS(w, req)

			resp := w.Result()
			defer resp.Body.Close() //nolint:errcheck

			if resp.StatusCode != tt.wantStatusCode {
				t.Fatalf("expected status %d, got %d", tt.wantStatusCode, resp.StatusCode)
			}
			if captured == nil {
				t.Fatal("SynthesizeFunc was not called")
			}
			if captured.LanguageCode != tt.wantLanguageCode {
				t.Errorf("expected SynthesisRequest.LanguageCode %q, got %q", tt.wantLanguageCode, captured.LanguageCode)
			}
		})
	}
}
