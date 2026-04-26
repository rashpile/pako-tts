# ElevenLabs API — Research Notes

Verified 2026-04-26 against the official ElevenLabs docs (`/elevenlabs/elevenlabs-docs` via context7 + `elevenlabs.io/docs/api-reference/...`).

Scope: parameters and endpoints relevant to extending pako-tts. Not exhaustive.

## What this repo currently uses

From `internal/provider/elevenlabs/client.go` and `provider.go`:

| Sent to ElevenLabs | Source |
|---|---|
| `text` | request body |
| `model_id` | **hardcoded** to `eleven_multilingual_v2` (`client.go:16`) |
| `output_format` | only `mp3_22050_32` or `pcm_22050` (computed from `mp3` / `wav`) |
| `voice_settings.stability` | from `domain.VoiceSettings` |
| `voice_settings.similarity_boost` | from `domain.VoiceSettings` |
| `voice_settings.style` | from `domain.VoiceSettings` |
| `voice_settings.use_speaker_boost` | from `domain.VoiceSettings` |
| `voice_id` | URL path |

**Bug:** `domain.VoiceSettings.Speed` exists in the domain (`internal/domain/voice.go:8`, defaulted to `1.0`) but is **never copied** into `VoiceSettingsReq` at `provider.go:72-77`. Setting `speed` via the API silently does nothing. ElevenLabs supports `speed` in `voice_settings` (typical range 0.7–1.2 depending on model).

## `GET /v1/models` — response schema

Each model object (`ModelResponseModel`) returns these fields:

| Field | Type | Notes |
|---|---|---|
| `model_id` | string | use as the `model_id` value when calling TTS |
| `name` | string | |
| `description` | string | |
| `languages` | array | each item: `{ language_id, name }` — supported language codes for THIS model |
| `can_do_text_to_speech` | bool | filter for TTS-capable models |
| `can_do_voice_conversion` | bool | |
| `can_be_finetuned` | bool | |
| `can_use_style` | bool | whether `voice_settings.style` is honored |
| `can_use_speaker_boost` | bool | whether `voice_settings.use_speaker_boost` is honored |
| `serves_pro_voices` | bool | |
| `requires_alpha_access` | bool | hide these from typical users (e.g. v3 alpha) |
| `token_cost_factor` | number | |
| `max_characters_request_free_user` | int | |
| `max_characters_request_subscribed_user` | int | |
| `maximum_text_length_per_request` | int | |
| `model_rates` | object | `{ character_cost_multiplier, cost_discount_multiplier }` |
| `concurrency_group` | string | |

**Implication for pako-tts UI**: one call to `GET /v1/models` provides both the model list AND per-model supported language codes. No separate "languages" endpoint needed.

## TTS body parameters not currently exposed

Per `POST /v1/text-to-speech/{voice_id}`:

| Param | Description |
|---|---|
| `language_code` | ISO 639-1 code to enforce a language. **No published whitelist of supporting models** — docs only state: *"If the model does not support provided language code, an error will be returned."* Surface the server error rather than gating client-side. |
| `seed` | Integer 0..4294967295 — deterministic output. |
| `previous_text` / `next_text` | Provide context for prosody continuity when concatenating multiple generations. |
| `previous_request_ids` / `next_request_ids` | Same purpose as above but by request ID (max 3 entries each). |
| `apply_text_normalization` | `auto` (default) / `on` / `off`. Controls whether numbers and abbreviations are spelled out. |
| `apply_language_text_normalization` | Boolean. Currently Japanese-only. **Heavily increases request latency.** |
| `pronunciation_dictionary_locators` | Array of `{ pronunciation_dictionary_id, version_id }`, up to 3 entries, applied in order. |
| `voice_settings.speed` | See bug note above — domain field exists, just isn't wired through. |

## Output format options

Per ElevenLabs docs (capabilities/text-to-speech), available `output_format` values include:

- **MP3**: `mp3_22050_32`, `mp3_44100_32`, `mp3_44100_64`, `mp3_44100_96`, `mp3_44100_128`, `mp3_44100_192`
- **PCM (S16LE)**: `pcm_8000`, `pcm_16000`, `pcm_22050`, `pcm_24000`, `pcm_44100`, `pcm_48000`
- **μ-law**: `ulaw_8000` (telephony)
- **A-law**: `alaw_8000` (telephony)
- **Opus**: `opus_48000_32`, `opus_48000_64`, `opus_48000_96`, `opus_48000_128`, `opus_48000_192`

Higher-quality variants are paid-tier only. This repo currently exposes only `mp3_22050_32` and `pcm_22050`.

## Models known to exist (not exhaustive)

From the docs and changelog:

- `eleven_multilingual_v2` — current default; high quality, multilingual
- `eleven_turbo_v2_5` — faster, supports `language_code`
- `eleven_flash_v2_5` — fastest, lowest quality
- `eleven_v3` — alpha, requires `requires_alpha_access`
- `eleven_multilingual_v1` — legacy multilingual
- `eleven_monolingual_v1` — legacy English-only (was the old default before 2025-05-19)
- `eleven_multilingual_ttv_v2` — voice-design (text-to-voice), not TTS
- `eleven_english_sts_v2` — voice changer (speech-to-speech), not TTS

Always rely on `GET /v1/models` rather than hardcoding this list — it changes.

## Endpoints we don't use today

| Endpoint | Purpose |
|---|---|
| `POST /v1/text-to-speech/{voice_id}/stream` | Streaming chunked audio (lower TTFB) |
| `POST /v1/text-to-speech/{voice_id}/with-timestamps` | Returns char/word timing alongside audio |
| `POST /v1/text-to-speech/{voice_id}/stream/with-timestamps` | Both streaming and timestamps |
| `POST /v1/sound-generation` | Generate sound effects from a prompt |
| `POST /v1/voice-changer/{voice_id}` | Voice conversion (audio in → audio out) |
| `GET /v1/models` | List models (the basis for the planned `/api/v1/providers/{name}/models` endpoint) |
| `GET /v1/voices` | List voices (already wired) |

## Other notes

- Default `model_id` for TTS endpoints was changed from `eleven_monolingual_v1` to `eleven_multilingual_v2` on 2025-05-19 (per ElevenLabs changelog).
- The `xi-api-key` header is the auth mechanism for all REST calls.
- Free tier has reduced concurrency and per-request character limits — see each model's `max_characters_request_free_user` / `max_characters_request_subscribed_user`.

## Sources

- ElevenLabs docs: https://elevenlabs.io/docs/api-reference
- Models endpoint: https://elevenlabs.io/docs/api-reference/models/list
- TTS convert endpoint: https://elevenlabs.io/docs/api-reference/text-to-speech/convert
- Capabilities/text-to-speech (formats, languages): https://elevenlabs.io/docs/capabilities/text-to-speech
- Changelog 2025-05-19 (default model change): per `/elevenlabs/elevenlabs-docs` via context7
