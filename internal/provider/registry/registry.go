// Package registry provides a provider registry for managing multiple TTS providers.
package registry

import (
	"context"
	"fmt"

	"github.com/pako-tts/server/internal/domain"
	"github.com/pako-tts/server/pkg/config"
)

// Registry implements domain.ProviderRegistry.
type Registry struct {
	providers   map[string]domain.TTSProvider
	defaultName string
	order       []string // Preserve insertion order for List()
}

// Ensure Registry implements ProviderRegistry.
var _ domain.ProviderRegistry = (*Registry)(nil)

// NewRegistry creates a new provider registry from configuration.
func NewRegistry(cfg *config.ProvidersConfig) (*Registry, error) {
	if cfg == nil {
		return nil, fmt.Errorf("providers config is nil")
	}

	r := &Registry{
		providers:   make(map[string]domain.TTSProvider),
		defaultName: cfg.Default,
		order:       make([]string, 0, len(cfg.List)),
	}

	// Create providers from config
	for _, providerCfg := range cfg.List {
		factory, ok := GetFactory(providerCfg.Type)
		if !ok {
			return nil, fmt.Errorf("unknown provider type: %q", providerCfg.Type)
		}

		isDefault := providerCfg.Name == cfg.Default
		provider, err := factory(providerCfg, isDefault)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %q: %w", providerCfg.Name, err)
		}

		r.providers[providerCfg.Name] = provider
		r.order = append(r.order, providerCfg.Name)
	}

	// Verify default provider exists
	if _, ok := r.providers[r.defaultName]; !ok {
		return nil, fmt.Errorf("default provider %q not found", r.defaultName)
	}

	return r, nil
}

// Get returns a provider by name.
func (r *Registry) Get(name string) (domain.TTSProvider, error) {
	provider, ok := r.providers[name]
	if !ok {
		return nil, domain.ErrProviderNotFound.WithMessage(fmt.Sprintf("Provider %q not found", name))
	}
	return provider, nil
}

// Default returns the default provider.
func (r *Registry) Default() domain.TTSProvider {
	return r.providers[r.defaultName]
}

// List returns all registered providers in registration order.
func (r *Registry) List() []domain.TTSProvider {
	result := make([]domain.TTSProvider, 0, len(r.order))
	for _, name := range r.order {
		result = append(result, r.providers[name])
	}
	return result
}

// ListInfo returns info for all providers.
func (r *Registry) ListInfo(ctx context.Context) []domain.ProviderInfo {
	result := make([]domain.ProviderInfo, 0, len(r.order))
	for _, name := range r.order {
		provider := r.providers[name]
		result = append(result, domain.ProviderInfo{
			Name:          provider.Name(),
			Type:          r.getProviderType(name),
			MaxConcurrent: provider.MaxConcurrent(),
			IsDefault:     name == r.defaultName,
			IsAvailable:   provider.IsAvailable(ctx),
		})
	}
	return result
}

// DefaultName returns the name of the default provider.
func (r *Registry) DefaultName() string {
	return r.defaultName
}

// getProviderType returns the type for a provider name.
// This is a simple implementation that relies on provider naming conventions.
func (r *Registry) getProviderType(name string) string {
	// For now, we'll use the provider's Name() method as a fallback
	// In a more complete implementation, we'd track the type separately
	if provider, ok := r.providers[name]; ok {
		return provider.Name()
	}
	return "unknown"
}
