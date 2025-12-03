# Tasks: TTS API Wrapper

**Input**: Design documents from `/specs/001-tts-api-wrapper/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/openapi.yaml

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Exact file paths included in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and Go module setup

- [x] T001 Initialize Go module with `go mod init github.com/pako-tts/server`
- [x] T002 [P] Create project directory structure per plan.md (cmd/, internal/, pkg/, tests/)
- [x] T003 [P] Create Makefile with build, test, lint, run targets
- [x] T004 [P] Create Dockerfile for containerized deployment
- [x] T005 [P] Create .env.example with all configuration variables

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

### Domain Interfaces (Ports)

- [x] T006 Define TTSProvider interface in `internal/domain/provider.go`
- [x] T007 [P] Define JobQueue interface in `internal/domain/queue.go`
- [x] T008 [P] Define AudioStorage interface in `internal/domain/storage.go`
- [x] T009 [P] Define Job entity and JobStatus enum in `internal/domain/job.go`
- [x] T010 [P] Define VoiceSettings and Voice types in `internal/domain/voice.go`
- [x] T011 [P] Define API error types in `internal/domain/errors.go`

### Configuration & Logging

- [x] T012 Implement Viper-based configuration in `pkg/config/config.go`
- [x] T013 [P] Setup Zap structured logging in `pkg/config/logger.go`

### HTTP Foundation

- [x] T014 Setup Chi router with middleware stack in `internal/api/routes.go`
- [x] T015 [P] Implement request logging middleware in `internal/api/middleware/logging.go`
- [x] T016 [P] Implement error response middleware in `internal/api/middleware/error.go`

### Core Adapters

- [x] T017 Implement ElevenLabs HTTP client in `internal/provider/elevenlabs/client.go`
- [x] T018 Implement ElevenLabs TTSProvider adapter in `internal/provider/elevenlabs/provider.go`
- [x] T019 [P] Define voice mapping constants in `internal/provider/elevenlabs/voices.go`
- [x] T020 Implement in-memory JobQueue in `internal/queue/memory/queue.go`
- [x] T021 [P] Implement background worker pool in `internal/queue/memory/worker.go`
- [x] T022 Implement filesystem AudioStorage in `internal/storage/filesystem/storage.go`

### Server Entry Point

- [x] T023 Create main.go with dependency wiring in `cmd/server/main.go`

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Synchronous TTS (Priority: P1) MVP

**Goal**: Direct audio response for short text (< 5,000 chars) via `POST /api/v1/tts`

**Independent Test**: Submit short text, receive playable MP3/WAV directly in response

### Implementation for User Story 1

- [x] T024 [US1] Implement TTSRequest validation in `internal/api/handlers/tts.go`
- [x] T025 [US1] Implement POST /api/v1/tts handler in `internal/api/handlers/tts.go`
- [x] T026 [US1] Add 30-second timeout handling for sync requests
- [x] T027 [US1] Add text length validation (max 5,000 chars) with 413 response
- [x] T028 [US1] Register /api/v1/tts route in `internal/api/routes.go`

**Checkpoint**: User Story 1 complete - sync TTS functional

---

## Phase 4: User Story 2 - Async Job Submission (Priority: P2)

**Goal**: Queue-based processing for any text length via job API

**Independent Test**: Submit text, receive job_id, poll status, download audio result

### Implementation for User Story 2

- [x] T029 [US2] Implement JobCreateRequest validation in `internal/api/handlers/jobs.go`
- [x] T030 [US2] Implement POST /api/v1/jobs handler (job submission) in `internal/api/handlers/jobs.go`
- [x] T031 [US2] Implement GET /api/v1/jobs/{job_id} handler (status) in `internal/api/handlers/jobs.go`
- [x] T032 [US2] Implement GET /api/v1/jobs/{job_id}/result handler (download) in `internal/api/handlers/jobs.go`
- [x] T033 [US2] Add job result expiration check (24h) with 410 Gone response
- [x] T034 [US2] Add 425 Too Early response for incomplete job result requests
- [x] T035 [US2] Register job routes in `internal/api/routes.go`

**Checkpoint**: User Story 2 complete - async job workflow functional

---

## Phase 5: User Story 3 - Track Job Progress (Priority: P3)

**Goal**: Progress percentage and estimated completion time for jobs

**Independent Test**: Submit job, poll status, verify progress_percentage updates

### Implementation for User Story 3

- [x] T036 [US3] Add progress tracking to worker in `internal/queue/memory/worker.go`
- [x] T037 [US3] Implement estimated_completion_at calculation based on text length
- [x] T038 [US3] Update JobStatusResponse to include progress fields in `internal/api/handlers/jobs.go`
- [x] T039 [US3] Add error_message field for failed jobs

**Checkpoint**: User Story 3 complete - progress tracking functional

---

## Phase 6: User Story 4 - Providers and Health (Priority: P4)

**Goal**: Health monitoring and provider information endpoints

**Independent Test**: Call /api/v1/health and /api/v1/providers, verify response structure

### Implementation for User Story 4

- [x] T040 [US4] Implement GET /api/v1/health handler in `internal/api/handlers/health.go`
- [x] T041 [US4] Add provider availability check to health response
- [x] T042 [US4] Add active_jobs and max_concurrent to health response
- [x] T043 [US4] Implement GET /api/v1/providers handler in `internal/api/handlers/providers.go`
- [x] T044 [US4] Register health and providers routes in `internal/api/routes.go`

**Checkpoint**: User Story 4 complete - operational endpoints functional

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Quality improvements and deployment readiness

- [x] T045 [P] Add request ID to all log entries and responses
- [x] T046 [P] Implement graceful shutdown in `cmd/server/main.go`
- [x] T047 [P] Add result cleanup goroutine (24h expiration) in `internal/storage/filesystem/storage.go`
- [x] T048 Validate all endpoints against OpenAPI spec in `contracts/openapi.yaml`
- [x] T049 Run quickstart.md examples and verify all work

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Stories (Phases 3-6)**: All depend on Foundational phase completion
  - US1 (P1): Can start immediately after Phase 2
  - US2 (P2): Can start immediately after Phase 2 (parallel with US1)
  - US3 (P3): Depends on US2 (extends job status)
  - US4 (P4): Can start immediately after Phase 2 (parallel with US1/US2)
- **Polish (Phase 7)**: Depends on all user stories being complete

### Task Dependencies Within Phases

**Phase 2 Critical Path**:
```
T006 (Provider interface) → T017 (ElevenLabs client) → T018 (Provider adapter)
T007 (Queue interface) → T020 (Memory queue) → T021 (Worker pool)
T008 (Storage interface) → T022 (Filesystem storage)
T012 (Config) → T023 (main.go)
T014 (Router) → T023 (main.go)
```

**User Story Dependencies**:
```
US1 (sync TTS): Requires T018 (Provider), T014 (Router)
US2 (async jobs): Requires T018 (Provider), T020 (Queue), T022 (Storage), T014 (Router)
US3 (progress): Requires US2 completion (extends job handlers)
US4 (health): Requires T018 (Provider), T014 (Router)
```

### Parallel Opportunities

**Phase 1**: T002, T003, T004, T005 can all run in parallel after T001

**Phase 2 Parallel Groups**:
- Group A: T007, T008, T009, T010, T011 (all independent of each other)
- Group B: T013, T015, T016 (independent utilities)
- Group C: T019 (independent voice mapping)

**Phase 3+**: User Stories 1, 2, and 4 can run in parallel once Phase 2 completes

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1 (Sync TTS)
4. **VALIDATE**: Test with curl examples from quickstart.md
5. Deploy if ready

### Full Feature Delivery

1. Setup + Foundational → Foundation ready
2. US1 (Sync) + US2 (Async) + US4 (Health) in parallel → Core API complete
3. US3 (Progress) → Enhanced UX
4. Polish → Production ready

---

## Notes

- [P] = Tasks that can run in parallel (different files, no dependencies)
- [USn] = Task belongs to User Story n
- All paths are relative to repository root
- Go 1.23 required (latest stable)
- Commit after each task or logical group
- Validate against OpenAPI spec in contracts/openapi.yaml