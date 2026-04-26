package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/pako-tts/server/internal/api/handlers/mocks"
	"github.com/pako-tts/server/internal/domain"
)

func TestProvidersHandler_ListVoices(t *testing.T) {
	knownVoices := []domain.Voice{
		{VoiceID: "v1", Name: "Voice One", Provider: "test-provider", Language: "en", Gender: "female"},
		{VoiceID: "v2", Name: "Voice Two", Provider: "test-provider", Language: "en", Gender: "male"},
	}

	tests := []struct {
		name           string
		providerName   string
		listVoicesFunc func(ctx context.Context) ([]domain.Voice, error)
		wantStatus     int
		wantErrorCode  string
		wantVoices     []domain.Voice
	}{
		{
			name:         "success returns voices for known provider",
			providerName: "test-provider",
			listVoicesFunc: func(ctx context.Context) ([]domain.Voice, error) {
				return knownVoices, nil
			},
			wantStatus: http.StatusOK,
			wantVoices: knownVoices,
		},
		{
			name:          "unknown provider returns 404",
			providerName:  "does-not-exist",
			wantStatus:    http.StatusNotFound,
			wantErrorCode: "PROVIDER_NOT_FOUND",
		},
		{
			name:         "provider error returns 503",
			providerName: "test-provider",
			listVoicesFunc: func(ctx context.Context) ([]domain.Voice, error) {
				return nil, errors.New("upstream failure")
			},
			wantStatus:    http.StatusServiceUnavailable,
			wantErrorCode: "PROVIDER_UNAVAILABLE",
		},
		{
			name:         "provider returns nil voices serializes as empty array",
			providerName: "test-provider",
			listVoicesFunc: func(ctx context.Context) ([]domain.Voice, error) {
				return nil, nil
			},
			wantStatus: http.StatusOK,
			wantVoices: []domain.Voice{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testLogger()
			mockProvider := &mocks.MockProvider{
				NameValue:      "test-provider",
				ListVoicesFunc: tt.listVoicesFunc,
			}
			registry := mocks.NewMockProviderRegistry(mockProvider)

			handler := NewProvidersHandler(registry, logger)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+tt.providerName+"/voices", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("name", tt.providerName)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.ListVoices(w, req)

			resp := w.Result()
			defer resp.Body.Close() //nolint:errcheck

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
			}

			if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Errorf("expected Content-Type application/json, got %q", ct)
			}

			if tt.wantErrorCode != "" {
				var errResp domain.ErrorResponse
				if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if errResp.Error == nil || errResp.Error.Code != tt.wantErrorCode {
					t.Fatalf("expected error code %q, got %+v", tt.wantErrorCode, errResp.Error)
				}
				if tt.wantErrorCode == "PROVIDER_NOT_FOUND" && !strings.Contains(errResp.Error.Message, tt.providerName) {
					t.Errorf("expected error message to contain provider name %q, got %q", tt.providerName, errResp.Error.Message)
				}
				return
			}

			rawBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}

			if len(tt.wantVoices) == 0 {
				if !strings.Contains(string(rawBody), `"voices":[]`) {
					t.Errorf("expected voices field to serialize as [] (not null), got %s", rawBody)
				}
			}

			var body VoicesListResponse
			if err := json.Unmarshal(rawBody, &body); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if body.Provider != tt.providerName {
				t.Errorf("expected provider %q, got %q", tt.providerName, body.Provider)
			}
			if len(body.Voices) != len(tt.wantVoices) {
				t.Fatalf("expected %d voices, got %d", len(tt.wantVoices), len(body.Voices))
			}
			for i, v := range tt.wantVoices {
				if body.Voices[i].VoiceID != v.VoiceID {
					t.Errorf("voice[%d]: expected ID %q, got %q", i, v.VoiceID, body.Voices[i].VoiceID)
				}
				if body.Voices[i].Name != v.Name {
					t.Errorf("voice[%d]: expected name %q, got %q", i, v.Name, body.Voices[i].Name)
				}
			}
		})
	}
}

