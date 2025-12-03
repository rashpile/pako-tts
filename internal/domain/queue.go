package domain

import (
	"context"
)

// JobQueue defines the interface for job queue implementations.
// This port allows swapping between in-memory and Redis-backed queues.
type JobQueue interface {
	// Enqueue adds a job to the queue for processing.
	Enqueue(ctx context.Context, job *Job) error

	// Dequeue retrieves the next job for processing (blocking).
	// Returns nil if the queue is closed.
	Dequeue(ctx context.Context) (*Job, error)

	// GetJob retrieves a job by ID.
	GetJob(ctx context.Context, jobID string) (*Job, error)

	// UpdateJob updates a job's status and metadata.
	UpdateJob(ctx context.Context, job *Job) error

	// ListJobs returns jobs matching the given status.
	ListJobs(ctx context.Context, status JobStatus) ([]*Job, error)

	// DeleteJob removes a job from the queue.
	DeleteJob(ctx context.Context, jobID string) error

	// Close shuts down the queue gracefully.
	Close() error

	// Stats returns current queue statistics.
	Stats() QueueStats
}

// QueueStats contains queue statistics for monitoring.
type QueueStats struct {
	TotalJobs      int `json:"total_jobs"`
	QueuedJobs     int `json:"queued_jobs"`
	ProcessingJobs int `json:"processing_jobs"`
	CompletedJobs  int `json:"completed_jobs"`
	FailedJobs     int `json:"failed_jobs"`
}
