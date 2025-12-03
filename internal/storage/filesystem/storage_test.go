package filesystem

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestNewStorage(t *testing.T) {
	tempDir := t.TempDir()
	logger := testLogger()

	storage, err := NewStorage(tempDir, logger)

	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	if storage == nil {
		t.Fatal("Expected non-nil storage")
	}
}

func TestNewStorage_CreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	newDir := filepath.Join(tempDir, "new-storage-dir")
	logger := testLogger()

	_, err := NewStorage(newDir, logger)

	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	info, err := os.Stat(newDir)
	if err != nil {
		t.Fatalf("Directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("Path should be a directory")
	}
}

func TestStorage_Store(t *testing.T) {
	tempDir := t.TempDir()
	logger := testLogger()
	storage, _ := NewStorage(tempDir, logger)

	ctx := context.Background()
	jobID := "test-job-123"
	audioData := []byte("fake audio data")
	format := "mp3"

	path, err := storage.Store(ctx, jobID, audioData, format)

	if err != nil {
		t.Fatalf("Failed to store audio: %v", err)
	}

	expectedPath := filepath.Join(tempDir, "test-job-123.mp3")
	if path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, path)
	}

	// Verify file was created
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read stored file: %v", err)
	}
	if string(data) != string(audioData) {
		t.Error("Stored data doesn't match original")
	}
}

func TestStorage_Retrieve(t *testing.T) {
	tempDir := t.TempDir()
	logger := testLogger()
	storage, _ := NewStorage(tempDir, logger)

	ctx := context.Background()
	jobID := "test-job-456"
	audioData := []byte("fake audio data for retrieval")

	// Store first
	_, err := storage.Store(ctx, jobID, audioData, "mp3")
	if err != nil {
		t.Fatalf("Failed to store audio: %v", err)
	}

	// Retrieve
	reader, contentType, err := storage.Retrieve(ctx, jobID)
	if err != nil {
		t.Fatalf("Failed to retrieve audio: %v", err)
	}
	defer reader.Close() //nolint:errcheck

	if contentType != "audio/mpeg" {
		t.Errorf("Expected content type audio/mpeg, got %s", contentType)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read retrieved audio: %v", err)
	}
	if string(data) != string(audioData) {
		t.Error("Retrieved data doesn't match original")
	}
}

func TestStorage_Retrieve_WAV(t *testing.T) {
	tempDir := t.TempDir()
	logger := testLogger()
	storage, _ := NewStorage(tempDir, logger)

	ctx := context.Background()
	jobID := "test-job-wav"
	audioData := []byte("fake wav data")

	// Store as WAV
	_, err := storage.Store(ctx, jobID, audioData, "wav")
	if err != nil {
		t.Fatalf("Failed to store audio: %v", err)
	}

	// Retrieve
	reader, contentType, err := storage.Retrieve(ctx, jobID)
	if err != nil {
		t.Fatalf("Failed to retrieve audio: %v", err)
	}
	defer reader.Close() //nolint:errcheck

	if contentType != "audio/wav" {
		t.Errorf("Expected content type audio/wav, got %s", contentType)
	}
}

func TestStorage_Retrieve_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	logger := testLogger()
	storage, _ := NewStorage(tempDir, logger)

	ctx := context.Background()

	_, _, err := storage.Retrieve(ctx, "non-existent-job")

	if err == nil {
		t.Error("Expected error for non-existent job")
	}
}

func TestStorage_Delete(t *testing.T) {
	tempDir := t.TempDir()
	logger := testLogger()
	storage, _ := NewStorage(tempDir, logger)

	ctx := context.Background()
	jobID := "test-job-delete"
	audioData := []byte("fake audio data")

	// Store
	path, _ := storage.Store(ctx, jobID, audioData, "mp3")

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("File should exist before deletion")
	}

	// Delete
	err := storage.Delete(ctx, jobID)
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("File should not exist after deletion")
	}
}

func TestStorage_Delete_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	logger := testLogger()
	storage, _ := NewStorage(tempDir, logger)

	ctx := context.Background()

	// Should not error for non-existent files
	err := storage.Delete(ctx, "non-existent-job")
	if err != nil {
		t.Errorf("Delete should not error for non-existent files: %v", err)
	}
}

func TestStorage_Exists(t *testing.T) {
	tempDir := t.TempDir()
	logger := testLogger()
	storage, _ := NewStorage(tempDir, logger)

	ctx := context.Background()
	jobID := "test-job-exists"

	// Should not exist initially
	if storage.Exists(ctx, jobID) {
		t.Error("Job should not exist initially")
	}

	// Store
	audioData := []byte("fake audio data")
	_, _ = storage.Store(ctx, jobID, audioData, "mp3")

	// Should exist now
	if !storage.Exists(ctx, jobID) {
		t.Error("Job should exist after storing")
	}
}

func TestStorage_GetPath(t *testing.T) {
	tempDir := t.TempDir()
	logger := testLogger()
	storage, _ := NewStorage(tempDir, logger)

	ctx := context.Background()
	jobID := "test-job-path"

	// Should return empty for non-existent
	path := storage.GetPath(ctx, jobID)
	if path != "" {
		t.Errorf("Expected empty path for non-existent job, got %s", path)
	}

	// Store
	audioData := []byte("fake audio data")
	storedPath, _ := storage.Store(ctx, jobID, audioData, "mp3")

	// Should return the path now
	path = storage.GetPath(ctx, jobID)
	if path != storedPath {
		t.Errorf("Expected path %s, got %s", storedPath, path)
	}
}

func TestStorage_CleanupExpired(t *testing.T) {
	tempDir := t.TempDir()
	logger := testLogger()
	storage, _ := NewStorage(tempDir, logger)

	ctx := context.Background()

	// Create some files with different modification times
	oldFile := filepath.Join(tempDir, "old-job.mp3")
	newFile := filepath.Join(tempDir, "new-job.mp3")

	os.WriteFile(oldFile, []byte("old"), 0644) //nolint:errcheck
	os.WriteFile(newFile, []byte("new"), 0644) //nolint:errcheck

	// Set old file modification time to 48 hours ago
	oldTime := time.Now().Add(-48 * time.Hour)
	os.Chtimes(oldFile, oldTime, oldTime) //nolint:errcheck

	// Cleanup with 24 hour retention
	deleted, err := storage.CleanupExpired(ctx, 24)
	if err != nil {
		t.Fatalf("CleanupExpired failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected 1 deleted file, got %d", deleted)
	}

	// Old file should be gone
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("Old file should be deleted")
	}

	// New file should still exist
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Error("New file should still exist")
	}
}
