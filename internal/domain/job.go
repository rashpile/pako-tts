package domain

import (
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the current state of a TTS job.
type JobStatus string

const (
	JobStatusQueued     JobStatus = "queued"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// Job represents a TTS synthesis request submitted for processing.
type Job struct {
	ID                    string         `json:"job_id"`
	Status                JobStatus      `json:"status"`
	Text                  string         `json:"text,omitempty"`
	VoiceID               string         `json:"voice_id"`
	ProviderName          string         `json:"provider_name"`
	OutputFormat          string         `json:"output_format"`
	VoiceSettings         *VoiceSettings `json:"voice_settings,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
	StartedAt             *time.Time     `json:"started_at,omitempty"`
	CompletedAt           *time.Time     `json:"completed_at,omitempty"`
	ProgressPercentage    float64        `json:"progress_percentage"`
	EstimatedCompletionAt *time.Time     `json:"estimated_completion_at,omitempty"`
	ErrorMessage          string         `json:"error_message,omitempty"`
	ResultPath            string         `json:"result_path,omitempty"`
	ExpiresAt             *time.Time     `json:"expires_at,omitempty"`
}

// NewJob creates a new job with default values.
func NewJob(text, voiceID, providerName, outputFormat string, settings *VoiceSettings) *Job {
	return &Job{
		ID:                 uuid.New().String(),
		Status:             JobStatusQueued,
		Text:               text,
		VoiceID:            voiceID,
		ProviderName:       providerName,
		OutputFormat:       outputFormat,
		VoiceSettings:      settings,
		CreatedAt:          time.Now().UTC(),
		ProgressPercentage: 0,
	}
}

// SetProcessing marks the job as processing.
func (j *Job) SetProcessing() {
	now := time.Now().UTC()
	j.Status = JobStatusProcessing
	j.StartedAt = &now
}

// SetCompleted marks the job as completed with the result path.
func (j *Job) SetCompleted(resultPath string, retentionHours int) {
	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(retentionHours) * time.Hour)
	j.Status = JobStatusCompleted
	j.CompletedAt = &now
	j.ResultPath = resultPath
	j.ExpiresAt = &expiresAt
	j.ProgressPercentage = 100
}

// SetFailed marks the job as failed with an error message.
func (j *Job) SetFailed(errMsg string) {
	now := time.Now().UTC()
	j.Status = JobStatusFailed
	j.CompletedAt = &now
	j.ErrorMessage = errMsg
}

// UpdateProgress updates the job's progress percentage and estimated completion.
func (j *Job) UpdateProgress(percentage float64, estimatedCompletion *time.Time) {
	j.ProgressPercentage = percentage
	j.EstimatedCompletionAt = estimatedCompletion
}

// IsExpired checks if the job result has expired.
func (j *Job) IsExpired() bool {
	if j.ExpiresAt == nil {
		return false
	}
	return time.Now().UTC().After(*j.ExpiresAt)
}

// IsComplete checks if the job has finished (completed or failed).
func (j *Job) IsComplete() bool {
	return j.Status == JobStatusCompleted || j.Status == JobStatusFailed
}
