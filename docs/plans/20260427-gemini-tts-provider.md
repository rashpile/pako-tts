# Add Gemini Flash TTS Provider

## Overview

Add a third TTS provider (`gemini`) using Google's `gemini-3.1-flash-tts-preview` model. Mirrors the structure of `internal/provider/elevenlabs/` for compatibility with the existing `domain.TTSProvider` interface, while exposing one Gemini-unique feature (free-text style instructions) via a new `VoiceSettings.StyleInstructions` field.

The provider must transparently honor the existing API contract (`mp3` and `wav` output) even though Gemini natively returns only PCM. This requires a new shared `internal/audio/transcode/` package: stdlib WAV-header wrap for `wav`, ffmpeg subprocess for `mp3`. ffmpeg is added to the runtime Docker image.

## Context (from discovery)

- Project: pako-tts — Go 1.25 TTS API server (Chi router, Viper, Zap). Two existing providers: `elevenlabs` (cloud), `selfhosted` (HTTP backend).
- Test command: `make test` → `go test -v -race ./...`. No e2e suite.
- Provider interface: `internal/domain/provider.go` — `Synthesize`, `ListVoices`, `ListModels`, `IsAvailable`, `MaxConcurrent`, `ActiveJobs`, `Status`, `Name`.
- Provider settings: `internal/domain/voice.go` — `VoiceSettings` already has `Stability`, `SimilarityBoost`, `Style *float64` (numeric, ElevenLabs-only), `UseSpeakerBoost`, `Speed`. Adding `StyleInstructions string` for Gemini's free-text style.
- Registry: `internal/provider/registry/factory.go` registers built-in providers via `init()` → `RegisterFactory(name, factory)`. Add `gemini` here.
- Config: `pkg/config/config.go` → `ProviderConfig` struct holds per-provider fields. Adding `DefaultStyle string`.
- Dockerfile: alpine 3.19 runtime, `apk --no-cache add ca-certificates` on line 25 — extend to add `ffmpeg`.
- Reference test script (proves API works): `/Users/pkoptilin/.openclaw/workspace/scripts/gemini_romanian_audiobook.py`. We adapt with: header-based auth (`x-goog-api-key`), direct response path access, no chunking in v1.
- Reference docs: <https://ai.google.dev/gemini-api/docs/speech-generation>, <https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-tts-preview>.
- Recent precedent: language-selection branch (mem #31, merged 2026-04-27) added top-level `LanguageCode` and updated all providers consistently. Same shape for `StyleInstructions` — providers ignore fields they don't honor (mem #28).

## Development Approach

- **testing approach**: Regular (code first, then tests in the same task)
- complete each task fully before moving to the next
- make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** — unit tests for new and modified functions, success + error paths
- **CRITICAL: all tests must pass before starting next task** — run `make test`
- **CRITICAL: update this plan file when scope changes during implementation**
- maintain backward compatibility — adding optional fields, no breaking changes to JSON API
- atomic commits per task — each commit builds and passes `make test` and `make lint`

## Testing Strategy

- **unit tests**: required for every task
  - new package `internal/audio/transcode/` — tests for WAV wrap byte correctness, ffmpeg PCM→MP3 happy path + missing-binary error path. CI workflow `.github/workflows/ci.yml` runs on `ubuntu-latest` which does NOT include ffmpeg by default — Task 1 must add an `apt-get install -y ffmpeg` step to the test job, OR the happy-path test must `t.Skip` when `exec.LookPath("ffmpeg")` fails. Pick the install path: tests run for real, no silent skips.
  - `internal/provider/gemini/` — table-driven tests with `httptest.Server` mocking the Gemini endpoint, mirroring `internal/provider/elevenlabs/provider_test.go`
  - `internal/domain/voice_test.go` — extend `VoiceSettings.Merge` test if Merge is updated
  - `pkg/config/config_test.go` — extend to cover `DefaultStyle` field load
- **integration**: registry routing test — confirm `Get("gemini")` returns the provider after factory registration
- **e2e**: project has no Playwright/Cypress setup; manual smoke via the running server is captured under Post-Completion

## Progress Tracking

- mark completed items with `[x]` immediately when done
- add newly discovered tasks with ➕ prefix
- document issues/blockers with ⚠️ prefix
- update plan if implementation deviates from original scope

## Solution Overview

**Architecture**: provider mirrors `elevenlabs/`. New `internal/audio/transcode/` package isolates format conversion so the provider stays thin. Synthesis flow:

```
SynthesisRequest
  ↓ (build prompt: language directive + style line + user text)
Gemini API (POST generateContent, returns base64 PCM)
  ↓ (decode base64)
PCM bytes
  ↓ (transcode based on req.OutputFormat)
  ├─ "wav" → transcode.PCMToWAV (stdlib, ~44-byte RIFF header prefix)
  └─ "mp3" → transcode.PCMToMP3 (ffmpeg subprocess)
SynthesisResult
```

**Key design decisions** (locked from brainstorm):
- Style: `VoiceSettings.StyleInstructions string` (free text). Per-request value wins over `ProviderConfig.DefaultStyle` fallback.
- Other providers ignore `StyleInstructions` silently — no `Capabilities()` interface in v1.
- Voices: 30 prebuilt names hardcoded with `Language: ""` (language-agnostic). Full list in Reference Data below.
- Model: single entry `gemini-3.1-flash-tts-preview`, advertises full 70+ ISO 639-1 list.
- `LanguageCode` injected into prompt as `"Speak in {name}.\n"` — Gemini has no `language_code` API field.
- Multi-speaker: out of scope (v2 follow-up).
- Token limits: surface upstream errors as-is (mem #12 dumb-client pattern).
- Auth: `x-goog-api-key` header.

## Reference Data

**30 prebuilt voice names** (verbatim, source: <https://ai.google.dev/gemini-api/docs/speech-generation>):

```
Zephyr, Puck, Charon, Kore, Fenrir, Leda, Orus, Aoede, Callirrhoe, Autonoe,
Enceladus, Iapetus, Umbriel, Algieba, Despina, Erinome, Algenib, Rasalgethi,
Laomedeia, Achernar, Alnilam, Schedar, Gacrux, Pulcherrima, Achird,
Zubenelgenubi, Vindemiatrix, Sadachbia, Sadaltager, Sulafat
```

**ISO 639-1 language list (~70 codes)**: source-of-truth is the model page <https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-tts-preview>. Implementer copies the listed codes verbatim into `supportedLanguages` and `isoToName` (e.g., `"en"` → `"English"`, `"ro"` → `"Romanian"`, `"zh"` → `"Chinese (Mandarin)"`, …). If the model page lists names like "Chinese (Mandarin)" map them to the standard ISO code (`zh`); preserve regional variants in the human-readable name only.

**API base URL**: `https://generativelanguage.googleapis.com/v1beta`

**ffmpeg invocation** (PCM 24kHz/16/mono → MP3 128kbps via stdin/stdout):
```
ffmpeg -f s16le -ar 24000 -ac 1 -i pipe:0 -f mp3 -b:a 128k pipe:1
```

## Technical Details

**Gemini request body**:
```json
{
  "contents": [{"parts": [{"text": "<composed prompt>"}]}],
  "generationConfig": {
    "responseModalities": ["AUDIO"],
    "speechConfig": {
      "voiceConfig": {"prebuiltVoiceConfig": {"voiceName": "Despina"}}
    }
  }
}
```

**Response shape**: `candidates[0].content.parts[0].inlineData.data` (base64 PCM string).

**Prompt composition**:
```go
parts := []string{}
if req.LanguageCode != "" {
    if name, ok := isoToName[req.LanguageCode]; ok {
        parts = append(parts, "Speak in "+name+".")
    }
}
style := ""
if req.Settings != nil { style = req.Settings.StyleInstructions }
if style == "" { style = p.defaultStyle }
if style != "" {
    parts = append(parts, "Style: "+style+".")
}
prompt := req.Text
if len(parts) > 0 {
    prompt = strings.Join(parts, "\n") + "\n\n" + req.Text
}
```

**WAV header (44 bytes, little-endian, mono 16-bit 24kHz)** — canonical layout per <http://soundfile.sapp.org/doc/WaveFormat/>:

| Offset | Size | Field             | Value (for our use)              |
|-------:|-----:|-------------------|-----------------------------------|
|      0 |    4 | ChunkID           | `"RIFF"` (ASCII)                  |
|      4 |    4 | ChunkSize         | `36 + dataSize` (uint32, LE)      |
|      8 |    4 | Format            | `"WAVE"` (ASCII)                  |
|     12 |    4 | Subchunk1ID       | `"fmt "` (ASCII, trailing space)  |
|     16 |    4 | Subchunk1Size     | `16` (PCM, uint32, LE)            |
|     20 |    2 | AudioFormat       | `1` (PCM, uint16, LE)             |
|     22 |    2 | NumChannels       | `1` (mono, uint16, LE)            |
|     24 |    4 | SampleRate        | `24000` (uint32, LE)              |
|     28 |    4 | ByteRate          | `sampleRate * channels * bps / 8` |
|     32 |    2 | BlockAlign        | `channels * bps / 8` (= 2)        |
|     34 |    2 | BitsPerSample     | `16` (uint16, LE)                 |
|     36 |    4 | Subchunk2ID       | `"data"` (ASCII)                  |
|     40 |    4 | Subchunk2Size     | `dataSize` (= len(pcm), uint32)   |
|     44 |    N | Data              | raw PCM bytes                     |

## What Goes Where

- **Implementation Steps** (`[ ]` checkboxes): code, tests, docs, Dockerfile.
- **Post-Completion** (no checkboxes): manual smoke test against real Gemini API; UI textarea for style_instructions deferred as v2.

## Implementation Steps

### Task 1: Shared audio transcode package + ffmpeg in Docker + CI

**Files:**
- Create: `internal/audio/transcode/wav.go`
- Create: `internal/audio/transcode/transcode.go`
- Create: `internal/audio/transcode/transcode_test.go`
- Modify: `Dockerfile`
- Modify: `.github/workflows/ci.yml`

- [x] create `internal/audio/transcode/wav.go` with `PCMToWAV(pcm []byte, sampleRate, channels, bitsPerSample int) []byte` — writes 44-byte RIFF header + PCM data using `encoding/binary` (little-endian) per the byte-offset table in Technical Details above
- [x] create `internal/audio/transcode/transcode.go` with `PCMToMP3(ctx context.Context, pcm []byte, sampleRate, channels int) ([]byte, error)` — shells out to `ffmpeg` via `os/exec`, pipes PCM in via stdin, reads MP3 from stdout, captures stderr for error context, respects context cancellation. Expose `var ffmpegBinary = "ffmpeg"` as a package-level variable so tests can override it without mutating the parent process's `PATH`
- [x] write tests for `PCMToWAV`: verify byte-for-byte header layout (RIFF/WAVE/fmt /data chunk markers, ChunkSize = 36+dataSize, Subchunk2Size = dataSize, sample rate, channels, bits per sample) for 1-second 24kHz mono PCM input
- [x] write tests for `PCMToMP3`: happy path (decode output frame and verify it's valid MP3 by signature `0xFF 0xFB` or `0xFF 0xFA`), context cancellation path (cancel mid-encode, expect error), missing-binary path (override `ffmpegBinary` to a non-existent path OR use `t.Setenv("PATH", t.TempDir())` — DO NOT use `os.Setenv` to avoid polluting parallel tests), structured error contains stderr context
- [x] modify `Dockerfile:25` from `RUN apk --no-cache add ca-certificates` to `RUN apk --no-cache add ca-certificates ffmpeg`
- [x] modify `.github/workflows/ci.yml` test job to install ffmpeg before `go test` runs (e.g., `- name: Install ffmpeg\n        run: sudo apt-get update && sudo apt-get install -y ffmpeg`) — required so PCMToMP3 happy-path test executes for real in CI
- [x] run `make test` — must pass before next task

### Task 2: Add `StyleInstructions` to VoiceSettings + `DefaultStyle` to ProviderConfig

**Files:**
- Modify: `internal/domain/voice.go`
- Modify: `internal/domain/voice_test.go` (or wherever Merge is tested)
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/config_test.go`

- [x] add `StyleInstructions string \`json:"style_instructions,omitempty"\`` to `domain.VoiceSettings` struct. **Note**: this is a `string` (not `*string`) — a deliberate divergence from the pointer-typed numeric fields. Empty string is treated as "unset"; the API consumer cannot signal "explicitly clear" separately from "not set", which is acceptable because there's no meaningful distinction for a free-text directive.
- [x] update `VoiceSettings.Merge` so `StyleInstructions` follows "non-empty other wins, otherwise base" (i.e., `if other.StyleInstructions != "" { result.StyleInstructions = other.StyleInstructions } else { result.StyleInstructions = v.StyleInstructions }`). Add a comment in the Merge function near this line clarifying the asymmetry vs pointer fields.
- [x] add `DefaultStyle string \`mapstructure:"default_style"\`` to `pkg/config/config.go` `ProviderConfig` struct (group with the existing per-provider fields, comment it as "For gemini")
- [x] add `DefaultStyle: getString(providerMap, "default_style"),` to the manual map decoder in `loadProvidersConfig` (around line 218-229 in `pkg/config/config.go`). The `mapstructure` tag alone is NOT enough — viper config goes through this manual decoder, not struct binding (see mem #97).
- [x] write/update tests for `VoiceSettings.Merge` covering `StyleInstructions` (both-empty, only-other-set, only-base-set, both-set → other wins)
- [x] write/update tests in `pkg/config/config_test.go` confirming `default_style` is loaded from yaml under a provider entry
- [x] run `make test` — must pass before next task

### Task 3: Gemini HTTP client + voices/languages tables

**Files:**
- Create: `internal/provider/gemini/client.go`
- Create: `internal/provider/gemini/voices.go`
- Create: `internal/provider/gemini/client_test.go`

- [x] create `internal/provider/gemini/client.go` with: `const baseURL = "https://generativelanguage.googleapis.com/v1beta"`; `Client` struct (apiKey, baseURL, httpClient with 120s timeout); `NewClient(apiKey)`, `NewClientWithBaseURL` for tests; request/response types (`TTSRequest`, `Contents`, `Parts`, `GenerationConfig`, `SpeechConfig`, `VoiceConfig`, `PrebuiltVoiceConfig`, `TTSResponse`, `Candidate`, `Content`, `Part`, `InlineData`); `GenerateAudio(ctx, model, prompt, voiceName)` returning raw PCM bytes (decodes base64 internally — use `base64.StdEncoding`; switch to `RawStdEncoding` only if Gemini ever returns unpadded); `CheckHealth(ctx)` (`GET <baseURL>/models/{defaultModelID}` — connectivity-only check, see caveat below)
- [x] auth via `x-goog-api-key` header (case is normalized by Go's `net/http` but tests should assert the exact lowercase literal `x-goog-api-key` to lock the contract); POST to `<baseURL>/models/{model}:generateContent`
- [x] **CheckHealth caveat**: the Gemini models GET endpoint is publicly readable, so `IsAvailable` is *connectivity-only* — an invalid API key may still return `true`. This is a deliberate trade-off: we avoid spending tokens on a probe request. Document this in `docs/gemini.md` (Task 8).
- [x] create `internal/provider/gemini/voices.go` with: `prebuiltVoices []domain.Voice` (30 entries from Reference Data above with `Language: ""`); `defaultModel domain.Model` (id `gemini-3.1-flash-tts-preview`, full ISO list as `Languages`); `isoToName map[string]string` (~70 entries — copy verbatim from <https://ai.google.dev/gemini-api/docs/models/gemini-3.1-flash-tts-preview>); `supportedLanguages []string` (same set as `defaultModel.Languages`, kept consistent — preferably define `supportedLanguages` once and assign to `defaultModel.Languages` so they cannot drift)
- [x] write tests for client `GenerateAudio` happy path (httptest.Server returns canned base64 PCM, verify decoded bytes match), error paths (HTTP 400 with structured error body, HTTP 500, network error, unparseable response, missing inlineData in candidates), exact header literal `x-goog-api-key` set with API key value, request body shape matches Google spec (decode and assert `contents[0].parts[0].text == prompt` and `generationConfig.speechConfig.voiceConfig.prebuiltVoiceConfig.voiceName == voiceName`)
- [x] write tests for `CheckHealth` (200 → true, 401 → false, network error → false)
- [x] run `make test` — must pass before next task

### Task 4: Gemini provider implementation

**Files:**
- Create: `internal/provider/gemini/provider.go`
- Create: `internal/provider/gemini/provider_test.go`

- [x] create `internal/provider/gemini/provider.go` with `Provider` struct (client, defaultModelID, defaultStyle, isDefault, activeJobs int32), `NewProvider`, `NewProviderFromConfig(cfg config.ProviderConfig, isDefault bool) (*Provider, error)` (requires `cfg.APIKey`, defaults `ModelID` to `gemini-3.1-flash-tts-preview`)
- [x] implement `Name()` returning `"gemini"` and `Type()` returning `"GeminiProvider"`. **Note**: `Type()` is NOT on the `domain.TTSProvider` interface — it's a concrete-struct convenience method called via type assertion (mirrors `elevenlabs.Provider.Type()`). Don't add it to mocks unless callers need it.
- [x] implement `Synthesize`: build prompt (language directive + style line + text), call `client.GenerateAudio`, route output via `switch req.OutputFormat { case "wav": ... ; default: ... // mp3 }` — mirrors `elevenlabs/provider.go:88-93`. `wav` → `transcode.PCMToWAV(pcm, 24000, 1, 16)` with content type `audio/wav`; `mp3` (and default for unknown formats) → `transcode.PCMToMP3(ctx, pcm, 24000, 1)` with content type `audio/mpeg`
- [x] implement `ListVoices` returning the static `prebuiltVoices` slice
- [x] implement `ListModels` returning `[]domain.Model{defaultModel}`
- [x] implement `IsAvailable` (delegates to `client.CheckHealth`), `MaxConcurrent` (returns `const maxConcurrent = 4` to mirror elevenlabs precedent — `cfg.MaxConcurrent` is plumbed through but not honored here, matching how `elevenlabs.NewProviderFromConfig` ignores it; revisit if rate-limit tuning becomes needed), `ActiveJobs` (atomic load), `Status`, `Info` — mirror elevenlabs shape
- [x] write tests for `Synthesize` with mocked httptest server: prompt composition cases (no lang/no style, lang only, style only, both — verify the `text` field sent upstream), output format cases (mp3 path verifies `audio/mpeg` ContentType + valid MP3 signature; wav path verifies `audio/wav` + RIFF header), per-request style overrides config default, config default used when request has empty style, error from upstream propagates
- [x] write tests for `ListVoices` (returns 30 entries, all `Language == ""`), `ListModels` (1 entry, ModelID matches, Languages non-empty)
- [x] write tests for `Info()` (returns expected `Type == "GeminiProvider"`, `Name == "gemini"`, `MaxConcurrent == 4`, `IsDefault` matches constructor flag)
- [x] write tests for `NewProviderFromConfig` (missing APIKey → error, empty ModelID → defaults to `gemini-3.1-flash-tts-preview`, DefaultStyle wired through)
- [x] run `make test` — must pass before next task

### Task 5: Register `gemini` in provider factory

**Files:**
- Modify: `internal/provider/registry/factory.go`
- Create: `internal/provider/registry/factory_test.go` (if not present)

- [x] add import `"github.com/pako-tts/server/internal/provider/gemini"` in `factory.go`
- [x] add `geminiFactory(cfg config.ProviderConfig, isDefault bool) (domain.TTSProvider, error)` returning `gemini.NewProviderFromConfig(cfg, isDefault)`
- [x] register inside `init()`: `RegisterFactory("gemini", geminiFactory)`
- [x] write tests in `factory_test.go` (or add to existing): `GetFactory("gemini")` returns non-nil; `GetFactory("bogus")` returns `(_, false)` (locks the existing contract while we're adding test coverage to this previously-untested package — see mem #14); building via the factory with a valid config returns a working provider; missing APIKey returns the expected error
- [x] run `make test` — must pass before next task

### Task 6: API schema + handler tests for `style_instructions`

**Files:**
- Modify: `cmd/server/openapi.yaml`
- Modify: `internal/api/handlers/tts_test.go` (or `jobs_test.go`)
- Possibly modify: `internal/api/handlers/mocks/provider.go`

- [x] add `style_instructions` field (type string, optional) to the `VoiceSettings` schema in `cmd/server/openapi.yaml`
- [x] **No code change expected** in `internal/api/handlers/tts.go` and `jobs.go` since `VoiceSettings` is embedded and JSON-decoded as a whole. But: inspect `internal/api/handlers/mocks/provider.go` (`MockProviderRegistry` per mem #18) — if its mock provider doesn't capture the incoming `*SynthesisRequest` for assertions, extend it as part of this task (add a `LastRequest *SynthesisRequest` field set inside `Synthesize`). This is the bit that could surprise the implementer.
- [x] add a handler-level test that posts a payload with `voice_settings.style_instructions` set, asserts the field round-trips into the `domain.SynthesisRequest` passed to the provider mock, and asserts the response is delivered correctly
- [x] run `make test` — must pass before next task

### Task 7: Verify acceptance criteria

- [ ] verify all decisions from Overview/Solution Overview are implemented (style hybrid, prompt composition, voices/model lists, language injection, output format routing, registry registration, Dockerfile)
- [ ] verify edge cases: empty `LanguageCode`, empty `StyleInstructions` with empty `DefaultStyle`, unknown ISO code (not in map → directive omitted), unknown voice (Gemini API returns error → propagated)
- [ ] run `make test` — full suite passes
- [ ] run `make lint` — passes (or document any pre-existing failures)
- [ ] run `make build` — produces `bin/pako-tts` successfully
- [ ] verify `docker build .` succeeds with the updated Dockerfile (image contains ffmpeg)

### Task 8: Documentation + move plan

**Files:**
- Create: `docs/gemini.md`
- Modify: `README.md`
- Move: `docs/plans/20260427-gemini-tts-provider.md` → `docs/plans/completed/`

- [ ] create `docs/gemini.md` (parallel to `docs/elevenlabs.md`) covering: model id, voice list (30 names), style instructions (config + per-request), language handling (auto-detect + ISO injection), audio tags pass-through (`[whispers]`, `[laughs]`, etc.), output format note (PCM native, server-side WAV/MP3 transcode), token limits, auth, sample request, **`IsAvailable` caveat** (connectivity-only — invalid API key may still report `true` because the Gemini models endpoint is publicly readable), **UI behavior note** (Gemini voices appear with blank language because they are language-agnostic — the user must set `language_code` per request to control language)
- [ ] update `README.md`: add `gemini` to providers list and provide a sample config block (api_key from env, model_id, default_style)
- [ ] mark all checkboxes complete in this plan
- [ ] move plan to `docs/plans/completed/20260427-gemini-tts-provider.md`
- [ ] run `make test` one last time — full suite green

## Post-Completion

*Items requiring manual intervention or external systems — no checkboxes, informational only.*

**Manual smoke test against real Gemini API:**
- Set `GEMINI_API_KEY` (or save to `~/.keys/GEMINI_ROMANIAN_LANG`).
- Run `bin/pako-tts` with a config that has a `gemini` provider entry.
- Curl `POST /synthesize` with body: `{"text":"Salut, ce mai faci?","voice_id":"Despina","model_id":"gemini-3.1-flash-tts-preview","language_code":"ro","output_format":"mp3","voice_settings":{"style_instructions":"warm, slightly slow, beginner-friendly"}}`
- Verify a valid MP3 is returned and plays correctly in Romanian with the requested style.
- Repeat with `output_format: "wav"` and verify WAV file is well-formed (`ffprobe` or `file <name>.wav`).

**Deferred to v2 (out of scope):**
- Multi-speaker dialogue support (`multiSpeakerVoiceConfig.speakerVoiceConfigs[]`) — adds new request shape; needs API schema change.
- UI textarea in `internal/ui/index.html` for `style_instructions` field that's only visible when the gemini provider is selected.
- Client-side text chunking for inputs exceeding the 8192-token Gemini input limit. Current behavior: surface upstream error verbatim.
- Streaming output (Gemini explicitly does not support; revisit if Google adds it).
