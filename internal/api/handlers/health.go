// Package handlers provides HTTP request handlers.
package handlers

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/pako-tts/server/internal/api/middleware"
	"github.com/pako-tts/server/internal/domain"
	"github.com/pako-tts/server/internal/provider/elevenlabs"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	provider domain.TTSProvider
	logger   *zap.Logger
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(provider domain.TTSProvider, logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		provider: provider,
		logger:   logger,
	}
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status    string                  `json:"status"`
	Version   string                  `json:"version"`
	Providers []domain.ProviderStatus `json:"providers"`
}

// HealthCheck handles GET /api/v1/health.
func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get provider status
	var providers []domain.ProviderStatus
	if ep, ok := h.provider.(*elevenlabs.Provider); ok {
		providers = append(providers, ep.Status(ctx))
	}

	// Determine overall status
	status := "healthy"
	for _, p := range providers {
		if !p.Available {
			status = "unhealthy"
			break
		}
	}

	response := HealthResponse{
		Status:    status,
		Version:   "0.0.1",
		Providers: providers,
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}
