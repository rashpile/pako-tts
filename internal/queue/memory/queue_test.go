package memory

import (
	"context"
	"testing"
	"time"

	"github.com/pako-tts/server/internal/domain"
)

func TestNewQueue(t *testing.T) {
	queue := NewQueue(10)

	if queue == nil {
		t.Fatal("Expected non-nil queue")
	}
	if queue.jobs == nil {
		t.Error("Expected jobs map to be initialized")
	}
	if queue.pending == nil {
		t.Error("Expected pending channel to be initialized")
	}
}

func TestQueue_Enqueue(t *testing.T) {
	queue := NewQueue(10)
	ctx := context.Background()

	job := domain.NewJob("test", "voice", "provider", "mp3", nil)

	err := queue.Enqueue(ctx, job)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Check job is stored
	storedJob, err := queue.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}
	if storedJob.ID != job.ID {
		t.Error("Stored job ID doesn't match")
	}
}

func TestQueue_Enqueue_ClosedQueue(t *testing.T) {
	queue := NewQueue(10)
	ctx := context.Background()

	queue.Close()

	job := domain.NewJob("test", "voice", "provider", "mp3", nil)
	err := queue.Enqueue(ctx, job)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestQueue_Enqueue_ContextCanceled(t *testing.T) {
	queue := NewQueue(1) // Small buffer
	ctx, cancel := context.WithCancel(context.Background())

	// Fill the buffer
	job1 := domain.NewJob("test1", "voice", "provider", "mp3", nil)
	queue.Enqueue(ctx, job1)

	// Cancel context before second enqueue
	cancel()

	job2 := domain.NewJob("test2", "voice", "provider", "mp3", nil)
	err := queue.Enqueue(ctx, job2)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestQueue_Dequeue(t *testing.T) {
	queue := NewQueue(10)
	ctx := context.Background()

	job := domain.NewJob("test", "voice", "provider", "mp3", nil)
	queue.Enqueue(ctx, job)

	dequeuedJob, err := queue.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Failed to dequeue job: %v", err)
	}
	if dequeuedJob.ID != job.ID {
		t.Error("Dequeued job ID doesn't match")
	}
}

func TestQueue_Dequeue_ClosedQueue(t *testing.T) {
	queue := NewQueue(10)
	ctx := context.Background()

	queue.Close()

	job, err := queue.Dequeue(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if job != nil {
		t.Error("Expected nil job from closed queue")
	}
}

func TestQueue_Dequeue_ContextCanceled(t *testing.T) {
	queue := NewQueue(10)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// No jobs enqueued, should timeout
	_, err := queue.Dequeue(ctx)

	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded error, got %v", err)
	}
}

func TestQueue_GetJob(t *testing.T) {
	queue := NewQueue(10)
	ctx := context.Background()

	job := domain.NewJob("test", "voice", "provider", "mp3", nil)
	queue.Enqueue(ctx, job)

	retrievedJob, err := queue.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}
	if retrievedJob.ID != job.ID {
		t.Error("Retrieved job ID doesn't match")
	}
}

func TestQueue_GetJob_NotFound(t *testing.T) {
	queue := NewQueue(10)
	ctx := context.Background()

	_, err := queue.GetJob(ctx, "non-existent-id")

	if err != domain.ErrJobNotFound {
		t.Errorf("Expected ErrJobNotFound, got %v", err)
	}
}

func TestQueue_UpdateJob(t *testing.T) {
	queue := NewQueue(10)
	ctx := context.Background()

	job := domain.NewJob("test", "voice", "provider", "mp3", nil)
	queue.Enqueue(ctx, job)

	job.SetProcessing()
	err := queue.UpdateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to update job: %v", err)
	}

	updatedJob, _ := queue.GetJob(ctx, job.ID)
	if updatedJob.Status != domain.JobStatusProcessing {
		t.Errorf("Expected status %s, got %s", domain.JobStatusProcessing, updatedJob.Status)
	}
}

func TestQueue_UpdateJob_NotFound(t *testing.T) {
	queue := NewQueue(10)
	ctx := context.Background()

	job := domain.NewJob("test", "voice", "provider", "mp3", nil)
	// Don't enqueue, just try to update

	err := queue.UpdateJob(ctx, job)

	if err != domain.ErrJobNotFound {
		t.Errorf("Expected ErrJobNotFound, got %v", err)
	}
}

