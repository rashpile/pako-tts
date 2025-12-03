// Package filesystem provides a filesystem-based audio storage implementation.
package filesystem

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Storage is a filesystem implementation of domain.AudioStorage.
type Storage struct {
	basePath string
	mu       sync.RWMutex
	logger   *zap.Logger
}

// NewStorage creates a new filesystem storage.
func NewStorage(basePath string, logger *zap.Logger) (*Storage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &Storage{
		basePath: basePath,
		logger:   logger,
	}, nil
}

// Store saves audio data and returns the storage path.
func (s *Storage) Store(ctx context.Context, jobID string, audio []byte, format string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%s.%s", jobID, format)
	filePath := filepath.Join(s.basePath, filename)

	if err := os.WriteFile(filePath, audio, 0644); err != nil {
		return "", fmt.Errorf("failed to write audio file: %w", err)
	}

	s.logger.Debug("Audio stored",
		zap.String("job_id", jobID),
		zap.String("path", filePath),
		zap.Int("size", len(audio)),
	)

	return filePath, nil
}

// Retrieve returns a reader for the stored audio file.
func (s *Storage) Retrieve(ctx context.Context, jobID string) (io.ReadCloser, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Try common formats
	for _, format := range []string{"mp3", "wav"} {
		filename := fmt.Sprintf("%s.%s", jobID, format)
		filePath := filepath.Join(s.basePath, filename)

		file, err := os.Open(filePath)
		if err == nil {
			contentType := "audio/mpeg"
			if format == "wav" {
				contentType = "audio/wav"
			}
			return file, contentType, nil
		}
	}

	return nil, "", fmt.Errorf("audio file not found for job %s", jobID)
}

// Delete removes the stored audio file.
func (s *Storage) Delete(ctx context.Context, jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Try to delete common formats
	for _, format := range []string{"mp3", "wav"} {
		filename := fmt.Sprintf("%s.%s", jobID, format)
		filePath := filepath.Join(s.basePath, filename)
		os.Remove(filePath) // Ignore errors for non-existent files
	}

	return nil
}

// Exists checks if audio exists for the given job.
func (s *Storage) Exists(ctx context.Context, jobID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, format := range []string{"mp3", "wav"} {
		filename := fmt.Sprintf("%s.%s", jobID, format)
		filePath := filepath.Join(s.basePath, filename)
		if _, err := os.Stat(filePath); err == nil {
			return true
		}
	}

	return false
}

// GetPath returns the storage path for a job's audio.
func (s *Storage) GetPath(ctx context.Context, jobID string) string {
	for _, format := range []string{"mp3", "wav"} {
		filename := fmt.Sprintf("%s.%s", jobID, format)
		filePath := filepath.Join(s.basePath, filename)
		if _, err := os.Stat(filePath); err == nil {
			return filePath
		}
	}
	return ""
}

// CleanupExpired removes audio files older than the retention period.
func (s *Storage) CleanupExpired(ctx context.Context, retentionHours int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-time.Duration(retentionHours) * time.Hour)
	deleted := 0

	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read storage directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			filePath := filepath.Join(s.basePath, entry.Name())
			if err := os.Remove(filePath); err == nil {
				deleted++
				s.logger.Debug("Deleted expired audio file",
					zap.String("path", filePath),
					zap.Time("modified", info.ModTime()),
				)
			}
		}
	}

	if deleted > 0 {
		s.logger.Info("Cleanup completed",
			zap.Int("deleted", deleted),
			zap.Int("retention_hours", retentionHours),
		)
	}

	return deleted, nil
}

// StartCleanupScheduler starts a goroutine that periodically cleans up expired files.
func (s *Storage) StartCleanupScheduler(ctx context.Context, retentionHours int, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := s.CleanupExpired(ctx, retentionHours); err != nil {
					s.logger.Error("Cleanup failed", zap.Error(err))
				}
			}
		}
	}()

	s.logger.Info("Cleanup scheduler started",
		zap.Int("retention_hours", retentionHours),
		zap.Duration("interval", interval),
	)
}
