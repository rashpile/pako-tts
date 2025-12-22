# Tasks: Multi-Provider Architecture with Local TTS Support

**Input**: Design documents from `/specs/002-local-tts-provider/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Test tasks are included as per constitution principle III (Test-First Mindset).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md, this project uses Go standard layout:
- `cmd/server/` - Application entry point
- `internal/` - Private application code
- `pkg/` - Public library code
- `tests/` - Test files

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and new package structure

- [x] T001 Create directory structure for new packages: `internal/provider/registry/` and `internal/provider/selfhosted/`
- [x] T002 [P] Create config.yaml template file with providers section at repository root
- [x] T003 [P] Update .gitignore to exclude local config files with secrets

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Add ProviderRegistry interface to `internal/domain/provider.go`
- [x] T005 Add ProvidersConfig and ProviderConfig structs to `pkg/config/config.go`
- [x] T006 Add config loading for providers section with Viper in `pkg/config/config.go`
- [x] T007 Add config validation for providers (at least one provider, default exists) in `pkg/config/config.go`
- [x] T008 [P] Add domain errors (ErrProviderNotFound, ErrProviderUnavailable) to `internal/domain/errors.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Configure Multiple TTS Providers (Priority: P1) 🎯 MVP

**Goal**: Enable loading and initializing multiple TTS providers from config.yaml at startup

**Independent Test**: Configure two providers in config.yaml and verify both are loaded and available at startup

### Tests for User Story 1

- [ ] T009 [P] [US1] Create unit test for registry factory pattern in `tests/unit/registry_test.go`
- [ ] T010 [P] [US1] Create unit test for config parsing with multiple providers in `tests/unit/config_providers_test.go`

### Implementation for User Story 1

- [x] T011 [P] [US1] Create ProviderFactory type and factory map in `internal/provider/registry/factory.go`
- [x] T012 [P] [US1] Create registry struct implementing ProviderRegistry in `internal/provider/registry/registry.go`
- [x] T013 [US1] Implement NewRegistry constructor that loads providers from config in `internal/provider/registry/registry.go`
- [x] T014 [US1] Add NewProviderFromConfig constructor to ElevenLabs provider in `internal/provider/elevenlabs/provider.go`
- [x] T015 [US1] Register ElevenLabs factory in `internal/provider/registry/factory.go`
- [x] T016 [US1] Update main.go to use registry instead of single provider in `cmd/server/main.go`
- [x] T017 [US1] Update RouterDeps to accept ProviderRegistry in `internal/api/routes.go`

**Checkpoint**: User Story 1 complete - application loads multiple providers from config

---

## Phase 4: User Story 2 - Use Default Provider for TTS Requests (Priority: P1)

**Goal**: Route TTS requests to default provider when none specified, allow explicit provider selection

**Independent Test**: Send TTS request without provider param and verify default handles it; send with explicit provider and verify correct routing

### Tests for User Story 2

- [ ] T018 [P] [US2] Create unit test for default provider selection in `tests/unit/tts_handler_test.go`
- [ ] T019 [P] [US2] Create unit test for explicit provider selection in `tests/unit/tts_handler_test.go`

### Implementation for User Story 2

- [x] T020 [US2] Add `provider` field to TTSRequest struct in `internal/api/handlers/tts.go`
- [x] T021 [US2] Update TTS handler to get provider from registry based on request in `internal/api/handlers/tts.go`
- [x] T022 [US2] Add error handling for unknown provider in `internal/api/handlers/tts.go`
- [x] T023 [US2] Update job creation handler to accept provider parameter in `internal/api/handlers/jobs.go`

**Checkpoint**: User Stories 1 AND 2 complete - TTS requests route to correct provider

---

## Phase 5: User Story 3 - List All Configured Providers (Priority: P2)

**Goal**: Expose API endpoint to list all providers with their status and default indicator

**Independent Test**: Call GET /api/v1/providers and verify all configured providers are returned with correct status

### Tests for User Story 3

- [ ] T024 [P] [US3] Create unit test for providers list endpoint in `tests/unit/providers_handler_test.go`

### Implementation for User Story 3

- [x] T025 [US3] Implement ListInfo method in registry in `internal/provider/registry/registry.go`
- [x] T026 [US3] Update providers handler to use registry.ListInfo in `internal/api/handlers/providers.go`
- [x] T027 [US3] Add default_provider field to providers list response in `internal/api/handlers/providers.go`

**Checkpoint**: User Story 3 complete - providers list API functional

---

## Phase 6: User Story 4 - Generate Speech Using Self-Hosted Provider (Priority: P2)

**Goal**: Implement selfhosted provider type that connects to local TTS REST API service

**Independent Test**: Configure selfhosted provider, send TTS request specifying it, verify audio returned from local service

### Tests for User Story 4