func TestProvidersHandler_ListModels(t *testing.T) {
	knownModels := []domain.Model{
		{ModelID: "eleven_multilingual_v2", Name: "Multilingual v2", Provider: "test-provider", Languages: []string{"en", "es"}},
		{ModelID: "eleven_flash_v2_5", Name: "Flash v2.5", Provider: "test-provider", Languages: []string{"en"}},
	}

	tests := []struct {
		name           string
		providerName   string
		listModelsFunc func(ctx context.Context) ([]domain.Model, error)
		wantStatus     int
		wantErrorCode  string
		wantModels     []domain.Model
	}{
		{
			name:         "success returns models for known provider",
			providerName: "test-provider",
			listModelsFunc: func(ctx context.Context) ([]domain.Model, error) {
				return knownModels, nil
			},
			wantStatus: http.StatusOK,
			wantModels: knownModels,
		},
		{
			name:          "unknown provider returns 404",
			providerName:  "does-not-exist",
			wantStatus:    http.StatusNotFound,
			wantErrorCode: "PROVIDER_NOT_FOUND",
		},
		{
			name:         "provider error returns 503",
			providerName: "test-provider",
			listModelsFunc: func(ctx context.Context) ([]domain.Model, error) {
				return nil, errors.New("upstream failure")
			},
			wantStatus:    http.StatusServiceUnavailable,
			wantErrorCode: "PROVIDER_UNAVAILABLE",
		},
		{
			name:         "provider returns nil models serializes as empty array",
			providerName: "test-provider",
			listModelsFunc: func(ctx context.Context) ([]domain.Model, error) {
				return nil, nil
			},
			wantStatus: http.StatusOK,
			wantModels: []domain.Model{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testLogger()
			mockProvider := &mocks.MockProvider{
				NameValue:      "test-provider",
				ListModelsFunc: tt.listModelsFunc,
			}
			registry := mocks.NewMockProviderRegistry(mockProvider)

			handler := NewProvidersHandler(registry, logger)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+tt.providerName+"/models", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("name", tt.providerName)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.ListModels(w, req)

			resp := w.Result()
			defer resp.Body.Close() //nolint:errcheck

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
			}

			if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Errorf("expected Content-Type application/json, got %q", ct)
			}

			if tt.wantErrorCode != "" {
				var errResp domain.ErrorResponse
				if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if errResp.Error == nil || errResp.Error.Code != tt.wantErrorCode {
					t.Fatalf("expected error code %q, got %+v", tt.wantErrorCode, errResp.Error)
				}
				if tt.wantErrorCode == "PROVIDER_NOT_FOUND" && !strings.Contains(errResp.Error.Message, tt.providerName) {
					t.Errorf("expected error message to contain provider name %q, got %q", tt.providerName, errResp.Error.Message)
				}
				return
			}

			rawBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}

			if len(tt.wantModels) == 0 {
				if !strings.Contains(string(rawBody), `"models":[]`) {
					t.Errorf("expected models field to serialize as [] (not null), got %s", rawBody)
				}
			}

			var body ModelsListResponse
			if err := json.Unmarshal(rawBody, &body); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if body.Provider != tt.providerName {
				t.Errorf("expected provider %q, got %q", tt.providerName, body.Provider)
			}
			if len(body.Models) != len(tt.wantModels) {
				t.Fatalf("expected %d models, got %d", len(tt.wantModels), len(body.Models))
			}
			for i, m := range tt.wantModels {
				if body.Models[i].ModelID != m.ModelID {
					t.Errorf("model[%d]: expected ID %q, got %q", i, m.ModelID, body.Models[i].ModelID)
				}
				if body.Models[i].Name != m.Name {
					t.Errorf("model[%d]: expected name %q, got %q", i, m.Name, body.Models[i].Name)
				}
			}
		})
	}
}
