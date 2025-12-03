// Package main is the entry point for the Pako TTS server.
package main

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/pako-tts/server/internal/api"
	"github.com/pako-tts/server/internal/provider/elevenlabs"
	"github.com/pako-tts/server/internal/queue/memory"
	"github.com/pako-tts/server/internal/storage/filesystem"
	"github.com/pako-tts/server/pkg/config"
)

//go:embed openapi.yaml
var openAPISpec []byte

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := config.NewLogger(&cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync() //nolint:errcheck

	logger.Info("Starting Pako TTS server",
		zap.Int("port", cfg.Server.Port),
		zap.String("log_level", cfg.Logging.Level),
	)

	// Validate API key
	if cfg.TTS.ElevenLabsAPIKey == "" {
		logger.Warn("ELEVENLABS_API_KEY not set - provider will be unavailable")
	}

	// Initialize provider
	provider := elevenlabs.NewProvider(cfg.TTS.ElevenLabsAPIKey, true)
	logger.Info("Provider initialized",
		zap.String("provider", provider.Name()),
		zap.Int("max_concurrent", provider.MaxConcurrent()),
	)

	// Initialize storage
	storage, err := filesystem.NewStorage(cfg.Storage.AudioStoragePath, logger)
	if err != nil {
		logger.Fatal("Failed to initialize storage", zap.Error(err))
	}
	logger.Info("Storage initialized",
		zap.String("path", cfg.Storage.AudioStoragePath),
	)

	// Initialize queue
	queue := memory.NewQueue(cfg.Queue.MaxConcurrentJobs)
	logger.Info("Queue initialized",
		zap.Int("max_concurrent", cfg.Queue.MaxConcurrentJobs),
	)

	// Start worker pool
	worker := memory.NewWorker(queue, provider, storage, logger, cfg.Storage.JobRetentionHours)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	worker.Start(ctx, cfg.Queue.WorkerCount)

	// Start cleanup scheduler (run every hour)
	storage.StartCleanupScheduler(ctx, cfg.Storage.JobRetentionHours, 1*time.Hour)

	// Setup router
	router := api.NewRouter(&api.RouterDeps{
		Logger:         logger,
		Provider:       provider,
		Queue:          queue,
		Storage:        storage,
		SyncTimeout:    cfg.TTS.SyncTimeout,
		MaxSyncTextLen: cfg.TTS.MaxSyncTextLength,
		DefaultVoiceID: cfg.TTS.DefaultVoiceID,
		RetentionHours: cfg.Storage.JobRetentionHours,
		OpenAPISpec:    openAPISpec,
	})

	// Setup HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("HTTP server starting",
			zap.String("addr", server.Addr),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop accepting new requests
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}

	// Stop workers
	cancel()
	worker.Stop()

	// Close queue
	queue.Close() //nolint:errcheck

	logger.Info("Server stopped")
}