- [ ] T028 [P] [US4] Create unit test for selfhosted provider with mock HTTP server in `tests/unit/selfhosted_provider_test.go`
- [ ] T029 [P] [US4] Create unit test for model-to-voice mapping in `tests/unit/selfhosted_voices_test.go`

### Implementation for User Story 4

- [x] T030 [P] [US4] Create selfhosted config struct in `internal/provider/selfhosted/config.go` (config in pkg/config/config.go)
- [x] T031 [P] [US4] Create HTTP client for local TTS API in `internal/provider/selfhosted/client.go`
- [x] T032 [US4] Create request/response types for local TTS API in `internal/provider/selfhosted/types.go`
- [x] T033 [US4] Implement selfhosted provider struct in `internal/provider/selfhosted/provider.go`
- [x] T034 [US4] Implement Synthesize method calling local TTS endpoint in `internal/provider/selfhosted/provider.go`
- [x] T035 [US4] Implement ListVoices method mapping models to voices in `internal/provider/selfhosted/provider.go`
- [x] T036 [US4] Implement IsAvailable using health endpoint in `internal/provider/selfhosted/provider.go`
- [x] T037 [US4] Add NewProviderFromConfig constructor in `internal/provider/selfhosted/provider.go`
- [x] T038 [US4] Register selfhosted factory in `internal/provider/registry/factory.go`
- [x] T039 [US4] Add concurrency control with atomic counter in `internal/provider/selfhosted/provider.go`

**Checkpoint**: User Story 4 complete - selfhosted provider fully functional

---

## Phase 7: User Story 5 - Check Provider Health Status (Priority: P3)

**Goal**: Report health status of all providers in health endpoint

**Independent Test**: Call health endpoint and verify each provider's status is correctly reported

### Tests for User Story 5

- [ ] T040 [P] [US5] Create unit test for health endpoint with multiple providers in `tests/unit/health_handler_test.go`

### Implementation for User Story 5

- [x] T041 [US5] Update health handler to iterate all providers in `internal/api/handlers/health.go`
- [x] T042 [US5] Add per-provider status to health response in `internal/api/handlers/health.go`
- [x] T043 [US5] Add overall health status logic (healthy if any provider available) in `internal/api/handlers/health.go`

**Checkpoint**: User Story 5 complete - health endpoint shows all provider statuses

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T044 [P] Update OpenAPI spec with provider parameter in `cmd/server/openapi.yaml`
- [x] T045 [P] Add example config.yaml with both providers to repository
- [ ] T046 [P] Update README.md with multi-provider configuration section
- [x] T047 Run golangci-lint and fix any issues
- [x] T048 Run all tests and verify passing
- [ ] T049 Validate against quickstart.md scenarios

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - US1 and US2 are both P1 priority and foundational for the others
  - US3 and US4 are P2 and can proceed after US1/US2
  - US5 is P3 and can proceed after US1/US2
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational - Foundation for all other stories
- **User Story 2 (P1)**: Depends on US1 (needs registry) - Can proceed immediately after US1
- **User Story 3 (P2)**: Depends on US1 (needs registry.ListInfo)
- **User Story 4 (P2)**: Depends on US1 (needs registry factory pattern)
- **User Story 5 (P3)**: Depends on US1 (needs registry.List)

### Within Each User Story

- Tests SHOULD be written first (per constitution Test-First Mindset)
- Config/Types before implementation
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel
- Tests within a story marked [P] can run in parallel
- US3, US4, US5 can proceed in parallel after US2 completes (if team capacity allows)
- All Polish tasks marked [P] can run in parallel

---

## Parallel Example: User Story 4

```bash
# Launch all tests for User Story 4 together:
Task: "Create unit test for selfhosted provider with mock HTTP server in tests/unit/selfhosted_provider_test.go"
Task: "Create unit test for model-to-voice mapping in tests/unit/selfhosted_voices_test.go"

# Launch config and client in parallel:
Task: "Create selfhosted config struct in internal/provider/selfhosted/config.go"
Task: "Create HTTP client for local TTS API in internal/provider/selfhosted/client.go"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Configure Multiple Providers)
4. Complete Phase 4: User Story 2 (Default Provider)
5. **STOP and VALIDATE**: Test with ElevenLabs only - verify default provider routing works
6. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 + US2 → Test with existing ElevenLabs → Deploy (MVP with config-based providers!)
3. Add US4 (Selfhosted) → Test with local TTS service → Deploy (Local TTS support!)
4. Add US3 (List Providers) → Test API discovery → Deploy
5. Add US5 (Health) → Test monitoring → Deploy

### Single Developer Flow

```
Phase 1 → Phase 2 → US1 → US2 → US4 → US3 → US5 → Phase 8
```

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Write tests first, ensure they fail before implementing (Test-First Mindset)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- US1 and US2 together form the MVP - everything else builds on them