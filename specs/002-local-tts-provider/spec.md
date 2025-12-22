# Feature Specification: Multi-Provider Architecture with Local TTS Support

**Feature Branch**: `002-local-tts-provider`
**Created**: 2025-12-22
**Status**: Draft
**Input**: User description: "Add local TTS provider"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Configure Multiple TTS Providers (Priority: P1)

An administrator wants to configure multiple TTS providers in config.yaml so the application can route requests to different services. Each provider has its own configuration (name, base URL, endpoints, credentials, concurrency limits) and implements a common interface.

**Why this priority**: This is the foundational architecture change required before any new provider can be added. Without multi-provider support, the system remains limited to a single hardcoded provider.

**Independent Test**: Can be fully tested by configuring two providers in config.yaml and verifying both are loaded and available at startup.

**Acceptance Scenarios**:

1. **Given** config.yaml contains multiple provider configurations, **When** the application starts, **Then** all configured providers are initialized and available for use
2. **Given** a provider is configured with custom properties (base URL, endpoints, credentials), **When** the application loads the config, **Then** each provider receives its specific configuration
3. **Given** config.yaml specifies a default provider, **When** the application starts, **Then** that provider is used for requests that don't specify a provider

---

### User Story 2 - Use Default Provider for TTS Requests (Priority: P1)

A user wants to send TTS requests without specifying a provider, and have the system use the configured default provider automatically.

**Why this priority**: Most users will rely on the default provider. This is essential for backwards compatibility and simple usage.

**Independent Test**: Can be fully tested by sending a TTS request without a provider parameter and verifying the default provider handles it.

**Acceptance Scenarios**:

1. **Given** a default provider is configured, **When** a user sends a TTS request without specifying a provider, **Then** the system uses the default provider
2. **Given** a default provider is configured, **When** a user sends a TTS request specifying a different provider, **Then** the system uses the specified provider

---

### User Story 3 - List All Configured Providers (Priority: P2)

A user wants to see all available TTS providers so they can choose which one to use for their requests.

**Why this priority**: Users need to discover available providers before they can explicitly select one. Secondary to basic synthesis functionality.

**Independent Test**: Can be fully tested by calling the providers list API and verifying all configured providers are returned.

**Acceptance Scenarios**:

1. **Given** multiple providers are configured, **When** a user calls the providers list endpoint, **Then** the system returns all configured providers with their status
2. **Given** a provider is marked as default, **When** a user queries the provider list, **Then** the default provider is indicated in the response

---

### User Story 4 - Generate Speech Using Self-Hosted Provider (Priority: P2)

A user wants to generate TTS audio using a self-hosted TTS service configured as one of the providers. This enables offline operation, cost savings, and data privacy.

**Why this priority**: This is the specific use case that motivated the refactoring. Requires multi-provider architecture to be in place first.

**Independent Test**: Can be fully tested by configuring a self-hosted provider and sending a TTS request specifying that provider.

**Acceptance Scenarios**:

1. **Given** a self-hosted provider is configured with its base URL and endpoints, **When** a user sends a TTS request specifying that provider, **Then** the system calls the self-hosted service and returns the generated audio
2. **Given** the self-hosted provider is configured, **When** a user requests available voices, **Then** the system queries the self-hosted service's voices endpoint and returns the results

---

### User Story 5 - Check Provider Health Status (Priority: P3)

A user wants to verify which providers are running and accessible through the application's health check endpoint.

**Why this priority**: Health monitoring is important for operations but requires providers to be functional first.

**Independent Test**: Can be fully tested by calling the health endpoint and verifying each provider's status is correctly reported.

**Acceptance Scenarios**:

1. **Given** multiple providers are configured, **When** a user checks the application health, **Then** each provider's availability status is reported
2. **Given** a provider becomes unavailable, **When** a user checks health, **Then** that provider shows as unavailable while others remain available

---

### Edge Cases

