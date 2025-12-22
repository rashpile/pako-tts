# Research: Multi-Provider Architecture with Local TTS Support

**Date**: 2025-12-22
**Feature**: 002-local-tts-provider

## Local TTS Service API Analysis

### Source

OpenAPI specification from `localhost:7021/openapi.json`

### Service Overview

- **Title**: TTS API Service
- **Description**: Local TTS synthesis with multiple engine backends
- **Version**: 1.0.0
- **Engines**: Coqui, Silero (enum: `coqui`, `silero`)

### Endpoints

#### 1. Health Check

```
GET /api/v1/health
```

**Response** (`HealthResponse`):
```json
{
  "status": "string",           // Overall service health
  "engines": [                  // Array of EngineHealth
    {
      "name": "string",
      "status": "loading|available|unavailable|disabled",
      "models_count": 0,
      "error": "string|null"
    }
  ],
  "version": "string|null",
  "uptime_seconds": 0
}
```

#### 2. Text-to-Speech Synthesis

```
POST /api/v1/tts
```

**Request** (`SynthesisRequest`):
```json
{
  "text": "string",              // Required: 1-5000 characters
  "model_id": "string|null",     // Optional: uses default if not specified
  "language": "string|null",     // Optional: uses model default
  "output_format": "wav|mp3",    // Default: wav
  "parameters": {}               // Optional: model-specific parameters
}
```

**Response**: Audio binary (WAV or MP3)

#### 3. List Models (equivalent to voices)

```
GET /api/v1/models
```

**Response** (`ModelsListResponse`):
```json
{
  "models": [
    {
      "id": "string",
      "name": "string",
      "engine": "coqui|silero",
      "languages": ["en", "es"],
      "is_available": true,
      "is_default": false
    }
  ],
  "default_model_id": "string|null"
}
```

#### 4. Get Model Details

```
GET /api/v1/models/{model_id}
```

**Response** (`ModelDetailResponse`):
```json
{
  "id": "string",
  "name": "string",
  "engine": "coqui|silero",
  "languages": ["en"],
  "default_language": "string|null",
  "sample_rate": 22050,
  "parameters": [
    {
      "name": "speed",
      "type": "float|int|string|bool",
      "description": "Speaking speed",
      "default": 1.0,
      "min_value": 0.5,
      "max_value": 2.0,
      "allowed_values": null
    }
  ],
  "is_available": true,
  "is_default": false
}
```

### API Mapping Decisions

| pako-tts Concept | Local TTS Concept | Mapping Strategy |
|------------------|-------------------|------------------|
| Voice | Model | Direct mapping: `model_id` → `voice_id` |
| VoiceID | Model ID | Use model `id` as voice identifier |
| VoiceSettings | Parameters | Map to `parameters` object |
| OutputFormat | output_format | Direct: "mp3" or "wav" |
| ListVoices | /api/v1/models | Transform ModelSummary → Voice |
| Synthesize | /api/v1/tts | Transform request/response |
| IsAvailable | /api/v1/health | Check engine status |

### Decision: Voice/Model Terminology

**Decision**: Map "model" to "voice" in pako-tts domain
**Rationale**: pako-tts uses voice terminology consistently; local TTS uses model terminology. The mapping is 1:1 since each model represents a distinct voice.
**Alternatives Considered**:
- Expose both concepts separately - rejected: adds complexity without benefit
- Rename domain to use "model" - rejected: breaking change to existing API

### Decision: Health Check Implementation

**Decision**: Use `/api/v1/health` endpoint to determine provider availability
**Rationale**: The health endpoint provides engine-level status which maps to provider availability
**Implementation**:
- Provider is available if at least one engine has status `available`
- Store engine status for detailed health reporting

### Decision: Voice Settings Mapping

**Decision**: Map VoiceSettings to local TTS `parameters` object dynamically
**Rationale**: Local TTS models have different parameter schemas; need flexible mapping
**Implementation**:
- Convert VoiceSettings fields to parameters map
- Speed → `speed` parameter (if supported by model)
- Ignore unsupported settings gracefully

## Provider Registry Architecture

### Decision: Factory Pattern for Provider Instantiation

**Decision**: Use a map of factory functions keyed by provider type
**Rationale**: Simple, explicit, type-safe approach without reflection
**Implementation**:
```go
type ProviderFactory func(cfg ProviderConfig) (domain.TTSProvider, error)

var factories = map[string]ProviderFactory{
    "elevenlabs": elevenlabs.NewProviderFromConfig,
    "selfhosted": selfhosted.NewProviderFromConfig,
}
```
**Alternatives Considered**:
- Reflection-based registration - rejected: adds complexity, reduces type safety
- Interface-based self-registration - rejected: requires init() side effects

### Decision: Configuration Structure

**Decision**: Use YAML array of provider configs with `type` discriminator
**Rationale**: Supports multiple providers of same type with different names
**Example**:
```yaml
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
      timeout: "30s"
```

### Decision: Provider Selection in API

**Decision**: Add optional `provider` query/body parameter to TTS endpoints
**Rationale**: Maintains backwards compatibility (uses default when omitted)
**Implementation**:
- `POST /api/v1/tts` accepts `provider` in request body
- If omitted, uses configured default provider
- Returns error if specified provider not found

## Selfhosted Provider Implementation

### Decision: HTTP Client Configuration

**Decision**: Create dedicated HTTP client per provider instance with configurable timeout
**Rationale**: Isolation between providers; configurable per-provider settings
**Implementation**:
- Use `http.Client` with configured timeout
- Include `Content-Type: application/json` header
- Handle response streaming for audio data

### Decision: Error Handling

**Decision**: Map local TTS HTTP errors to domain errors
**Rationale**: Consistent error handling across providers
**Mapping**:
- 422 Validation Error → ErrValidation
- 503 Unavailable → ErrProviderUnavailable
- Network errors → ErrProviderUnavailable
- Timeout → ErrProviderUnavailable

### Decision: Concurrency Control

**Decision**: Use atomic counter for active jobs (same as ElevenLabs provider)
**Rationale**: Consistent approach; proven pattern in existing code
**Implementation**: Copy pattern from `internal/provider/elevenlabs/provider.go`

## Best Practices Applied

### Go Provider Pattern

- Each provider in its own package under `internal/provider/`
- Implement `domain.TTSProvider` interface
- Use constructor function `NewProviderFromConfig(cfg) (TTSProvider, error)`
- Keep HTTP client internal to provider package

### Configuration Best Practices

- Use Viper for config loading
- Support environment variable substitution (`${VAR}`)
- Validate config at startup (fail fast)
- Log loaded configuration (redact secrets)

### Testing Strategy

- Contract tests verify TTSProvider interface compliance
- Mock HTTP server for selfhosted provider unit tests
- Integration tests require running local TTS service
- Use testify for assertions