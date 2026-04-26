# pako-tts Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-12-03

## Active Technologies
- Go 1.23+ + Chi router (v5.1.0), Viper (config), Zap (logging), UUID (002-local-tts-provider)
- Filesystem (audio_cache directory) (002-local-tts-provider)

- Go 1.23 (latest stable) + Chi router (v5.1.0), Viper (config), Zap (logging), UUID (001-tts-api-wrapper)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go 1.23 (latest stable)

## Code Style

Go 1.23 (latest stable): Follow standard conventions

## Recent Changes
- 002-local-tts-provider: Added Go 1.23+ + Chi router (v5.1.0), Viper (config), Zap (logging), UUID

- 001-tts-api-wrapper: Added Go 1.23 (latest stable) + Chi router (v5.1.0), Viper (config), Zap (logging), UUID

<!-- MANUAL ADDITIONS START -->

## Where to find project info

Before searching the codebase, check these directories — they often have the answer:

- **`./docs/`** — general project documentation (README context, provider docs like `docs/elevenlabs.md`).
- **`./docs/research/`** — research notes about external APIs and design investigations (e.g. `docs/research/research-elevenlab.md` for the ElevenLabs API surface, schemas, and parameters not yet exposed). Read these before re-researching the same topic.
- **`./docs/todo/*.md`** — backlog of pending tasks and ideas. Useful when picking up unfinished work or checking whether something is already planned.
- **`./docs/plans/`** — active implementation plans. **`./docs/plans/completed/`** — archived plans (read these for historical context on how features were built).

<!-- MANUAL ADDITIONS END -->
