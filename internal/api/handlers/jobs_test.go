package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/pako-tts/server/internal/api/handlers/mocks"
	"github.com/pako-tts/server/internal/domain"
	"github.com/pako-tts/server/internal/queue/memory"
)

func TestJobsHandler_SubmitJob(t *testing.T) {
	logger := testLogger()
	mockProvider := &mocks.MockProvider{NameValue: "test-provider"}
	queue := memory.NewQueue(10)
	mockStorage := mocks.NewMockStorage()

	handler := NewJobsHandler(mockProvider, queue, mockStorage, logger, "default-voice", 24)

	reqBody := JobCreateRequest{
		Text:         "Hello, world!",
		VoiceID:      "voice123",
		OutputFormat: "mp3",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SubmitJob(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var jobResp JobCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if jobResp.JobID == "" {
		t.Error("Expected job ID to be set")
	}
	if jobResp.Status != string(domain.JobStatusQueued) {
		t.Errorf("Expected status 'queued', got %s", jobResp.Status)
	}
}

func TestJobsHandler_SubmitJob_InvalidJSON(t *testing.T) {
	logger := testLogger()
	mockProvider := &mocks.MockProvider{NameValue: "test-provider"}
	queue := memory.NewQueue(10)
	mockStorage := mocks.NewMockStorage()

	handler := NewJobsHandler(mockProvider, queue, mockStorage, logger, "default-voice", 24)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SubmitJob(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", resp.StatusCode)
	}
}

func TestJobsHandler_SubmitJob_EmptyText(t *testing.T) {
	logger := testLogger()
	mockProvider := &mocks.MockProvider{NameValue: "test-provider"}
	queue := memory.NewQueue(10)
	mockStorage := mocks.NewMockStorage()

	handler := NewJobsHandler(mockProvider, queue, mockStorage, logger, "default-voice", 24)

	reqBody := JobCreateRequest{
		Text:    "",
		VoiceID: "voice123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SubmitJob(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", resp.StatusCode)
	}
}

func TestJobsHandler_SubmitJob_InvalidFormat(t *testing.T) {
	logger := testLogger()
	mockProvider := &mocks.MockProvider{NameValue: "test-provider"}
	queue := memory.NewQueue(10)
	mockStorage := mocks.NewMockStorage()

	handler := NewJobsHandler(mockProvider, queue, mockStorage, logger, "default-voice", 24)

	reqBody := JobCreateRequest{
		Text:         "Hello",
		VoiceID:      "voice123",
		OutputFormat: "invalid",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SubmitJob(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", resp.StatusCode)
	}
}

func TestJobsHandler_GetJobStatus(t *testing.T) {
	logger := testLogger()
	mockProvider := &mocks.MockProvider{NameValue: "test-provider"}
	queue := memory.NewQueue(10)
	mockStorage := mocks.NewMockStorage()

	handler := NewJobsHandler(mockProvider, queue, mockStorage, logger, "default-voice", 24)

	// Create a job first
	ctx := context.Background()
	job := domain.NewJob("test text", "voice123", "test-provider", "mp3", nil)
	queue.Enqueue(ctx, job)

	// Create request with chi URL params
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+job.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", job.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.GetJobStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var statusResp JobStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if statusResp.JobID != job.ID {
		t.Errorf("Expected job ID %s, got %s", job.ID, statusResp.JobID)
	}
	if statusResp.Status != string(domain.JobStatusQueued) {
		t.Errorf("Expected status 'queued', got %s", statusResp.Status)
	}
}

func TestJobsHandler_GetJobStatus_NotFound(t *testing.T) {
	logger := testLogger()
	mockProvider := &mocks.MockProvider{NameValue: "test-provider"}
	queue := memory.NewQueue(10)
	mockStorage := mocks.NewMockStorage()

	handler := NewJobsHandler(mockProvider, queue, mockStorage, logger, "default-voice", 24)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/non-existent", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", "non-existent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.GetJobStatus(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestJobsHandler_GetJobResult_NotComplete(t *testing.T) {
	logger := testLogger()
	mockProvider := &mocks.MockProvider{NameValue: "test-provider"}
	queue := memory.NewQueue(10)
	mockStorage := mocks.NewMockStorage()

	handler := NewJobsHandler(mockProvider, queue, mockStorage, logger, "default-voice", 24)

	// Create a job (still queued, not completed)
	ctx := context.Background()
	job := domain.NewJob("test text", "voice123", "test-provider", "mp3", nil)
	queue.Enqueue(ctx, job)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+job.ID+"/result", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", job.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.GetJobResult(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooEarly {
		t.Errorf("Expected status 425 (TooEarly), got %d", resp.StatusCode)
	}
}

func TestJobsHandler_GetJobResult_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockProvider := &mocks.MockProvider{NameValue: "test-provider"}
	queue := memory.NewQueue(10)
	mockStorage := mocks.NewMockStorage()

	handler := NewJobsHandler(mockProvider, queue, mockStorage, logger, "default-voice", 24)

	// Create and complete a job
	ctx := context.Background()
	job := domain.NewJob("test text", "voice123", "test-provider", "mp3", nil)
	queue.Enqueue(ctx, job)
	job.SetCompleted("/storage/"+job.ID+".mp3", 24)
	queue.UpdateJob(ctx, job)

	// Store audio data
	mockStorage.StoredFiles[job.ID] = []byte("fake audio content")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/"+job.ID+"/result", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", job.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.GetJobResult(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "audio/mpeg" {
		t.Errorf("Expected Content-Type audio/mpeg, got %s", contentType)
	}
}
