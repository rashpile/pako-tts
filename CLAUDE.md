# pako-tts Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-12-03

## Active Technologies
- Go 1.23+ + Chi router (v5.1.0), Viper (config), Zap (logging), UUID
- ffmpeg (runtime: PCM→MP3 transcoding via subprocess in `internal/audio/transcode/`)
- Filesystem (audio_cache directory)

## Project Structure

```text
internal/
  api/         — HTTP handlers, middleware, router
  audio/
    transcode/ — PCM→WAV (stdlib) and PCM→MP3 (ffmpeg subprocess)
  domain/      — shared types (TTSProvider interface, VoiceSettings, Voice, Model, ...)
  provider/
    elevenlabs/
    gemini/
    selfhosted/
    registry/  — factory registration and provider lookup
  ui/          — embedded browser UI
cmd/server/    — main entrypoint, OpenAPI spec
pkg/config/    — Viper-based config loading
```

## Commands

```bash
make fmt    # gofmt
make lint   # golangci-lint
make test   # go test -v -race ./...
make build  # produces bin/pako-tts
make run    # run server locally
```

## Code Style

Go 1.23+: Follow standard conventions

## Architecture Notes

- Provider VoiceSettings contract: providers silently ignore fields they don't support (e.g. Gemini ignores stability/speed; ElevenLabs ignores style_instructions). No Capabilities() interface — dumb pass-through pattern (v1).
- VoiceSettings.StyleInstructions is `string` not `*string`; empty == unset. Deliberate divergence from pointer-typed numeric fields.
- loadProvidersConfig uses a manual map decoder, NOT mapstructure struct binding. New provider config fields MUST be added to both the struct tag AND the manual getString/getInt call in loadProvidersConfig (pkg/config/config.go). Adding only the struct tag will silently produce zero values from YAML config.

<!-- MANUAL ADDITIONS START -->

## Where to find project info

Before searching the codebase, check these directories — they often have the answer:

- **`./docs/`** — general project documentation (README context, provider docs like `docs/elevenlabs.md`).
- **`./docs/research/`** — research notes about external APIs and design investigations (e.g. `docs/research/research-elevenlab.md` for the ElevenLabs API surface, schemas, and parameters not yet exposed). Read these before re-researching the same topic.
- **`./docs/todo/*.md`** — backlog of pending tasks and ideas. Useful when picking up unfinished work or checking whether something is already planned.
- **`./docs/plans/`** — active implementation plans. **`./docs/plans/completed/`** — archived plans (read these for historical context on how features were built).

<!-- MANUAL ADDITIONS END -->
