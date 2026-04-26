package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
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

// VoicesListResponse represents the voices list response for a provider.
type VoicesListResponse struct {
	Provider string         `json:"provider"`
	Voices   []domain.Voice `json:"voices"`
}

// ListVoices handles GET /api/v1/providers/{name}/voices.
func (h *ProvidersHandler) ListVoices(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	provider, err := h.registry.Get(name)
	if err != nil {
		middleware.WriteError(w, domain.ErrProviderNotFound.WithMessage("Provider '"+name+"' not found"))
		return
	}

	voices, err := provider.ListVoices(r.Context())
	if err != nil {
		h.logger.Error("ListVoices failed", zap.String("provider", name), zap.Error(err))
		middleware.WriteError(w, domain.ErrProviderUnavailable.WithMessage(err.Error()))
		return
	}

	if voices == nil {
		voices = []domain.Voice{}
	}

	middleware.WriteJSON(w, http.StatusOK, VoicesListResponse{Provider: name, Voices: voices})
}
