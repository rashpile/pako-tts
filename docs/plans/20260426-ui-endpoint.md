# UI Endpoint + Voices Endpoint

## Overview

Add a simple browser-based UI at `/ui/` so users can try the TTS API without writing curl commands or wiring up a client. The UI mirrors the look-and-feel of `pako-transcriber`'s `/static/index.html` (single file, inline CSS/JS, ~640px card layout, blue accent `#2563eb`, ES5 JavaScript).

The UI uses the **synchronous** `POST /api/v1/tts` endpoint: text in, audio bytes out, played via an HTML5 `<audio>` element.

To support a friendly **Voice** dropdown that updates when the provider changes, this plan also adds a new endpoint `GET /api/v1/providers/{name}/voices` backed by the existing `TTSProvider.ListVoices(ctx)` method (already implemented by both the `selfhosted` and `elevenlabs` providers).

The OpenAPI spec at `cmd/server/openapi.yaml` is updated to document the new endpoint and the `Voice` / `VoicesListResponse` schemas.

## Context (from discovery)

- **Project**: Go 1.23 HTTP server (Chi router, Zap logger), TTS API wrapper with multi-provider support (ElevenLabs + selfhosted)
- **Entry / wiring**: `cmd/server/main.go` builds a `RouterDeps`, calls `api.NewRouter(...)`. OpenAPI spec is `//go:embed openapi.yaml`
- **Routes**: `internal/api/routes.go` mounts everything under `/api/v1/...`. Existing `r.Get("/providers", providersHandler.ListProviders)` is the closest analog for the new voices route
- **Handlers package**: `internal/api/handlers/` (health, jobs, openapi, providers, tts). `providers.go` is small and uses `middleware.WriteJSON` for success responses and `middleware.WriteError(domain.APIError)` for failures. **No `providers_test.go` exists yet** — this plan creates one
- **Domain types ready to reuse**:
  - `domain.Voice` with JSON tags `voice_id`, `name`, `provider`, `language`, `gender`, `preview_url` (`internal/domain/voice.go:13`)
  - `domain.TTSProvider.ListVoices(ctx)` (`internal/domain/provider.go:20`)
  - `domain.ErrProviderNotFound` for 404 responses
- **Sync TTS endpoint** (`POST /api/v1/tts`) returns audio bytes with `Content-Type` from `result.ContentType` — directly usable as a blob in JS
- **Reference UI**: `pako-transcriber/app/static/index.html` — single file, inline `<style>` + `<script>`, plain ES5, optimistic light theme

## Development Approach

- **testing approach**: Regular (code first, then tests within the same task)
- complete each task fully before moving to the next
- make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
  - tests are not optional — they are a required part of the checklist
  - write unit tests for new functions/methods
  - write unit tests for modified functions/methods
  - tests cover both success and error scenarios
- **CRITICAL: all tests must pass before starting next task** — no exceptions
- **CRITICAL: update this plan file when scope changes during implementation**
- run tests after each change: `go test ./...`
- maintain backward compatibility (additive changes only — no existing routes change)

## Testing Strategy

- **unit tests**: required for every task
  - `internal/api/handlers/providers_test.go` (new) — table-driven tests for `ListVoices` handler covering: success, unknown provider (404), provider error (500/502)
  - `internal/ui/ui_test.go` (new) — verify embedded HTML is served at `/ui/` with `Content-Type: text/html` and non-empty body
- **e2e tests**: project has no UI-based e2e harness today; manual smoke test in browser is part of acceptance verification (see Post-Completion)
- **mock provider**: extend `MockProvider.ListVoicesFunc` in tests as needed (already supports it)

## Progress Tracking

- mark completed items with `[x]` immediately when done
- add newly discovered tasks with `+` prefix
- document issues/blockers with `WARN` prefix
- update plan if implementation deviates from original scope

## Solution Overview

1. **New `internal/ui` package** containing `index.html` (embedded via `//go:embed`) and a tiny handler that serves it. Why: same pattern as the existing OpenAPI byte-slice embed, single-binary deployment, no Docker volume changes.
2. **New `ListVoices` handler method** on `ProvidersHandler` reading `{name}` from the URL, looking up the provider via `registry.Get`, calling `provider.ListVoices(ctx)`, and writing `{provider, voices}`. Why: leverages existing interface, minimal surface.
3. **Two new routes** wired in `internal/api/routes.go`:
   - `GET /ui/` (and `GET /ui` redirect) → UI handler
   - `GET /api/v1/providers/{name}/voices` → new `ListVoices` handler
4. **OpenAPI spec update**: add the new path entry and `Voice` + `VoicesListResponse` schemas mirroring the existing style.

## Technical Details

### UI handler (`internal/ui/ui.go`)
```go
package ui

import (
    _ "embed"
    "net/http"
)

//go:embed index.html
var indexHTML []byte

// Handler serves the embedded UI at /ui/.
type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.Header().Set("Cache-Control", "no-cache")
    w.WriteHeader(http.StatusOK)
    w.Write(indexHTML) //nolint:errcheck
}
```

