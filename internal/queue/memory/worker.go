package memory

import (
	"context"
	"io"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/pako-tts/server/internal/domain"
)

// Worker processes jobs from the queue.
type Worker struct {
	queue          *Queue
	provider       domain.TTSProvider
	storage        domain.AudioStorage
	logger         *zap.Logger
	retentionHours int
	wg             sync.WaitGroup
	cancel         context.CancelFunc
}

// NewWorker creates a new worker.
func NewWorker(
	queue *Queue,
	provider domain.TTSProvider,
	storage domain.AudioStorage,
	logger *zap.Logger,
	retentionHours int,
) *Worker {
	return &Worker{
		queue:          queue,
		provider:       provider,
		storage:        storage,
		logger:         logger,
		retentionHours: retentionHours,
	}
}

// Start starts the worker pool with the given number of workers.
func (w *Worker) Start(ctx context.Context, numWorkers int) {
	ctx, w.cancel = context.WithCancel(ctx)

	for i := 0; i < numWorkers; i++ {
		w.wg.Add(1)
		go w.run(ctx, i)
	}

	w.logger.Info("Worker pool started", zap.Int("workers", numWorkers))
}

// Stop stops all workers gracefully.
func (w *Worker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
	w.logger.Info("Worker pool stopped")
}

func (w *Worker) run(ctx context.Context, workerID int) {
	defer w.wg.Done()

	logger := w.logger.With(zap.Int("worker_id", workerID))
	logger.Debug("Worker started")

	for {
		select {
		case <-ctx.Done():
			logger.Debug("Worker stopping")
			return
		default:
			job, err := w.queue.Dequeue(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.Error("Failed to dequeue job", zap.Error(err))
				continue
			}
			if job == nil {
				// Queue closed
				return
			}

			w.processJob(ctx, job, logger)
		}
	}
}

func (w *Worker) processJob(ctx context.Context, job *domain.Job, logger *zap.Logger) {
	logger = logger.With(zap.String("job_id", job.ID))
	logger.Info("Processing job")

	// Mark as processing
	job.SetProcessing()
	if err := w.queue.UpdateJob(ctx, job); err != nil {
		logger.Error("Failed to update job status", zap.Error(err))
		return
	}

	// Estimate completion time based on text length
	estimatedDuration := w.estimateDuration(len(job.Text))
	estimatedCompletion := time.Now().Add(estimatedDuration)
	job.UpdateProgress(10, &estimatedCompletion)
	w.queue.UpdateJob(ctx, job) //nolint:errcheck

	// Build synthesis request
	req := &domain.SynthesisRequest{
		Text:         job.Text,
		VoiceID:      job.VoiceID,
		OutputFormat: job.OutputFormat,
		Settings:     job.VoiceSettings,
	}

	// Update progress to 30%
	job.UpdateProgress(30, &estimatedCompletion)
	w.queue.UpdateJob(ctx, job) //nolint:errcheck

	// Synthesize audio
	result, err := w.provider.Synthesize(ctx, req)
	if err != nil {
		logger.Error("Synthesis failed", zap.Error(err))
		job.SetFailed(err.Error())
		w.queue.UpdateJob(ctx, job) //nolint:errcheck
		return
	}

	// Update progress to 70%
	job.UpdateProgress(70, &estimatedCompletion)
	w.queue.UpdateJob(ctx, job) //nolint:errcheck

	// Read audio data
	audioData, err := io.ReadAll(result.Audio)
	if err != nil {
		logger.Error("Failed to read audio data", zap.Error(err))
		job.SetFailed(err.Error())
		w.queue.UpdateJob(ctx, job) //nolint:errcheck
		return
	}

	// Update progress to 90%
	job.UpdateProgress(90, nil)
	w.queue.UpdateJob(ctx, job) //nolint:errcheck

	// Store audio
	resultPath, err := w.storage.Store(ctx, job.ID, audioData, job.OutputFormat)
	if err != nil {
		logger.Error("Failed to store audio", zap.Error(err))
		job.SetFailed(err.Error())
		w.queue.UpdateJob(ctx, job) //nolint:errcheck
		return
	}

	// Mark as completed
	job.SetCompleted(resultPath, w.retentionHours)
	if err := w.queue.UpdateJob(ctx, job); err != nil {
		logger.Error("Failed to update job status", zap.Error(err))
		return
	}

	logger.Info("Job completed successfully",
		zap.String("result_path", resultPath),
		zap.Int("audio_size", len(audioData)),
	)
}

// estimateDuration estimates synthesis duration based on text length.
// Rough estimate: 1000 characters ≈ 5 seconds of synthesis time.
func (w *Worker) estimateDuration(textLength int) time.Duration {
	// Base time + per-character time
	baseTime := 2 * time.Second
	perChar := 5 * time.Millisecond

	return baseTime + time.Duration(textLength)*perChar
}
