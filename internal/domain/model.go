package domain

// Model represents a TTS model (e.g., "eleven_multilingual_v2").
type Model struct {
	ModelID     string   `json:"model_id"`
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Description string   `json:"description,omitempty"`
	Languages   []string `json:"languages,omitempty"`
}
