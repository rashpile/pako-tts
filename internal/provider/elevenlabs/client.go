// Package elevenlabs provides the ElevenLabs TTS provider implementation.
package elevenlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL      = "https://api.elevenlabs.io/v1"
	defaultModel = "eleven_multilingual_v2"
)

// Client is an HTTP client for the ElevenLabs API.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new ElevenLabs API client.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// TTSRequest represents a text-to-speech request to ElevenLabs.
type TTSRequest struct {
	Text          string            `json:"text"`
	ModelID       string            `json:"model_id"`
	OutputFormat  string            `json:"output_format,omitempty"`
	VoiceSettings *VoiceSettingsReq `json:"voice_settings,omitempty"`
}

// VoiceSettingsReq represents voice settings for ElevenLabs API.
type VoiceSettingsReq struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
	Style           float64 `json:"style,omitempty"`
	UseSpeakerBoost bool    `json:"use_speaker_boost,omitempty"`
}

// VoiceResponse represents a voice from the ElevenLabs API.
type VoiceResponse struct {
	VoiceID    string            `json:"voice_id"`
	Name       string            `json:"name"`
	Category   string            `json:"category"`
	Labels     map[string]string `json:"labels"`
	PreviewURL string            `json:"preview_url"`
}

// VoicesResponse represents the response from the voices endpoint.
type VoicesResponse struct {
	Voices []VoiceResponse `json:"voices"`
}

// TextToSpeech converts text to speech using ElevenLabs API.
func (c *Client) TextToSpeech(ctx context.Context, voiceID string, req *TTSRequest) (io.ReadCloser, string, error) {
	url := fmt.Sprintf("%s/text-to-speech/%s", baseURL, voiceID)

	if req.ModelID == "" {
		req.ModelID = defaultModel
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("xi-api-key", c.apiKey)
	httpReq.Header.Set("Accept", "audio/mpeg")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("ElevenLabs API error (status %d): %s", resp.StatusCode, string(errBody))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "audio/mpeg"
	}

	return resp.Body, contentType, nil
}

// GetVoices retrieves available voices from ElevenLabs API.
func (c *Client) GetVoices(ctx context.Context) (*VoicesResponse, error) {
	url := fmt.Sprintf("%s/voices", baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("xi-api-key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ElevenLabs API error (status %d): %s", resp.StatusCode, string(errBody))
	}

	var voices VoicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&voices); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &voices, nil
}

// CheckHealth checks if the ElevenLabs API is available.
func (c *Client) CheckHealth(ctx context.Context) bool {
	url := fmt.Sprintf("%s/user", baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	httpReq.Header.Set("xi-api-key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
