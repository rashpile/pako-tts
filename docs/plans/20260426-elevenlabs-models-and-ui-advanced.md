# ElevenLabs Model Selection + UI Advanced Voice Settings

## Overview

Implement the five outstanding items from `docs/todo/todo.md`:

1. ElevenLabs `model_id` becomes configurable via `config.yaml` (default `eleven_multilingual_v2`); the hardcoded fallback in the client is removed.
2. New endpoint `GET /api/v1/providers/{name}/models` returns the provider's available models. Mirrors the shape of the existing `/voices` endpoint.
3. Optional `model_id` accepted on `POST /api/v1/tts` and `POST /api/v1/jobs`. Plumbed through `domain.SynthesisRequest` to `elevenlabs.TTSRequest.ModelID`. Falls back to the configured per-provider default when omitted.
4. Browser UI gains a **Model** `<select>` populated from the new models endpoint. Repopulates on provider change (same pattern as the voice dropdown).
5. Browser UI gains a collapsible **Advanced** section exposing provider-specific voice settings (ElevenLabs: `stability`, `similarity_boost`, `style`, `use_speaker_boost`). Provider-keyed JS map keeps the form layout simple and gives a clear extension point for future providers.

These changes are additive — no existing endpoint contract changes for clients that don't set the new fields.

## Context (from discovery)

- **Project**: Go 1.23 HTTP server (Chi router, Zap logger). Multi-provider (ElevenLabs + selfhosted), single-binary deployment, embedded OpenAPI spec at `cmd/server/openapi.yaml`.
- **Files involved**:
  - `pkg/config/config.go:30` — `ProviderConfig` struct (no `model_id` field today).
  - `internal/provider/elevenlabs/client.go:16` — `defaultModel = "eleven_multilingual_v2"` hardcoded; `client.go:69` applies it as fallback when `req.ModelID == ""`.
  - `internal/provider/elevenlabs/provider.go:36` — `NewProviderFromConfig` (does not read `cfg.ModelID` today).
  - `internal/provider/selfhosted/provider.go:131` — `ListVoices` already calls `client.GetModels`. Selfhosted client already has `GetModels` returning model objects.
  - `internal/provider/registry/registry.go` and `factory.go` — registry / factory wiring.
  - `internal/domain/provider.go:12` — `TTSProvider` interface; `SynthesisRequest` struct (`provider.go:36`, no `ModelID` field today).
  - `internal/domain/voice.go:13` — `Voice` type (template for `Model`). `VoiceSettings` already supports stability/similarity_boost/style/speed/use_speaker_boost.
  - `internal/domain/job.go:39` — `NewJob(text, voiceID, providerName, outputFormat, settings)` — signature must accept new `modelID`.
  - `internal/queue/memory/worker.go:120` — builds `domain.SynthesisRequest` from `*domain.Job`; must copy new `ModelID` field.
  - `internal/api/handlers/providers.go:52` — existing `ListVoices` handler is the 1:1 template for new `ListModels` handler.
  - `internal/api/handlers/tts.go:42` — `TTSRequest` struct adds optional `model_id`.
  - `internal/api/handlers/jobs.go:45` — `JobCreateRequest` struct adds optional `model_id`.
  - `internal/api/handlers/mocks/provider.go:11` — `MockProvider` needs `ListModelsFunc` and `ListModels(ctx)` method (interface compliance).
  - `internal/api/handlers/providers_test.go:19` — table-driven test pattern for new `TestProvidersHandler_ListModels`.
  - `internal/api/routes.go:105` — registers `GET /providers/{name}/voices`; new `GET /providers/{name}/models` registers next to it.
  - `cmd/server/openapi.yaml:319,612` — existing `/api/v1/providers/{name}/voices` path and `VoicesListResponse` schema serve as exact template.
  - `internal/ui/index.html` — single-file vanilla ES5 UI; existing `loadVoices(name)` is the 1:1 template for `loadModels(name)`.
  - `config.yaml.example` — needs an updated commented `model_id` example for elevenlabs.
- **Patterns**:
  - Handlers use `middleware.WriteJSON` for success and `middleware.WriteError(domain.APIError)` for errors. Provider not-found → `domain.ErrProviderNotFound` (404). Upstream provider failure → `domain.ErrProviderUnavailable` (503).
  - Tests use `mocks.MockProvider` + `mocks.NewMockProviderRegistry`, table-driven, with `chi.RouteCtxKey` injection (see `providers_test.go:79`).
  - `MockProvider.ListVoicesFunc` returning `nil, nil` is normalized to `[]Voice{}` in the handler (`providers.go:68`). Apply the same pattern for models so the JSON serializes `"models":[]` not `"models":null`.
