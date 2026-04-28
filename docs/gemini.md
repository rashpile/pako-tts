# Gemini Provider

This document describes how to use the Gemini TTS provider through the pako-tts API. It covers configuration, voice list, language handling, style instructions, output formats, and known limitations.

## Configuration

| Config field | Env var | Default | Required | Notes |
|---|---|---|---|---|
| `api_key` | `GEMINI_API_KEY` (or any name you bind) | — | yes | Your Google AI API key |
| `model_id` | — | `gemini-3.1-flash-tts-preview` | no | Only one model is supported today |
| `default_style` | — | `""` | no | Free-text style applied when a request omits `voice_settings.style_instructions` |

Sample `config.yaml` entry:

```yaml
providers:
  default: "gemini"
  list:
    - name: "gemini"
      type: "gemini"
      api_key: "${GEMINI_API_KEY}"
      model_id: "gemini-3.1-flash-tts-preview"   # optional; this is the default
      default_style: "warm, conversational"        # optional; per-request style overrides this
```

Authentication uses the `x-goog-api-key` HTTP header. The API key is never sent as a query parameter.

## Voices

Gemini exposes 30 prebuilt voices. All voices are language-agnostic — the spoken language is controlled via the `language_code` request field, not by choosing a voice. In the browser UI, Gemini voices therefore appear with a blank language column; this is expected.

```bash
curl http://localhost:8080/api/v1/providers/gemini/voices
```

Full voice list:

| Voice ID | Voice ID | Voice ID |
|---|---|---|
| Zephyr | Puck | Charon |
| Kore | Fenrir | Leda |
| Orus | Aoede | Callirrhoe |
| Autonoe | Enceladus | Iapetus |
| Umbriel | Algieba | Despina |
| Erinome | Algenib | Rasalgethi |
| Laomedeia | Achernar | Alnilam |
| Schedar | Gacrux | Pulcherrima |
| Achird | Zubenelgenubi | Vindemiatrix |
| Sadachbia | Sadaltager | Sulafat |

## Model

```bash
curl http://localhost:8080/api/v1/providers/gemini/models
```

One model is currently exposed: `gemini-3.1-flash-tts-preview`. It supports 72 languages (ISO 639-1 codes).

## Language handling

Set `language_code` (ISO 639-1) in the request body to control the spoken language:

```json
{ "language_code": "ro" }
```

The server injects a spoken-language directive into the Gemini prompt: `"Speak in Romanian."`. The Gemini API has no native `language_code` field — the directive is the only mechanism.

If you omit `language_code`, Gemini auto-detects the language from the text. For multilingual or short texts this is often reliable, but explicit codes are recommended for production use.

