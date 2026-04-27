package ui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_ServeHTTP(t *testing.T) {
	handler := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/ui/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("expected Content-Type to start with text/html, got %q", contentType)
	}

	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %q", got)
	}

	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("expected X-Content-Type-Options nosniff, got %q", got)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	body := string(bodyBytes)
	if len(body) == 0 {
		t.Fatal("expected non-empty body")
	}

	wantMarkers := []string{
		"<title>Pako TTS</title>",
		`id="provider-select"`,
		`id="voice-select"`,
		`id="model-select"`,
		`id="language-select"`,
		`id="format-select"`,
		`id="advanced-section"`,
		"ADVANCED_SCHEMAS",
		"'ElevenLabsProvider'",
		"/api/v1/tts",
		"'/api/v1/providers'",
		"'/voices'",
		"'/models'",
		"language_code",
		"rebuildLanguageSelect",
		"voiceLanguages",
	}

	for _, marker := range wantMarkers {
		if !strings.Contains(body, marker) {
			t.Errorf("expected body to contain %q", marker)
		}
	}
}
