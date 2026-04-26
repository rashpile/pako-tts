// Package domain defines the core business entities and interfaces (ports).
package domain

import (
	"context"
	"io"
	"time"
)

// TTSProvider defines the interface for text-to-speech providers.
// This is the primary port for TTS functionality.
type TTSProvider interface {
	// Name returns the provider identifier (e.g., "elevenlabs").
	Name() string

	// Synthesize converts text to speech and returns audio data.
	Synthesize(ctx context.Context, req *SynthesisRequest) (*SynthesisResult, error)

	// ListVoices returns available voices for this provider.
	ListVoices(ctx context.Context) ([]Voice, error)

	// ListModels returns available models for this provider.
	// Providers that have no concept of a model distinct from a voice may return (nil, nil).
	ListModels(ctx context.Context) ([]Model, error)

	// IsAvailable checks if the provider is currently available.
	IsAvailable(ctx context.Context) bool

	// MaxConcurrent returns the maximum number of concurrent synthesis jobs.
	MaxConcurrent() int

	// ActiveJobs returns the current number of active jobs.
	ActiveJobs() int

	// Status returns the provider's runtime status for health checks.
	Status(ctx context.Context) ProviderStatus
}

// SynthesisRequest contains parameters for a TTS synthesis request.
type SynthesisRequest struct {
	Text         string
	VoiceID      string
	ModelID      string // optional; provider falls back to its configured default when empty
	OutputFormat string // "mp3" or "wav"
	Settings     *VoiceSettings
}

// SynthesisResult contains the result of a TTS synthesis operation.
type SynthesisResult struct {
	Audio       io.Reader
	ContentType string
	Duration    time.Duration
	SizeBytes   int64
}

// ProviderInfo contains metadata about a TTS provider for API responses.
type ProviderInfo struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	MaxConcurrent int    `json:"max_concurrent"`
	IsDefault     bool   `json:"is_default"`
	IsAvailable   bool   `json:"is_available"`
}

// ProviderStatus contains runtime status of a provider for health checks.
type ProviderStatus struct {
	Name          string `json:"name"`
	Available     bool   `json:"available"`
	ActiveJobs    int    `json:"active_jobs"`
	MaxConcurrent int    `json:"max_concurrent"`
}

// ProviderRegistry manages multiple TTS providers.
// It handles provider lookup, default provider selection, and provider listing.
type ProviderRegistry interface {
	// Get returns a provider by name.
	// Returns ErrProviderNotFound if the provider doesn't exist.
	Get(name string) (TTSProvider, error)

	// Default returns the default provider.
	// The default provider is used when no provider is specified in requests.
	Default() TTSProvider

	// List returns all registered providers.
	List() []TTSProvider

	// ListInfo returns info for all providers (for API response).
	// Includes availability status for each provider.
	ListInfo(ctx context.Context) []ProviderInfo

	// DefaultName returns the name of the default provider.
	DefaultName() string
}
