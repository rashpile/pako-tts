// Package memory provides an in-memory job queue implementation.
package memory

import (
	"context"
	"sync"

	"github.com/pako-tts/server/internal/domain"
)

// Queue is an in-memory implementation of domain.JobQueue.
type Queue struct {
	mu      sync.RWMutex
	jobs    map[string]*domain.Job
	pending chan *domain.Job
	closed  bool
}

// NewQueue creates a new in-memory job queue.
func NewQueue(bufferSize int) *Queue {
	return &Queue{
		jobs:    make(map[string]*domain.Job),
		pending: make(chan *domain.Job, bufferSize),
	}
}

// Enqueue adds a job to the queue for processing.
func (q *Queue) Enqueue(ctx context.Context, job *domain.Job) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return context.Canceled
	}
	q.jobs[job.ID] = job
	q.mu.Unlock()

	select {
	case q.pending <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Dequeue retrieves the next job for processing.
func (q *Queue) Dequeue(ctx context.Context) (*domain.Job, error) {
	select {
	case job, ok := <-q.pending:
		if !ok {
			return nil, nil
		}
		return job, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// GetJob retrieves a job by ID.
func (q *Queue) GetJob(ctx context.Context, jobID string) (*domain.Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	job, ok := q.jobs[jobID]
	if !ok {
		return nil, domain.ErrJobNotFound
	}
	return job, nil
}

// UpdateJob updates a job's status and metadata.
func (q *Queue) UpdateJob(ctx context.Context, job *domain.Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.jobs[job.ID]; !ok {
		return domain.ErrJobNotFound
	}
	q.jobs[job.ID] = job
	return nil
}

// ListJobs returns jobs matching the given status.
func (q *Queue) ListJobs(ctx context.Context, status domain.JobStatus) ([]*domain.Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var result []*domain.Job
	for _, job := range q.jobs {
		if job.Status == status {
			result = append(result, job)
		}
	}
	return result, nil
}

// DeleteJob removes a job from the queue.
func (q *Queue) DeleteJob(ctx context.Context, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	delete(q.jobs, jobID)
	return nil
}

// Close shuts down the queue gracefully.
func (q *Queue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		q.closed = true
		close(q.pending)
	}
	return nil
}

// Stats returns current queue statistics.
func (q *Queue) Stats() domain.QueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := domain.QueueStats{}
	for _, job := range q.jobs {
		stats.TotalJobs++
		switch job.Status {
		case domain.JobStatusQueued:
			stats.QueuedJobs++
		case domain.JobStatusProcessing:
			stats.ProcessingJobs++
		case domain.JobStatusCompleted:
			stats.CompletedJobs++
		case domain.JobStatusFailed:
			stats.FailedJobs++
		}
	}
	return stats
}
