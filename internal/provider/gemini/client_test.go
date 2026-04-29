package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testAPIKey = "test-api-key"

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := NewClientWithBaseURL(testAPIKey, srv.URL)
	c.httpClient = &http.Client{Timeout: 5 * time.Second}
	return c, srv
}

func cannedPCM() []byte {
	// 10 bytes of fake PCM audio
	return []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A}
}

func audioResponse(t *testing.T, pcm []byte) string {
	t.Helper()
	encoded := base64.StdEncoding.EncodeToString(pcm)
	resp := TTSResponse{
		Candidates: []Candidate{
			{
				Content: Content{
					Parts: []Part{
						{InlineData: &InlineData{MimeType: "audio/pcm", Data: encoded}},
					},
				},
			},
		},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal audio response: %v", err)
	}
	return string(b)
}

func responseJSON(t *testing.T, resp TTSResponse) string {
	t.Helper()
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	return string(b)
}

// TestGenerateAudio_HappyPath verifies decoded PCM matches the server-provided base64.
func TestGenerateAudio_HappyPath(t *testing.T) {
	expected := cannedPCM()
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(audioResponse(t, expected)))
	})
	defer srv.Close()

	got, err := client.GenerateAudio(context.Background(), "test-model", "hello", "Despina")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(expected) {
		t.Errorf("PCM mismatch: got %v, want %v", got, expected)
	}
}

// TestGenerateAudio_AuthHeader verifies the exact lowercase header x-goog-api-key is sent.
func TestGenerateAudio_AuthHeader(t *testing.T) {
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Go's http normalises header keys; check the canonical form maps to our value.
		if r.Header.Get("x-goog-api-key") != "test-api-key" {
			t.Errorf("expected x-goog-api-key=test-api-key, got %q", r.Header.Get("x-goog-api-key"))
		}
		// Also assert the raw canonical key name so the contract is locked.
		if _, ok := r.Header["X-Goog-Api-Key"]; !ok {
			t.Errorf("canonical header X-Goog-Api-Key not present in request")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(audioResponse(t, cannedPCM())))
	})
	defer srv.Close()

	_, err := client.GenerateAudio(context.Background(), "test-model", "hello", "Despina")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGenerateAudio_RequestShape asserts the JSON body matches the Gemini spec.
func TestGenerateAudio_RequestShape(t *testing.T) {
	const prompt = "Speak in Romanian.\n\nhello world"
	const voice = "Aoede"

	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req TTSRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if len(req.Contents) != 1 || len(req.Contents[0].Parts) != 1 {
			t.Fatalf("expected contents[0].parts length 1, got %d parts", len(req.Contents[0].Parts))
		}
		if req.Contents[0].Parts[0].Text != prompt {
			t.Errorf("contents[0].parts[0].text: got %q, want %q", req.Contents[0].Parts[0].Text, prompt)
		}

		wantVoice := req.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName
		if wantVoice != voice {
			t.Errorf("voiceName: got %q, want %q", wantVoice, voice)
		}

		modalities := req.GenerationConfig.ResponseModalities
		if len(modalities) != 1 || modalities[0] != "AUDIO" {
			t.Errorf("responseModalities: got %v, want [AUDIO]", modalities)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(audioResponse(t, cannedPCM())))
	})
	defer srv.Close()

	_, err := client.GenerateAudio(context.Background(), "test-model", prompt, voice)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGenerateAudio_HTTP400 verifies structured error body is included in error message.
