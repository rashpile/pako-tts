package handlers

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/pako-tts/server/internal/api/middleware"
	"github.com/pako-tts/server/internal/domain"
	"github.com/pako-tts/server/internal/provider/elevenlabs"
)

// ProvidersHandler handles provider-related requests.
type ProvidersHandler struct {
	provider domain.TTSProvider
	logger   *zap.Logger
}

// NewProvidersHandler creates a new providers handler.
func NewProvidersHandler(provider domain.TTSProvider, logger *zap.Logger) *ProvidersHandler {
	return &ProvidersHandler{
		provider: provider,
		logger:   logger,
	}
}

// ProvidersListResponse represents the providers list response.
type ProvidersListResponse struct {
	Providers       []domain.ProviderInfo `json:"providers"`
	DefaultProvider string                `json:"default_provider"`
}

// ListProviders handles GET /api/v1/providers.
func (h *ProvidersHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var providers []domain.ProviderInfo
	defaultProvider := ""

	if ep, ok := h.provider.(*elevenlabs.Provider); ok {
		info := ep.Info(ctx)
		providers = append(providers, info)
		if info.IsDefault {
			defaultProvider = info.Name
		}
	}

	response := ProvidersListResponse{
		Providers:       providers,
		DefaultProvider: defaultProvider,
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}
