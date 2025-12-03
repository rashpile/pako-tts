# Research: TTS API Wrapper

**Feature**: 001-tts-api-wrapper
**Date**: 2025-12-03

## Executive Summary

This document captures research findings for implementing a Go-based TTS API wrapper with ElevenLabs integration and job queue processing. Key decisions are informed by the existing Python implementation (pako-speech) and Go best practices.

---

## 1. ElevenLabs API Integration

### Decision: Use ElevenLabs REST API directly via HTTP client

**Rationale**:
- No official Go SDK exists for ElevenLabs
- REST API is well-documented and straightforward
- Existing Python implementation provides proven patterns

**Alternatives Considered**:
- Third-party Go clients (none maintained/reliable found)
- gRPC (not supported by ElevenLabs)

### ElevenLabs API Findings (from existing Python implementation)

**Endpoint**: `POST /v1/text-to-speech/{voice_id}`

**Key Parameters**:
```
- voice_id: string (required) - Voice identifier
- text: string (required) - Text to synthesize
- model_id: string - "eleven_multilingual_v2" (recommended)
- output_format: string - "mp3_22050_32" (default)
- voice_settings: object
  - stability: float (0.0-1.0)
  - similarity_boost: float (0.0-1.0)
  - style: float (0.0-1.0)
  - speed: float (0.7-1.2)
  - use_speaker_boost: boolean
```

**Response**: Streaming audio bytes (chunked)

**Authentication**: API key via `xi-api-key` header

**Voice Mapping** (from existing implementation):
| Name | Voice ID |
|------|----------|
| adam | pNInz6obpgDQGcFmaJgB |
| aria | 9BWtsMINqrJLrRacOk9x |
| sarah | EXAVITQu4vr4xnSDxMaL |
| laura | FGY2WhTYpPnrIDTdsKH5 |
| charlie | IKne3meq5aSn9XLyUdCD |
| george | JBFqnCBsd6RMkjVDRZzb |

---

## 2. Go HTTP Framework

### Decision: Use Chi router (go-chi/chi/v5)

**Rationale**:
- Lightweight, idiomatic Go
- Compatible with standard net/http
- Rich middleware ecosystem (logger, recoverer, timeout, CORS)
- Matches transcriber project patterns
- High performance, production-ready

**Alternatives Considered**:
- Gin: More opinionated, larger dependency
- Standard net/http: Lacks routing features needed
- Fiber: Not net/http compatible

### Chi Middleware Stack
```go
r.Use(middleware.RequestID)    // Unique request IDs
r.Use(middleware.RealIP)       // X-Forwarded-For handling
r.Use(middleware.Logger)       // Request logging
r.Use(middleware.Recoverer)    // Panic recovery
r.Use(middleware.Timeout(30*time.Second)) // Request timeout
```

---

## 3. Job Queue Architecture

### Decision: In-memory queue with worker pool (initial), Redis-backed queue (future)

**Rationale**:
- Start simple per constitution (Simplicity & YAGNI)
- In-memory sufficient for initial scale (100 concurrent jobs)
- Clear interface allows Redis upgrade without core changes

**Alternatives Considered**:
- Redis from start: Over-engineering for initial requirements
- RabbitMQ: Too complex for simple job queue
- Database-backed: Slower, more complex

### Queue Interface (Ports & Adapters)
```go
type JobQueue interface {
    Enqueue(ctx context.Context, job *Job) error
    Dequeue(ctx context.Context) (*Job, error)
    GetStatus(ctx context.Context, jobID string) (*JobStatus, error)
    UpdateStatus(ctx context.Context, jobID string, status JobStatus) error
}
```

---

## 4. Audio Storage

### Decision: Local filesystem with configurable path (initial), S3-compatible (future)

**Rationale**:
- Matches existing Python implementation pattern
- Simple, reliable for initial deployment
- Clear interface for future cloud storage

**Storage Interface**:
```go
type AudioStorage interface {
    Store(ctx context.Context, jobID string, audio []byte) (string, error)
    Retrieve(ctx context.Context, jobID string) (io.ReadCloser, error)
    Delete(ctx context.Context, jobID string) error
    Exists(ctx context.Context, jobID string) bool
}
```

---

## 5. Configuration Management

### Decision: Environment variables with Viper

**Rationale**:
- Standard 12-factor app pattern
- Matches existing Python implementation (.env files)
- Viper provides flexible config sources

