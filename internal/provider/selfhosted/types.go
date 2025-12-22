// Package selfhosted provides a TTS provider for self-hosted TTS services.
package selfhosted

// SynthesisRequest represents a TTS synthesis request for the local TTS API.
type SynthesisRequest struct {
	Text         string         `json:"text"`
	ModelID      string         `json:"model_id,omitempty"`
	Language     string         `json:"language,omitempty"`
	OutputFormat string         `json:"output_format,omitempty"`
	Parameters   map[string]any `json:"parameters,omitempty"`
}

// ModelSummary represents a model in the models list response.
type ModelSummary struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Engine      string   `json:"engine"`
	Languages   []string `json:"languages"`
	IsAvailable bool     `json:"is_available"`
	IsDefault   bool     `json:"is_default"`
}

// ModelsListResponse represents the response from /api/v1/models.
type ModelsListResponse struct {
	Models         []ModelSummary `json:"models"`
	DefaultModelID string         `json:"default_model_id"`
}

// EngineHealth represents the health status of a TTS engine.
type EngineHealth struct {
	Name        string  `json:"name"`
	Status      string  `json:"status"` // loading, available, unavailable, disabled
	ModelsCount int     `json:"models_count"`
	Error       *string `json:"error"`
}

// HealthResponse represents the response from /api/v1/health.
type HealthResponse struct {
	Status        string         `json:"status"`
	Engines       []EngineHealth `json:"engines"`
	Version       *string        `json:"version"`
	UptimeSeconds float64        `json:"uptime_seconds"`
}

// ErrorResponse represents an error response from the local TTS API.
type ErrorResponse struct {
	Detail string `json:"detail"`
}
