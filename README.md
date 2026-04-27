# Pako TTS

High-performance Text-to-Speech API service powered by ElevenLabs.

## Features

- **Synchronous API**: Direct audio response for short texts (< 5,000 chars)
- **Asynchronous Jobs**: Queue-based processing for long texts
- **Multiple Voices**: Wide selection of high-quality voices
- **Voice Customization**: Adjustable stability, speed, and style
- **Job Management**: Real-time status tracking and result retrieval
- **Result Expiration**: Automatic cleanup after 24 hours

## Quick Start

### Prerequisites

- Go 1.23+
- ElevenLabs API key

### Configuration

Copy the example environment file and set your API key:

```bash
cp .env.example .env
# Edit .env and set ELEVENLABS_API_KEY
```

### Run Locally

```bash
# Install dependencies
make deps

# Run the server
make run
```

The server starts at `http://localhost:8080`.

### Run with Docker

```bash
# Build and run
docker-compose up --build

# Or manually
docker build -t pako-tts .
docker run -p 8080:8080 -e ELEVENLABS_API_KEY=your-key pako-tts
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/health` | GET | Health check |
| `/api/v1/providers` | GET | List TTS providers |
| `/api/v1/providers/{name}/voices` | GET | List voices for a provider |
| `/api/v1/providers/{name}/models` | GET | List models for a provider |
| `/api/v1/tts` | POST | Synchronous TTS (< 5000 chars) |
| `/api/v1/jobs` | POST | Submit async job |
| `/api/v1/jobs/{id}` | GET | Get job status |
| `/api/v1/jobs/{id}/result` | GET | Download audio result |
| `/openapi.json` | GET | OpenAPI specification |
| `/ui/` | GET | Browser UI for trying the API |

Both `POST /api/v1/tts` and `POST /api/v1/jobs` accept an optional `model_id` field. When omitted, the provider's configured default model is used (for ElevenLabs, set via `model_id` in `config.yaml` — defaults to `eleven_multilingual_v2`).

Both endpoints also accept an optional `language_code` field (ISO 639-1, e.g. `"en"`, `"es"`). When set, the chosen model is forced to render in that language; if the model does not support the requested language, the upstream error is surfaced as a 503. When omitted, the provider/model default applies. The selfhosted provider forwards `language_code` to its upstream `language` field via the API. The browser UI Language picker is currently populated only from ElevenLabs' models endpoint; selfhosted users wanting to set a language must do so via the API directly (not the UI).

## Web UI

A simple browser UI is available at [`/ui/`](http://localhost:8080/ui/) for trying the API without writing curl commands. It lets you pick a provider, choose a voice, model, and language (ISO 639-1 code; populated from the union of languages advertised by the loaded models), enter text, select an output format (mp3/wav), and play or download the synthesized audio in-browser. A collapsible **Advanced** section exposes provider-specific voice settings (for ElevenLabs: `stability`, `similarity_boost`, `style`, `use_speaker_boost`). The UI is a single embedded HTML file served by the same Go binary — no extra build step or static-asset hosting required.

## Providers

Provider-specific docs (parameters, voice settings, examples, known limitations):

- **[ElevenLabs](docs/elevenlabs.md)** — voice settings (stability, similarity_boost, style, use_speaker_boost), output formats, examples
- **[Gemini](docs/gemini.md)** — 30 prebuilt voices, 72 languages, free-text style instructions, server-side WAV/MP3 transcode from PCM

### Sample Gemini config

```yaml
providers:
  default: "gemini"
  list:
    - name: "gemini"
      type: "gemini"
      api_key: "${GEMINI_API_KEY}"
      model_id: "gemini-3.1-flash-tts-preview"  # optional; this is the default
      default_style: "warm, conversational"       # optional; per-request style overrides this
```

## Usage Examples

### Synchronous TTS (short text)

```bash
curl -X POST http://localhost:8080/api/v1/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello world!", "voice_id": "pNInz6obpgDQGcFmaJgB"}' \
  --output hello.mp3
```

### Async Job (long text)

```bash
# Submit job
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{"text": "Your long text here..."}'

# Check status
curl http://localhost:8080/api/v1/jobs/{job_id}

# Download result (when completed)
curl http://localhost:8080/api/v1/jobs/{job_id}/result --output audio.mp3
```

## Development

```bash
# Format code
make fmt

# Run linter
make lint

# Run tests
make test

# Build binary
make build
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ELEVENLABS_API_KEY` | - | ElevenLabs API key (required) |
| `HTTP_PORT` | 8080 | Server port |
| `DEFAULT_VOICE_ID` | pNInz6obpgDQGcFmaJgB | Default voice |
| `MAX_SYNC_TEXT_LENGTH` | 5000 | Max chars for sync endpoint |
| `SYNC_TIMEOUT` | 30s | Sync request timeout |
| `WORKER_COUNT` | 4 | Background workers |
| `AUDIO_STORAGE_PATH` | ./audio_cache | Audio file storage |
| `JOB_RETENTION_HOURS` | 24 | Result retention period |
| `LOG_LEVEL` | info | Log level |
| `LOG_FORMAT` | json | Log format (json/console) |

## License

MIT