package gemini

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/pako-tts/server/internal/audio/transcode"
	"github.com/pako-tts/server/internal/domain"
	"github.com/pako-tts/server/pkg/config"
)

const (
	providerType  = "GeminiProvider"
	maxConcurrent = 4
)

// Provider implements domain.TTSProvider for Google Gemini TTS.
type Provider struct {
	client         *Client
	defaultModelID string
	defaultStyle   string
	isDefault      bool
	activeJobs     int32
}

// NewProvider creates a new Gemini provider with default model.
func NewProvider(apiKey string, isDefault bool) *Provider {
	return &Provider{
		client:         NewClient(apiKey),
		defaultModelID: defaultModelID,
		isDefault:      isDefault,
	}
}

// NewProviderFromConfig creates a Gemini provider from a ProviderConfig.
func NewProviderFromConfig(cfg config.ProviderConfig, isDefault bool) (*Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("gemini provider requires api_key")
	}

	modelID := cfg.ModelID
	if modelID == "" {
		modelID = defaultModelID
	}

	return &Provider{
		client:         NewClient(cfg.APIKey),
		defaultModelID: modelID,
		defaultStyle:   cfg.DefaultStyle,
		isDefault:      isDefault,
	}, nil
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return providerName
}

// Type returns the stable provider type identifier (not on the TTSProvider interface;
// available via type assertion, mirrors elevenlabs.Provider.Type()).
func (p *Provider) Type() string {
	return providerType
}

// Synthesize converts text to speech via the Gemini API, transcoding PCM to the
// requested output format server-side (wav = stdlib RIFF wrap, mp3 = ffmpeg).
func (p *Provider) Synthesize(ctx context.Context, req *domain.SynthesisRequest) (*domain.SynthesisResult, error) {
	atomic.AddInt32(&p.activeJobs, 1)
	defer atomic.AddInt32(&p.activeJobs, -1)

	model := req.ModelID
	if model == "" {
		model = p.defaultModelID
	}

	prompt := p.buildPrompt(req)

	pcm, err := p.client.GenerateAudio(ctx, model, prompt, req.VoiceID)
	if err != nil {
		return nil, err
	}

	var audio []byte
	var contentType string

	switch req.OutputFormat {
	case "wav":
		audio = transcode.PCMToWAV(pcm, 24000, 1, 16)
		contentType = "audio/wav"
	default:
		audio, err = transcode.PCMToMP3(ctx, pcm, 24000, 1)
		if err != nil {
			return nil, err
		}
		contentType = "audio/mpeg"
	}

	return &domain.SynthesisResult{
		Audio:       bytes.NewReader(audio),
		ContentType: contentType,
		SizeBytes:   int64(len(audio)),
	}, nil
}

// buildPrompt assembles the Gemini prompt: optional language directive, optional style
// line, then the user text — per the prompt composition spec in Solution Overview.
func (p *Provider) buildPrompt(req *domain.SynthesisRequest) string {
	var parts []string

	if req.LanguageCode != "" {
		if name, ok := isoToName[req.LanguageCode]; ok {
			parts = append(parts, "Speak in "+name+".")
		}
	}

	style := ""
	if req.Settings != nil {
		style = req.Settings.StyleInstructions
	}
	if style == "" {
		style = p.defaultStyle
	}
	if style != "" {
		parts = append(parts, "Style: "+style+".")
	}

	if len(parts) == 0 {
		return req.Text
	}
	return strings.Join(parts, "\n") + "\n\n" + req.Text
}

// ListVoices returns the static list of 30 prebuilt Gemini voices.
func (p *Provider) ListVoices(_ context.Context) ([]domain.Voice, error) {
	return prebuiltVoices, nil
}

// ListModels returns the single Gemini TTS model entry.
func (p *Provider) ListModels(_ context.Context) ([]domain.Model, error) {
	return []domain.Model{defaultModel}, nil
}

// IsAvailable checks connectivity to the Gemini API (connectivity-only; see CheckHealth caveat).
func (p *Provider) IsAvailable(ctx context.Context) bool {
	return p.client.CheckHealth(ctx)
}

// MaxConcurrent returns the maximum number of concurrent synthesis jobs.
func (p *Provider) MaxConcurrent() int {
	return maxConcurrent
}

// ActiveJobs returns the current number of active synthesis jobs.
func (p *Provider) ActiveJobs() int {
	return int(atomic.LoadInt32(&p.activeJobs))
}

// Info returns provider metadata for API responses.
func (p *Provider) Info(ctx context.Context) domain.ProviderInfo {
	return domain.ProviderInfo{
		Name:          providerName,
		Type:          providerType,
		MaxConcurrent: maxConcurrent,
		IsDefault:     p.isDefault,
		IsAvailable:   p.IsAvailable(ctx),
	}
}

// Status returns provider runtime status for health checks.
func (p *Provider) Status(ctx context.Context) domain.ProviderStatus {
	return domain.ProviderStatus{
		Name:          providerName,
		Available:     p.IsAvailable(ctx),
		ActiveJobs:    p.ActiveJobs(),
		MaxConcurrent: maxConcurrent,
	}
}
