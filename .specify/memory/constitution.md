<!--
Sync Impact Report
==================
Version change: N/A → 1.0.0 (Initial ratification)

Added Principles:
- I. Interface-First Design
- II. Ports & Adapters Architecture
- III. Test-First Mindset
- IV. Simplicity & YAGNI
- V. Code Reuse & Discovery
- VI. Quality & Performance Standards

Added Sections:
- Go Development Standards
- Development Workflow
- Governance

Templates Status:
- plan-template.md: ✅ Compatible (Constitution Check section exists)
- spec-template.md: ✅ Compatible (no constitution-specific references)
- tasks-template.md: ✅ Compatible (test-first patterns align)

Follow-up TODOs: None
-->

# pako-tts Constitution

## Core Principles

### I. Interface-First Design

Every new module MUST begin by defining **interfaces and data contracts** that describe its responsibilities and interactions before any implementation code is written.

- Define domain-level interfaces (ports) before writing implementation code
- Interfaces MUST specify input/output contracts clearly
- Data contracts MUST be documented and versioned
- Implementation details MUST NOT leak into interface definitions

**Rationale**: Clear interfaces enable parallel development, easier testing, and reduce coupling between components.

### II. Ports & Adapters Architecture

For real systems (databases, external APIs, message queues), create **adapter layers** that implement domain ports.

- Domain logic MUST depend only on interfaces (ports), never on concrete implementations
- Adapters MUST implement domain interfaces and handle external system specifics
- Use Go modules for dependency management
- Each adapter MUST be independently replaceable without affecting domain logic

**Rationale**: This architecture enables testing with mocks, swapping implementations, and isolating external system changes from business logic.

### III. Test-First Mindset

Apply test-first development pragmatically: write tests before implementation where it adds value.

- Tests SHOULD be written before implementation for non-trivial logic
- Red-Green-Refactor cycle is the preferred workflow
- Contract tests MUST exist for all public interfaces
- Integration tests MUST cover critical paths and external system interactions
- Test coverage is a tool, not a goal — focus on testing behavior, not lines

**Rationale**: Tests written first clarify requirements and catch design issues early. Pragmatism over dogma means not testing trivial code.

### IV. Simplicity & YAGNI

No abstractions without proven need. Start simple and extract interfaces only when multiple implementations or clear extension points are identified.

- MUST NOT design for hypothetical future requirements
- Abstractions require justification in code review
- Prefer duplication over premature abstraction
- Three similar implementations MAY justify an abstraction; one or two do not
- Complexity MUST be justified in the Complexity Tracking section of plans

**Rationale**: Premature abstraction creates maintenance burden and cognitive overhead without delivering value.

### V. Code Reuse & Discovery

Prevent duplicate implementations by ensuring developers check existing code before creating new functionality.

- Before implementing new functionality, MUST search for existing implementations
- Shared utilities MUST be discoverable (documented, well-named, in expected locations)
- Duplication across modules MUST be flagged in code review
- When duplicates are found, refactor to shared code if both usages are stable

**Rationale**: Duplicate code increases maintenance cost and introduces inconsistency risks.

### VI. Quality & Performance Standards

Code quality, testing standards, and performance requirements are non-negotiable.

- All code MUST pass linting (golangci-lint) before merge
- All code MUST be formatted with gofmt/goimports
- Error handling MUST be explicit — no ignored errors without documented justification
- Performance-critical paths MUST have benchmarks
- API response times MUST meet documented SLAs

**Rationale**: Consistent quality standards reduce bugs, improve maintainability, and ensure reliable user experience.

## Go Development Standards

- **Go Modules**: All dependencies MUST be managed via Go modules
- **Error Handling**: Use explicit error returns; wrap errors with context using `fmt.Errorf` or `errors.Join`
- **Logging**: Use structured logging; include request IDs for traceability
- **Configuration**: Use environment variables or config files; never hardcode secrets
- **Documentation**: Public APIs MUST have godoc comments
- **Naming**: Follow Go naming conventions (MixedCaps, not underscores)

## Development Workflow

1. **Before Implementation**: Read existing code; define interfaces; write tests (for non-trivial code)
2. **During Implementation**: Follow red-green-refactor; commit frequently; keep changes focused
3. **Before PR**: Run linting; ensure all tests pass; update documentation if needed
4. **Code Review**: Verify constitution compliance; check for duplication; validate complexity justification
5. **After Merge**: Monitor for regressions; update shared documentation if patterns change

## Governance

This constitution supersedes all other development practices for the pako-tts project.

- All PRs MUST verify compliance with these principles
- Amendments require: documented rationale, team review, migration plan for breaking changes
- Constitution violations MUST be resolved before merge or explicitly justified in the Complexity Tracking section
- Version updates follow semantic versioning: MAJOR (principle removal/redefinition), MINOR (principle addition), PATCH (clarification)

**Version**: 1.0.0 | **Ratified**: 2025-12-03 | **Last Amended**: 2025-12-03