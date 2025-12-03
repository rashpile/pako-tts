package domain

import (
	"context"
	"io"
)

// AudioStorage defines the interface for storing and retrieving audio files.
// This port allows swapping between filesystem and cloud storage implementations.
type AudioStorage interface {
	// Store saves audio data and returns the storage path.
	Store(ctx context.Context, jobID string, audio []byte, format string) (string, error)

	// Retrieve returns a reader for the stored audio file.
	Retrieve(ctx context.Context, jobID string) (io.ReadCloser, string, error)

	// Delete removes the stored audio file.
	Delete(ctx context.Context, jobID string) error

	// Exists checks if audio exists for the given job.
	Exists(ctx context.Context, jobID string) bool

	// GetPath returns the storage path for a job's audio.
	GetPath(ctx context.Context, jobID string) string
}