func TestQueue_ListJobs(t *testing.T) {
	queue := NewQueue(10)
	ctx := context.Background()

	// Create jobs with different statuses
	job1 := domain.NewJob("test1", "voice", "provider", "mp3", nil)
	job2 := domain.NewJob("test2", "voice", "provider", "mp3", nil)
	job3 := domain.NewJob("test3", "voice", "provider", "mp3", nil)

	queue.Enqueue(ctx, job1)
	queue.Enqueue(ctx, job2)
	queue.Enqueue(ctx, job3)

	// Update job2 to processing
	job2.SetProcessing()
	queue.UpdateJob(ctx, job2)

	// Update job3 to completed
	job3.SetCompleted("/path/to/result", 24)
	queue.UpdateJob(ctx, job3)

	// List queued jobs
	queuedJobs, err := queue.ListJobs(ctx, domain.JobStatusQueued)
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}
	if len(queuedJobs) != 1 {
		t.Errorf("Expected 1 queued job, got %d", len(queuedJobs))
	}

	// List processing jobs
	processingJobs, _ := queue.ListJobs(ctx, domain.JobStatusProcessing)
	if len(processingJobs) != 1 {
		t.Errorf("Expected 1 processing job, got %d", len(processingJobs))
	}

	// List completed jobs
	completedJobs, _ := queue.ListJobs(ctx, domain.JobStatusCompleted)
	if len(completedJobs) != 1 {
		t.Errorf("Expected 1 completed job, got %d", len(completedJobs))
	}
}

func TestQueue_DeleteJob(t *testing.T) {
	queue := NewQueue(10)
	ctx := context.Background()

	job := domain.NewJob("test", "voice", "provider", "mp3", nil)
	queue.Enqueue(ctx, job)

	err := queue.DeleteJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("Failed to delete job: %v", err)
	}

	_, err = queue.GetJob(ctx, job.ID)
	if err != domain.ErrJobNotFound {
		t.Error("Job should not exist after deletion")
	}
}

func TestQueue_Close(t *testing.T) {
	queue := NewQueue(10)

	err := queue.Close()
	if err != nil {
		t.Fatalf("Failed to close queue: %v", err)
	}

	// Double close should be safe
	err = queue.Close()
	if err != nil {
		t.Fatalf("Double close should not error: %v", err)
	}
}

func TestQueue_Stats(t *testing.T) {
	queue := NewQueue(10)
	ctx := context.Background()

	// Create jobs with different statuses
	job1 := domain.NewJob("test1", "voice", "provider", "mp3", nil)
	job2 := domain.NewJob("test2", "voice", "provider", "mp3", nil)
	job3 := domain.NewJob("test3", "voice", "provider", "mp3", nil)
	job4 := domain.NewJob("test4", "voice", "provider", "mp3", nil)

	queue.Enqueue(ctx, job1)
	queue.Enqueue(ctx, job2)
	queue.Enqueue(ctx, job3)
	queue.Enqueue(ctx, job4)

	job2.SetProcessing()
	queue.UpdateJob(ctx, job2)

	job3.SetCompleted("/path", 24)
	queue.UpdateJob(ctx, job3)

	job4.SetFailed("error")
	queue.UpdateJob(ctx, job4)

	stats := queue.Stats()

	if stats.TotalJobs != 4 {
		t.Errorf("Expected TotalJobs 4, got %d", stats.TotalJobs)
	}
	if stats.QueuedJobs != 1 {
		t.Errorf("Expected QueuedJobs 1, got %d", stats.QueuedJobs)
	}
	if stats.ProcessingJobs != 1 {
		t.Errorf("Expected ProcessingJobs 1, got %d", stats.ProcessingJobs)
	}
	if stats.CompletedJobs != 1 {
		t.Errorf("Expected CompletedJobs 1, got %d", stats.CompletedJobs)
	}
	if stats.FailedJobs != 1 {
		t.Errorf("Expected FailedJobs 1, got %d", stats.FailedJobs)
	}
}
