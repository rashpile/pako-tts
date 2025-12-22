package handlers

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/pako-tts/server/internal/api/middleware"
	"github.com/pako-tts/server/internal/domain"
)

// ProvidersHandler handles provider-related requests.
type ProvidersHandler struct {
	registry domain.ProviderRegistry
	logger   *zap.Logger
}

// NewProvidersHandler creates a new providers handler.
func NewProvidersHandler(registry domain.ProviderRegistry, logger *zap.Logger) *ProvidersHandler {
	return &ProvidersHandler{
		registry: registry,
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

	response := ProvidersListResponse{
		Providers:       h.registry.ListInfo(ctx),
		DefaultProvider: h.registry.DefaultName(),
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}