### Voices handler (`internal/api/handlers/providers.go`, additive)
```go
type VoicesListResponse struct {
    Provider string         `json:"provider"`
    Voices   []domain.Voice `json:"voices"`
}

func (h *ProvidersHandler) ListVoices(w http.ResponseWriter, r *http.Request) {
    name := chi.URLParam(r, "name")
    provider, err := h.registry.Get(name)
    if err != nil {
        middleware.WriteError(w, domain.ErrProviderNotFound.WithMessage("Provider '"+name+"' not found"))
        return
    }
    voices, err := provider.ListVoices(r.Context())
    if err != nil {
        h.logger.Error("ListVoices failed", zap.String("provider", name), zap.Error(err))
        // Match tts.go's synthesis-error handling: upstream/provider failures are 503.
        middleware.WriteError(w, domain.ErrProviderUnavailable.WithMessage(err.Error()))
        return
    }
    middleware.WriteJSON(w, http.StatusOK, VoicesListResponse{Provider: name, Voices: voices})
}
```

### Route wiring (`internal/api/routes.go`)
Add before the existing API route group:
```go
uiHandler := ui.NewHandler()
r.Get("/ui", func(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/ui/", http.StatusMovedPermanently)
})
r.Get("/ui/", uiHandler.ServeHTTP)
```
Inside the existing `r.Route("/api/v1", ...)`:
```go
r.Get("/providers/{name}/voices", providersHandler.ListVoices)
```

### HTML structure (`internal/ui/index.html`)
Single file, ES5, no external assets. Sections:
- `<style>` — copied/adapted from transcriber: 640px container, white cards, blue button
- Form card: `<textarea>` (text), `<select>` (provider), `<select>` (voice — empty until provider chosen), `<select>` (output_format: mp3/wav), submit button, error message
- Result card (hidden initially): `<audio controls>`, download button, size/format text

### JS flow (in `<script>`)
1. `loadProviders()` on page load → `fetch('/api/v1/providers')` → populate provider `<select>`, mark default selected, then call `loadVoices(default)`
2. `loadVoices(name)` → `fetch('/api/v1/providers/' + name + '/voices')` → populate voice `<select>`. On failure or empty list: single option `value=""` labeled "Default voice"
3. Provider `change` listener → `loadVoices(newName)`
4. Form submit → `fetch('/api/v1/tts', {method: 'POST', body: JSON.stringify({text, voice_id, provider, output_format})})` →
   - on 2xx: `res.blob()` → `URL.createObjectURL(blob)` → set `<audio src>`, show result card, show download button with `download="tts-<timestamp>.<format>"`
   - on non-2xx: parse JSON error, show in error msg
5. Button shows "Synthesizing…" + disabled while in flight

### OpenAPI updates (`cmd/server/openapi.yaml`)
Add path under `paths:` (after `/api/v1/providers`):
```yaml
  /api/v1/providers/{name}/voices:
    get:
      tags: [Providers]
      summary: List Voices for Provider
      description: Returns voices available for the named provider.
      operationId: listProviderVoices
      parameters:
        - name: name
          in: path
          required: true
          description: Provider identifier
          schema: { type: string }
      responses:
        "200":
          description: Voice list
          content:
            application/json:
              schema: { $ref: "#/components/schemas/VoicesListResponse" }
        "404":
          description: Provider not found
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }
        "503":
          description: Provider unavailable (upstream call failed)
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ErrorResponse" }
```
Add schemas under `components.schemas:`:
```yaml
    Voice:
      type: object
      required: [voice_id, name, provider]
      properties:
        voice_id: { type: string }
        name:     { type: string }
        provider: { type: string }
        language: { type: string }
        gender:   { type: string }
        preview_url: { type: string, format: uri }

    VoicesListResponse:
      type: object
      required: [provider, voices]
      properties:
        provider: { type: string }
        voices:
          type: array
          items: { $ref: "#/components/schemas/Voice" }
```

## What Goes Where

- **Implementation Steps** (`[ ]` checkboxes): Go code, embedded HTML, OpenAPI yaml, unit tests, route wiring
- **Post-Completion** (no checkboxes): manual browser smoke test, screenshots, optional Docker rebuild

## Implementation Steps

### Task 1: Add ListVoices handler + route + OpenAPI

**Files:**
- Modify: `internal/api/handlers/providers.go` (add `"github.com/go-chi/chi/v5"` import for `chi.URLParam`)
- Create: `internal/api/handlers/providers_test.go`
- Modify: `internal/api/routes.go`
- Modify: `cmd/server/openapi.yaml`

