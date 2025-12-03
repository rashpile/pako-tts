# Data Model: TTS API Wrapper

**Feature**: 001-tts-api-wrapper
**Date**: 2025-12-03

---

## Entities

### 1. Job

Represents a TTS synthesis request submitted for processing.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| job_id | UUID | Yes | Unique identifier for the job |
| status | JobStatus | Yes | Current processing status |
| text | string | Yes | Input text to synthesize |
| voice_id | string | Yes | Selected voice identifier |
| provider_name | string | Yes | TTS provider used (e.g., "elevenlabs") |
| output_format | string | Yes | Audio format (mp3, wav) |
| voice_settings | VoiceSettings | No | Optional voice customization |
| created_at | timestamp | Yes | When job was submitted |
| started_at | timestamp | No | When processing began |
| completed_at | timestamp | No | When processing finished |
| progress_percentage | float | Yes | Progress 0-100 |
| estimated_completion_at | timestamp | No | Estimated completion time |
| error_message | string | No | Error details if failed |
| result_path | string | No | Path to audio file (when completed) |
| expires_at | timestamp | No | When result expires (24h after completion) |

**State Transitions**:
```
[created] → queued → processing → completed
                  ↘            ↗
                    → failed
```

### 2. JobStatus (Enum)

| Value | Description |
|-------|-------------|
| queued | Job is waiting in queue |
| processing | Job is being processed |
| completed | Job finished successfully |
| failed | Job failed with error |

### 3. VoiceSettings

Optional voice customization parameters.

| Field | Type | Required | Default | Range | Description |
|-------|------|----------|---------|-------|-------------|
| stability | float | No | 0.0 | 0.0-1.0 | Voice stability |
| similarity_boost | float | No | 1.0 | 0.0-1.0 | Voice similarity |
| style | float | No | 0.0 | 0.0-1.0 | Style exaggeration |
| speed | float | No | 1.0 | 0.7-1.2 | Speaking speed |
| use_speaker_boost | bool | No | true | - | Speaker boost toggle |

### 4. Provider

Represents a TTS service provider.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Provider identifier (e.g., "elevenlabs") |
| type | string | Yes | Provider type class name |
| max_concurrent | int | Yes | Maximum concurrent jobs |
| is_default | bool | Yes | Whether this is the default provider |
| is_available | bool | Yes | Current availability status |
| active_jobs | int | Yes | Number of jobs currently processing |

### 5. Voice

Represents an available voice option.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| voice_id | string | Yes | Unique voice identifier |
| name | string | Yes | Human-readable voice name |
| provider | string | Yes | Provider this voice belongs to |
| language | string | No | Primary language code (ISO 639-1) |
| gender | string | No | Voice gender (male/female/neutral) |
| preview_url | string | No | URL to voice sample |

### 6. HealthStatus

Service health information.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| status | string | Yes | Overall status (healthy/unhealthy) |
| version | string | Yes | API version |
| providers | []ProviderStatus | Yes | Provider health details |

### 7. ProviderStatus

Provider health in health check response.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Provider identifier |
| available | bool | Yes | Availability status |
| active_jobs | int | Yes | Currently processing jobs |
| max_concurrent | int | Yes | Maximum capacity |

---

## Request/Response Models

### TTSRequest (Sync Endpoint)

```json
{
  "text": "string (required, max 5000 chars)",
  "voice_id": "string (optional, defaults to provider default)",
  "output_format": "string (optional, mp3|wav, default: mp3)",
  "voice_settings": {
    "stability": "float (optional, 0.0-1.0)",
    "similarity_boost": "float (optional, 0.0-1.0)",
    "style": "float (optional, 0.0-1.0)",
    "speed": "float (optional, 0.7-1.2)",
    "use_speaker_boost": "bool (optional)"
  }
}
```

### JobCreateRequest (Async Endpoint)

```json
{
  "text": "string (required)",
  "voice_id": "string (optional)",
  "provider": "string (optional, defaults to default provider)",
  "output_format": "string (optional, mp3|wav, default: mp3)",
  "voice_settings": {
    "stability": "float (optional)",
    "similarity_boost": "float (optional)",
    "style": "float (optional)",
    "speed": "float (optional)",
    "use_speaker_boost": "bool (optional)"
  }
}
```

### JobCreateResponse

```json
{
  "job_id": "uuid",
  "status": "queued",
  "created_at": "2025-12-03T10:30:00Z"
}
```

### JobStatusResponse

```json
{
  "job_id": "uuid",
  "status": "processing",
  "provider_name": "elevenlabs",
  "created_at": "2025-12-03T10:30:00Z",
  "started_at": "2025-12-03T10:30:05Z",
  "completed_at": null,
  "progress_percentage": 45.0,
  "estimated_completion_at": "2025-12-03T10:32:00Z",
  "error_message": null
}
```

### HealthResponse

```json
{
  "status": "healthy",
  "version": "0.0.1",
  "providers": [
    {
      "name": "elevenlabs",
      "available": true,
      "active_jobs": 2,
      "max_concurrent": 4
    }
  ]
}
```

### ProvidersListResponse

```json
{
  "providers": [
    {
      "name": "elevenlabs",
      "type": "ElevenLabsProvider",
      "max_concurrent": 4,
      "is_default": true,
      "is_available": true
    }
  ],
  "default_provider": "elevenlabs"
}
```

### ErrorResponse

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Text exceeds maximum length of 5000 characters",
    "details": {
      "field": "text",
      "max_length": 5000,
      "actual_length": 5234
    }
  }
}
```

---

## Validation Rules

### Text Input
- **Sync endpoint**: Max 5,000 characters
- **Async endpoint**: No maximum (queue handles long text)
- Empty text: Rejected with 422

### Voice Settings
- stability: 0.0 ≤ value ≤ 1.0
- similarity_boost: 0.0 ≤ value ≤ 1.0
- style: 0.0 ≤ value ≤ 1.0
- speed: 0.7 ≤ value ≤ 1.2

### Output Format
- Allowed values: `mp3`, `wav`
- Default: `mp3`

### Job ID
- Must be valid UUID v4
- Non-existent ID returns 404

---

## Indexes (for future persistence)

| Entity | Index | Purpose |
|--------|-------|---------|
| Job | job_id (PK) | Primary lookup |
| Job | status + created_at | Queue ordering |
| Job | expires_at | Cleanup queries |
| Job | provider_name + status | Provider load balancing |