# Language Selection on TTS Request and UI

## Overview

Implement the only remaining item in `docs/todo/todo.md`:

- Accept an optional `language_code` (ISO 639-1, e.g. `"en"`) on `POST /api/v1/tts` and `POST /api/v1/jobs`.
- Plumb it through `domain.SynthesisRequest` → `domain.Job` (+ `NewJob`) → the worker → `elevenlabs.TTSRequest.LanguageCode`.
- ElevenLabs returns a "model does not support language" error verbatim — surface it as 503 like other provider errors.
- Add a Language `<select>` to `internal/ui/index.html`, populated from the **union** of `languages[]` across the loaded models for the selected provider. Send `language_code` on submit; leave empty to use the model's default.
- Selfhosted provider ignores the field for now.
- Update OpenAPI spec.

This change mirrors the just-merged `model_id` work (PR #4) one-for-one — same plumbing, same surfaces, same test patterns. No new architecture.

## Context (from discovery)

- **Project**: Go 1.23 HTTP server (Chi router, Zap logger), multi-provider TTS (ElevenLabs + selfhosted), single-binary deployment, embedded OpenAPI at `cmd/server/openapi.yaml`.
- **Files involved (ordered by request flow)**:
  - `internal/domain/provider.go:43` — `SynthesisRequest` (`ModelID` already present; add `LanguageCode` next to it).
  - `internal/domain/job.go:25` — `Job.ModelID` precedent; add `LanguageCode`. `NewJob(text, voiceID, modelID, providerName, outputFormat string, settings *VoiceSettings)` becomes `NewJob(text, voiceID, modelID, languageCode, providerName, outputFormat string, settings *VoiceSettings)`.
  - `internal/api/handlers/tts.go:45,118` — `TTSRequest.ModelID` precedent.
  - `internal/api/handlers/jobs.go:48,122` — `JobCreateRequest.ModelID` precedent + the single `domain.NewJob` call site.
  - `internal/queue/memory/worker.go:123` — copies `ModelID` from job to SynthesisRequest; add `LanguageCode` line.
  - `internal/provider/elevenlabs/client.go:42` — `TTSRequest.ModelID` JSON struct; add `LanguageCode string \`json:"language_code,omitempty"\``.
  - `internal/provider/elevenlabs/provider.go:77-81` — sets `ttsReq.ModelID`; add a similar (simpler — no fallback) line for `LanguageCode`.
  - `internal/provider/selfhosted/provider.go:99-103` — explicit per-todo decision: ignore `req.LanguageCode`.
  - `internal/api/handlers/mocks/provider.go` — mock provider (no change needed; it doesn't filter request fields).
  - `cmd/server/openapi.yaml:401,423` — add `language_code` to both `TTSRequest` and `JobCreateRequest` schemas.
  - `internal/ui/index.html:331-359` — `loadModels(name)` is the precedent; add a Language `<select>` populated from `union(model.languages[])`. The `<option>` building block is at line 426 for `loadVoices`.
- **Patterns observed**:
  - All optional request fields use `string \`json:"<name>,omitempty"\`` and zero-value `""` means "use default."
  - `domain.NewJob` call sites that don't care about the new field pass `""`. Verified count for this change: **32 sites** (see Task 1 callout).
  - ElevenLabs handler-side errors funnel through `domain.ErrProviderUnavailable.WithMessage(err.Error())` (`internal/api/handlers/tts.go:128` for sync; jobs handler logs and stores `ErrorMessage` on the failed job).
  - JSON tag for the new field on the wire: `language_code` (snake_case, matches `model_id`, `voice_id`, `output_format`).
- **Existing language-related code**: `internal/domain/voice.go:17` has `Voice.Language` (a per-voice label, not a request param), `domain.Model.Languages []string` (already populated by `elevenlabs.Provider.ListModels` from `/v1/models`). The UI's `loadVoices` already shows `(en)`-style labels (`internal/ui/index.html:426`) but does not surface model languages.
- **Dependencies**: no new third-party packages.
- **Reference**: `docs/research/research-elevenlab.md:56` documents the `language_code` ElevenLabs param ("If the model does not support provided language code, an error will be returned"). **No published whitelist of supporting models** — surface upstream errors rather than gating client-side.

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
- maintain backward compatibility (additive changes only — `language_code` is optional everywhere; existing clients that don't send it get today's behavior)

## Testing Strategy

- **unit tests** required for every task:
  - `internal/provider/elevenlabs/provider_test.go` — extend the existing `Synthesize` test to assert `req.LanguageCode` is forwarded into the marshalled JSON body of the upstream call. Capture the inbound HTTP body via `httptest.Server` (existing pattern).
  - `internal/api/handlers/tts_test.go` — extend `TestSynthesizeTTS_PassesModelID` (or add a sibling) to assert `req.LanguageCode` is plumbed through to the captured `*domain.SynthesisRequest`.
  - `internal/api/handlers/jobs_test.go` — extend the existing `TestSubmitJob_PassesModelID` to assert `job.LanguageCode` is set when the request body sets it.
  - `internal/domain/job_test.go` — extend `TestNewJob` to assert the new `LanguageCode` field is stored.
  - `internal/queue/memory/worker_test.go` — extend `TestWorker_PropagatesJobModelIDToSynthesisRequest` (or add a parallel test) to assert `job.LanguageCode` propagates to the captured `req.LanguageCode`.
  - `internal/provider/selfhosted/provider_test.go` — assert `Synthesize` ignores `req.LanguageCode` (no-op; the existing TTSRequest sent to upstream does not include the field).
  - `internal/ui/ui_test.go` — extend body-marker assertions to include `id="language-select"` and `'/api/v1/providers/'` already present (no new endpoint).
- **e2e tests**: project has no UI-based e2e harness today; manual smoke tests in browser are part of acceptance verification (Task 6).

## Progress Tracking

- mark completed items with `[x]` immediately when done
- add newly discovered tasks with `+` prefix
- document issues/blockers with `WARN` prefix
- update plan if implementation deviates from original scope

## Solution Overview

1. **Domain layer**: `SynthesisRequest`, `Job`, and `NewJob` gain a `LanguageCode` field/parameter (string, optional).
2. **Worker** copies `LanguageCode` from `Job` to the constructed `SynthesisRequest`.
3. **TTS + Jobs handlers** decode optional `language_code` from request bodies. The TTS handler passes it on `domain.SynthesisRequest`; the jobs handler passes it through `domain.NewJob`.
4. **ElevenLabs** `TTSRequest` adds `LanguageCode string \`json:"language_code,omitempty"\``. Provider sets `ttsReq.LanguageCode = req.LanguageCode` unconditionally — no fallback (the field is genuinely optional; the provider/model default applies when empty).
5. **Selfhosted** ignores the field (per todo). Document this with a comment in `Synthesize`.
6. **OpenAPI** documents the new optional field on both request schemas.
7. **UI** adds a Language `<select>` (next to Model). After `loadModels(name)` resolves, the union of `model.languages[]` populates the language dropdown. Submit includes `language_code` only when non-empty.

## Technical Details

### Domain — `internal/domain/provider.go`

Extend `SynthesisRequest`:
```go
type SynthesisRequest struct {
    Text         string
    VoiceID      string
    ModelID      string // optional; provider falls back to its configured default when empty
    LanguageCode string // optional; ISO 639-1 (e.g. "en"). Provider/model default when empty.
    OutputFormat string
    Settings     *VoiceSettings
}
```

### Domain — `internal/domain/job.go`

Add `LanguageCode string \`json:"language_code,omitempty"\`` to `Job`. Update `NewJob` signature:
```go
func NewJob(text, voiceID, modelID, languageCode, providerName, outputFormat string, settings *VoiceSettings) *Job
```

The new field is inserted after `modelID` to keep the optional-string cluster together at the front.

### Worker — `internal/queue/memory/worker.go:120-126`

Add one line:
```go
req := &domain.SynthesisRequest{
    Text:         job.Text,
    VoiceID:      job.VoiceID,
    ModelID:      job.ModelID,
    LanguageCode: job.LanguageCode,   // NEW
    OutputFormat: job.OutputFormat,
    Settings:     job.VoiceSettings,
}
```

### TTS handler — `internal/api/handlers/tts.go`

```go
type TTSRequest struct {
    Text          string                `json:"text"`
    VoiceID       string                `json:"voice_id,omitempty"`
    Provider      string                `json:"provider,omitempty"`
    ModelID       string                `json:"model_id,omitempty"`
    LanguageCode  string                `json:"language_code,omitempty"` // NEW
    OutputFormat  string                `json:"output_format,omitempty"`
    VoiceSettings *domain.VoiceSettings `json:"voice_settings,omitempty"`
}
```

In `SynthesizeTTS`, set `synthReq.LanguageCode = req.LanguageCode` next to the existing `synthReq.ModelID = req.ModelID`.

### Jobs handler — `internal/api/handlers/jobs.go`

Same field on `JobCreateRequest`. Pass `req.LanguageCode` through to `domain.NewJob` at the existing call site (`jobs.go:122`):
```go
job := domain.NewJob(req.Text, voiceID, req.ModelID, req.LanguageCode, providerName, outputFormat, req.VoiceSettings)
```

### ElevenLabs client — `internal/provider/elevenlabs/client.go`

Add to `TTSRequest`:
```go
type TTSRequest struct {
    Text          string            `json:"text"`
    ModelID       string            `json:"model_id"`
    LanguageCode  string            `json:"language_code,omitempty"` // NEW
    OutputFormat  string            `json:"output_format,omitempty"`
    VoiceSettings *VoiceSettingsReq `json:"voice_settings,omitempty"`
}
```

### ElevenLabs provider — `internal/provider/elevenlabs/provider.go`

In `Synthesize`, after the existing `ModelID` assignment block:
```go
if req.LanguageCode != "" {
    ttsReq.LanguageCode = req.LanguageCode
}
// (no else — empty means "let model use its default")
```

### Selfhosted provider — `internal/provider/selfhosted/provider.go`

`Synthesize`: do **not** touch `req.LanguageCode`. Add a one-line doc comment near the existing model-resolution switch noting that `language_code` is intentionally ignored for selfhosted (the local TTS API does not have a comparable parameter; future selfhosted implementations can opt-in by reading `req.LanguageCode`).

### OpenAPI — `cmd/server/openapi.yaml`

Add to both `TTSRequest` (after `model_id` at line 401) and `JobCreateRequest` (after `model_id` at line 423):
```yaml
        language_code:
          type: string
          description: ISO 639-1 language code (e.g. "en"). Forces the chosen model to render in this language. Provider/model default when omitted; some models do not support all languages and will return an upstream error.
```

### UI — `internal/ui/index.html`

**HTML — insert after the Model `<select>` block at lines 156-159, immediately before the Output Format `<label>` at line 161:**
```html
<label for="language-select">Language</label>
<select id="language-select" name="language_code">
    <option value="">Default language</option>
</select>
```

**JS additions (vanilla ES5, mirrors `loadModels` and `resetModels`):**

```js
var languageSelect = document.getElementById('language-select');

function resetLanguages(message) {
    languageSelect.innerHTML = '';
    var opt = document.createElement('option');
    opt.value = '';
    opt.textContent = message || 'Default language';
    languageSelect.appendChild(opt);
}

// Compute the union of language codes across an array of models. Sorts result
// for deterministic option ordering.
function unionModelLanguages(models) {
    var seen = {};
    for (var i = 0; i < models.length; i++) {
        var langs = models[i].languages || [];
        for (var j = 0; j < langs.length; j++) seen[langs[j]] = true;
    }
    var out = [];
    for (var k in seen) if (seen.hasOwnProperty(k)) out.push(k);
    out.sort();
    return out;
}
```

**Hook into `loadModels(name)`** — after the existing `models.forEach` that populates `modelSelect`, also populate the language select:
```js
var langs = unionModelLanguages(models);
resetLanguages('Default language');
langs.forEach(function (code) {
    var opt = document.createElement('option');
    opt.value = code;
    opt.textContent = code;
    languageSelect.appendChild(opt);
});
```

In the `loadModels` `.catch`, also call `resetLanguages('Default language')` so failed loads don't leave stale options.

In the provider `change` listener (`internal/ui/index.html:438-447`), the `else` branch (no provider selected) should also call `resetLanguages('Default language')`.

In `setFormDisabled(disabled, label)`, add `languageSelect.disabled = disabled;` so the field is locked while a request is in flight (mirrors `modelSelect.disabled`).

**Form submit body construction** (next to existing `body.model_id`):
```js
var languageCode = languageSelect.value;
if (languageCode) body.language_code = languageCode;
```

## What Goes Where

- **Implementation Steps** (`[ ]` checkboxes): Go code, embedded HTML, OpenAPI yaml, unit tests, route wiring.
- **Post-Completion** (no checkboxes): manual browser smoke test, manual `curl` verification with real ElevenLabs API.

## Implementation Steps

### Task 1: Plumb `LanguageCode` through domain, worker, and handlers

**Files:**
- Modify: `internal/domain/provider.go` (add `LanguageCode` to `SynthesisRequest`)
- Modify: `internal/domain/job.go` (add `LanguageCode` field + parameter to `NewJob`)
- Modify: `internal/domain/job_test.go` (**11** `NewJob` call sites — pass `""`; extend `TestNewJob` to assert the new field)
- Modify: `internal/queue/memory/worker.go` (copy `LanguageCode` to `SynthesisRequest`)
- Modify: `internal/queue/memory/queue_test.go` (**16** `domain.NewJob(...)` call sites — all need new `""` arg at position 4)
- Modify: `internal/queue/memory/worker_test.go` (**1** `domain.NewJob(...)` call site at line 104; extend the existing model-id propagation test for language_code, OR add a sibling)
- Modify: `internal/api/handlers/tts.go` (add `LanguageCode` to `TTSRequest`; set on `synthReq`)
- Modify: `internal/api/handlers/jobs.go` (**1** `domain.NewJob(...)` call site; add `LanguageCode` to `JobCreateRequest`; pass to `NewJob`)
- Modify: `internal/api/handlers/tts_test.go` (assert `req.LanguageCode` plumbed through)
- Modify: `internal/api/handlers/jobs_test.go` (**3** `NewJob` call sites + assert `job.LanguageCode` is set when request body sets `language_code`)
- Modify: `cmd/server/openapi.yaml` (add `language_code` to `TTSRequest` + `JobCreateRequest` schemas)

> **Mass-update callout:** the `NewJob` signature change touches **32 call sites** total: 11 in `internal/domain/job_test.go` + 16 in `internal/queue/memory/queue_test.go` + 1 in `internal/queue/memory/worker_test.go` (line 104, easy to miss) + 1 in `internal/api/handlers/jobs.go` + 3 in `internal/api/handlers/jobs_test.go`. Update them all in the same commit so the build is never broken. Insert `""` at position 4 (after `modelID`). Quick verification: `grep -rn "NewJob(" --include="*.go" .` should return 32 results.

- [ ] add `LanguageCode string` to `domain.SynthesisRequest`
- [ ] add `LanguageCode string \`json:"language_code,omitempty"\`` to `domain.Job`; update `NewJob` signature to insert `languageCode string` after `modelID`
- [ ] update all **32** existing `domain.NewJob` call sites to pass the new `""` argument at position 4
- [ ] update worker (`internal/queue/memory/worker.go:123`) to copy `LanguageCode: job.LanguageCode` into the `SynthesisRequest`
- [ ] add `LanguageCode string \`json:"language_code,omitempty"\`` to `handlers.TTSRequest`; set `synthReq.LanguageCode = req.LanguageCode` in `SynthesizeTTS`
- [ ] add `LanguageCode string \`json:"language_code,omitempty"\`` to `handlers.JobCreateRequest`; pass it to `domain.NewJob`
- [ ] update `cmd/server/openapi.yaml`: add `language_code` (with description per Technical Details) to both `TTSRequest` and `JobCreateRequest` schemas — placed immediately after `model_id`
- [ ] extend `TestNewJob` in `internal/domain/job_test.go` to assert the new `LanguageCode` field is stored on the returned `*Job`
- [ ] extend `internal/api/handlers/tts_test.go` with an assertion (or a new sibling test) that posting `{"text":"x","language_code":"en"}` causes the captured `*domain.SynthesisRequest` to have `LanguageCode == "en"`; assert it remains `""` when the field is omitted
- [ ] extend `internal/api/handlers/jobs_test.go` with `TestJobsHandler_SubmitJob_PassesLanguageCode` (matching the existing `TestJobsHandler_SubmitJob_PassesModelID` naming): post `{"text":"x","language_code":"en"}`; fetch the enqueued job from the mock queue; assert `job.LanguageCode == "en"`
- [ ] extend `internal/queue/memory/worker_test.go` with a propagation assertion: enqueue a job with `LanguageCode: "en"`, run one tick of the worker against a `MockProvider` whose `SynthesizeFunc` captures the request, assert captured `req.LanguageCode == "en"`
- [ ] run `go test ./...` — must pass before next task

### Task 2: ElevenLabs provider sends `language_code` to upstream

**Files:**
- Modify: `internal/provider/elevenlabs/client.go` (add `LanguageCode` to `TTSRequest` JSON)
- Modify: `internal/provider/elevenlabs/provider.go` (set `ttsReq.LanguageCode` from `req.LanguageCode`)
- Modify: `internal/provider/elevenlabs/provider_test.go` (assert outbound JSON body contains `language_code` when set)

- [ ] add `LanguageCode string \`json:"language_code,omitempty"\`` to `elevenlabs.TTSRequest` in `client.go`, immediately after `ModelID`
- [ ] in `elevenlabs.Provider.Synthesize` (`provider.go`), after the existing `ModelID` assignment block, add: `if req.LanguageCode != "" { ttsReq.LanguageCode = req.LanguageCode }` (no else — empty stays empty so the upstream uses model default)
- [ ] extend the existing `Synthesize` test in `provider_test.go` to: (a) call `Synthesize` with `req.LanguageCode = "en"` and assert the captured upstream HTTP body contains `"language_code":"en"`; (b) call without `LanguageCode` and assert the body does NOT contain a `language_code` key (verifies `omitempty`)
- [ ] run `go test ./internal/provider/elevenlabs/...` — must pass before next task

### Task 3: Selfhosted provider explicitly ignores `language_code`

**Files:**
- Modify: `internal/provider/selfhosted/provider.go` (add doc comment near `Synthesize`)
- Modify: `internal/provider/selfhosted/provider_test.go` (assert `req.LanguageCode` is not forwarded)

- [ ] add a comment in `selfhosted.Provider.Synthesize` near the existing model-id resolution switch noting that `req.LanguageCode` is intentionally ignored — the local TTS API has no comparable parameter
- [ ] add a test in `provider_test.go` that calls `Synthesize` with `req.LanguageCode = "en"` and asserts `language_code` is absent from the upstream request body. **Important**: do NOT decode into the local `selfhosted.SynthesisRequest` struct (it has no `LanguageCode` field, so a JSON decode would silently drop the key and the assertion would be vacuous). Instead, capture the raw body bytes and assert with `bytes.Contains(rawBody, []byte("language_code"))` (must be `false`), or unmarshal into `map[string]any` and check `_, ok := m["language_code"]; !ok`. This catches a future regression where the field gets accidentally forwarded.
- [ ] run `go test ./internal/provider/selfhosted/...` — must pass before next task

### Task 4: UI — Language `<select>` populated from union of model languages

**Files:**
- Modify: `internal/ui/index.html`
- Modify: `internal/ui/ui_test.go`

- [ ] add `<label for="language-select">` + `<select id="language-select" name="language_code">` to the form, immediately after the Model select
- [ ] add the `languageSelect` JS variable, `resetLanguages(message)` helper, and `unionModelLanguages(models)` helper per Technical Details
- [ ] hook into `loadModels(name)` success path: compute `unionModelLanguages(models)` and populate `languageSelect` (preserving the empty `'Default language'` option at the top)
- [ ] hook into `loadModels(name)` `.catch` and the provider `change` listener's no-provider branch to call `resetLanguages('Default language')`
- [ ] add `languageSelect.disabled = disabled;` to `setFormDisabled(disabled, label)` so the field is locked during in-flight requests
- [ ] update form `submit` handler: include `body.language_code = languageCode` when non-empty (next to the existing `body.model_id` block)
- [ ] extend `internal/ui/ui_test.go` body-marker assertions to include `id="language-select"` and `language_code` (so the submit-body wiring is locked in). Mirror the existing `id="model-select"` assertion at `ui_test.go:53`. Do NOT assert on internal helper names like `unionModelLanguages` — those lock the implementation without a real behavior signal.
- [ ] run `go test ./internal/ui/...` and `go test ./...` — must pass before next task

### Task 5: Verify acceptance criteria

- [ ] `go vet ./...` and `go test ./...` (full suite) pass
- [ ] start the server locally (`make run`) with a minimal config containing the ElevenLabs provider; confirm:
  - [ ] `POST /api/v1/tts` with `{"text":"hola mundo","language_code":"es","model_id":"eleven_flash_v2_5"}` returns audio that sounds Spanish (manual listen)
  - [ ] `POST /api/v1/tts` with `{"text":"hello","language_code":"zz"}` (invalid code) returns 503 with the upstream ElevenLabs error message in `error.message`
  - [ ] `POST /api/v1/tts` without `language_code` still works exactly as before (no regression)
  - [ ] `POST /api/v1/jobs` with `language_code` enqueues, processes, and returns audio in the requested language
- [ ] visit `/ui/` in a browser:
  - [ ] Language dropdown populates after page load and after changing provider (with codes like `en`, `es`, `de`, …)
  - [ ] Submitting with a language selected sends `language_code` in the request body (verify in DevTools Network tab)
  - [ ] Submitting with the default language option leaves `language_code` out of the request body
- [ ] confirm `/openapi.json` includes `language_code` on `TTSRequest` and `JobCreateRequest`
- [ ] verify no regressions: existing tests, voices endpoint, models endpoint, sync TTS path, async jobs path

### Task 6: [Final] Update documentation and finalize

**Files:**
- Modify: `README.md` (mention `language_code` next to the existing `model_id` mention)
- Modify: `docs/todo/todo.md` (mark the language selection item as complete)

- [ ] add a short note to `README.md` mentioning the optional `language_code` request field and the UI Language picker
- [ ] update `docs/todo/todo.md`: check off the language selection item (or remove the section if no other todos remain in it)
- [ ] move this plan to `docs/plans/completed/` (`mkdir -p docs/plans/completed && mv docs/plans/20260426-language-selection.md docs/plans/completed/`)

## Post-Completion

*Items requiring manual intervention or external systems — informational only*

**Manual verification:**
- Browser smoke test in Chrome and Safari: pick provider → models load → language dropdown populates → pick a language → synthesize → audio plays in expected language.
- Try a model that doesn't support a given language (e.g. `eleven_english_sts_v2` + `language_code=es` if exposed) and confirm the 503 surfaces a useful upstream error message.
- Confirm the language list reflects only models advertised by `/api/v1/providers/elevenlabs/models` — i.e. it changes if the upstream model catalog changes.

**Known limitations / out of scope:**
- The Language dropdown shows raw ISO 639-1 codes (e.g. `en`, `es`). No human-readable label. The upstream `/v1/models` `languages` array does include a `name` field per language but `domain.Model.Languages` flattens it to codes only — adding the names would require a `[]Language{Code, Name}` shape change. Out of scope.
- Selfhosted provider deliberately ignores `language_code`. Future selfhosted backends can opt-in by reading `req.LanguageCode`.
- No client-side validation — bad codes get a 503 with the upstream error. Acceptable per the validation decision in the plan; revisit if support burden grows.
- The Language dropdown is independent of the Model dropdown. A user can pick a `language_code` the chosen Model does not support, get a 503, and have to retry. This is intentional (cleaner UI than a coupled state machine) but may surprise users.

**Optional follow-ups (not in scope):**
- Couple Language and Model dropdowns: filter Model options when a Language is chosen (and vice versa) so impossible combinations are unselectable.
- Show language name alongside code in the dropdown (`English (en)` instead of `en`) — needs a model-shape change to carry names.
- Support `language_code` on the selfhosted provider for backends that do accept it (would need a config flag or capability discovery).
- Expose the remaining ElevenLabs TTS body params (`seed`, `previous_text`/`next_text`, `apply_text_normalization`, `pronunciation_dictionary_locators`) using the same plumbing pattern.
