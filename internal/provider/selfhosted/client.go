package selfhosted

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the self-hosted TTS API.
type Client struct {
	baseURL        string
	ttsEndpoint    string
	voicesEndpoint string
	healthEndpoint string
	httpClient     *http.Client
}

// NewClient creates a new selfhosted TTS API client.
func NewClient(baseURL string, ttsEndpoint, voicesEndpoint, healthEndpoint string, timeout time.Duration) *Client {
	return &Client{
		baseURL:        baseURL,
		ttsEndpoint:    ttsEndpoint,
		voicesEndpoint: voicesEndpoint,
		healthEndpoint: healthEndpoint,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// TextToSpeech sends a TTS request and returns the audio stream.
func (c *Client) TextToSpeech(ctx context.Context, req *SynthesisRequest) (io.ReadCloser, string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+c.ttsEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close() //nolint:errcheck
		var errResp ErrorResponse
		if decodeErr := json.NewDecoder(resp.Body).Decode(&errResp); decodeErr == nil && errResp.Detail != "" {
			return nil, "", fmt.Errorf("TTS failed: %s", errResp.Detail)
		}
		return nil, "", fmt.Errorf("TTS failed with status %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "audio/wav"
	}

	return resp.Body, contentType, nil
}

// GetModels retrieves the list of available models (voices).
func (c *Client) GetModels(ctx context.Context) (*ModelsListResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+c.voicesEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get models failed with status %d", resp.StatusCode)
	}

	var models ModelsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &models, nil
}

// CheckHealth checks if the TTS service is healthy.
func (c *Client) CheckHealth(ctx context.Context) (*HealthResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+c.healthEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err // Return raw error for availability check
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &health, nil
}