If you supply an unrecognised ISO code (not in the server's 72-code table), the language directive is silently omitted and Gemini falls back to auto-detection.

Supported ISO 639-1 codes: `af`, `sq`, `am`, `ar`, `hy`, `az`, `eu`, `be`, `bn`, `bs`, `bg`, `ca`, `zh`, `hr`, `cs`, `da`, `nl`, `en`, `et`, `fi`, `fr`, `gl`, `ka`, `de`, `el`, `gu`, `he`, `hi`, `hu`, `is`, `id`, `it`, `ja`, `kn`, `kk`, `km`, `ko`, `lo`, `lv`, `lt`, `mk`, `ms`, `ml`, `mt`, `mr`, `mn`, `ne`, `nb`, `fa`, `pl`, `pt`, `ro`, `ru`, `sr`, `si`, `sk`, `sl`, `es`, `su`, `sw`, `sv`, `ta`, `te`, `th`, `tr`, `uk`, `ur`, `uz`, `vi`, `cy`, `yo`, `zu`.

## Style instructions

`style_instructions` is a free-text field that directs the voice's delivery style. It is unique to the Gemini provider; all other providers silently ignore it.

Two layers of style control:

1. **Config default** (`default_style` in `config.yaml`) — applied when a request omits `style_instructions` or sends an empty string.
2. **Per-request override** (`voice_settings.style_instructions`) — takes priority over the config default when non-empty.

The style directive is injected into the Gemini prompt as `"Style: <text>."`. For example, `"warm, slightly slow, beginner-friendly"` becomes `Style: warm, slightly slow, beginner-friendly.`

You can also use Gemini's audio tags inline in the `text` field: `[laughs]`, `[whispers]`, `[sighs]`, etc. These are passed through as-is and interpreted by the model.

### Synthesis request

```json
{
  "text": "Salut, ce mai faci?",
  "voice_id": "Despina",
  "model_id": "gemini-3.1-flash-tts-preview",
  "provider": "gemini",
  "output_format": "mp3",
  "language_code": "ro",
  "voice_settings": {
    "style_instructions": "warm, slightly slow, beginner-friendly"
  }
}
```

| Field | Type | Default | Notes |
|---|---|---|---|
| `text` | string | — | Required. Sync endpoint enforces `MAX_SYNC_TEXT_LENGTH` (5000 chars). Async has no length limit. |
| `voice_id` | string | server `DEFAULT_VOICE_ID` | Use any name from the voice list above. |
| `model_id` | string | `gemini-3.1-flash-tts-preview` | Only one model is currently supported. |
| `provider` | string | server default provider | Use `"gemini"` to force this provider. |
| `output_format` | string | `mp3` | `mp3` or `wav`. See Output formats below. |
| `language_code` | string | `""` (auto-detect) | ISO 639-1 code. |
| `voice_settings.style_instructions` | string | `""` (uses config default) | Free-text style directive. |

## Output formats

Gemini natively returns raw 16-bit PCM (24 kHz, mono). The server transcodes on the fly:

| `output_format` | Transcoding | Content-Type |
|---|---|---|
| `mp3` (default) | ffmpeg: PCM → MP3 128 kbps | `audio/mpeg` |
| `wav` | stdlib: RIFF header prepended to PCM | `audio/wav` |

ffmpeg is included in the Docker runtime image. If you run the server outside Docker, install ffmpeg and ensure it is on `PATH`.

## Token limits

Gemini has an 8192-token input limit. If your text exceeds this, the upstream error is returned as-is (HTTP 503 with the Gemini error body). Client-side chunking is a v2 roadmap item.

## Auth and availability check

- Auth: `x-goog-api-key: <api_key>` header on every request.
- `IsAvailable` / health check: makes a `GET` request to the Gemini models endpoint. **This endpoint is publicly readable**, so a connectivity check may return `true` even when the API key is invalid or revoked. A full availability check would require spending tokens on a probe synthesis request, which is deliberately avoided. If you need to validate the API key, make a test synthesis call.

## Examples

### Synthesise Romanian speech

```bash
curl -X POST http://localhost:8080/api/v1/tts \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Salut, ce mai faci?",
    "voice_id": "Despina",
    "language_code": "ro",
    "output_format": "mp3",
    "voice_settings": {"style_instructions": "warm, conversational"}
  }' \
  --output salut.mp3
```

### Synthesise with a whisper

```bash
curl -X POST http://localhost:8080/api/v1/tts \
  -H "Content-Type: application/json" \
  -d '{
    "text": "[whispers] This is a secret.",
    "voice_id": "Kore",
    "language_code": "en"
  }' \
  --output whisper.mp3
```

### Async job for long text

```bash
JOB_ID=$(curl -s -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "text": "<paste long text here>",
    "provider": "gemini",
    "voice_id": "Aoede",
    "language_code": "en",
    "voice_settings": {"style_instructions": "audiobook narrator, clear and measured"}
  }' | jq -r .job_id)

curl http://localhost:8080/api/v1/jobs/$JOB_ID
curl http://localhost:8080/api/v1/jobs/$JOB_ID/result --output chapter.mp3
```

## Known limitations

- **`IsAvailable` caveat**: connectivity-only — an invalid API key may still report the provider as available (see Auth section above).
- **Blank language in UI**: Gemini voices are language-agnostic and appear with no language tag in the browser UI. Set `language_code` per request via the API to control the spoken language.
- **Token limit**: 8192-token input limit; upstream error is surfaced as-is when exceeded.
- **Single model**: only `gemini-3.1-flash-tts-preview` is available today.
- **Multi-speaker**: not supported (v2 roadmap).
- **Streaming**: Gemini does not support streaming TTS; not supported.
- **`voice_settings.stability` / `similarity_boost` / `style` / `use_speaker_boost` / `speed`**: accepted in the request but silently ignored by the Gemini provider — these are ElevenLabs-specific numeric controls.