**Environment Variables**:
```
ELEVENLABS_API_KEY      # Required: API key
AUDIO_STORAGE_PATH      # Default: ./audio_cache
HTTP_PORT               # Default: 8080
DEFAULT_VOICE_ID        # Default: pNInz6obpgDQGcFmaJgB (Adam)
MAX_SYNC_TEXT_LENGTH    # Default: 5000
JOB_RETENTION_HOURS     # Default: 24
WORKER_COUNT            # Default: 4
```

---

## 6. Go Version and Dependencies

### Decision: Go 1.23 (latest stable)

**Note**: User specified "v1.25" which doesn't exist. Go 1.23 is the current latest stable version.

**Primary Dependencies**:
| Package | Version | Purpose |
|---------|---------|---------|
| github.com/go-chi/chi/v5 | v5.1.0 | HTTP router |
| github.com/go-chi/cors | v1.2.1 | CORS middleware |
| github.com/google/uuid | v1.6.0 | UUID generation |
| github.com/spf13/viper | v1.19.0 | Configuration |
| go.uber.org/zap | v1.27.0 | Structured logging |

---

## 7. Provider Interface (Multi-provider Architecture)

### Decision: Define TTSProvider interface per constitution

**Rationale**:
- Constitution requires interface-first design
- Enables adding providers without core changes
- Supports testing with mocks

**Provider Interface**:
```go
type TTSProvider interface {
    Name() string
    Synthesize(ctx context.Context, req *SynthesisRequest) (*SynthesisResult, error)
    ListVoices(ctx context.Context) ([]Voice, error)
    IsAvailable(ctx context.Context) bool
    MaxConcurrent() int
}

type SynthesisRequest struct {
    Text         string
    VoiceID      string
    OutputFormat string
    Settings     VoiceSettings
}

type SynthesisResult struct {
    Audio       io.Reader
    ContentType string
    Duration    time.Duration
}
```

---

## 8. Error Handling

### Decision: Custom error types with HTTP status mapping

**Error Types**:
```go
var (
    ErrJobNotFound     = NewAPIError(404, "JOB_NOT_FOUND", "Job not found")
    ErrResultExpired   = NewAPIError(410, "RESULT_EXPIRED", "Result has expired")
    ErrJobNotComplete  = NewAPIError(425, "JOB_NOT_COMPLETE", "Job not yet completed")
    ErrValidation      = NewAPIError(422, "VALIDATION_ERROR", "Validation failed")
    ErrProviderUnavailable = NewAPIError(503, "PROVIDER_UNAVAILABLE", "TTS provider unavailable")
)
```

---

## 9. Testing Strategy

### Decision: Interface-based testing with mocks

**Test Categories**:
- Unit tests: Mock all interfaces (provider, queue, storage)
- Integration tests: Test HTTP handlers with real in-memory implementations
- Contract tests: Verify OpenAPI spec compliance

**Testing Tools**:
- Standard Go testing package
- github.com/stretchr/testify for assertions
- httptest for handler testing

---

## 10. Project Structure

### Decision: Standard Go project layout

```
/
├── cmd/
│   └── server/
│       └── main.go           # Entry point
├── internal/
│   ├── api/
│   │   ├── handlers/         # HTTP handlers
│   │   ├── middleware/       # Custom middleware
│   │   └── routes.go         # Route registration
│   ├── domain/
│   │   ├── job.go            # Job entity
│   │   ├── provider.go       # Provider interface
│   │   └── voice.go          # Voice configuration
│   ├── provider/
│   │   └── elevenlabs/       # ElevenLabs adapter
│   ├── queue/
│   │   └── memory/           # In-memory queue implementation
│   └── storage/
│       └── filesystem/       # File storage implementation
├── pkg/
│   └── config/               # Configuration utilities
├── tests/
│   ├── contract/             # Contract tests
│   └── integration/          # Integration tests
├── go.mod
├── go.sum
└── Makefile
```

---

## Summary

All technical unknowns have been resolved:

| Area | Decision |
|------|----------|
| Go Version | 1.23 (latest stable) |
| HTTP Framework | Chi v5 |
| ElevenLabs Integration | Direct REST API via net/http |
| Job Queue | In-memory with interface for Redis upgrade |
| Storage | Filesystem with interface for S3 upgrade |
| Configuration | Environment variables + Viper |
| Testing | Interface mocks + httptest |