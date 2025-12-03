# Quickstart: Pako TTS API

This guide covers basic usage of the Pako TTS API for text-to-speech conversion.

## Prerequisites

- Running Pako TTS server (default: `http://localhost:8080`)
- ElevenLabs API key configured on server

## Quick Examples

### 1. Synchronous TTS (Short Text)

For texts under 5,000 characters, get audio directly:

```bash
# Simple request
curl -X POST http://localhost:8080/api/v1/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello, this is a test."}' \
  --output hello.mp3

# With voice selection
curl -X POST http://localhost:8080/api/v1/tts \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Hello, this is a test.",
    "voice_id": "JBFqnCBsd6RMkjVDRZzb",
    "output_format": "mp3"
  }' \
  --output hello.mp3
```

### 2. Async Job (Long Text)

For longer texts, use the job queue:

```bash
# Step 1: Submit job
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "text": "This is a very long text that will be processed asynchronously...",
    "voice_id": "pNInz6obpgDQGcFmaJgB"
  }'

# Response:
# {"job_id": "550e8400-e29b-41d4-a716-446655440000", "status": "queued", "created_at": "..."}

# Step 2: Check status (poll until completed)
curl http://localhost:8080/api/v1/jobs/550e8400-e29b-41d4-a716-446655440000

# Response when processing:
# {"job_id": "...", "status": "processing", "progress_percentage": 45.0, ...}

# Response when completed:
# {"job_id": "...", "status": "completed", ...}

# Step 3: Download result
curl http://localhost:8080/api/v1/jobs/550e8400-e29b-41d4-a716-446655440000/result \
  --output audio.mp3
```

### 3. Check Available Providers

```bash
curl http://localhost:8080/api/v1/providers

# Response:
# {
#   "providers": [
#     {"name": "elevenlabs", "is_default": true, "is_available": true, ...}
#   ],
#   "default_provider": "elevenlabs"
# }
```

### 4. Health Check

```bash
curl http://localhost:8080/api/v1/health

# Response:
# {"status": "healthy", "version": "0.0.1", "providers": [...]}
```

## Voice Settings

Customize voice output with settings:

```bash
curl -X POST http://localhost:8080/api/v1/tts \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Hello with custom settings.",
    "voice_id": "pNInz6obpgDQGcFmaJgB",
    "voice_settings": {
      "stability": 0.5,
      "similarity_boost": 0.8,
      "style": 0.0,
      "speed": 1.1
    }
  }' \
  --output custom.mp3
```

| Setting | Range | Default | Description |
|---------|-------|---------|-------------|
| stability | 0.0-1.0 | 0.0 | Higher = more consistent |
| similarity_boost | 0.0-1.0 | 1.0 | Voice clarity |
| style | 0.0-1.0 | 0.0 | Style exaggeration |
| speed | 0.7-1.2 | 1.0 | Speaking speed |

## Available Voices

Common voice IDs:

| Name | Voice ID |
|------|----------|
| Adam | pNInz6obpgDQGcFmaJgB |
| Aria | 9BWtsMINqrJLrRacOk9x |
| Sarah | EXAVITQu4vr4xnSDxMaL |
| Laura | FGY2WhTYpPnrIDTdsKH5 |
| George | JBFqnCBsd6RMkjVDRZzb |
| Charlie | IKne3meq5aSn9XLyUdCD |

## Error Handling

### Common Errors

| Status | Code | Meaning |
|--------|------|---------|
| 404 | JOB_NOT_FOUND | Job ID doesn't exist |
| 410 | RESULT_EXPIRED | Result older than 24h |
| 413 | TEXT_TOO_LONG | Sync text > 5000 chars |
| 422 | VALIDATION_ERROR | Invalid input |
| 425 | JOB_NOT_COMPLETE | Job still processing |
| 503 | PROVIDER_UNAVAILABLE | TTS provider down |

### Example Error Response

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid voice_id",
    "details": {
      "field": "voice_id",
      "value": "invalid-id"
    }
  }
}
```

## Go Client Example

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
)

func main() {
    // Sync TTS
    req := map[string]interface{}{
        "text":     "Hello from Go!",
        "voice_id": "pNInz6obpgDQGcFmaJgB",
    }

    body, _ := json.Marshal(req)
    resp, err := http.Post(
        "http://localhost:8080/api/v1/tts",
        "application/json",
        bytes.NewReader(body),
    )
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        errBody, _ := io.ReadAll(resp.Body)
        fmt.Printf("Error: %s\n", errBody)
        return
    }

    // Save audio
    audio, _ := io.ReadAll(resp.Body)
    os.WriteFile("output.mp3", audio, 0644)
    fmt.Println("Audio saved to output.mp3")
}
```

## Next Steps

- See full [API Reference](./contracts/openapi.yaml) for all endpoints
- Configure voice settings for your use case
- Set up job status webhooks (coming soon)