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
| `/api/v1/tts` | POST | Synchronous TTS (< 5000 chars) |
| `/api/v1/jobs` | POST | Submit async job |
| `/api/v1/jobs/{id}` | GET | Get job status |
| `/api/v1/jobs/{id}/result` | GET | Download audio result |
| `/openapi.json` | GET | OpenAPI specification |

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