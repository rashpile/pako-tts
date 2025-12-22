// Package handlers provides HTTP request handlers.
package handlers

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/pako-tts/server/internal/api/middleware"
	"github.com/pako-tts/server/internal/domain"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	registry domain.ProviderRegistry
	logger   *zap.Logger
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(registry domain.ProviderRegistry, logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		registry: registry,
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

	// Get status for all providers
	var providers []domain.ProviderStatus
	for _, p := range h.registry.List() {
		providers = append(providers, p.Status(ctx))
	}

	// Determine overall status - healthy if at least one provider is available
	status := "unhealthy"
	for _, p := range providers {
		if p.Available {
			status = "healthy"
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
