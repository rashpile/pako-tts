package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/pako-tts/server/internal/api/handlers/mocks"
)

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestHealthCheck(t *testing.T) {
	logger := testLogger()
	mockProvider := &mocks.MockProvider{
		NameValue:      "mock-provider",
		AvailableValue: true,
	}

	handler := NewHealthHandler(mockProvider, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	handler.HealthCheck(w, req)

	resp := w.Result()
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if healthResp.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got %s", healthResp.Status)
	}
	if healthResp.Version == "" {
		t.Error("Expected version to be set")
	}
}
