// Package registry provides a provider registry for managing multiple TTS providers.
package registry

import (
	"github.com/pako-tts/server/internal/domain"
	"github.com/pako-tts/server/internal/provider/elevenlabs"
	"github.com/pako-tts/server/internal/provider/selfhosted"
	"github.com/pako-tts/server/pkg/config"
)

// ProviderFactory creates a TTSProvider from configuration.
// The isDefault parameter indicates if this provider is the default.
type ProviderFactory func(cfg config.ProviderConfig, isDefault bool) (domain.TTSProvider, error)

// factories holds registered provider factories keyed by provider type.
var factories = make(map[string]ProviderFactory)

func init() {
	// Register built-in provider factories
	RegisterFactory("elevenlabs", elevenlabsFactory)
	RegisterFactory("selfhosted", selfhostedFactory)
}

// RegisterFactory registers a provider factory for a given type.
func RegisterFactory(providerType string, factory ProviderFactory) {
	factories[providerType] = factory
}

// GetFactory returns the factory for a provider type.
func GetFactory(providerType string) (ProviderFactory, bool) {
	factory, ok := factories[providerType]
	return factory, ok
}

// elevenlabsFactory creates an ElevenLabs provider from config.
func elevenlabsFactory(cfg config.ProviderConfig, isDefault bool) (domain.TTSProvider, error) {
	return elevenlabs.NewProviderFromConfig(cfg, isDefault)
}

// selfhostedFactory creates a selfhosted provider from config.
func selfhostedFactory(cfg config.ProviderConfig, isDefault bool) (domain.TTSProvider, error) {
	return selfhosted.NewProviderFromConfig(cfg, isDefault)
}
