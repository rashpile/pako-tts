package domain

import (
	"testing"
)

func TestDefaultVoiceSettings(t *testing.T) {
	settings := DefaultVoiceSettings()

	if settings == nil {
		t.Fatal("Expected non-nil settings")
	}

	if settings.Stability == nil || *settings.Stability != 0.0 {
		t.Errorf("Expected Stability to be 0.0, got %v", settings.Stability)
	}
	if settings.SimilarityBoost == nil || *settings.SimilarityBoost != 1.0 {
		t.Errorf("Expected SimilarityBoost to be 1.0, got %v", settings.SimilarityBoost)
	}
	if settings.Style == nil || *settings.Style != 0.0 {
		t.Errorf("Expected Style to be 0.0, got %v", settings.Style)
	}
	if settings.Speed == nil || *settings.Speed != 1.0 {
		t.Errorf("Expected Speed to be 1.0, got %v", settings.Speed)
	}
	if settings.UseSpeakerBoost == nil || *settings.UseSpeakerBoost != true {
		t.Errorf("Expected UseSpeakerBoost to be true, got %v", settings.UseSpeakerBoost)
	}
}

func TestVoiceSettings_Merge_NilOther(t *testing.T) {
	settings := DefaultVoiceSettings()

	result := settings.Merge(nil)

	if result != settings {
		t.Error("Expected Merge(nil) to return the original settings")
	}
}

func TestVoiceSettings_Merge_OverrideValues(t *testing.T) {
	base := DefaultVoiceSettings()

	stability := 0.8
	other := &VoiceSettings{
		Stability: &stability,
	}

	result := base.Merge(other)

	if result.Stability == nil || *result.Stability != 0.8 {
		t.Errorf("Expected Stability to be 0.8, got %v", result.Stability)
	}
	// Other values should come from base
	if result.SimilarityBoost == nil || *result.SimilarityBoost != 1.0 {
		t.Errorf("Expected SimilarityBoost to be 1.0, got %v", result.SimilarityBoost)
	}
}

func TestVoiceSettings_Merge_AllValues(t *testing.T) {
	base := DefaultVoiceSettings()

	stability := 0.5
	similarityBoost := 0.6
	style := 0.7
	speed := 1.5
	useSpeakerBoost := false

	other := &VoiceSettings{
		Stability:       &stability,
		SimilarityBoost: &similarityBoost,
		Style:           &style,
		Speed:           &speed,
		UseSpeakerBoost: &useSpeakerBoost,
	}

	result := base.Merge(other)

	if result.Stability == nil || *result.Stability != 0.5 {
		t.Errorf("Expected Stability to be 0.5, got %v", result.Stability)
	}
	if result.SimilarityBoost == nil || *result.SimilarityBoost != 0.6 {
		t.Errorf("Expected SimilarityBoost to be 0.6, got %v", result.SimilarityBoost)
	}
	if result.Style == nil || *result.Style != 0.7 {
		t.Errorf("Expected Style to be 0.7, got %v", result.Style)
	}
	if result.Speed == nil || *result.Speed != 1.5 {
		t.Errorf("Expected Speed to be 1.5, got %v", result.Speed)
	}
	if result.UseSpeakerBoost == nil || *result.UseSpeakerBoost != false {
		t.Errorf("Expected UseSpeakerBoost to be false, got %v", result.UseSpeakerBoost)
	}
}

func TestVoiceSettings_Merge_NilBase(t *testing.T) {
	var base *VoiceSettings = nil

	stability := 0.5
	other := &VoiceSettings{
		Stability: &stability,
	}

	result := base.Merge(other)

	if result.Stability == nil || *result.Stability != 0.5 {
		t.Errorf("Expected Stability to be 0.5, got %v", result.Stability)
	}
	// Values not in other should be nil
	if result.SimilarityBoost != nil {
		t.Errorf("Expected SimilarityBoost to be nil, got %v", result.SimilarityBoost)
	}
}
