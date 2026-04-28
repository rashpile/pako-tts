package registry

import (
	"testing"

	"github.com/pako-tts/server/pkg/config"
)

func TestGetFactory_KnownProviders(t *testing.T) {
	for _, name := range []string{"elevenlabs", "selfhosted", "gemini"} {
		f, ok := GetFactory(name)
		if !ok {
			t.Errorf("GetFactory(%q) returned false", name)
		}
		if f == nil {
			t.Errorf("GetFactory(%q) returned nil factory", name)
		}
	}
}

func TestGetFactory_Unknown(t *testing.T) {
	_, ok := GetFactory("bogus")
	if ok {
		t.Error("GetFactory(\"bogus\") should return false")
	}
}

func TestGeminiFactory_MissingAPIKey(t *testing.T) {
	f, ok := GetFactory("gemini")
	if !ok {
		t.Fatal("gemini factory not registered")
	}
	_, err := f(config.ProviderConfig{Name: "g", Type: "gemini"}, false)
	if err == nil {
		t.Error("expected error when api_key is missing")
	}
}

func TestGeminiFactory_ValidConfig(t *testing.T) {
	f, ok := GetFactory("gemini")
	if !ok {
		t.Fatal("gemini factory not registered")
	}
	p, err := f(config.ProviderConfig{
		Name:   "g",
		Type:   "gemini",
		APIKey: "test-key",
	}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Name() != "gemini" {
		t.Errorf("expected provider name 'gemini', got %q", p.Name())
	}
}

func TestGeminiFactory_DefaultModelID(t *testing.T) {
	f, _ := GetFactory("gemini")
	p, err := f(config.ProviderConfig{
		Name:   "g",
		Type:   "gemini",
		APIKey: "test-key",
	}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	models, err := p.ListModels(t.Context())
	if err != nil {
		t.Fatalf("ListModels error: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("expected at least one model")
	}
	if models[0].ModelID != "gemini-3.1-flash-tts-preview" {
		t.Errorf("unexpected default model: %q", models[0].ModelID)
	}
}
