package domain

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

// DefaultVoiceSettings returns the default voice settings.
func DefaultVoiceSettings() *VoiceSettings {
	stability := 0.0
	similarityBoost := 1.0
	style := 0.0
	speed := 1.0
	useSpeakerBoost := true

	return &VoiceSettings{
		Stability:       &stability,
		SimilarityBoost: &similarityBoost,
		Style:           &style,
		Speed:           &speed,
		UseSpeakerBoost: &useSpeakerBoost,
	}
}

// Merge merges non-nil values from other settings into this settings.
func (v *VoiceSettings) Merge(other *VoiceSettings) *VoiceSettings {
	if other == nil {
		return v
	}

	result := &VoiceSettings{}

	// Use other's values if set, otherwise use v's values
	if other.Stability != nil {
		result.Stability = other.Stability
	} else if v != nil {
		result.Stability = v.Stability
	}

	if other.SimilarityBoost != nil {
		result.SimilarityBoost = other.SimilarityBoost
	} else if v != nil {
		result.SimilarityBoost = v.SimilarityBoost
	}

	if other.Style != nil {
		result.Style = other.Style
	} else if v != nil {
		result.Style = v.Style
	}

	if other.Speed != nil {
		result.Speed = other.Speed
	} else if v != nil {
		result.Speed = v.Speed
	}

	if other.UseSpeakerBoost != nil {
		result.UseSpeakerBoost = other.UseSpeakerBoost
	} else if v != nil {
		result.UseSpeakerBoost = v.UseSpeakerBoost
	}

	return result
}
