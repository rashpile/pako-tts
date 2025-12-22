# Implementation Plan: Multi-Provider Architecture with Local TTS Support

**Branch**: `002-local-tts-provider` | **Date**: 2025-12-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-local-tts-provider/spec.md`

## Summary

Refactor the pako-tts application from a single hardcoded ElevenLabs provider to a multi-provider architecture where providers are configured via config.yaml. Add a new "selfhosted" provider type that connects to any REST API TTS service (specifically targeting the local TTS service at localhost:7021). The architecture uses a provider registry pattern with factory-based instantiation keyed by provider `type`.

## Technical Context

**Language/Version**: Go 1.23+
**Primary Dependencies**: Chi router (v5.1.0), Viper (config), Zap (logging), UUID
**Storage**: Filesystem (audio_cache directory)
**Testing**: Go standard testing with testify
**Target Platform**: Linux server, Docker container
**Project Type**: Single API server
**Performance Goals**: Match existing ElevenLabs provider performance; local provider latency dependent on local TTS service
**Constraints**: Must maintain backwards compatibility with existing API; config-driven provider initialization
**Scale/Scope**: Support 2-10 concurrent providers; primary use case is 2 providers (ElevenLabs + local)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Interface-First Design | ✅ PASS | TTSProvider interface already exists; will extend with ProviderRegistry interface |
| II. Ports & Adapters | ✅ PASS | Provider adapters in `internal/provider/`; domain interfaces in `internal/domain/` |
| III. Test-First Mindset | ✅ PASS | Contract tests for TTSProvider; integration tests for config loading |
| IV. Simplicity & YAGNI | ✅ PASS | Multi-provider justified by explicit user requirement; factory pattern is minimal abstraction |
| V. Code Reuse & Discovery | ✅ PASS | Selfhosted provider reuses domain types; shared HTTP client utilities |
| VI. Quality & Performance | ✅ PASS | golangci-lint required; benchmarks for synthesis path |

**Gate Status**: PASS - Proceeding to Phase 0

## Project Structure

### Documentation (this feature)

```text
specs/002-local-tts-provider/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/
└── server/
    └── main.go              # Entry point - MODIFY for provider registry

internal/
├── api/
│   ├── handlers/
│   │   ├── tts.go           # MODIFY - accept provider param
│   │   ├── providers.go     # EXISTS - may need updates
│   │   └── health.go        # MODIFY - report all providers
│   └── routes.go            # May need updates
├── domain/
│   ├── provider.go          # MODIFY - add ProviderRegistry interface
│   └── voice.go             # Unchanged
├── provider/
│   ├── elevenlabs/          # EXISTS - MODIFY for config-based init
│   │   ├── provider.go
│   │   ├── client.go
│   │   └── voices.go
│   ├── selfhosted/          # NEW - self-hosted provider
│   │   ├── provider.go      # TTSProvider implementation
│   │   ├── client.go        # HTTP client for local TTS API
│   │   └── config.go        # Provider-specific config
│   └── registry/            # NEW - provider registry
│       ├── registry.go      # Provider factory and registry
│       └── config.go        # Config parsing for providers
├── queue/                   # Unchanged
└── storage/                 # Unchanged

pkg/
└── config/
    └── config.go            # MODIFY - add providers section

tests/
├── contract/                # NEW - provider contract tests
├── integration/             # Provider integration tests
└── unit/                    # Unit tests
```

**Structure Decision**: Single project structure maintained. New packages added under `internal/provider/` for registry and selfhosted provider.

## Complexity Tracking

> No constitution violations requiring justification.

| Decision | Rationale | Alternative Considered |
|----------|-----------|------------------------|
| Provider Registry pattern | Required for multi-provider support per spec | Direct provider instantiation in main.go would not scale |
| Factory function map | Minimal abstraction for type-based instantiation | Reflection-based registration adds complexity |