- What happens when config.yaml has no providers configured? The application should fail to start with a clear error message
- What happens when the default provider name doesn't match any configured provider? The application should fail to start with a clear error message
- What happens when a user requests a provider that doesn't exist? The system should return a clear error indicating the provider is not configured
- How does the system handle a provider with invalid configuration (e.g., malformed URL)? The provider should be marked as unavailable with logged errors
- What happens when all providers are unavailable? The system should return appropriate error responses for TTS requests

## Requirements *(mandatory)*

### Functional Requirements

**Multi-Provider Architecture:**
- **FR-001**: System MUST load provider configurations from config.yaml at startup
- **FR-002**: System MUST support configuring multiple providers, each with a unique name
- **FR-003**: System MUST use the `type` field in each provider config to determine which implementation to instantiate (e.g., `elevenlabs`, `selfhosted`)
- **FR-004**: System MUST initialize each configured provider using a common TTSProvider interface
- **FR-005**: System MUST support a default provider setting in config.yaml
- **FR-006**: System MUST use the default provider when no provider is specified in a request
- **FR-007**: System MUST allow each provider to have custom configuration properties (base URL, endpoints, credentials, etc.)
- **FR-008**: Each provider implementation MUST be in its own package

**Provider API:**
- **FR-009**: System MUST expose an API endpoint to list all configured providers
- **FR-010**: System MUST indicate which provider is the default in the provider list response
- **FR-011**: System MUST report each provider's availability status

**Self-Hosted Provider:**
- **FR-012**: System MUST support a self-hosted provider type that connects to any REST API TTS service
- **FR-013**: Self-hosted provider MUST support configurable TTS endpoint path
- **FR-014**: Self-hosted provider MUST support configurable voices endpoint path
- **FR-015**: Self-hosted provider MUST support both MP3 and WAV output formats
- **FR-016**: Self-hosted provider MUST respect configured concurrency limits
- **FR-017**: Self-hosted provider MUST handle connection failures and timeouts gracefully

### Key Entities

- **Provider Configuration**: Settings for a TTS provider in config.yaml; includes provider name, type, base URL, endpoint paths, credentials (if needed), timeout settings, concurrency limits
- **TTSProvider Interface**: Common interface all providers implement; provides synthesis, voice listing, health check, and status methods
- **Provider Registry**: Collection of initialized providers loaded from configuration; tracks default provider
- **Self-Hosted Provider**: Provider implementation for generic REST API TTS services; configurable endpoints and request/response mapping

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: System successfully loads and initializes multiple providers from config.yaml
- **SC-002**: Users can send TTS requests without specifying a provider and have the default provider handle them
- **SC-003**: Users can explicitly select any configured provider for their TTS requests
- **SC-004**: Provider list API returns all configured providers with correct status and default indicator
- **SC-005**: Self-hosted provider successfully generates audio from a configured REST API TTS service
- **SC-006**: System handles provider unavailability gracefully with appropriate error messages
- **SC-007**: Adding a new provider type requires only implementing the TTSProvider interface in a new package

## Clarifications

### Session 2025-12-22

- Q: What identifier name does the local provider use? → A: Provider name is configurable via config.yaml
- Q: Does the local service have a voices endpoint? → A: Voices endpoint path is configurable in provider config
- Q: How are multiple providers supported? → A: Refactor to read provider list from config.yaml, each provider implements common interface, each in separate package
- Q: How is the default provider determined? → A: config.yaml contains default provider name setting
- Q: What is the self-hosted provider? → A: One type of custom provider for REST API TTS services; later can add opensource/commercial provider implementations
- Q: How does the system identify provider type? → A: `type` field in config (e.g., `type: elevenlabs`, `type: selfhosted`)

## Assumptions

- The existing TTSProvider interface is suitable for all provider types (or can be extended as needed)
- Each provider type (elevenlabs, self-hosted, future providers) will be in its own package under internal/provider/
- Provider configuration structure in config.yaml will be flexible enough to accommodate different provider types with varying configuration needs
- The ElevenLabs provider will be migrated to use the new config-based initialization