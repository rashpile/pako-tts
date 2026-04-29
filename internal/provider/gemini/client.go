// Package gemini provides the Google Gemini TTS provider implementation.
package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"
)

const (
	baseURL = "https://generativelanguage.googleapis.com/v1beta"
)

// Client is an HTTP client for the Gemini API.
type Client struct {
	apiKey       string
	baseURL      string
	httpClient   *http.Client
	healthClient *http.Client
}

// NewClient creates a new Gemini API client.
func NewClient(apiKey string) *Client {
	return NewClientWithBaseURL(apiKey, baseURL)
}

// NewClientWithBaseURL creates a new Gemini API client with a custom base URL (used in tests).
func NewClientWithBaseURL(apiKey, base string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: base,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		healthClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// TTSRequest is the JSON body sent to the Gemini generateContent endpoint.
type TTSRequest struct {
	Contents         []Content        `json:"contents"`
	GenerationConfig GenerationConfig `json:"generationConfig"`
}

// Content holds a list of parts for the Gemini API.
type Content struct {
	Role  string `json:"role,omitempty"`
	Parts []Part `json:"parts"`
}

// Part holds a text prompt (requests) or inline audio data (responses).
type Part struct {
	Text       string      `json:"text,omitempty"`
	InlineData *InlineData `json:"inlineData,omitempty"`
}

// InlineData carries base64-encoded audio returned by Gemini.
type InlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// GenerationConfig configures the response modality and voice.
type GenerationConfig struct {
	ResponseModalities []string     `json:"responseModalities"`
	SpeechConfig       SpeechConfig `json:"speechConfig"`
}

// SpeechConfig selects the voice configuration.
type SpeechConfig struct {
	VoiceConfig VoiceConfig `json:"voiceConfig"`
}

// VoiceConfig wraps the prebuilt voice selection.
type VoiceConfig struct {
	PrebuiltVoiceConfig PrebuiltVoiceConfig `json:"prebuiltVoiceConfig"`
}

// PrebuiltVoiceConfig specifies a prebuilt Gemini voice by name.
type PrebuiltVoiceConfig struct {
	VoiceName string `json:"voiceName"`
}

// TTSResponse is the JSON response from a Gemini generateContent call.
type TTSResponse struct {
	Candidates     []Candidate     `json:"candidates"`
	PromptFeedback *PromptFeedback `json:"promptFeedback,omitempty"`
}

// Candidate is a single response candidate.
type Candidate struct {
	Content       Content `json:"content"`
	FinishReason  string  `json:"finishReason,omitempty"`
	FinishMessage string  `json:"finishMessage,omitempty"`
}

// PromptFeedback describes why Gemini returned no candidates for a prompt.
type PromptFeedback struct {
	BlockReason string `json:"blockReason,omitempty"`
}

// GenerateAudio calls the Gemini generateContent endpoint and returns raw PCM bytes.
// The prompt should already include any language directive and style prefix.
func (c *Client) GenerateAudio(ctx context.Context, model, prompt, voiceName string) ([]byte, error) {
	url := fmt.Sprintf("%s/models/%s:generateContent", c.baseURL, neturl.PathEscape(model))

	reqBody := TTSRequest{
		Contents: []Content{
			{Parts: []Part{{Text: prompt}}},
		},
		GenerationConfig: GenerationConfig{
			ResponseModalities: []string{"AUDIO"},
			SpeechConfig: SpeechConfig{
				VoiceConfig: VoiceConfig{
					PrebuiltVoiceConfig: PrebuiltVoiceConfig{VoiceName: voiceName},
				},
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	const maxBody = 256 * 1024 * 1024 // 256 MiB — large enough for ~60 min of audio
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if int64(len(respBody)) > maxBody {
		return nil, fmt.Errorf("gemini response body exceeded %d bytes", maxBody)
	}

	if resp.StatusCode != http.StatusOK {
		snippet := respBody
		if len(snippet) > 512 {
			snippet = snippet[:512]
		}
		return nil, fmt.Errorf("gemini API error (status %d): %s", resp.StatusCode, snippet)
	}

	var ttsResp TTSResponse
	if err := json.Unmarshal(respBody, &ttsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	audioData, diag := findInlineAudio(ttsResp)
	if audioData == "" {
		return nil, fmt.Errorf("gemini API returned no audio data (%s)", diag)
	}

	pcm, err := base64.StdEncoding.DecodeString(audioData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio data: %w", err)
	}

	return pcm, nil
}

// CheckHealth checks if the Gemini API is reachable by probing the given model endpoint.
// This is connectivity-only: an invalid API key may still return true because the
// models endpoint is publicly readable.
func (c *Client) CheckHealth(ctx context.Context, model string) bool {
	url := fmt.Sprintf("%s/models/%s", c.baseURL, neturl.PathEscape(model))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	httpReq.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.healthClient.Do(httpReq)
	if err != nil {
		return false
	}
	defer resp.Body.Close() //nolint:errcheck
	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode == http.StatusOK
}

func findInlineAudio(resp TTSResponse) (string, string) {
	emptyAudioParts := 0
	finishReasons := make([]string, 0, len(resp.Candidates))
	firstTextSnippet := ""
	firstFinishMessage := ""

	for _, candidate := range resp.Candidates {
		if candidate.FinishReason != "" {
			finishReasons = append(finishReasons, candidate.FinishReason)
		}
		if firstFinishMessage == "" && candidate.FinishMessage != "" {
			firstFinishMessage = candidate.FinishMessage
		}

		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil {
				if strings.TrimSpace(part.InlineData.Data) != "" {
					return part.InlineData.Data, ""
				}
				emptyAudioParts++
			}
			if firstTextSnippet == "" && strings.TrimSpace(part.Text) != "" {
				firstTextSnippet = compactWhitespace(part.Text)
			}
		}
	}

	diag := []string{fmt.Sprintf("candidates=%d", len(resp.Candidates))}
	if len(finishReasons) > 0 {
		diag = append(diag, "finish_reasons="+strings.Join(finishReasons, ","))
	}
	if resp.PromptFeedback != nil && resp.PromptFeedback.BlockReason != "" {
		diag = append(diag, "prompt_block_reason="+resp.PromptFeedback.BlockReason)
	}
	if emptyAudioParts > 0 {
		diag = append(diag, fmt.Sprintf("empty_audio_parts=%d", emptyAudioParts))
	}
	if firstFinishMessage != "" {
		diag = append(diag, fmt.Sprintf("finish_message=%q", truncateForError(compactWhitespace(firstFinishMessage), 120)))
	}
	if firstTextSnippet != "" {
		diag = append(diag, fmt.Sprintf("text_snippet=%q", truncateForError(firstTextSnippet, 120)))
	}

	return "", strings.Join(diag, ", ")
}

func compactWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func truncateForError(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
