# ElevenLabs Provider

This document describes how to use the ElevenLabs TTS provider through the pako-tts API. It covers the parameters you can send, what each one does, and what the current implementation accepts vs. ignores.

For the underlying ElevenLabs research notes (model schema, parameters not yet exposed, known limits), see [`research-elevenlab.md`](research-elevenlab.md).

## Configuration

| Env var | Default | Required | Notes |
|---|---|---|---|
| `ELEVENLABS_API_KEY` | — | yes | Your ElevenLabs API key |
| `DEFAULT_VOICE_ID` | `pNInz6obpgDQGcFmaJgB` | no | Used when a request omits `voice_id` |

The default model is configurable in `config.yaml` under the elevenlabs provider entry:

```yaml
providers:
  list:
    - name: "elevenlabs"
      type: "elevenlabs"
      api_key: "${ELEVENLABS_API_KEY}"
      model_id: "eleven_multilingual_v2"  # optional; default when blank
```

If `model_id` is omitted, the server falls back to `eleven_multilingual_v2`. Per-request `model_id` (see below) overrides the default.

## Listing voices

```bash
curl http://localhost:8080/api/v1/providers/elevenlabs/voices
```

Response:

```json
{
  "provider": "elevenlabs",
  "voices": [
    {
      "voice_id": "21m00Tcm4TlvDq8ikWAM",
      "name": "Rachel",
      "provider": "elevenlabs",
      "language": "en",
      "gender": "female",
      "preview_url": "https://..."
    }
  ]
}
```

Use any returned `voice_id` in subsequent TTS requests.

## Listing models

```bash
curl http://localhost:8080/api/v1/providers/elevenlabs/models
```

Response:

```json
{
  "provider": "elevenlabs",
  "models": [
    {
      "model_id": "eleven_multilingual_v2",
      "name": "Eleven Multilingual v2",
      "provider": "elevenlabs",
      "description": "...",
      "languages": ["en", "es", "fr"]
    }
  ]
}
```

Use any returned `model_id` in the `model_id` field of `/tts` or `/jobs` requests.

## Synthesis request shape

Both `POST /api/v1/tts` (sync) and `POST /api/v1/jobs` (async) accept the same body:

```json
{
  "text": "Hello world",
  "voice_id": "21m00Tcm4TlvDq8ikWAM",
  "model_id": "eleven_multilingual_v2",
  "provider": "elevenlabs",
  "output_format": "mp3",
  "voice_settings": {
    "stability": 0.5,
    "similarity_boost": 0.75,
    "style": 0.0,
    "use_speaker_boost": true
  }
}
```

