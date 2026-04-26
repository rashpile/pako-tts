package selfhosted

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/pako-tts/server/internal/domain"
	"github.com/pako-tts/server/pkg/config"
)

const (
	providerType       = "SelfhostedProvider"
	defaultTTSEndpoint = "/api/v1/tts"
	defaultVoices      = "/api/v1/models"
	defaultHealth      = "/api/v1/health"
	defaultTimeout     = 30 * time.Second
	defaultConcurrent  = 2
)

// Provider implements the TTSProvider interface for self-hosted TTS services.
type Provider struct {
	name          string
	client        *Client
	maxConcurrent int
	activeJobs    int32
	isDefault     bool
}

// NewProviderFromConfig creates a new selfhosted provider from configuration.
func NewProviderFromConfig(cfg config.ProviderConfig, isDefault bool) (*Provider, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("selfhosted provider requires base_url")
	}

	// Set defaults for endpoints
	ttsEndpoint := cfg.TTSEndpoint
	if ttsEndpoint == "" {
		ttsEndpoint = defaultTTSEndpoint
	}

	voicesEndpoint := cfg.VoicesEndpoint
	if voicesEndpoint == "" {
		voicesEndpoint = defaultVoices
	}

	healthEndpoint := cfg.HealthEndpoint
	if healthEndpoint == "" {
		healthEndpoint = defaultHealth
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	maxConcurrent := cfg.MaxConcurrent
	if maxConcurrent == 0 {
		maxConcurrent = defaultConcurrent
	}

	client := NewClient(cfg.BaseURL, ttsEndpoint, voicesEndpoint, healthEndpoint, timeout)

	return &Provider{
		name:          cfg.Name,
		client:        client,
		maxConcurrent: maxConcurrent,
		isDefault:     isDefault,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return p.name
}

// Type returns the stable provider type identifier (independent of user-configured name).
func (p *Provider) Type() string {
	return providerType
}

// Synthesize converts text to speech.
func (p *Provider) Synthesize(ctx context.Context, req *domain.SynthesisRequest) (*domain.SynthesisResult, error) {
	atomic.AddInt32(&p.activeJobs, 1)
	defer atomic.AddInt32(&p.activeJobs, -1)

	// Build selfhosted TTS request
	// Map voice_id to model_id (only if explicitly provided, not a default)
	ttsReq := &SynthesisRequest{
		Text: req.Text,
	}

	// Only set model_id if voice_id looks like a local model (not an ElevenLabs ID)
	// ElevenLabs IDs are 20+ chars, local models are typically shorter descriptive names
	if req.VoiceID != "" && len(req.VoiceID) < 20 {
		ttsReq.ModelID = req.VoiceID
	}

	// Set output format
	switch req.OutputFormat {
	case "mp3":
		ttsReq.OutputFormat = "mp3"
	default:
		ttsReq.OutputFormat = "wav"
	}

	// Map voice settings to parameters if provided
	if req.Settings != nil {
		ttsReq.Parameters = mapVoiceSettingsToParams(req.Settings)
	}

	// Call local TTS API
	audioReader, contentType, err := p.client.TextToSpeech(ctx, ttsReq)
	if err != nil {
		return nil, err
	}

	// Read all audio data
	audioData, err := io.ReadAll(audioReader)
	audioReader.Close() //nolint:errcheck
	if err != nil {
		return nil, err
	}

	return &domain.SynthesisResult{
		Audio:       bytes.NewReader(audioData),
		ContentType: contentType,
		SizeBytes:   int64(len(audioData)),
	}, nil
}

// ListVoices returns available voices (mapped from models).
func (p *Provider) ListVoices(ctx context.Context) ([]domain.Voice, error) {
	resp, err := p.client.GetModels(ctx)
	if err != nil {
		return nil, err
	}

	voices := make([]domain.Voice, 0, len(resp.Models))
	for _, m := range resp.Models {
		if !m.IsAvailable {
			continue // Skip unavailable models
		}

		voice := domain.Voice{
			VoiceID:  m.ID,
			Name:     m.Name,
			Provider: p.name,
		}

		// Use first language if available
		if len(m.Languages) > 0 {
			voice.Language = m.Languages[0]
		}

		voices = append(voices, voice)
	}

	return voices, nil
}

// ListModels returns available models for selfhosted.
//
// Selfhosted's upstream `voices_endpoint` defaults to `/api/v1/models`, so
// `ListVoices` already returns the upstream model list. Returning the same
// items here would duplicate the dropdowns in the UI and produce confusing
// `voice_id == model_id` request bodies. Returning (nil, nil) is intentional —
// the handler normalizes nil to `[]`, and the UI's Model dropdown then shows
// only "Default model", which is the correct UX for selfhosted (no separate
// model concept distinct from voices).
func (p *Provider) ListModels(ctx context.Context) ([]domain.Model, error) {
	return nil, nil
}

// IsAvailable checks if the provider is available.
func (p *Provider) IsAvailable(ctx context.Context) bool {
	health, err := p.client.CheckHealth(ctx)
	if err != nil {
		return false
	}

	// Provider is available if at least one engine is available
	for _, engine := range health.Engines {
		if engine.Status == "available" {
			return true
		}
	}

	return false
}

// MaxConcurrent returns the maximum concurrent jobs.
func (p *Provider) MaxConcurrent() int {
	return p.maxConcurrent
}

// ActiveJobs returns the current number of active jobs.
func (p *Provider) ActiveJobs() int {
	return int(atomic.LoadInt32(&p.activeJobs))
}

// Status returns provider status for health checks.
func (p *Provider) Status(ctx context.Context) domain.ProviderStatus {
	return domain.ProviderStatus{
		Name:          p.name,
		Available:     p.IsAvailable(ctx),
		ActiveJobs:    p.ActiveJobs(),
		MaxConcurrent: p.maxConcurrent,
	}
}

// mapVoiceSettingsToParams converts domain.VoiceSettings to a parameters map.
func mapVoiceSettingsToParams(settings *domain.VoiceSettings) map[string]any {
	params := make(map[string]any)

	// Map stability to speed (if applicable)
	// Note: This mapping is model-dependent; adjust based on actual model parameters
	if settings.Stability != nil {
		// Some local TTS models use "speed" parameter
		params["speed"] = *settings.Stability
	}

	return params
}