- [x] add `VoicesListResponse` struct and `ListVoices(w, r)` method to `internal/api/handlers/providers.go` per Technical Details (handler returns 503 via `ErrProviderUnavailable` on upstream failures, matching `tts.go:125`)
- [x] register `r.Get("/providers/{name}/voices", providersHandler.ListVoices)` inside the existing `r.Route("/api/v1", ...)` block in `internal/api/routes.go`
- [x] add `/api/v1/providers/{name}/voices` path (with 200 / 404 / 503 responses) and `Voice` + `VoicesListResponse` schemas to `cmd/server/openapi.yaml` per Technical Details
- [x] create `internal/api/handlers/providers_test.go` with table-driven tests for the new `ListVoices` handler only (keep PR scope tight; do NOT backfill `ListProviders` tests). Cover: success (returns voices for known provider), unknown provider (404 with `PROVIDER_NOT_FOUND`), provider returns error (503 with `PROVIDER_UNAVAILABLE`). Use `MockProvider.ListVoicesFunc` and a test registry similar to `jobs_test.go`
- [x] run `go test ./internal/api/handlers/...` — must pass before next task

### Task 2: Add internal/ui package with embedded HTML and route wiring

**Files:**
- Create: `internal/ui/ui.go`
- Create: `internal/ui/index.html`
- Create: `internal/ui/ui_test.go`
- Modify: `internal/api/routes.go`

- [ ] create `internal/ui/ui.go` with `//go:embed index.html`, `Handler` struct, and `ServeHTTP` writing `Content-Type: text/html; charset=utf-8` and the embedded bytes (per Technical Details)
- [ ] create `internal/ui/index.html` as a single-file UI:
  - `<style>` block adapted from `pako-transcriber/app/static/index.html` (640px container, white card, blue `#2563eb` button)
  - form: `<textarea name="text">` (rows=5, required), `<select id="provider-select">`, `<select id="voice-select">`, `<select id="format-select">` with `mp3`/`wav`, submit button, error message div
  - hidden result card: `<audio controls id="audio-player">`, download `<a>`, size/format span
  - ES5 `<script>`: `loadProviders()` → `loadVoices(name)` on init and provider change; form submit → `POST /api/v1/tts` JSON body → handle blob success and JSON error
- [ ] wire UI routes in `internal/api/routes.go`: `r.Get("/ui/", uiHandler.ServeHTTP)` plus an inline redirect for the trailing-slash-less form: `r.Get("/ui", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/ui/", http.StatusMovedPermanently) })`. Place above the `/api/v1` route group. Add `"github.com/pako-tts/server/internal/ui"` and `"net/http"` imports as needed
- [ ] create `internal/ui/ui_test.go` with `TestHandler_ServeHTTP` verifying: 200 status, `Content-Type` starts with `text/html`, body is non-empty, body contains recognizable markers (e.g. `<title>Pako TTS</title>`, `id="provider-select"`) AND the API URL constants (`/api/v1/tts`, `/api/v1/providers/`) so a future refactor that breaks JS endpoint paths is caught
- [ ] run `go test ./...` — must pass before next task

### Task 3: Verify acceptance criteria

- [ ] start the server locally (`go run ./cmd/server`) and confirm `GET /ui/` returns the HTML page in a browser
- [ ] in the browser: select a provider, confirm voices dropdown populates, type sample text, click Synthesize, confirm `<audio>` plays the result
- [ ] confirm `GET /api/v1/providers/<name>/voices` returns expected JSON shape via `curl`
- [ ] confirm OpenAPI spec at `/openapi.json` includes the new path and schemas
- [ ] run full test suite: `go test ./...` and `go vet ./...`
- [ ] verify no existing tests broke

### Task 4: [Final] Update documentation and finalize

**Files:**
- Modify: `README.md`

- [ ] add a short "Web UI" section to `README.md` mentioning `/ui/` is available for trying the API in a browser
- [ ] move this plan to `docs/plans/completed/`

## Post-Completion

*Items requiring manual intervention or external systems — informational only*

**Manual verification:**
- browser smoke test of the synthesis flow with at least one configured provider (ElevenLabs or selfhosted) end-to-end
- confirm `audio/mpeg` and `audio/wav` both render correctly in the `<audio>` element across Chrome/Safari
- check console for unexpected errors or CORS issues (CORS is currently `AllowedOrigins: ["*"]`, so same-origin should be trivial)

**Known limitations (informational):**
- The server's `DefaultVoiceID` is global (one default for the whole server). When the user picks a non-default provider and the voices dropdown is empty (provider unreachable / no voices), the UI sends an empty `voice_id`, which the TTS handler resolves to `DefaultVoiceID` — that ID may not be valid for the selected provider and yield `INVALID_VOICE`. Acceptable for a "try the API" UI; surface the error message clearly. Future work could send no `provider` field when the user hasn't explicitly switched away from the default.

**Optional follow-ups (not in scope):**
- expose voice settings sliders (stability, similarity_boost, style, speed) in the UI
- caching for the voices endpoint if traffic is heavier than expected
- swap the simple `<audio>` element for a richer player with waveform / scrubbing
- per-provider default voice resolution server-side
