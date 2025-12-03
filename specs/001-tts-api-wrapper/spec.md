# Feature Specification: TTS API Wrapper with Job Queue

**Feature Branch**: `001-tts-api-wrapper`
**Created**: 2025-12-03
**Status**: Draft
**Input**: User description: "API wrapper for text2speech services, support queue/jobs processing."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Synchronous Text-to-Speech (Priority: P1)

As an application developer, I want to submit short text and receive audio directly in the response, so that I can quickly convert small texts without polling for job status.

**Why this priority**: This is the simplest use case - immediate audio for short texts. Essential for real-time applications like chatbots or notifications.

**Independent Test**: Can be fully tested by submitting a short text string and receiving a playable audio file directly in the response.

**Acceptance Scenarios**:

1. **Given** a valid request with short text (under 5,000 characters), **When** the user calls the sync endpoint, **Then** the system returns audio directly in the response
2. **Given** a request with text exceeding the sync limit, **When** the user calls the sync endpoint, **Then** the system returns an error suggesting to use the async job endpoint
3. **Given** a request with an invalid voice ID, **When** the user submits the request, **Then** the system returns a validation error

---

### User Story 2 - Async Job Submission for Long Text (Priority: P2)

As an application developer, I want to submit large text documents for asynchronous processing, so that I can handle long-form content without request timeouts.

**Why this priority**: Essential for production use where text length exceeds real-time processing limits. Enables audiobook-style content generation.

**Independent Test**: Can be tested by submitting a large text document, receiving a job ID, and later retrieving the completed audio file.

**Acceptance Scenarios**:

1. **Given** text content of any length, **When** the user submits a job request, **Then** the system returns a job ID and status (queued) immediately
2. **Given** a valid job ID, **When** the user queries job status, **Then** the system returns current status (queued, processing, completed, failed)
3. **Given** a completed job, **When** the user requests the result, **Then** the system provides the audio file for download

---

### User Story 3 - Track Job Progress (Priority: P3)

As an application developer, I want to monitor job progress and estimated completion time, so that I can provide feedback to my users.

**Why this priority**: User experience for long-running jobs - knowing progress and ETA reduces uncertainty.

**Independent Test**: Can be tested by submitting a job and polling status to verify progress updates.

**Acceptance Scenarios**:

1. **Given** a job in processing state, **When** the user queries status, **Then** the system returns progress percentage (0-100)
2. **Given** a job in processing state, **When** the user queries status, **Then** the system returns estimated completion time (if available)
3. **Given** a failed job, **When** the user queries status, **Then** the system provides an error message explaining the failure

---

### User Story 4 - View Providers and Health (Priority: P4)

As an application developer, I want to see available TTS providers and service health, so that I can choose providers and integrate with monitoring.

**Why this priority**: Operational feature for production deployments. Enables provider selection and health monitoring.

**Independent Test**: Can be tested by calling health and providers endpoints and verifying response structure.

**Acceptance Scenarios**:

1. **Given** a request for available providers, **When** the user queries the providers endpoint, **Then** the system returns provider list with availability status
2. **Given** a request specifying a provider in job submission, **When** the user submits, **Then** the system uses the specified provider
3. **Given** the service is running, **When** the user calls health endpoint, **Then** the system returns health status and provider capacity

---

### Edge Cases

- What happens when the upstream TTS provider is unavailable? System returns a service unavailable error with provider status
- What happens when the audio result expires before download? System returns 410 Gone with expiration info
- How does the system handle malformed or invalid text input? System returns 422 with validation error details
- What happens when a user queries a job ID that does not exist? System returns 404 Not Found
- What happens when result is requested for a still-processing job? System returns 425 Too Early

## Requirements *(mandatory)*

### Functional Requirements

**Sync API (for small text)**
- **FR-001**: System MUST provide sync endpoint `POST /api/v1/tts` that returns audio directly for text under 5,000 characters
- **FR-002**: System MUST return audio in MP3 format (default) or WAV format based on request

**Async Job API (matching transcriber patterns)**
- **FR-003**: System MUST accept job submission via `POST /api/v1/jobs` returning job_id, status, created_at
- **FR-004**: System MUST provide job status via `GET /api/v1/jobs/{job_id}` with progress_percentage and estimated_completion_at
- **FR-005**: System MUST provide audio result via `GET /api/v1/jobs/{job_id}/result`
- **FR-006**: System MUST support job status values: queued, processing, completed, failed

**Service Endpoints (matching transcriber patterns)**
- **FR-007**: System MUST provide health check via `GET /api/v1/health` with provider status and capacity
- **FR-008**: System MUST list providers via `GET /api/v1/providers` with name, type, max_concurrent, is_default, is_available

**Provider & Voice**
- **FR-009**: System MUST support multiple TTS providers with ElevenLabs as the primary; architecture MUST allow adding providers without core changes
- **FR-010**: System MUST allow voice selection (voice_id, language, speed) in both sync and async requests

**Error Handling & Retention**
- **FR-011**: System MUST retain job results for 24 hours, then return 410 Gone for expired results
- **FR-012**: System MUST return 425 Too Early for result requests on incomplete jobs
- **FR-013**: System MUST provide clear error messages with validation details (422 for validation errors)

### Key Entities

- **Job**: Represents a TTS synthesis request. Attributes: job_id (UUID), status, provider_name, created_at, started_at, completed_at, progress_percentage, estimated_completion_at, error_message (if failed)
- **Provider**: Represents an external TTS service. Attributes: name, type, max_concurrent, is_default, is_available
- **Voice Configuration**: Synthesis parameters: voice_id, language, speed, output_format

## Assumptions

- API follows same patterns as Pako Transcriber (`/api/v1/` prefix, similar response structures)
- Default audio format is MP3 if not specified
- Default voice is determined by provider's default if not specified
- Job results are stored temporarily and cleaned up after 24 hours
- No authentication required initially (same as transcriber - can be added later)
- Sync endpoint has 30-second timeout; longer texts must use async job API

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Sync endpoint returns audio within 5 seconds for text under 1,000 characters
- **SC-002**: Job submission returns job ID within 1 second
- **SC-003**: System successfully processes 95% of submissions without errors
- **SC-004**: System supports at least 100 concurrent job submissions
- **SC-005**: Failed jobs provide actionable error messages in 100% of cases
- **SC-006**: Health endpoint responds within 500ms with accurate provider status