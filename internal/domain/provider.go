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

	// IsAvailable checks if the provider is currently available.
	IsAvailable(ctx context.Context) bool

	// MaxConcurrent returns the maximum number of concurrent synthesis jobs.
	MaxConcurrent() int

	// ActiveJobs returns the current number of active jobs.
	ActiveJobs() int
}

// SynthesisRequest contains parameters for a TTS synthesis request.
type SynthesisRequest struct {
	Text         string
	VoiceID      string
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
