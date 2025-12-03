package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/pako-tts/server/internal/api/middleware"
	"github.com/pako-tts/server/internal/domain"
)

// JobsHandler handles job-related requests.
type JobsHandler struct {
	provider       domain.TTSProvider
	queue          domain.JobQueue
	storage        domain.AudioStorage
	logger         *zap.Logger
	defaultVoiceID string
	retentionHours int
}

// NewJobsHandler creates a new jobs handler.
func NewJobsHandler(
	provider domain.TTSProvider,
	queue domain.JobQueue,
	storage domain.AudioStorage,
	logger *zap.Logger,
	defaultVoiceID string,
	retentionHours int,
) *JobsHandler {
	return &JobsHandler{
		provider:       provider,
		queue:          queue,
		storage:        storage,
		logger:         logger,
		defaultVoiceID: defaultVoiceID,
		retentionHours: retentionHours,
	}
}

// JobCreateRequest represents a job creation request.
type JobCreateRequest struct {
	Text          string                `json:"text"`
	VoiceID       string                `json:"voice_id,omitempty"`
	Provider      string                `json:"provider,omitempty"`
	OutputFormat  string                `json:"output_format,omitempty"`
	VoiceSettings *domain.VoiceSettings `json:"voice_settings,omitempty"`
}

// JobCreateResponse represents a job creation response.
type JobCreateResponse struct {
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// JobStatusResponse represents a job status response.
type JobStatusResponse struct {
	JobID                 string  `json:"job_id"`
	Status                string  `json:"status"`
	ProviderName          string  `json:"provider_name"`
	CreatedAt             string  `json:"created_at"`
	StartedAt             *string `json:"started_at,omitempty"`
	CompletedAt           *string `json:"completed_at,omitempty"`
	ProgressPercentage    float64 `json:"progress_percentage"`
	EstimatedCompletionAt *string `json:"estimated_completion_at,omitempty"`
	ErrorMessage          *string `json:"error_message,omitempty"`
}

// SubmitJob handles POST /api/v1/jobs.
func (h *JobsHandler) SubmitJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req JobCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, domain.ErrValidation.WithMessage("Invalid JSON body"))
		return
	}

	// Validate text
	if req.Text == "" {
		middleware.WriteError(w, domain.ErrValidation.WithDetails(map[string]any{
			"field":   "text",
			"message": "Text is required",
		}))
		return
	}

	// Set defaults
	voiceID := req.VoiceID
	if voiceID == "" {
		voiceID = h.defaultVoiceID
	}

	outputFormat := req.OutputFormat
	if outputFormat == "" {
		outputFormat = "mp3"
	}

	// Validate output format
	if outputFormat != "mp3" && outputFormat != "wav" {
		middleware.WriteError(w, domain.ErrInvalidFormat)
		return
	}

	providerName := req.Provider
	if providerName == "" {
		providerName = h.provider.Name()
	}

	// Create job
	job := domain.NewJob(req.Text, voiceID, providerName, outputFormat, req.VoiceSettings)

	// Enqueue job
	if err := h.queue.Enqueue(ctx, job); err != nil {
		h.logger.Error("Failed to enqueue job", zap.Error(err))
		middleware.WriteError(w, domain.ErrInternalServer)
		return
	}

	h.logger.Info("Job created",
		zap.String("job_id", job.ID),
		zap.Int("text_length", len(req.Text)),
	)

	response := JobCreateResponse{
		JobID:     job.ID,
		Status:    string(job.Status),
		CreatedAt: job.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	middleware.WriteJSON(w, http.StatusCreated, response)
}

// GetJobStatus handles GET /api/v1/jobs/{jobID}.
func (h *JobsHandler) GetJobStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jobID := chi.URLParam(r, "jobID")

	job, err := h.queue.GetJob(ctx, jobID)
	if err != nil {
		if apiErr, ok := err.(*domain.APIError); ok {
			middleware.WriteError(w, apiErr)
		} else {
			middleware.WriteError(w, domain.ErrJobNotFound)
		}
		return
	}

	response := JobStatusResponse{
		JobID:              job.ID,
		Status:             string(job.Status),
		ProviderName:       job.ProviderName,
		CreatedAt:          job.CreatedAt.Format("2006-01-02T15:04:05Z"),
		ProgressPercentage: job.ProgressPercentage,
	}

	if job.StartedAt != nil {
		startedAt := job.StartedAt.Format("2006-01-02T15:04:05Z")
		response.StartedAt = &startedAt
	}

	if job.CompletedAt != nil {
		completedAt := job.CompletedAt.Format("2006-01-02T15:04:05Z")
		response.CompletedAt = &completedAt
	}

	if job.EstimatedCompletionAt != nil {
		estimatedAt := job.EstimatedCompletionAt.Format("2006-01-02T15:04:05Z")
		response.EstimatedCompletionAt = &estimatedAt
	}

	if job.ErrorMessage != "" {
		response.ErrorMessage = &job.ErrorMessage
	}

	middleware.WriteJSON(w, http.StatusOK, response)
}

// GetJobResult handles GET /api/v1/jobs/{jobID}/result.
func (h *JobsHandler) GetJobResult(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jobID := chi.URLParam(r, "jobID")

	job, err := h.queue.GetJob(ctx, jobID)
	if err != nil {
		if apiErr, ok := err.(*domain.APIError); ok {
			middleware.WriteError(w, apiErr)
		} else {
			middleware.WriteError(w, domain.ErrJobNotFound)
		}
		return
	}

	// Check if job is complete
	if job.Status != domain.JobStatusCompleted {
		middleware.WriteError(w, domain.ErrJobNotComplete.WithDetails(map[string]any{
			"current_status": string(job.Status),
		}))
		return
	}

	// Check if result has expired
	if job.IsExpired() {
		middleware.WriteError(w, domain.ErrResultExpired)
		return
	}

	// Retrieve audio
	reader, contentType, err := h.storage.Retrieve(ctx, jobID)
	if err != nil {
		h.logger.Error("Failed to retrieve audio", zap.Error(err), zap.String("job_id", jobID))
		middleware.WriteError(w, domain.ErrResultExpired)
		return
	}
	defer reader.Close()

	// Stream audio response
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+jobID+"."+job.OutputFormat+"\"")
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, reader); err != nil {
		h.logger.Error("Failed to write audio response", zap.Error(err))
	}
}
