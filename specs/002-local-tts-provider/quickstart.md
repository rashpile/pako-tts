# Quickstart: Multi-Provider Architecture with Local TTS Support

This guide helps developers quickly understand and implement the multi-provider architecture.

## Overview

The feature adds support for multiple TTS providers configured via `config.yaml`. Each provider implements the `TTSProvider` interface and can be selected at request time.

## Key Files to Modify/Create

### New Files

| File | Purpose |
|------|---------|
| `internal/provider/registry/registry.go` | Provider registry implementation |
| `internal/provider/registry/config.go` | Provider config parsing |
| `internal/provider/selfhosted/provider.go` | Selfhosted provider implementation |
| `internal/provider/selfhosted/client.go` | HTTP client for local TTS API |
| `config.yaml` | Multi-provider configuration |

### Modified Files

| File | Changes |
|------|---------|
| `pkg/config/config.go` | Add ProvidersConfig struct |
| `internal/domain/provider.go` | Add ProviderRegistry interface |
| `cmd/server/main.go` | Use registry instead of single provider |
| `internal/api/handlers/tts.go` | Accept provider parameter |
| `internal/api/routes.go` | Pass registry to handlers |

## Quick Implementation Steps

### 1. Add ProviderRegistry Interface

```go
// internal/domain/provider.go

type ProviderRegistry interface {
    Get(name string) (TTSProvider, error)
    Default() TTSProvider
    List() []TTSProvider
    ListInfo(ctx context.Context) []ProviderInfo
    DefaultName() string
}
```

### 2. Create Provider Config Structure

```go
// pkg/config/config.go

type ProvidersConfig struct {
    Default string           `mapstructure:"default"`
    List    []ProviderConfig `mapstructure:"list"`
}

type ProviderConfig struct {
    Name           string        `mapstructure:"name"`
    Type           string        `mapstructure:"type"`
    MaxConcurrent  int           `mapstructure:"max_concurrent"`
    Timeout        time.Duration `mapstructure:"timeout"`
    // Type-specific fields loaded dynamically
}
```

### 3. Implement Registry

```go
// internal/provider/registry/registry.go

type Registry struct {
    providers   map[string]domain.TTSProvider
    defaultName string
}

func NewRegistry(cfg *config.ProvidersConfig) (*Registry, error) {
    // Create providers from config
    // Validate default exists
    // Return registry
}
```

### 4. Implement Selfhosted Provider

```go
// internal/provider/selfhosted/provider.go

type Provider struct {
    name       string
    client     *Client
    activeJobs int32
}

func NewProviderFromConfig(cfg ProviderConfig) (*Provider, error) {
    // Extract base_url, endpoints from config
    // Create HTTP client
    // Return provider
}
```

### 5. Update main.go

```go
// cmd/server/main.go

// Before:
provider := elevenlabs.NewProvider(cfg.TTS.ElevenLabsAPIKey, true)

// After:
registry, err := registry.NewRegistry(&cfg.Providers)
if err != nil {
    logger.Fatal("Failed to create provider registry", zap.Error(err))
}
```

### 6. Update TTS Handler

```go
// internal/api/handlers/tts.go

type TTSRequest struct {
    Text          string `json:"text"`
    VoiceID       string `json:"voice_id,omitempty"`
    Provider      string `json:"provider,omitempty"`  // NEW
    OutputFormat  string `json:"output_format,omitempty"`
    VoiceSettings *domain.VoiceSettings `json:"voice_settings,omitempty"`
}

func (h *TTSHandler) SynthesizeTTS(w http.ResponseWriter, r *http.Request) {
    // ...
    var provider domain.TTSProvider
    if req.Provider != "" {
        provider, err = h.registry.Get(req.Provider)
    } else {
        provider = h.registry.Default()
    }
    // ...
}
```

## Configuration Example

```yaml
# config.yaml
providers:
  default: "local-tts"
  list:
    - name: "elevenlabs"
      type: "elevenlabs"
      api_key: "${ELEVENLABS_API_KEY}"
      max_concurrent: 4

    - name: "local-tts"
      type: "selfhosted"
      base_url: "http://localhost:7021"
      tts_endpoint: "/api/v1/tts"
      voices_endpoint: "/api/v1/models"
      health_endpoint: "/api/v1/health"
      max_concurrent: 2
      timeout: "60s"
```

## Testing

### Unit Test: Provider Registry

```go
func TestRegistry_Get(t *testing.T) {
    cfg := &config.ProvidersConfig{
        Default: "test",
        List: []config.ProviderConfig{
            {Name: "test", Type: "mock"},
        },
    }
    reg, _ := registry.NewRegistry(cfg)

    provider, err := reg.Get("test")
    assert.NoError(t, err)
    assert.Equal(t, "test", provider.Name())
}
```

### Integration Test: Selfhosted Provider

```go
func TestSelfhostedProvider_Synthesize(t *testing.T) {
    // Requires local TTS service running at localhost:7021
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    cfg := selfhosted.Config{
        Name:           "test",
        BaseURL:        "http://localhost:7021",
        TTSEndpoint:    "/api/v1/tts",
        MaxConcurrent:  2,
    }
    provider, _ := selfhosted.NewProvider(cfg)

    req := &domain.SynthesisRequest{
        Text:         "Hello, world!",
        OutputFormat: "wav",
    }
    result, err := provider.Synthesize(context.Background(), req)

    assert.NoError(t, err)
    assert.NotNil(t, result.Audio)
}
```

## API Usage Examples

### Use Default Provider

```bash
curl -X POST http://localhost:8080/api/v1/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello, world!"}' \
  --output audio.mp3
```

### Specify Provider

```bash
curl -X POST http://localhost:8080/api/v1/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello, world!", "provider": "local-tts"}' \
  --output audio.wav
```

### List Providers

```bash
curl http://localhost:8080/api/v1/providers
```

Response:
```json
{
  "providers": [
    {
      "name": "elevenlabs",
      "type": "elevenlabs",
      "max_concurrent": 4,
      "is_default": false,
      "is_available": true
    },
    {
      "name": "local-tts",
      "type": "selfhosted",
      "max_concurrent": 2,
      "is_default": true,
      "is_available": true
    }
  ],
  "default_provider": "local-tts"
}
```

## Common Issues

### Provider Not Found

```json
{"error": {"code": "PROVIDER_NOT_FOUND", "message": "Provider 'unknown' not found"}}
```

**Fix**: Check provider name in config.yaml and request matches exactly.

### Provider Unavailable

```json
{"error": {"code": "PROVIDER_UNAVAILABLE", "message": "Provider 'local-tts' is not available"}}
```

**Fix**: Ensure local TTS service is running and accessible.

### Config Validation Error

```
Failed to create provider registry: default provider 'missing' not found in list
```

**Fix**: Ensure `providers.default` matches a provider name in `providers.list`.