package elevenlabs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/pako-tts/server/internal/domain"
	"github.com/pako-tts/server/pkg/config"
)

const (
	providerName     = "elevenlabs"
	providerType     = "ElevenLabsProvider"
	maxConcurrent    = 4
	fallbackModelID  = "eleven_multilingual_v2"
)

// Provider implements the TTSProvider interface for ElevenLabs.
type Provider struct {
	client         *Client
	activeJobs     int32
	isDefault      bool
	defaultModelID string
}

// NewProvider creates a new ElevenLabs provider.
func NewProvider(apiKey string, isDefault bool) *Provider {
	return &Provider{
		client:         NewClient(apiKey),
		isDefault:      isDefault,
		defaultModelID: fallbackModelID,
	}
}

// NewProviderFromConfig creates a new ElevenLabs provider from configuration.
func NewProviderFromConfig(cfg config.ProviderConfig, isDefault bool) (*Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("elevenlabs provider requires api_key")
	}

	modelID := cfg.ModelID
	if modelID == "" {
		modelID = fallbackModelID
	}

	return &Provider{
		client:         NewClient(cfg.APIKey),
		isDefault:      isDefault,
		defaultModelID: modelID,
	}, nil
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return providerName
}

// Type returns the stable provider type identifier (independent of user-configured name).
func (p *Provider) Type() string {
	return providerType
}

// Synthesize converts text to speech.
func (p *Provider) Synthesize(ctx context.Context, req *domain.SynthesisRequest) (*domain.SynthesisResult, error) {
	atomic.AddInt32(&p.activeJobs, 1)
	defer atomic.AddInt32(&p.activeJobs, -1)

	// Build ElevenLabs request
	ttsReq := &TTSRequest{
		Text: req.Text,
	}

	// Resolve model id: explicit request value wins; otherwise fall back to provider default.
	if req.ModelID != "" {
		ttsReq.ModelID = req.ModelID
	} else {
		ttsReq.ModelID = p.defaultModelID
	}

	// Forward optional ISO 639-1 language code; empty means "let model use its default".
	if req.LanguageCode != "" {
		ttsReq.LanguageCode = req.LanguageCode
	}

	// Set output format
	switch req.OutputFormat {
	case "wav":
		ttsReq.OutputFormat = "pcm_22050"
	default:
		ttsReq.OutputFormat = "mp3_22050_32"
	}

	// Apply voice settings if provided
	if req.Settings != nil {
		ttsReq.VoiceSettings = &VoiceSettingsReq{
			Stability:       getFloatValue(req.Settings.Stability, 0.5),
			SimilarityBoost: getFloatValue(req.Settings.SimilarityBoost, 0.75),
			Style:           getFloatValue(req.Settings.Style, 0.0),
			UseSpeakerBoost: getBoolValue(req.Settings.UseSpeakerBoost, true),
		}
	}

	// Call ElevenLabs API
	audioReader, contentType, err := p.client.TextToSpeech(ctx, req.VoiceID, ttsReq)
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

// ListVoices returns available voices.
func (p *Provider) ListVoices(ctx context.Context) ([]domain.Voice, error) {
	resp, err := p.client.GetVoices(ctx)
	if err != nil {
		return nil, err
	}

	voices := make([]domain.Voice, 0, len(resp.Voices))
	for _, v := range resp.Voices {
		voice := domain.Voice{
			VoiceID:    v.VoiceID,
			Name:       v.Name,
			Provider:   providerName,
			PreviewURL: v.PreviewURL,
		}

		// Extract labels
		if lang, ok := v.Labels["language"]; ok {
			voice.Language = lang
		}
		if gender, ok := v.Labels["gender"]; ok {
			voice.Gender = gender
		}

		voices = append(voices, voice)
	}

	return voices, nil
}

// ListModels returns available text-to-speech models for ElevenLabs.
func (p *Provider) ListModels(ctx context.Context) ([]domain.Model, error) {
	resp, err := p.client.GetModels(ctx)
	if err != nil {
		return nil, err
	}

	models := make([]domain.Model, 0, len(resp))
	for _, m := range resp {
		if !m.CanDoTextToSpeech {
			continue
		}
		langs := make([]string, 0, len(m.Languages))
		for _, l := range m.Languages {
			if l.LanguageID != "" {
				langs = append(langs, l.LanguageID)
			}
		}
		models = append(models, domain.Model{
			ModelID:     m.ModelID,
			Name:        m.Name,
			Provider:    providerName,
			Description: m.Description,
			Languages:   langs,
		})
	}
	return models, nil
}

// IsAvailable checks if the provider is available.
func (p *Provider) IsAvailable(ctx context.Context) bool {
	return p.client.CheckHealth(ctx)
}

// MaxConcurrent returns the maximum concurrent jobs.
func (p *Provider) MaxConcurrent() int {
	return maxConcurrent
}

// ActiveJobs returns the current number of active jobs.
func (p *Provider) ActiveJobs() int {
	return int(atomic.LoadInt32(&p.activeJobs))
}

// Info returns provider info for API responses.
func (p *Provider) Info(ctx context.Context) domain.ProviderInfo {
	return domain.ProviderInfo{
		Name:          providerName,
		Type:          providerType,
		MaxConcurrent: maxConcurrent,
		IsDefault:     p.isDefault,
		IsAvailable:   p.IsAvailable(ctx),
	}
}

// Status returns provider status for health checks.
func (p *Provider) Status(ctx context.Context) domain.ProviderStatus {
	return domain.ProviderStatus{
		Name:          providerName,
		Available:     p.IsAvailable(ctx),
		ActiveJobs:    p.ActiveJobs(),
		MaxConcurrent: maxConcurrent,
	}
}

func getFloatValue(ptr *float64, defaultVal float64) float64 {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

func getBoolValue(ptr *bool, defaultVal bool) bool {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}
