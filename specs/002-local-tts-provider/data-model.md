# Data Model: Multi-Provider Architecture with Local TTS Support

**Date**: 2025-12-22
**Feature**: 002-local-tts-provider

## Entity Overview

```
┌─────────────────────┐     ┌──────────────────────┐
│   ProviderConfig    │────▶│   TTSProvider        │
│   (config.yaml)     │     │   (interface)        │
└─────────────────────┘     └──────────────────────┘
         │                           ▲
         │                           │
         ▼                  ┌────────┴────────┐
┌─────────────────────┐     │                 │
│  ProviderRegistry   │     │                 │
│  (runtime)          │     ▼                 ▼
└─────────────────────┘  ┌──────────┐  ┌──────────────┐
                         │ElevenLabs│  │ SelfHosted   │
                         │Provider  │  │ Provider     │
                         └──────────┘  └──────────────┘
```

## Domain Entities

### ProviderConfig (New)

Configuration for a single TTS provider loaded from config.yaml.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Unique provider identifier |
| type | string | Yes | Provider type: "elevenlabs", "selfhosted" |
| is_default | bool | No | Whether this is the default provider (only one) |
| max_concurrent | int | No | Max concurrent synthesis jobs (default: 4) |
| timeout | duration | No | Request timeout (default: 30s) |
| *type-specific* | varies | Depends | Provider-specific configuration fields |

**ElevenLabs-specific fields:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| api_key | string | Yes | ElevenLabs API key |

**SelfHosted-specific fields:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| base_url | string | Yes | Base URL of the TTS service |
| tts_endpoint | string | No | TTS synthesis endpoint (default: /api/v1/tts) |
| voices_endpoint | string | No | Voices/models list endpoint (default: /api/v1/models) |
| health_endpoint | string | No | Health check endpoint (default: /api/v1/health) |

**Validation Rules:**
- `name` must be non-empty and unique across all providers
- `type` must be a registered provider type
- Exactly one provider must have `is_default: true` OR a separate `default` field specifies the default provider name
- `max_concurrent` must be positive if specified
- `timeout` must be positive duration if specified
- For ElevenLabs: `api_key` must be non-empty
- For SelfHosted: `base_url` must be valid URL

### ProvidersConfig (New)

Top-level configuration section for all providers.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| default | string | Yes | Name of the default provider |
| list | []ProviderConfig | Yes | List of provider configurations |

**Validation Rules:**
- `list` must have at least one provider
- `default` must match a provider name in `list`
- No duplicate provider names in `list`

### ProviderRegistry (New Interface)

Runtime registry of initialized providers.

```go
type ProviderRegistry interface {
    // Get returns a provider by name, or error if not found
    Get(name string) (TTSProvider, error)

    // Default returns the default provider
    Default() TTSProvider

    // List returns all registered providers
    List() []TTSProvider

    // ListInfo returns info for all providers (for API response)
    ListInfo(ctx context.Context) []ProviderInfo
}
```

**State Transitions:** None (immutable after initialization)

### TTSProvider (Existing Interface - No Changes)

```go
type TTSProvider interface {
    Name() string
    Synthesize(ctx context.Context, req *SynthesisRequest) (*SynthesisResult, error)
    ListVoices(ctx context.Context) ([]Voice, error)
    IsAvailable(ctx context.Context) bool
    MaxConcurrent() int
    ActiveJobs() int
}
```

### ProviderInfo (Existing - No Changes)

```go
type ProviderInfo struct {
    Name          string `json:"name"`
    Type          string `json:"type"`
    MaxConcurrent int    `json:"max_concurrent"`
    IsDefault     bool   `json:"is_default"`
    IsAvailable   bool   `json:"is_available"`
}
```

### Voice (Existing - No Changes)

```go
type Voice struct {
    VoiceID    string `json:"voice_id"`
    Name       string `json:"name"`
    Provider   string `json:"provider"`
    Language   string `json:"language,omitempty"`
    Gender     string `json:"gender,omitempty"`
    PreviewURL string `json:"preview_url,omitempty"`
}
```

## SelfHosted Provider Internal Entities

### LocalTTSRequest (Internal)

Request body sent to local TTS service.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| text | string | Yes | Text to synthesize |
| model_id | string | No | Model/voice ID |
| language | string | No | Language code |
| output_format | string | No | "wav" or "mp3" |
| parameters | map[string]any | No | Model-specific parameters |

### LocalTTSModelSummary (Internal)

Model information from local TTS service.

| Field | Type | Description |
|-------|------|-------------|
| id | string | Model identifier |
| name | string | Human-readable name |
| engine | string | Engine type (coqui, silero) |
| languages | []string | Supported language codes |
| is_available | bool | Whether model is available |
| is_default | bool | Whether this is default model |

### LocalTTSHealthResponse (Internal)

Health response from local TTS service.

| Field | Type | Description |
|-------|------|-------------|
| status | string | Overall status |
| engines | []EngineHealth | Per-engine status |
| version | string | Service version |
| uptime_seconds | int | Uptime |

## Configuration File Structure

### config.yaml Example

```yaml
server:
  port: 8080
  read_timeout: 60s
  write_timeout: 60s

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
      timeout: 30s

tts:
  default_voice_id: "pNInz6obpgDQGcFmaJgB"
  max_sync_text_length: 5000
  sync_timeout: 30s

queue:
  worker_count: 4
  max_concurrent_jobs: 100

storage:
  audio_storage_path: "./audio_cache"
  job_retention_hours: 24

logging:
  level: info
  format: json
```

## Relationship Diagram

```
config.yaml
    │
    ▼
┌─────────────────────────────────────────────────────────┐
│                    ProvidersConfig                       │
│  ┌─────────────────┐  ┌─────────────────┐              │
│  │ ProviderConfig  │  │ ProviderConfig  │  ...         │
│  │ name: elevenlabs│  │ name: local-tts │              │
│  │ type: elevenlabs│  │ type: selfhosted│              │
│  └────────┬────────┘  └────────┬────────┘              │
└───────────┼────────────────────┼────────────────────────┘
            │                    │
            ▼                    ▼
┌───────────────────┐  ┌─────────────────────┐
│ ElevenLabsProvider│  │ SelfHostedProvider  │
│                   │  │                     │
│ implements        │  │ implements          │
│ TTSProvider       │  │ TTSProvider         │
└─────────┬─────────┘  └──────────┬──────────┘
          │                       │
          └───────────┬───────────┘
                      ▼
            ┌──────────────────┐
            │ ProviderRegistry │
            │                  │
            │ Get(name)        │
            │ Default()        │
            │ List()           │
            └──────────────────┘
```