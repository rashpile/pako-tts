package domain

import (
	"testing"
	"time"
)

func TestNewJob(t *testing.T) {
	text := "Hello, world!"
	voiceID := "voice123"
	providerName := "elevenlabs"
	outputFormat := "mp3"

	job := NewJob(text, voiceID, providerName, outputFormat, nil)

	if job.ID == "" {
		t.Error("Expected job ID to be generated")
	}
	if job.Status != JobStatusQueued {
		t.Errorf("Expected status %s, got %s", JobStatusQueued, job.Status)
	}
	if job.Text != text {
		t.Errorf("Expected text %s, got %s", text, job.Text)
	}
	if job.VoiceID != voiceID {
		t.Errorf("Expected voiceID %s, got %s", voiceID, job.VoiceID)
	}
	if job.ProviderName != providerName {
		t.Errorf("Expected providerName %s, got %s", providerName, job.ProviderName)
	}
	if job.OutputFormat != outputFormat {
		t.Errorf("Expected outputFormat %s, got %s", outputFormat, job.OutputFormat)
	}
	if job.ProgressPercentage != 0 {
		t.Errorf("Expected progress 0, got %f", job.ProgressPercentage)
	}
	if job.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

func TestNewJobWithSettings(t *testing.T) {
	settings := DefaultVoiceSettings()
	job := NewJob("test", "voice", "provider", "mp3", settings)

	if job.VoiceSettings == nil {
		t.Error("Expected VoiceSettings to be set")
	}
}

func TestJob_SetProcessing(t *testing.T) {
	job := NewJob("test", "voice", "provider", "mp3", nil)

	job.SetProcessing()

	if job.Status != JobStatusProcessing {
		t.Errorf("Expected status %s, got %s", JobStatusProcessing, job.Status)
	}
	if job.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	}
}

func TestJob_SetCompleted(t *testing.T) {
	job := NewJob("test", "voice", "provider", "mp3", nil)
	resultPath := "/storage/audio/test.mp3"
	retentionHours := 24

	job.SetCompleted(resultPath, retentionHours)

	if job.Status != JobStatusCompleted {
		t.Errorf("Expected status %s, got %s", JobStatusCompleted, job.Status)
	}
	if job.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
	if job.ResultPath != resultPath {
		t.Errorf("Expected resultPath %s, got %s", resultPath, job.ResultPath)
	}
	if job.ExpiresAt == nil {
		t.Error("Expected ExpiresAt to be set")
	}
	if job.ProgressPercentage != 100 {
		t.Errorf("Expected progress 100, got %f", job.ProgressPercentage)
	}
}

func TestJob_SetFailed(t *testing.T) {
	job := NewJob("test", "voice", "provider", "mp3", nil)
	errMsg := "synthesis failed"

	job.SetFailed(errMsg)

	if job.Status != JobStatusFailed {
		t.Errorf("Expected status %s, got %s", JobStatusFailed, job.Status)
	}
	if job.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
	if job.ErrorMessage != errMsg {
		t.Errorf("Expected errorMessage %s, got %s", errMsg, job.ErrorMessage)
	}
}

func TestJob_UpdateProgress(t *testing.T) {
	job := NewJob("test", "voice", "provider", "mp3", nil)
	percentage := 50.0
	estimatedCompletion := time.Now().Add(10 * time.Second)

	job.UpdateProgress(percentage, &estimatedCompletion)

	if job.ProgressPercentage != percentage {
		t.Errorf("Expected progress %f, got %f", percentage, job.ProgressPercentage)
	}
	if job.EstimatedCompletionAt == nil {
		t.Error("Expected EstimatedCompletionAt to be set")
	}
}

func TestJob_IsExpired(t *testing.T) {
	t.Run("nil ExpiresAt", func(t *testing.T) {
		job := NewJob("test", "voice", "provider", "mp3", nil)
		if job.IsExpired() {
			t.Error("Expected job with nil ExpiresAt to not be expired")
		}
	})

	t.Run("not expired", func(t *testing.T) {
		job := NewJob("test", "voice", "provider", "mp3", nil)
		future := time.Now().Add(1 * time.Hour)
		job.ExpiresAt = &future

		if job.IsExpired() {
			t.Error("Expected job with future ExpiresAt to not be expired")
		}
	})

	t.Run("expired", func(t *testing.T) {
		job := NewJob("test", "voice", "provider", "mp3", nil)
		past := time.Now().Add(-1 * time.Hour)
		job.ExpiresAt = &past

		if !job.IsExpired() {
			t.Error("Expected job with past ExpiresAt to be expired")
		}
	})
}

func TestJob_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		status   JobStatus
		expected bool
	}{
		{"queued", JobStatusQueued, false},
		{"processing", JobStatusProcessing, false},
		{"completed", JobStatusCompleted, true},
		{"failed", JobStatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := NewJob("test", "voice", "provider", "mp3", nil)
			job.Status = tt.status

			if job.IsComplete() != tt.expected {
				t.Errorf("Expected IsComplete() to be %v for status %s", tt.expected, tt.status)
			}
		})
	}
}
