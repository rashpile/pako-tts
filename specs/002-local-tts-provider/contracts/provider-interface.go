// Package contracts defines the interface contracts for the multi-provider architecture.
// This file documents the Go interfaces that providers must implement.
//
// NOTE: This is a contract specification file, not production code.
// The actual implementation will be in internal/domain/provider.go

package contracts

import (
	"context"
	"io"
	"time"
)

// =============================================================================
// EXISTING INTERFACES (no changes)
// =============================================================================

// TTSProvider defines the interface for text-to-speech providers.
// All provider implementations must satisfy this interface.
type TTSProvider interface {
	// Name returns the provider identifier (e.g., "elevenlabs", "local-tts").
	Name() string

	// Synthesize converts text to speech and returns audio data.
	// The request contains text, voice ID, output format, and optional settings.
	// Returns audio as an io.Reader with content type and size information.
	Synthesize(ctx context.Context, req *SynthesisRequest) (*SynthesisResult, error)

	// ListVoices returns available voices for this provider.
	// Each voice includes ID, name, provider, and optional metadata.
	ListVoices(ctx context.Context) ([]Voice, error)

	// IsAvailable checks if the provider is currently available.
	// Should perform a lightweight health check.
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

// VoiceSettings contains voice customization parameters.
type VoiceSettings struct {
	Stability       *float64 `json:"stability,omitempty"`
	SimilarityBoost *float64 `json:"similarity_boost,omitempty"`
	Style           *float64 `json:"style,omitempty"`
	Speed           *float64 `json:"speed,omitempty"`
	UseSpeakerBoost *bool    `json:"use_speaker_boost,omitempty"`
}

// Voice represents an available voice option.
type Voice struct {
	VoiceID    string `json:"voice_id"`
	Name       string `json:"name"`
	Provider   string `json:"provider"`
	Language   string `json:"language,omitempty"`
	Gender     string `json:"gender,omitempty"`
	PreviewURL string `json:"preview_url,omitempty"`
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

// =============================================================================
// NEW INTERFACES
// =============================================================================

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

// ProviderFactory creates a TTSProvider from configuration.
// Each provider type has its own factory function.
type ProviderFactory func(cfg ProviderConfig) (TTSProvider, error)

// ProviderConfig contains the configuration for a single provider.
// Type-specific fields are accessed through the Get methods.
type ProviderConfig interface {
	// Name returns the unique provider identifier.
	Name() string

	// Type returns the provider type (e.g., "elevenlabs", "selfhosted").
	Type() string

	// MaxConcurrent returns the maximum concurrent jobs.
	MaxConcurrent() int

	// Timeout returns the request timeout.
	Timeout() time.Duration

	// GetString returns a string configuration value.
	GetString(key string) string

	// GetInt returns an integer configuration value.
	GetInt(key string) int

	// GetBool returns a boolean configuration value.
	GetBool(key string) bool
}

// =============================================================================
// ERROR DEFINITIONS
// =============================================================================

// ErrProviderNotFound is returned when a requested provider doesn't exist.
// var ErrProviderNotFound = errors.New("provider not found")

// ErrProviderUnavailable is returned when a provider exists but is not available.
// var ErrProviderUnavailable = errors.New("provider unavailable")

// =============================================================================
// FACTORY REGISTRATION
// =============================================================================

// Expected factory registration pattern (in registry package):
//
// var factories = map[string]ProviderFactory{
//     "elevenlabs": elevenlabs.NewProviderFromConfig,
//     "selfhosted": selfhosted.NewProviderFromConfig,
// }
//
// func NewRegistry(cfg ProvidersConfig) (ProviderRegistry, error) {
//     providers := make(map[string]TTSProvider)
//     for _, providerCfg := range cfg.List {
//         factory, ok := factories[providerCfg.Type()]
//         if !ok {
//             return nil, fmt.Errorf("unknown provider type: %s", providerCfg.Type())
//         }
//         provider, err := factory(providerCfg)
//         if err != nil {
//             return nil, fmt.Errorf("failed to create provider %s: %w", providerCfg.Name(), err)
//         }
//         providers[providerCfg.Name()] = provider
//     }
//     return &registry{providers: providers, defaultName: cfg.Default}, nil
// }