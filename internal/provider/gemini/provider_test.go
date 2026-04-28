package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pako-tts/server/internal/domain"
	"github.com/pako-tts/server/pkg/config"
)

// silentPCM returns nBytes of silence (zeros) suitable for ffmpeg encoding.
func silentPCM(nBytes int) []byte {
	return make([]byte, nBytes)
}

// newTestProvider creates a Provider backed by a test server that returns the given PCM.
func newTestProvider(t *testing.T, pcm []byte, isDefault bool) (*Provider, func()) {
	t.Helper()
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(audioResponse(t, pcm)))
	})
	p := &Provider{
		client:         client,
		defaultModelID: defaultModelID,
		isDefault:      isDefault,
	}
	return p, srv.Close
}

// --- NewProviderFromConfig ---

func TestNewProviderFromConfig_MissingAPIKey(t *testing.T) {
	_, err := NewProviderFromConfig(config.ProviderConfig{Type: "gemini"}, false)
	if err == nil {
		t.Fatal("expected error when api_key missing")
	}
	if !strings.Contains(err.Error(), "api_key") {
		t.Errorf("expected api_key mention in error, got: %v", err)
	}
}

func TestNewProviderFromConfig_DefaultModelID(t *testing.T) {
	p, err := NewProviderFromConfig(config.ProviderConfig{APIKey: "key"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.defaultModelID != defaultModelID {
		t.Errorf("expected defaultModelID %q, got %q", defaultModelID, p.defaultModelID)
	}
}

func TestNewProviderFromConfig_CustomModelID(t *testing.T) {
	p, err := NewProviderFromConfig(config.ProviderConfig{APIKey: "key", ModelID: "custom-model"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.defaultModelID != "custom-model" {
		t.Errorf("expected custom-model, got %q", p.defaultModelID)
	}
}

func TestNewProviderFromConfig_DefaultStyleWiredThrough(t *testing.T) {
	p, err := NewProviderFromConfig(config.ProviderConfig{APIKey: "key", DefaultStyle: "calm"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.defaultStyle != "calm" {
		t.Errorf("expected defaultStyle 'calm', got %q", p.defaultStyle)
	}
}

// --- Name / Type ---

func TestProvider_Name(t *testing.T) {
	p := NewProvider("key", false)
	if p.Name() != "gemini" {
		t.Errorf("expected Name 'gemini', got %q", p.Name())
	}
}

func TestProvider_Type(t *testing.T) {
	p := NewProvider("key", false)
	if p.Type() != "GeminiProvider" {
		t.Errorf("expected Type 'GeminiProvider', got %q", p.Type())
	}
}

// --- MaxConcurrent / ActiveJobs ---

func TestProvider_MaxConcurrent(t *testing.T) {
	p := NewProvider("key", false)
	if p.MaxConcurrent() != 4 {
		t.Errorf("expected MaxConcurrent 4, got %d", p.MaxConcurrent())
	}
}

func TestProvider_ActiveJobs_Initial(t *testing.T) {
	p := NewProvider("key", false)
	if p.ActiveJobs() != 0 {
		t.Errorf("expected ActiveJobs 0, got %d", p.ActiveJobs())
	}
}

// --- Info ---

func TestProvider_Info_IsDefault(t *testing.T) {
	p, cleanup := newTestProvider(t, cannedPCM(), true)
	defer cleanup()

	info := p.Info(context.Background())
	if info.Name != "gemini" {
		t.Errorf("Info.Name: got %q, want 'gemini'", info.Name)
	}
	if info.Type != "GeminiProvider" {
		t.Errorf("Info.Type: got %q, want 'GeminiProvider'", info.Type)
	}
	if info.MaxConcurrent != 4 {
		t.Errorf("Info.MaxConcurrent: got %d, want 4", info.MaxConcurrent)
	}
	if !info.IsDefault {
		t.Error("Info.IsDefault: expected true")
	}
}

func TestProvider_Info_NonDefault(t *testing.T) {
	p, cleanup := newTestProvider(t, cannedPCM(), false)
	defer cleanup()

	info := p.Info(context.Background())
	if info.IsDefault {
		t.Error("Info.IsDefault: expected false for non-default provider")
	}
}

// --- ListVoices ---

func TestProvider_ListVoices_Count(t *testing.T) {
	p := NewProvider("key", false)
	voices, err := p.ListVoices(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(voices) != 30 {
		t.Errorf("expected 30 voices, got %d", len(voices))
	}
}

func TestProvider_ListVoices_LanguageEmpty(t *testing.T) {
	p := NewProvider("key", false)
	voices, err := p.ListVoices(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range voices {
		if v.Language != "" {
			t.Errorf("voice %q: expected empty Language, got %q", v.Name, v.Language)
		}
	}
}

// --- ListModels ---

func TestProvider_ListModels_OneEntry(t *testing.T) {
	p := NewProvider("key", false)
	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if models[0].ModelID != defaultModelID {
		t.Errorf("ModelID: got %q, want %q", models[0].ModelID, defaultModelID)
	}
	if len(models[0].Languages) == 0 {
		t.Error("expected non-empty Languages")
	}
}

// --- buildPrompt ---

func TestBuildPrompt_NoLangNoStyle(t *testing.T) {
	p := &Provider{}
	got := p.buildPrompt(&domain.SynthesisRequest{Text: "hello"})
	if got != "hello" {
		t.Errorf("expected bare text, got %q", got)
	}
}

func TestBuildPrompt_LangOnly(t *testing.T) {
	p := &Provider{}
	got := p.buildPrompt(&domain.SynthesisRequest{Text: "salut", LanguageCode: "ro"})
	want := "Speak in Romanian.\n\nsalut"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildPrompt_StyleOnly_FromDefault(t *testing.T) {
	p := &Provider{defaultStyle: "warm"}
	got := p.buildPrompt(&domain.SynthesisRequest{Text: "hello"})
	want := "Style: warm.\n\nhello"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildPrompt_LangAndStyle(t *testing.T) {
	p := &Provider{}
	got := p.buildPrompt(&domain.SynthesisRequest{
		Text:         "bonjour",
		LanguageCode: "fr",
		Settings:     &domain.VoiceSettings{StyleInstructions: "cheerful"},
	})
	want := "Speak in French.\nStyle: cheerful.\n\nbonjour"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildPrompt_RequestStyleOverridesDefault(t *testing.T) {
	p := &Provider{defaultStyle: "calm"}
	got := p.buildPrompt(&domain.SynthesisRequest{
		Text:     "hello",
		Settings: &domain.VoiceSettings{StyleInstructions: "excited"},
	})
	if !strings.Contains(got, "excited") {
		t.Errorf("expected per-request style 'excited' in prompt, got %q", got)
	}
	if strings.Contains(got, "calm") {
		t.Errorf("per-request style should override default 'calm', got %q", got)
	}
}

func TestBuildPrompt_DefaultStyleUsedWhenRequestEmpty(t *testing.T) {
	p := &Provider{defaultStyle: "slow"}
	got := p.buildPrompt(&domain.SynthesisRequest{
		Text:     "hello",
		Settings: &domain.VoiceSettings{},
	})
	if !strings.Contains(got, "slow") {
		t.Errorf("expected default style 'slow' in prompt, got %q", got)
	}
}

func TestBuildPrompt_UnknownLangCodeOmitsDirective(t *testing.T) {
	p := &Provider{}
	got := p.buildPrompt(&domain.SynthesisRequest{Text: "hello", LanguageCode: "xx"})
	if got != "hello" {
		t.Errorf("expected bare text for unknown lang code, got %q", got)
	}
}

// --- Synthesize output format ---

func TestProvider_Synthesize_WAV_ContentTypeAndHeader(t *testing.T) {
	// 100 ms of silence at 24kHz mono 16-bit = 4800 bytes
	p, cleanup := newTestProvider(t, silentPCM(4800), false)
	defer cleanup()

	result, err := p.Synthesize(context.Background(), &domain.SynthesisRequest{
		Text:         "hello",
		VoiceID:      "Despina",
		OutputFormat: "wav",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ContentType != "audio/wav" {
		t.Errorf("ContentType: got %q, want 'audio/wav'", result.ContentType)
	}

	buf := make([]byte, 4)
	if _, err := io.ReadFull(result.Audio, buf); err != nil {
		t.Fatalf("read RIFF header: %v", err)
	}
	if string(buf) != "RIFF" {
		t.Errorf("expected RIFF header, got %q", string(buf))
	}
}

func TestProvider_Synthesize_MP3_ContentTypeAndSignature(t *testing.T) {
	// 100 ms of silence at 24kHz mono 16-bit
	p, cleanup := newTestProvider(t, silentPCM(4800), false)
	defer cleanup()

	result, err := p.Synthesize(context.Background(), &domain.SynthesisRequest{
		Text:         "hello",
		VoiceID:      "Despina",
		OutputFormat: "mp3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ContentType != "audio/mpeg" {
		t.Errorf("ContentType: got %q, want 'audio/mpeg'", result.ContentType)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, result.Audio); err != nil {
		t.Fatalf("read audio: %v", err)
	}
	data := buf.Bytes()
	if len(data) < 3 {
		t.Fatal("MP3 output too short")
	}
	// Valid MP3 starts with either an ID3 tag ("ID3") or an MPEG sync word (0xFF 0xEx).
	hasID3 := data[0] == 0x49 && data[1] == 0x44 && data[2] == 0x33
	hasSyncWord := data[0] == 0xFF && (data[1]&0xE0) == 0xE0
	if !hasID3 && !hasSyncWord {
		t.Errorf("expected MP3 signature (ID3 or 0xFF sync), got 0x%02X 0x%02X 0x%02X", data[0], data[1], data[2])
	}
}

func TestProvider_Synthesize_DefaultFormatIsMP3(t *testing.T) {
	p, cleanup := newTestProvider(t, silentPCM(4800), false)
	defer cleanup()

	result, err := p.Synthesize(context.Background(), &domain.SynthesisRequest{
		Text:    "hello",
		VoiceID: "Despina",
		// OutputFormat empty → should default to mp3
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ContentType != "audio/mpeg" {
		t.Errorf("expected default format audio/mpeg, got %q", result.ContentType)
	}
}

func TestProvider_Synthesize_UnknownVoiceIDFallsBackToDefault(t *testing.T) {
	var capturedVoice string
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req TTSRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			capturedVoice = req.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(audioResponse(t, silentPCM(4800))))
	})
	defer srv.Close()

	p := &Provider{client: client, defaultModelID: defaultModelID}
	_, err := p.Synthesize(context.Background(), &domain.SynthesisRequest{
		Text:    "hello",
		VoiceID: "pNInz6obpgDQGcFmaJgB", // ElevenLabs ID injected by handler default
	})
	if err != nil {
		t.Fatalf("unexpected error for non-Gemini voice ID: %v", err)
	}
	if capturedVoice != defaultVoiceName {
		t.Errorf("expected fallback to default voice %q, got %q", defaultVoiceName, capturedVoice)
	}
}

func TestProvider_Synthesize_EmptyVoiceIDUsesDefault(t *testing.T) {
	var capturedVoice string
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req TTSRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			capturedVoice = req.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(audioResponse(t, silentPCM(4800))))
	})
	defer srv.Close()

	p := &Provider{client: client, defaultModelID: defaultModelID}
	_, err := p.Synthesize(context.Background(), &domain.SynthesisRequest{
		Text:    "hello",
		VoiceID: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedVoice != defaultVoiceName {
		t.Errorf("expected default voice %q, got %q", defaultVoiceName, capturedVoice)
	}
}

func TestProvider_Synthesize_UpstreamErrorPropagates(t *testing.T) {
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"quota exceeded"}}`, http.StatusTooManyRequests)
	})
	defer srv.Close()

	p := &Provider{client: client, defaultModelID: defaultModelID}
	_, err := p.Synthesize(context.Background(), &domain.SynthesisRequest{
		Text:    "hello",
		VoiceID: "Despina",
	})
	if err == nil {
		t.Fatal("expected error from upstream, got nil")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("expected 429 in error, got: %v", err)
	}
}

func TestProvider_Synthesize_PromptSentToUpstream(t *testing.T) {
	var capturedPrompt string
	var capturedVoice string

	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var req TTSRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(req.Contents) > 0 && len(req.Contents[0].Parts) > 0 {
			capturedPrompt = req.Contents[0].Parts[0].Text
		}
		capturedVoice = req.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(audioResponse(t, silentPCM(4800))))
	})
	defer srv.Close()

	p := &Provider{
		client:         client,
		defaultModelID: defaultModelID,
		defaultStyle:   "calm",
	}
	_, err := p.Synthesize(context.Background(), &domain.SynthesisRequest{
		Text:         "hello",
		VoiceID:      "Aoede",
		LanguageCode: "en",
		OutputFormat: "wav",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantPrompt := "Speak in English.\nStyle: calm.\n\nhello"
	if capturedPrompt != wantPrompt {
		t.Errorf("prompt: got %q, want %q", capturedPrompt, wantPrompt)
	}
	if capturedVoice != "Aoede" {
		t.Errorf("voice: got %q, want 'Aoede'", capturedVoice)
	}
}