- **Dependencies**: no new third-party packages required. ElevenLabs `GET /v1/models` already returns the data we need (per `docs/research/research-elevenlab.md`).
- **Note**: The selfhosted provider's existing `Synthesize` (`provider.go:91-95`) opportunistically maps a short `voice_id` to `model_id`. Adding an explicit `req.ModelID` does not change that behavior — when `req.ModelID` is set, it takes precedence; when blank, today's fallback continues to work.
- **Selfhosted ListModels semantics (decision)**: The selfhosted upstream `voices_endpoint` defaults to `/api/v1/models`, so today `ListVoices` already returns the upstream model list. To avoid the UI showing the same items in both Voice and Model dropdowns (and avoid sending `voice_id == model_id` redundantly), `selfhosted.ListModels` returns `(nil, nil)` (empty slice serialized as `[]`). The UI's Model dropdown then shows only "Default model" for selfhosted, which is the correct UX (selfhosted has no separate "model id" concept distinct from its voices). ElevenLabs is the only provider with a meaningful Model dropdown for now.
- **MockProvider order callout**: The `domain.TTSProvider` interface change (adding `ListModels`) breaks compilation of every test that uses `MockProvider` until `MockProvider.ListModels` is added. In Task 1 the MockProvider edit must land in the **same commit** as the interface change (or before it).
- **Full `domain.NewJob` caller inventory** (so Task 4 doesn't surprise the implementer): 11 calls in `internal/domain/job_test.go`, 16 calls in `internal/queue/memory/queue_test.go`, 3 calls in `internal/api/handlers/jobs_test.go`, 1 call in `internal/api/handlers/jobs.go` — **31 call sites** that all need a `""` (or test-supplied) `modelID` argument inserted at position 3.

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
- maintain backward compatibility (additive changes only — `model_id` is optional everywhere; existing clients that don't send it use the configured per-provider default)

## Testing Strategy

- **unit tests**: required for every task
  - `internal/provider/elevenlabs/provider_test.go` — extend (or create) tests asserting `NewProviderFromConfig` reads `ModelID` from config and defaults to `eleven_multilingual_v2` when blank; `Synthesize` forwards explicit `req.ModelID` to `client.TTSRequest.ModelID` and falls back to provider's stored default.
  - `internal/api/handlers/providers_test.go` — add `TestProvidersHandler_ListModels` mirroring the existing `TestProvidersHandler_ListVoices` cases (success, unknown provider 404, provider error 503, nil normalized to `[]`).
  - `internal/api/handlers/tts_test.go` (create if missing) or `jobs_test.go` (extend) — verify that `model_id` in the request body is plumbed through to `domain.SynthesisRequest.ModelID` (or `domain.Job.ModelID` for jobs) using a `MockProvider.SynthesizeFunc` capture.
  - `internal/domain/job_test.go` — extend `TestNewJob` to verify the new `modelID` parameter is stored on the job.
  - `internal/api/handlers/mocks/provider.go` — extend `MockProvider` with `ListModelsFunc` so existing tests still compile and the new handler tests work.
  - `internal/ui/ui_test.go` — extend `TestHandler_ServeHTTP` body-marker assertions to include `id="model-select"`, `/api/v1/providers/` (already present, but assert presence of the models path component) and `id="advanced-section"` so a future refactor that breaks the UI is caught.
- **e2e tests**: project has no UI-based e2e harness today; manual smoke tests in browser are part of acceptance verification (Task 6).
- **mock provider**: `MockProvider` will gain `ListModelsFunc` mirroring the existing `ListVoicesFunc`.

## Progress Tracking

- mark completed items with `[x]` immediately when done
- add newly discovered tasks with `+` prefix
- document issues/blockers with `WARN` prefix
- update plan if implementation deviates from original scope

## Solution Overview

1. **Domain layer** gets a `Model` type and a new `ListModels(ctx) ([]Model, error)` method on the `TTSProvider` interface. Both providers implement it (elevenlabs from `/v1/models`, selfhosted from its existing `client.GetModels` — same source already used for `ListVoices`). `domain.SynthesisRequest` gains a `ModelID string` field. `domain.Job` and `domain.NewJob` gain a `ModelID` field/parameter.
2. **ElevenLabs provider** stores a `defaultModelID string` (read from `cfg.ModelID`, defaulting to `eleven_multilingual_v2` when blank). On `Synthesize`, sets `ttsReq.ModelID = req.ModelID` if non-empty, else `defaultModelID`. The `defaultModel` constant + `if req.ModelID == "" { req.ModelID = defaultModel }` fallback in `client.go` are removed.
3. **Config** gains `ModelID` on `ProviderConfig` (mapstructure tag `model_id`). Read in `loadProvidersConfig` like other string fields.
4. **API handler `ListModels`** mirrors `ListVoices`: lookup provider, call `provider.ListModels(ctx)`, return `{provider, models}`. New OpenAPI path + schemas.
5. **Sync TTS + jobs handlers** decode optional `model_id`. TTS handler passes it on `domain.SynthesisRequest`; jobs handler passes it via `domain.NewJob` and worker copies it onto `SynthesisRequest`.
6. **UI**:
   - New Model `<select>` next to the Voice select. Populated by `loadModels(provider)` on init and provider change.
   - New collapsible `<details><summary>Advanced</summary>` section. JS map keyed by provider name (or provider type) defines which controls render. ElevenLabs gets four controls. Submit only includes `voice_settings` keys whose values were touched (or always include all when the section is open).

## Technical Details

### Domain — `internal/domain/provider.go`

Add:
```go
// Model represents a TTS model (e.g., "eleven_multilingual_v2").
type Model struct {
    ModelID     string   `json:"model_id"`
    Name        string   `json:"name"`
    Provider    string   `json:"provider"`
    Description string   `json:"description,omitempty"`
    Languages   []string `json:"languages,omitempty"`
}
```

Extend `TTSProvider` interface:
```go
type TTSProvider interface {
    // ... existing methods ...
    ListModels(ctx context.Context) ([]Model, error)
}
```

Extend `SynthesisRequest`:
```go
type SynthesisRequest struct {
    Text         string
    VoiceID      string
    ModelID      string         // optional; provider falls back to its configured default
    OutputFormat string
    Settings     *VoiceSettings
}
```

### Domain — `internal/domain/job.go`

Add `ModelID string` field on `Job` (json tag `model_id,omitempty`). Update `NewJob` signature:
```go
func NewJob(text, voiceID, modelID, providerName, outputFormat string, settings *VoiceSettings) *Job
```
All callers (handlers + tests in `internal/queue/memory/queue_test.go`) updated accordingly.

### Config — `pkg/config/config.go`

Add `ModelID string \`mapstructure:"model_id"\`` to `ProviderConfig`. In `loadProvidersConfig` (`config.go:217`), add `ModelID: getString(providerMap, "model_id"),`.

`config.yaml.example` — under the `elevenlabs` provider entry add:
```yaml
      model_id: "eleven_multilingual_v2"  # optional; ElevenLabs model id
```

### ElevenLabs provider — `internal/provider/elevenlabs/{client.go,provider.go}`

`client.go`:
- Remove the `defaultModel = "eleven_multilingual_v2"` constant **and** the `if req.ModelID == "" { req.ModelID = defaultModel }` block in `TextToSpeech`. Do **not** add a defensive error — keep `Client` dumb. The invariant is documented in a `// model_id must be set by the caller` comment on `TTSRequest.ModelID`. The `Provider.Synthesize` path always sets it; if a future caller forgets, the upstream ElevenLabs API returns its own validation error, surfaced verbatim to the client.
- Add `GetModels(ctx)` calling `GET /v1/models`. Response struct mirrors only the fields we use: `model_id`, `name`, `description`, `can_do_text_to_speech`, `languages` (subset). Filter to `can_do_text_to_speech == true`.

`provider.go`:
- Add `defaultModelID string` field on `Provider`.
- `NewProviderFromConfig`: read `cfg.ModelID`; if empty, use `"eleven_multilingual_v2"`. Store on the provider.
- `Synthesize`: set `ttsReq.ModelID = req.ModelID` if non-empty, else `p.defaultModelID`.
- New `ListModels(ctx)`: call `client.GetModels`, map each to `domain.Model{ModelID, Name, Provider: providerName, Description, Languages}` (extract languages from the nested `[{language_id, name}]` array — flatten to `[]string{language_id...}`). Skip models where `can_do_text_to_speech == false`.

### Selfhosted provider — `internal/provider/selfhosted/provider.go`

Add `ListModels(ctx)` that returns `(nil, nil)`. Rationale: the selfhosted upstream `voices_endpoint` defaults to `/api/v1/models`, so `ListVoices` already exposes the same upstream model list — returning the same items from `ListModels` would make the UI show duplicate dropdowns and produce confusing `voice_id == model_id` request bodies. The handler's nil-normalization yields `"models": []`, which is the correct empty-state for selfhosted ("no separate model concept"). Add a doc comment on the method explaining this so the next reader does not undo the deliberate empty.

`Synthesize` change: replace the existing block

```go
if req.VoiceID != "" && len(req.VoiceID) < 20 {
    ttsReq.ModelID = req.VoiceID
}
```

with

```go
switch {
case req.ModelID != "":
    ttsReq.ModelID = req.ModelID
case req.VoiceID != "" && len(req.VoiceID) < 20:
    ttsReq.ModelID = req.VoiceID
}
```

— so an explicit `req.ModelID` wins; otherwise legacy heuristic is preserved unchanged.

### MockProvider — `internal/api/handlers/mocks/provider.go`

Add `ListModelsFunc func(ctx context.Context) ([]domain.Model, error)` field and `ListModels(ctx)` method (returns `ListModelsFunc(ctx)` if set, else a default 2-element slice for ergonomics). This satisfies the new interface and lets existing tests compile unchanged.

### API handler — `internal/api/handlers/providers.go`

Add (mirroring `ListVoices`):
```go
type ModelsListResponse struct {
    Provider string         `json:"provider"`
    Models   []domain.Model `json:"models"`
}

func (h *ProvidersHandler) ListModels(w http.ResponseWriter, r *http.Request) {
    name := chi.URLParam(r, "name")
    provider, err := h.registry.Get(name)
    if err != nil {
        middleware.WriteError(w, domain.ErrProviderNotFound.WithMessage("Provider '"+name+"' not found"))
        return
    }
    models, err := provider.ListModels(r.Context())
    if err != nil {
        h.logger.Error("ListModels failed", zap.String("provider", name), zap.Error(err))
        middleware.WriteError(w, domain.ErrProviderUnavailable.WithMessage(err.Error()))
        return
    }
    if models == nil {
        models = []domain.Model{}
    }
    middleware.WriteJSON(w, http.StatusOK, ModelsListResponse{Provider: name, Models: models})
}
```

### TTS handler — `internal/api/handlers/tts.go`

Add `ModelID string \`json:"model_id,omitempty"\`` to `TTSRequest`. In `SynthesizeTTS`, set `synthReq.ModelID = req.ModelID` (zero-value string is fine — provider treats empty as "use default").

### Jobs handler — `internal/api/handlers/jobs.go`

Add `ModelID string \`json:"model_id,omitempty"\`` to `JobCreateRequest`. Pass `req.ModelID` through to the new `NewJob(...)` parameter.

### Worker — `internal/queue/memory/worker.go`

`worker.go:120`: copy `ModelID: job.ModelID` into the constructed `SynthesisRequest`.

### Routes — `internal/api/routes.go`

Add inside the existing `r.Route("/api/v1", ...)` block, next to the voices route:
```go
r.Get("/providers/{name}/models", providersHandler.ListModels)
```

### OpenAPI — `cmd/server/openapi.yaml`

1. Add `model_id` (optional string) to both `TTSRequest` and `JobCreateRequest` schemas.
2. Add new path `/api/v1/providers/{name}/models` with 200 / 404 / 503 responses (clone the existing `/voices` path entry).
3. Add `Model` and `ModelsListResponse` schemas under `components.schemas`:
```yaml
    Model:
      type: object
      required: [model_id, name, provider]
      properties:
        model_id:
          type: string
          description: Provider-specific model identifier (e.g. "eleven_multilingual_v2")
        name:
          type: string
          description: Human-readable model name
        provider:
          type: string
          description: Provider identifier this model belongs to
        description:
          type: string
          description: Provider-supplied model description
        languages:
          type: array
          description: Language codes (ISO 639-1) the model supports
          items: { type: string }

    ModelsListResponse:
      type: object
      required: [provider, models]
      properties:
        provider: { type: string }
        models:
          type: array
          items: { $ref: "#/components/schemas/Model" }
```

### UI — `internal/ui/index.html`

**HTML (form additions):**
```html
<label for="model-select">Model</label>
<select id="model-select" name="model_id">
    <option value="">Default model</option>
</select>

<details id="advanced-section" class="advanced">
    <summary>Advanced</summary>
    <div id="advanced-controls"></div>
</details>
```

Plus a small `<style>` block: `.advanced summary { cursor: pointer; font-size: 0.875rem; color: #555; margin-top: 8px; } .advanced .control-row { display: flex; gap: 12px; align-items: center; margin-bottom: 12px; } .advanced .control-row input[type=range] { flex: 1; }`.

**JS additions** (vanilla ES5, mirrors existing `loadVoices` pattern):

```js
var modelSelect = document.getElementById('model-select');
var advancedSection = document.getElementById('advanced-section');
var advancedControls = document.getElementById('advanced-controls');
var modelLoadToken = 0;

// Maps provider name -> provider type (populated from GET /api/v1/providers).
// Keying advanced controls by *type* (not name) lets users rename their provider
// in config.yaml without losing the UI controls. Existing provider types today:
// "ElevenLabsProvider" and "SelfhostedProvider" (see internal/provider/*/provider.go).
var providerTypes = {};

// Type-keyed advanced control schemas. Add entries when new provider types gain controls.
// Defaults MUST match domain.DefaultVoiceSettings() (internal/domain/voice.go:24)
// so that opening Advanced and submitting without changes is a no-op vs. closing it.
var ADVANCED_SCHEMAS = {
    'ElevenLabsProvider': [
        { key: 'stability',         label: 'Stability',         type: 'range', min: 0, max: 1,   step: 0.05, default: 0.0 },
        { key: 'similarity_boost',  label: 'Similarity boost',  type: 'range', min: 0, max: 1,   step: 0.05, default: 1.0 },
        { key: 'style',             label: 'Style',             type: 'range', min: 0, max: 1,   step: 0.05, default: 0.0 },
        { key: 'use_speaker_boost', label: 'Speaker boost',     type: 'checkbox',                            default: true }
    ]
};

function renderAdvanced(name) {
    advancedControls.innerHTML = '';
    var schema = ADVANCED_SCHEMAS[providerTypes[name]];
    if (!schema) {
        advancedSection.style.display = 'none';
        return;
    }
    advancedSection.style.display = '';
    schema.forEach(function (ctrl) {
        var row = document.createElement('div');
        row.className = 'control-row';
        var label = document.createElement('label');
        label.htmlFor = 'adv-' + ctrl.key;
        label.textContent = ctrl.label;
        row.appendChild(label);

        var input = document.createElement('input');
        input.id = 'adv-' + ctrl.key;
        input.dataset.key = ctrl.key;
        input.dataset.kind = ctrl.type;
        if (ctrl.type === 'range') {
            input.type = 'range';
            input.min = ctrl.min; input.max = ctrl.max; input.step = ctrl.step;
            input.value = ctrl.default;
            var valSpan = document.createElement('span');
            valSpan.textContent = ctrl.default;
            input.addEventListener('input', function () { valSpan.textContent = input.value; });
            row.appendChild(input);
            row.appendChild(valSpan);
        } else if (ctrl.type === 'checkbox') {
            input.type = 'checkbox';
            input.checked = !!ctrl.default;
            row.appendChild(input);
        }
        advancedControls.appendChild(row);
    });
}

function collectVoiceSettings() {
    // Include voice_settings only when Advanced is open. The control defaults are
    // aligned to domain.DefaultVoiceSettings(), so opening Advanced and submitting
    // without changes is a no-op (the server-side merge produces the same result
    // as if voice_settings were absent).
    if (!advancedSection.open) return null;
    var inputs = advancedControls.querySelectorAll('input[data-key]');
    if (inputs.length === 0) return null;
    var out = {};
    for (var i = 0; i < inputs.length; i++) {
        var el = inputs[i];
        if (el.dataset.kind === 'range') out[el.dataset.key] = parseFloat(el.value);
        else if (el.dataset.kind === 'checkbox') out[el.dataset.key] = el.checked;
    }
    return out;
}

function loadModels(name) {
    modelSelect.innerHTML = '';
    var loading = document.createElement('option');
    loading.value = ''; loading.textContent = 'Loading models...';
    modelSelect.appendChild(loading);
    var token = ++modelLoadToken;
    fetch('/api/v1/providers/' + encodeURIComponent(name) + '/models')
        .then(function (res) {
            if (!res.ok) throw new Error('HTTP ' + res.status);
            return res.json();
        })
        .then(function (data) {
            if (token !== modelLoadToken) return;
            var models = data.models || [];
            modelSelect.innerHTML = '';
            var def = document.createElement('option');
            def.value = ''; def.textContent = 'Default model';
            modelSelect.appendChild(def);
            models.forEach(function (m) {
                var opt = document.createElement('option');
                opt.value = m.model_id;
                opt.textContent = m.name || m.model_id;
                modelSelect.appendChild(opt);
            });
        })
        .catch(function () {
            if (token !== modelLoadToken) return;
            modelSelect.innerHTML = '';
            var def = document.createElement('option');
            def.value = ''; def.textContent = 'Default model';
            modelSelect.appendChild(def);
            // No error message here — model list is best-effort; default works.
        });
}
```

Hook into existing `loadProviders` — alongside the existing `loadVoices` call. Inside the success handler that iterates `data.providers`, also populate the `providerTypes` map (`providerTypes[p.name] = p.type;`) BEFORE calling `renderAdvanced(initialName)`:
```js
providers.forEach(function(p) {
    providerTypes[p.name] = p.type;
    // ... existing option creation ...
});
// after the forEach:
if (initialName) { loadVoices(initialName); loadModels(initialName); renderAdvanced(initialName); }
```

And update the provider `change` listener:
```js
providerSelect.addEventListener('change', function () {
    var name = providerSelect.value;
    if (name) { loadVoices(name); loadModels(name); renderAdvanced(name); }
    else {
        resetVoices('Default voice');
        modelSelect.innerHTML = '<option value="">Default model</option>';
        advancedSection.style.display = 'none';
    }
});
```

Update the existing `setFormDisabled` function to also disable the Model select while a request is in flight:
```js
function setFormDisabled(disabled, label) {
    textInput.disabled = disabled;
    providerSelect.disabled = disabled;
    voiceSelect.disabled = disabled;
    modelSelect.disabled = disabled;       // NEW
    formatSelect.disabled = disabled;
    submitBtn.disabled = disabled;
    submitBtn.textContent = label || 'Synthesize';
}
```

Form submit body construction:
```js
var modelID = modelSelect.value;
if (modelID) body.model_id = modelID;
var settings = collectVoiceSettings();
if (settings) body.voice_settings = settings;
```

## What Goes Where

- **Implementation Steps** (`[ ]` checkboxes): Go code, embedded HTML, OpenAPI yaml, unit tests, route wiring, README/config example updates.
- **Post-Completion** (no checkboxes): manual browser smoke test, deploy considerations, screenshots.

## Implementation Steps

### Task 1: Add `domain.Model` + `ListModels` to `TTSProvider` interface

**Files:**
- Create: `internal/domain/model.go`
- Modify: `internal/domain/provider.go`
- Modify: `internal/api/handlers/mocks/provider.go`
- Modify: `internal/provider/elevenlabs/provider.go`
- Modify: `internal/provider/elevenlabs/client.go`
- Modify: `internal/provider/selfhosted/provider.go`
- Modify: `internal/provider/elevenlabs/provider_test.go`
- Create or modify: `internal/provider/selfhosted/provider_test.go`

> **Order matters:** the interface change in step 2 below will break compilation of every test that uses `MockProvider` until step 3 (MockProvider) is added. Land steps 2–6 in a single commit, or use the order below which adds MockProvider's `ListModels` before the new providers' implementations.

- [x] create `internal/domain/model.go` with the `Model` struct (fields: `ModelID`, `Name`, `Provider`, `Description`, `Languages []string`) and JSON tags per Technical Details
- [x] add `ListModels(ctx context.Context) ([]Model, error)` to the `TTSProvider` interface in `internal/domain/provider.go`
- [x] add `ListModelsFunc` field and `ListModels(ctx)` method on `mocks.MockProvider` — do this **immediately** so the test build does not fail mid-task
- [x] add `GetModels(ctx)` to `internal/provider/elevenlabs/client.go` calling `GET /v1/models` with internal response structs that decode only `model_id`, `name`, `description`, `can_do_text_to_speech`, and the nested `languages: [{language_id, name}]` array
- [x] implement `Provider.ListModels` in `internal/provider/elevenlabs/provider.go`: skip models where `can_do_text_to_speech == false`, flatten languages to `[]string{language_id...}`
- [x] implement `Provider.ListModels` in `internal/provider/selfhosted/provider.go` returning `(nil, nil)` with a doc comment explaining why (selfhosted's "voices" upstream IS its model list — exposing them again as "models" would just duplicate the dropdown). Do NOT delegate to `client.GetModels` here
- [x] write unit tests for `elevenlabs.Provider.ListModels` using `httptest.Server` (success: returns mapped `[]domain.Model` with languages flattened; success: filters out non-TTS models; error: returns wrapped error). Mirror existing test file patterns in `provider_test.go`
- [x] write unit test for `selfhosted.Provider.ListModels` asserting it returns `(nil, nil)` (no upstream call) — and that the existing `ListVoices` still works
- [x] run `go test ./internal/...` — must pass before next task

### Task 2: Wire `model_id` config + remove client fallback

**Files:**
- Modify: `pkg/config/config.go`
- Modify: `internal/provider/elevenlabs/provider.go`
- Modify: `internal/provider/elevenlabs/client.go`
- Modify: `internal/provider/elevenlabs/provider_test.go`
- Modify: `config.yaml.example`

- [x] add `ModelID string \`mapstructure:"model_id"\`` to `pkg/config/config.go:30` `ProviderConfig`
- [x] read `model_id` in `loadProvidersConfig` (`config.go:217`) via `getString(providerMap, "model_id")`
- [x] add `defaultModelID string` field on `elevenlabs.Provider`; `NewProviderFromConfig` sets it from `cfg.ModelID` (default `"eleven_multilingual_v2"` when blank)
- [x] update `Provider.Synthesize` to set `ttsReq.ModelID = req.ModelID` (if non-empty) else `p.defaultModelID` (note: also added `ModelID` to `domain.SynthesisRequest` here since this update requires it; the equivalent Task 4 checkbox is now satisfied)
- [x] remove the `defaultModel` constant from `client.go` and the `if req.ModelID == "" { req.ModelID = defaultModel }` block in `TextToSpeech`. Do NOT add a defensive error — keep `Client` dumb. Add a `// model_id must be set by the caller` comment on `TTSRequest.ModelID` so the invariant is documented; the `Provider.Synthesize` path always sets it
- [x] update `config.yaml.example` to document the new optional `model_id` field on the elevenlabs provider entry
- [x] write tests in `provider_test.go`: (a) `NewProviderFromConfig` defaults `defaultModelID` to `"eleven_multilingual_v2"` when `cfg.ModelID == ""`; (b) `NewProviderFromConfig` honors a custom `cfg.ModelID`; (c) `Synthesize` sends `req.ModelID` when set; (d) `Synthesize` sends `defaultModelID` when `req.ModelID == ""`. Use `httptest.Server` capturing the inbound JSON body
- [x] write a config test (extend `pkg/config/config_test.go` if it exists, otherwise add one) verifying `ModelID` is loaded from `providers.list[].model_id`
- [x] run `go test ./...` — must pass before next task

### Task 3: Add `GET /api/v1/providers/{name}/models` handler + route + OpenAPI

**Files:**
- Modify: `internal/api/handlers/providers.go`
- Modify: `internal/api/handlers/providers_test.go`
- Modify: `internal/api/routes.go`
- Modify: `cmd/server/openapi.yaml`

- [ ] add `ModelsListResponse` struct and `ListModels(w, r)` method to `internal/api/handlers/providers.go` per Technical Details (returns 503 via `ErrProviderUnavailable` on upstream failures, 404 via `ErrProviderNotFound` on unknown provider, normalizes nil to `[]Model{}`)
- [ ] register `r.Get("/providers/{name}/models", providersHandler.ListModels)` inside the existing `r.Route("/api/v1", ...)` block in `internal/api/routes.go` (next to the existing voices route)
- [ ] add `/api/v1/providers/{name}/models` path entry (200 / 404 / 503) to `cmd/server/openapi.yaml`, plus `Model` and `ModelsListResponse` schemas under `components.schemas`
- [ ] add `TestProvidersHandler_ListModels` to `internal/api/handlers/providers_test.go` mirroring `TestProvidersHandler_ListVoices` (success returns models for known provider, unknown provider returns 404 `PROVIDER_NOT_FOUND`, provider error returns 503 `PROVIDER_UNAVAILABLE`, nil normalized to `[]`)
- [ ] run `go test ./internal/api/handlers/...` — must pass before next task

### Task 4: Plumb optional `model_id` through TTS request, Job, and worker

**Files:**
- Modify: `internal/domain/provider.go` (add `ModelID` to `SynthesisRequest`)
- Modify: `internal/domain/job.go` (add `ModelID` field + parameter to `NewJob`)
- Modify: `internal/domain/job_test.go` (11 `NewJob` call sites — all need a new `""` arg at position 3)
- Modify: `internal/queue/memory/worker.go`
- Modify: `internal/queue/memory/queue_test.go` (16 `domain.NewJob(...)` call sites — all need new `""` arg)
- Create: `internal/queue/memory/worker_test.go` (does not exist yet)
- Modify: `internal/api/handlers/tts.go`
- Modify: `internal/api/handlers/jobs.go` (1 `NewJob` call)
- Create: `internal/api/handlers/tts_test.go` (does not exist yet — sync TTS handler is currently untested)
- Modify: `internal/api/handlers/jobs_test.go` (3 `NewJob` call sites)
- Modify: `cmd/server/openapi.yaml`

> **Mass-update callout:** the `NewJob` signature change touches **31 call sites** total (11 + 16 + 1 + 3). Update them all in the same commit so the build is never broken. A simple `find . -name '*.go' -exec sed -i '' 's/NewJob(\("[^"]*"\), \("[^"]*"\), /NewJob(\1, \2, "", /' {} \;` won't fully work because of mixed identifier vs. literal arg styles — do it manually or with a careful gofmt-aware tool.

- [x] add `ModelID string` to `domain.SynthesisRequest` (done in Task 2 as a forward dependency)
- [ ] add `ModelID string \`json:"model_id,omitempty"\`` to `domain.Job`; update `NewJob` signature to `NewJob(text, voiceID, modelID, providerName, outputFormat string, settings *VoiceSettings) *Job`
- [ ] update worker at `internal/queue/memory/worker.go:120` to copy `ModelID: job.ModelID` into the constructed `SynthesisRequest`
- [ ] update **all 31** existing `domain.NewJob` call sites to pass the new `modelID` argument (use `""` for non-test sites; tests should pass `""` unless they're the new model-propagation test below)
- [ ] add `ModelID string \`json:"model_id,omitempty"\`` to `handlers.TTSRequest`; in `SynthesizeTTS`, set `synthReq.ModelID = req.ModelID`
- [ ] add `ModelID string \`json:"model_id,omitempty"\`` to `handlers.JobCreateRequest`; pass it as the new param to `domain.NewJob`
- [ ] update `cmd/server/openapi.yaml` `TTSRequest` and `JobCreateRequest` schemas to document optional `model_id` (description: "Provider-specific model id; uses provider's configured default when omitted")
- [ ] update `internal/domain/job_test.go` (`TestNewJob`) to assert the new `ModelID` field is stored on the returned `*Job`
- [ ] create `internal/api/handlers/tts_test.go` with `TestSynthesizeTTS_PassesModelID`: build a `MockProvider` whose `SynthesizeFunc` captures the inbound `*domain.SynthesisRequest`; POST a JSON body with `"model_id": "eleven_v3"`; assert the captured `req.ModelID == "eleven_v3"`. Also assert `req.ModelID == ""` is left untouched when the field is omitted
- [ ] extend `internal/api/handlers/jobs_test.go` with a `TestSubmitJob_PassesModelID`: POST `{"text":"x","model_id":"eleven_v3"}`; fetch the enqueued job from the mock queue; assert `job.ModelID == "eleven_v3"`
- [ ] create `internal/queue/memory/worker_test.go` with `TestWorker_PropagatesJobModelIDToSynthesisRequest`: enqueue a job with `ModelID: "eleven_v3"`, run one tick of the worker against a `MockProvider` whose `SynthesizeFunc` captures the request, assert captured `req.ModelID == "eleven_v3"`. (Without this test, the worker wiring is uncovered.)
- [ ] run `go test ./...` — must pass before next task

### Task 5: UI — Model select + provider-keyed Advanced section

**Files:**
- Modify: `internal/ui/index.html`
- Modify: `internal/ui/ui_test.go`

- [ ] add Model `<label>` + `<select id="model-select">` to the form, after the Voice select (per Technical Details)
- [ ] add collapsible `<details id="advanced-section" class="advanced"><summary>Advanced</summary><div id="advanced-controls"></div></details>` after the Format select
- [ ] add the `.advanced` CSS rules (summary cursor, `.control-row` flex layout, range input flex sizing) to the existing `<style>` block
- [ ] add the JS additions per Technical Details: `providerTypes` name→type map, `ADVANCED_SCHEMAS` keyed by provider **type** (`'ElevenLabsProvider'`) with defaults aligned to `domain.DefaultVoiceSettings()` (stability=0.0, similarity_boost=1.0, style=0.0, use_speaker_boost=true), `renderAdvanced(name)`, `collectVoiceSettings()`, `loadModels(name)`
- [ ] update `loadProviders()` success path: in the `providers.forEach` populate `providerTypes[p.name] = p.type`; after the loop call `loadVoices(initialName)` AND `loadModels(initialName)` AND `renderAdvanced(initialName)` (alongside, not chained — they're independent fetches)
- [ ] update the provider `change` listener to call `loadVoices(name)`, `loadModels(name)`, `renderAdvanced(name)` when name is set; otherwise clear voices, clear models, hide the advanced section
- [ ] update `setFormDisabled(disabled, label)` to also disable `modelSelect` while a request is in flight
- [ ] update form `submit` handler: include `body.model_id = modelID` when non-empty; include `body.voice_settings = collectVoiceSettings()` when truthy (i.e., when Advanced is open)
- [ ] extend `internal/ui/ui_test.go` body-marker assertions to include `id="model-select"`, `id="advanced-section"`, `ADVANCED_SCHEMAS`, `'ElevenLabsProvider'` (so a future refactor that breaks the type-keyed map is caught), and the new endpoint path fragment `'/models'` using the same single-quote bracketing pattern as the existing `'/voices'` assertion
- [ ] run `go test ./internal/ui/...` and `go test ./...` — must pass before next task

### Task 6: Verify acceptance criteria

- [ ] `go vet ./...` and `go test ./...` (full suite) pass
- [ ] start the server locally (`go run ./cmd/server`) with a minimal config containing the ElevenLabs provider; confirm:
  - [ ] `GET /api/v1/providers/elevenlabs/models` returns 200 with `{provider, models[]}` (run via curl). Models include `eleven_multilingual_v2`. Unknown provider returns 404 `PROVIDER_NOT_FOUND`
  - [ ] `POST /api/v1/tts` with `{"text":"hello","model_id":"eleven_flash_v2_5"}` (or similar valid model id) returns audio. Without `model_id`, still works using default
  - [ ] `POST /api/v1/jobs` with `model_id` enqueues a job that carries the model id (verify via `GET /api/v1/jobs/{jobID}` response if it surfaces; otherwise add a debug log line and check)
- [ ] visit `/ui/` in a browser (or skip with note if not automatable):
  - [ ] Model dropdown populates after page load and after changing provider
  - [ ] Advanced section renders elevenlabs sliders + checkbox; submitting with the section open includes `voice_settings` in the request
  - [ ] Submitting with the section closed produces a request without `voice_settings`
- [ ] confirm `/openapi.json` includes the `/api/v1/providers/{name}/models` path and the `Model` + `ModelsListResponse` schemas, plus `model_id` is present on `TTSRequest` and `JobCreateRequest`
- [ ] verify existing tests still pass (no regressions in the voices endpoint, the existing UI route, or the sync TTS path)

### Task 7: [Final] Update documentation and finalize

**Files:**
- Modify: `README.md`
- Modify: `docs/todo/todo.md` (remove or strike-through completed items; or remove the file if empty)

- [ ] add a short note to `README.md` mentioning the new `model_id` config field, the new `/api/v1/providers/{name}/models` endpoint, the optional `model_id` request field, and that the UI now exposes Model + Advanced controls
- [ ] update `docs/todo/todo.md`: check off the completed items (or delete them if the file becomes empty). If the file ends up empty, leave it with just the `# TODO` header so future ideas have a home
- [ ] move this plan to `docs/plans/completed/` (`mkdir -p docs/plans/completed && mv docs/plans/20260426-elevenlabs-models-and-ui-advanced.md docs/plans/completed/`)

## Post-Completion

*Items requiring manual intervention or external systems — informational only*

**Manual verification:**
- Browser smoke test in Chrome and Safari: pick provider → pick model → open Advanced → tweak sliders → submit → audio plays. Confirm `<details>` collapse animation and that closing the section drops `voice_settings` from the next request.
- Try an invalid `model_id` in the API (e.g. `"does_not_exist"`) and confirm the surfaced error is the upstream ElevenLabs error (503 with the upstream message), not a generic timeout.
- Confirm the selfhosted provider's existing `ListVoices` endpoint (`GET /api/v1/providers/<name>/voices`) is unaffected by this change. The new `ListModels` endpoint should return `{provider, models: []}` for selfhosted (deliberately empty — see selfhosted decision in Context). The UI's Model dropdown for selfhosted should show only "Default model".

**Known limitations / out of scope:**
- The `ADVANCED_SCHEMAS` JS map is hand-authored per provider. A future enhancement could derive it from a server-side per-provider capabilities endpoint (e.g. `GET /api/v1/providers/{name}/capabilities`) so the UI auto-discovers controls. Not in scope.
- `domain.VoiceSettings.Speed` is still not wired into the elevenlabs `TTSRequest` (separate bug noted in `docs/research/research-elevenlab.md`). Out of scope for this plan; tracked separately.
- The selfhosted provider's `Synthesize` continues to use the heuristic of `voice_id` < 20 chars implying a model id. If `req.ModelID` is set explicitly, it now takes precedence; otherwise the legacy heuristic remains. Documented behavior, not a regression.
- Per-provider DEFAULT model surfacing: the UI shows "Default model" as the placeholder option but doesn't display *which* model the server will use. Acceptable for v1; revisit if users find it confusing.

**Optional follow-ups (not in scope):**
- Expose remaining ElevenLabs TTS body params (`language_code`, `seed`, `previous_text`/`next_text`, `apply_text_normalization`, `pronunciation_dictionary_locators`) in the Advanced section (see `docs/research/research-elevenlab.md`).
- Wire `domain.VoiceSettings.Speed` into the elevenlabs `VoiceSettingsReq` (the documented bug).
- Surface the configured per-provider default model in the UI's "Default model" placeholder (e.g. `"Default model (eleven_multilingual_v2)"`) so users know what they'll get without picking explicitly. Requires a small extension to `GET /api/v1/providers` to include `default_model_id`.
