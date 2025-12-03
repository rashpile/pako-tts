// Package api provides HTTP API routing and handlers.
package api

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"

	"github.com/pako-tts/server/internal/api/handlers"
	apimiddleware "github.com/pako-tts/server/internal/api/middleware"
	"github.com/pako-tts/server/internal/domain"
)

// RouterDeps contains dependencies for the router.
type RouterDeps struct {
	Logger         *zap.Logger
	Provider       domain.TTSProvider
	Queue          domain.JobQueue
	Storage        domain.AudioStorage
	SyncTimeout    time.Duration
	MaxSyncTextLen int
	DefaultVoiceID string
	RetentionHours int
	OpenAPISpec    []byte
}

// NewRouter creates a new Chi router with all routes and middleware.
func NewRouter(deps *RouterDeps) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(apimiddleware.NewLogging(deps.Logger))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Create handlers
	healthHandler := handlers.NewHealthHandler(deps.Provider, deps.Logger)
	providersHandler := handlers.NewProvidersHandler(deps.Provider, deps.Logger)

	// OpenAPI handler (if spec provided)
	var openAPIHandler *handlers.OpenAPIHandler
	if len(deps.OpenAPISpec) > 0 {
		var err error
		openAPIHandler, err = handlers.NewOpenAPIHandler(deps.OpenAPISpec)
		if err != nil {
			deps.Logger.Warn("Failed to parse OpenAPI spec", zap.Error(err))
		}
	}
	ttsHandler := handlers.NewTTSHandler(
		deps.Provider,
		deps.Logger,
		deps.SyncTimeout,
		deps.MaxSyncTextLen,
		deps.DefaultVoiceID,
	)
	jobsHandler := handlers.NewJobsHandler(
		deps.Provider,
		deps.Queue,
		deps.Storage,
		deps.Logger,
		deps.DefaultVoiceID,
		deps.RetentionHours,
	)

	// OpenAPI spec at root
	if openAPIHandler != nil {
		r.Get("/openapi.json", openAPIHandler.ServeSpecJSON)
		r.Get("/openapi.yaml", openAPIHandler.ServeSpecYAML)
	}

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// OpenAPI spec
		if openAPIHandler != nil {
			r.Get("/openapi.json", openAPIHandler.ServeSpecJSON)
			r.Get("/openapi.yaml", openAPIHandler.ServeSpecYAML)
		}

		// Health check
		r.Get("/health", healthHandler.HealthCheck)

		// Providers
		r.Get("/providers", providersHandler.ListProviders)

		// Synchronous TTS
		r.With(middleware.Timeout(deps.SyncTimeout)).Post("/tts", ttsHandler.SynthesizeTTS)

		// Async Jobs
		r.Post("/jobs", jobsHandler.SubmitJob)
		r.Get("/jobs/{jobID}", jobsHandler.GetJobStatus)
		r.Get("/jobs/{jobID}/result", jobsHandler.GetJobResult)
	})

	return r
}