| Field | Type | Default | Notes |
|---|---|---|---|
| `text` | string | — | Required. Sync endpoint enforces `MAX_SYNC_TEXT_LENGTH` (5000 chars). Async has no length limit. |
| `voice_id` | string | server `DEFAULT_VOICE_ID` | Optional. Get IDs from `/api/v1/providers/elevenlabs/voices`. |
| `model_id` | string | provider's configured default | Optional. Get IDs from `/api/v1/providers/elevenlabs/models`. Falls back to the `model_id` in `config.yaml` (or `eleven_multilingual_v2`) when omitted. |
| `provider` | string | server default provider | Optional. Use `"elevenlabs"` to force this provider. |
| `output_format` | string | `mp3` | One of `mp3` or `wav`. See limits below. |
| `voice_settings` | object | server defaults | Optional. See [Voice settings](#voice-settings) below. |

## Voice settings

The `voice_settings` object lets you tune how the voice is rendered. All four fields are optional — omit any to use the server defaults (which fall back to ElevenLabs' own defaults).

### `stability` (0.0 – 1.0)

Controls how consistent the voice sounds across the request.

- **Lower** (e.g. `0.0–0.3`) — more emotional/expressive but may vary noticeably between runs and within a single render.
- **Higher** (e.g. `0.7–1.0`) — more monotone and predictable; the voice "sticks" to one delivery.
- **Default in this server**: `0.5` (when not provided).
- **ElevenLabs default**: `0.5`. Long-form narration usually benefits from `0.5–0.75`.

### `similarity_boost` (0.0 – 1.0)

Controls how strictly the model adheres to the original voice's timbre.

- **Lower** — looser interpretation, can sound "off" or generic.
- **Higher** (e.g. `0.75+`) — closer to the source voice, but can also amplify noise/artifacts present in the original sample.
- **Default in this server**: `0.75`.
- **ElevenLabs default**: `0.75`. Increasing it does not always improve quality — past `0.9` can introduce hissing or breathing artifacts.

### `style` (0.0 – 1.0)

Style exaggeration — how strongly the model amplifies the voice's expressive style.

- `0.0` — neutral, most stable, fastest.
- `0.3` — mild expressiveness.
- `0.6` — strongly expressive (more emotion, more drama). Some occasional artifacts.
- `1.0` — maximum exaggeration. Often produces hallucinations, slurring, or weird pacing.
- **Default in this server**: `0.0`.
- **Trade-offs**: higher values **increase latency** and **decrease stability**. ElevenLabs' guidance is to keep `style = 0.0` unless you have a specific reason to push it.
- **Model support**: only models where `can_use_style: true` honor this. The default model `eleven_multilingual_v2` does. Flash/Turbo models may silently ignore it.

### `use_speaker_boost` (bool)

When `true`, applies an enhancement that increases similarity to the original speaker.

- **Cost**: slightly higher latency.
- **Default in this server**: `true`.
- **Model support**: only models where `can_use_speaker_boost: true` honor it.

## Output formats

The `output_format` field accepts:

| Value | Sent to ElevenLabs as | Audio |
|---|---|---|
| `mp3` (default) | `mp3_22050_32` | MP3, 22.05 kHz, 32 kbps |
| `wav` | `pcm_22050` | PCM 16-bit, 22.05 kHz (returned as raw PCM, not a WAV container) |

Higher-quality formats (44.1 kHz MP3, Opus, μ-law, etc.) are **not currently exposed**. See `research-elevenlab.md` for the full list ElevenLabs supports.

## Examples

### Plain synthesis

```bash
curl -X POST http://localhost:8080/api/v1/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello, this is a test."}' \
  --output hello.mp3
```

### Pick a specific voice

```bash
curl -X POST http://localhost:8080/api/v1/tts \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Hello, this is a test.",
    "voice_id": "21m00Tcm4TlvDq8ikWAM"
  }' \
  --output rachel.mp3
```

### Tune the voice settings

```bash
curl -X POST http://localhost:8080/api/v1/tts \
  -H "Content-Type: application/json" \
  -d '{
    "text": "She said excitedly, this is going to be wonderful!",
    "voice_settings": {
      "stability": 0.4,
      "similarity_boost": 0.8,
      "style": 0.6,
      "use_speaker_boost": true
    }
  }' \
  --output expressive.mp3
```

### Long text via async job

```bash
JOB_ID=$(curl -s -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{"text": "<paste a long article here>"}' | jq -r .job_id)

# Poll status
curl http://localhost:8080/api/v1/jobs/$JOB_ID

# When status == "completed", download
curl http://localhost:8080/api/v1/jobs/$JOB_ID/result --output article.mp3
```

## Known limitations

These are accepted-by-the-API but **silently dropped** before reaching ElevenLabs, or not currently supported. Track in [`todo.md`](todo.md).

- **`voice_settings.speed`** — accepted in the request but **never sent** to ElevenLabs (mapping bug). Setting it has no effect today.
- **`language_code`** — not exposed. ElevenLabs supports it for some models (Turbo/Flash/v3); when added, it will let you force a specific language for normalization.
- **`seed`** — not exposed. Would enable deterministic output for reproducible runs.
- **`previous_text` / `next_text`** — not exposed. Would help prosody continuity when stitching together long text in chunks.
- **`pronunciation_dictionary_locators`** — not exposed.
- **`apply_text_normalization`** — not exposed (always uses ElevenLabs default `auto`).
- **High-bitrate / Opus / telephony output formats** — not exposed.

## Tips

- The browser UI at [`/ui/`](http://localhost:8080/ui/) is the fastest way to try voices and hear what `style` / `stability` actually sound like. It exposes a Model dropdown and a collapsible **Advanced** section with `stability`, `similarity_boost`, `style`, and `use_speaker_boost` controls.
- Voice IDs from the `/voices` endpoint are stable per ElevenLabs account. If you change accounts (different API key), the IDs change too.
- For very long text, prefer the async `/jobs` endpoint — sync hits the 5000-char cap and may also exceed `SYNC_TIMEOUT`.
- ElevenLabs free tier has tighter per-request character limits and concurrency. Check `max_characters_request_free_user` / `max_characters_request_subscribed_user` on the model object (see `research-elevenlab.md`).
