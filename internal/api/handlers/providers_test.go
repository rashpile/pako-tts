package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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

			if tt.wantErrorCode != "" {
				var errResp domain.ErrorResponse
				if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if errResp.Error == nil || errResp.Error.Code != tt.wantErrorCode {
					t.Fatalf("expected error code %q, got %+v", tt.wantErrorCode, errResp.Error)
				}
				return
			}

			var body VoicesListResponse
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
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
