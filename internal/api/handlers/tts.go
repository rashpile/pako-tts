package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/pako-tts/server/internal/api/middleware"
	"github.com/pako-tts/server/internal/domain"
)

// TTSHandler handles synchronous TTS requests.
type TTSHandler struct {
	provider       domain.TTSProvider
	logger         *zap.Logger
	syncTimeout    time.Duration
	maxTextLen     int
	defaultVoiceID string
}

// NewTTSHandler creates a new TTS handler.
func NewTTSHandler(
	provider domain.TTSProvider,
	logger *zap.Logger,
	syncTimeout time.Duration,
	maxTextLen int,
	defaultVoiceID string,
) *TTSHandler {
	return &TTSHandler{
		provider:       provider,
		logger:         logger,
		syncTimeout:    syncTimeout,
		maxTextLen:     maxTextLen,
		defaultVoiceID: defaultVoiceID,
	}
}

// TTSRequest represents a synchronous TTS request.
type TTSRequest struct {
	Text          string                `json:"text"`
	VoiceID       string                `json:"voice_id,omitempty"`
	OutputFormat  string                `json:"output_format,omitempty"`
	VoiceSettings *domain.VoiceSettings `json:"voice_settings,omitempty"`
}

// SynthesizeTTS handles POST /api/v1/tts.
func (h *TTSHandler) SynthesizeTTS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req TTSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, domain.ErrValidation.WithMessage("Invalid JSON body"))
		return
	}

	// Validate text
	if req.Text == "" {
		middleware.WriteError(w, domain.ErrValidation.WithDetails(map[string]any{
			"field":   "text",
			"message": "Text is required",
		}))
		return
	}

	if len(req.Text) > h.maxTextLen {
		middleware.WriteError(w, domain.ErrTextTooLong.WithDetails(map[string]any{
			"max_length":    h.maxTextLen,
			"actual_length": len(req.Text),
		}))
		return
	}

	// Set defaults
	voiceID := req.VoiceID
	if voiceID == "" {
		voiceID = h.defaultVoiceID
	}

	outputFormat := req.OutputFormat
	if outputFormat == "" {
		outputFormat = "mp3"
	}

	// Validate output format
	if outputFormat != "mp3" && outputFormat != "wav" {
		middleware.WriteError(w, domain.ErrInvalidFormat)
		return
	}

	// Check provider availability
	if !h.provider.IsAvailable(ctx) {
		middleware.WriteError(w, domain.ErrProviderUnavailable)
		return
	}

	// Build synthesis request
	synthReq := &domain.SynthesisRequest{
		Text:         req.Text,
		VoiceID:      voiceID,
		OutputFormat: outputFormat,
		Settings:     req.VoiceSettings,
	}

	// Synthesize
	result, err := h.provider.Synthesize(ctx, synthReq)
	if err != nil {
		h.logger.Error("Synthesis failed", zap.Error(err))
		middleware.WriteError(w, domain.ErrProviderUnavailable.WithMessage(err.Error()))
		return
	}

	// Stream audio response
	w.Header().Set("Content-Type", result.ContentType)
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, result.Audio); err != nil {
		h.logger.Error("Failed to write audio response", zap.Error(err))
	}
}