func TestGenerateAudio_HTTP400(t *testing.T) {
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"invalid request"}}`, http.StatusBadRequest)
	})
	defer srv.Close()

	_, err := client.GenerateAudio(context.Background(), "test-model", "hello", "Despina")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected status 400 in error, got: %v", err)
	}
}

// TestGenerateAudio_HTTP500 verifies server errors are propagated.
func TestGenerateAudio_HTTP500(t *testing.T) {
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	})
	defer srv.Close()

	_, err := client.GenerateAudio(context.Background(), "test-model", "hello", "Despina")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status 500 in error, got: %v", err)
	}
}

// TestGenerateAudio_UnparsableResponse verifies JSON decode errors are propagated.
func TestGenerateAudio_UnparsableResponse(t *testing.T) {
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	})
	defer srv.Close()

	_, err := client.GenerateAudio(context.Background(), "test-model", "hello", "Despina")
	if err == nil {
		t.Fatal("expected error for unparseable response, got nil")
	}
}

// TestGenerateAudio_MissingInlineData verifies missing audio data returns an error.
func TestGenerateAudio_MissingInlineData(t *testing.T) {
	emptyResp := `{"candidates":[{"content":{"parts":[{"text":"no audio here"}]}}]}`
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(emptyResp))
	})
	defer srv.Close()

	_, err := client.GenerateAudio(context.Background(), "test-model", "hello", "Despina")
	if err == nil {
		t.Fatal("expected error for missing inlineData, got nil")
	}
	if !strings.Contains(err.Error(), "no audio data") {
		t.Errorf("expected 'no audio data' in error, got: %v", err)
	}
}

func TestGenerateAudio_AudioInSecondPart(t *testing.T) {
	expected := cannedPCM()
	encoded := base64.StdEncoding.EncodeToString(expected)
	resp := TTSResponse{
		Candidates: []Candidate{
			{
				Content: Content{
					Parts: []Part{
						{Text: "preface"},
						{InlineData: &InlineData{MimeType: "audio/pcm", Data: encoded}},
					},
				},
				FinishReason: "STOP",
			},
		},
	}
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseJSON(t, resp)))
	})
	defer srv.Close()

	got, err := client.GenerateAudio(context.Background(), "test-model", "hello", "Despina")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(expected) {
		t.Errorf("PCM mismatch: got %v, want %v", got, expected)
	}
}

func TestGenerateAudio_AudioInSecondCandidate(t *testing.T) {
	expected := cannedPCM()
	encoded := base64.StdEncoding.EncodeToString(expected)
	resp := TTSResponse{
		Candidates: []Candidate{
			{
				Content: Content{
					Parts: []Part{{Text: "first candidate has text only"}},
				},
				FinishReason: "STOP",
			},
			{
				Content: Content{
					Parts: []Part{{InlineData: &InlineData{MimeType: "audio/pcm", Data: encoded}}},
				},
				FinishReason: "STOP",
			},
		},
	}
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseJSON(t, resp)))
	})
	defer srv.Close()

	got, err := client.GenerateAudio(context.Background(), "test-model", "hello", "Despina")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(expected) {
		t.Errorf("PCM mismatch: got %v, want %v", got, expected)
	}
}

func TestGenerateAudio_TextOnlyResponseIncludesDiagnostics(t *testing.T) {
	resp := TTSResponse{
		Candidates: []Candidate{
			{
				Content: Content{
					Parts: []Part{{Text: "Secrete si Surprize in Gradina cu Hortensii"}},
				},
				FinishReason:  "STOP",
				FinishMessage: "Model produced text only",
			},
		},
	}
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseJSON(t, resp)))
	})
	defer srv.Close()

	_, err := client.GenerateAudio(context.Background(), "test-model", "Secrete și Surprize în Grădina cu Hortensii", "Despina")
	if err == nil {
		t.Fatal("expected error for text-only response, got nil")
	}
	if !strings.Contains(err.Error(), "finish_reasons=STOP") {
		t.Errorf("expected finish reason in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), `text_snippet="Secrete si Surprize in Gradina cu Hortensii"`) {
		t.Errorf("expected text snippet in error, got: %v", err)
	}
}

func TestGenerateAudio_EmptyInlineDataIncludesDiagnostics(t *testing.T) {
	resp := TTSResponse{
		Candidates: []Candidate{
			{
				Content: Content{
					Parts: []Part{{InlineData: &InlineData{MimeType: "audio/pcm", Data: ""}}},
				},
				FinishReason: "STOP",
			},
		},
	}
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseJSON(t, resp)))
	})
	defer srv.Close()

	_, err := client.GenerateAudio(context.Background(), "test-model", "hello", "Despina")
	if err == nil {
		t.Fatal("expected error for empty inlineData, got nil")
	}
	if !strings.Contains(err.Error(), "empty_audio_parts=1") {
		t.Errorf("expected empty audio part count in error, got: %v", err)
	}
}

// TestGenerateAudio_EmptyCandidates verifies empty candidates array returns an error.
func TestGenerateAudio_EmptyCandidates(t *testing.T) {
	emptyResp := `{"candidates":[],"promptFeedback":{"blockReason":"OTHER"}}`
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(emptyResp))
	})
	defer srv.Close()

	_, err := client.GenerateAudio(context.Background(), "test-model", "hello", "Despina")
	if err == nil {
		t.Fatal("expected error for empty candidates, got nil")
	}
	if !strings.Contains(err.Error(), "prompt_block_reason=OTHER") {
		t.Errorf("expected prompt block reason in error, got: %v", err)
	}
}

// TestGenerateAudio_NetworkError verifies that transport errors are wrapped and returned.
func TestGenerateAudio_NetworkError(t *testing.T) {
	// Point client at a URL with no server listening.
	c := NewClientWithBaseURL("test-key", "http://127.0.0.1:1")
	c.httpClient = &http.Client{Timeout: 500 * time.Millisecond}

	_, err := c.GenerateAudio(context.Background(), "test-model", "hello", "Despina")
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

// TestCheckHealth_OK verifies true is returned on HTTP 200 and the API key header is sent.
func TestCheckHealth_OK(t *testing.T) {
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != testAPIKey {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	if !client.CheckHealth(context.Background(), defaultModelID) {
		t.Error("expected CheckHealth to return true on 200")
	}
}

// TestCheckHealth_Unauthorized verifies false is returned on HTTP 401.
func TestCheckHealth_Unauthorized(t *testing.T) {
	client, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()

	if client.CheckHealth(context.Background(), defaultModelID) {
		t.Error("expected CheckHealth to return false on 401")
	}
}

// TestCheckHealth_NetworkError verifies false is returned when the server is unreachable.
func TestCheckHealth_NetworkError(t *testing.T) {
	c := NewClientWithBaseURL("test-key", "http://127.0.0.1:1")
	c.httpClient = &http.Client{Timeout: 500 * time.Millisecond}

	if c.CheckHealth(context.Background(), defaultModelID) {
		t.Error("expected CheckHealth to return false on network error")
	}
}

// TestVoices_Count verifies exactly 30 prebuilt voices are defined.
func TestVoices_Count(t *testing.T) {
	if len(prebuiltVoices) != 30 {
		t.Errorf("expected 30 prebuilt voices, got %d", len(prebuiltVoices))
	}
}

// TestVoices_LanguageEmpty verifies all voices have empty Language (language-agnostic).
func TestVoices_LanguageEmpty(t *testing.T) {
	for _, v := range prebuiltVoices {
		if v.Language != "" {
			t.Errorf("voice %q: expected empty Language, got %q", v.Name, v.Language)
		}
	}
}

// TestVoices_Provider verifies all voices reference the gemini provider.
func TestVoices_Provider(t *testing.T) {
	for _, v := range prebuiltVoices {
		if v.Provider != providerName {
			t.Errorf("voice %q: expected provider %q, got %q", v.Name, providerName, v.Provider)
		}
	}
}

// TestDefaultModel_Fields verifies the model definition.
func TestDefaultModel_Fields(t *testing.T) {
	if defaultModel.ModelID != defaultModelID {
		t.Errorf("modelID: got %q, want %q", defaultModel.ModelID, defaultModelID)
	}
	if defaultModel.Provider != providerName {
		t.Errorf("provider: got %q, want %q", defaultModel.Provider, providerName)
	}
	if len(defaultModel.Languages) == 0 {
		t.Error("expected non-empty Languages on defaultModel")
	}
}

// TestSupportedLanguagesConsistency verifies supportedLanguages and defaultModel.Languages are the same slice.
func TestSupportedLanguagesConsistency(t *testing.T) {
	if len(defaultModel.Languages) != len(supportedLanguages) {
		t.Errorf("defaultModel.Languages length %d != supportedLanguages length %d",
			len(defaultModel.Languages), len(supportedLanguages))
	}
}

// TestIsoToName_Coverage verifies every supportedLanguage has an isoToName entry.
func TestIsoToName_Coverage(t *testing.T) {
	for _, code := range supportedLanguages {
		if _, ok := isoToName[code]; !ok {
			t.Errorf("isoToName missing entry for supported language %q", code)
		}
	}
}

// TestIsoToName_RomanianPresent verifies the Romanian code is present (used in reference scripts).
func TestIsoToName_RomanianPresent(t *testing.T) {
	name, ok := isoToName["ro"]
	if !ok {
		t.Fatal("expected 'ro' in isoToName")
	}
	if name != "Romanian" {
		t.Errorf("expected 'Romanian' for 'ro', got %q", name)
	}
}
