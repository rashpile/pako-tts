## ElevenLabs model selection

- [x] **Add `model_id` to config file** — extend `pkg/config` provider config so ElevenLabs can read a `model_id` value from `config.yaml` / env. Default to `eleven_multilingual_v2`. Pass it through `NewProviderFromConfig` into the client; remove the hardcoded `defaultModel` fallback as the only source.
- [x] **New API endpoint to list models** — add `GET /api/v1/providers/{name}/models` returning the provider's available models (id, name, description, supported languages). For ElevenLabs, fetch from `GET /v1/models`. Mirror the shape of the recently added `/voices` endpoint.
- [x] **Optional `model_id` in TTS request** — accept an optional `model_id` field on `POST /api/v1/tts` and `POST /api/v1/jobs`. Plumb it through `domain.SynthesisRequest` → `elevenlabs.TTSRequest.ModelID`. Falls back to the configured default when omitted. Update OpenAPI spec.
- [x] **UI select for model** — add a Model `<select>` to `internal/ui/index.html` populated from the new models endpoint. Repopulate on provider change (same pattern as the voice dropdown). Send the selected value as `model_id` on submit; leave empty to use the server default.

## UI

- [x] **Show provider-specific parameters in the UI** — surface provider-specific knobs (e.g. ElevenLabs `voice_settings.stability`, `similarity_boost`, `style`, `use_speaker_boost`) in `internal/ui/index.html` so users can experiment without curl. Render the relevant controls when a provider is selected (other providers will get their own param sets). Keep the form layout simple — sliders or number inputs grouped under a collapsible "Advanced" section so the basic flow stays uncluttered. Send chosen values as `voice_settings` in the POST body.
