package transcode

import (
	"context"
	"encoding/binary"
	"strings"
	"testing"
)

// oneSec24kHzMono16Bit produces 1 second of silent 24 kHz / 16-bit / mono PCM.
func oneSec24kHzMono16Bit() []byte {
	// 24000 samples × 2 bytes = 48000 bytes
	return make([]byte, 48000)
}

// TestPCMToWAV verifies the canonical 44-byte RIFF/WAVE header for a 1-second 24 kHz mono PCM buffer.
func TestPCMToWAV(t *testing.T) {
	pcm := oneSec24kHzMono16Bit()
	sampleRate := 24000
	channels := 1
	bitsPerSample := 16

	result := PCMToWAV(pcm, sampleRate, channels, bitsPerSample)

	expectedTotal := 44 + len(pcm)
	if len(result) != expectedTotal {
		t.Fatalf("expected total length %d, got %d", expectedTotal, len(result))
	}

	// ChunkID: "RIFF"
	if string(result[0:4]) != "RIFF" {
		t.Errorf("offset 0: expected RIFF, got %q", result[0:4])
	}

	// ChunkSize: 36 + dataSize
	chunkSize := binary.LittleEndian.Uint32(result[4:8])
	expectedChunkSize := uint32(36 + len(pcm))
	if chunkSize != expectedChunkSize {
		t.Errorf("offset 4 ChunkSize: expected %d, got %d", expectedChunkSize, chunkSize)
	}

	// Format: "WAVE"
	if string(result[8:12]) != "WAVE" {
		t.Errorf("offset 8: expected WAVE, got %q", result[8:12])
	}

	// Subchunk1ID: "fmt "
	if string(result[12:16]) != "fmt " {
		t.Errorf("offset 12: expected \"fmt \", got %q", result[12:16])
	}

	// Subchunk1Size: 16 (PCM)
	sub1Size := binary.LittleEndian.Uint32(result[16:20])
	if sub1Size != 16 {
		t.Errorf("offset 16 Subchunk1Size: expected 16, got %d", sub1Size)
	}

	// AudioFormat: 1 (PCM)
	audioFmt := binary.LittleEndian.Uint16(result[20:22])
	if audioFmt != 1 {
		t.Errorf("offset 20 AudioFormat: expected 1, got %d", audioFmt)
	}

	// NumChannels
	numChans := binary.LittleEndian.Uint16(result[22:24])
	if numChans != uint16(channels) {
		t.Errorf("offset 22 NumChannels: expected %d, got %d", channels, numChans)
	}

	// SampleRate
	sr := binary.LittleEndian.Uint32(result[24:28])
	if sr != uint32(sampleRate) {
		t.Errorf("offset 24 SampleRate: expected %d, got %d", sampleRate, sr)
	}

	// ByteRate: sampleRate × channels × bitsPerSample / 8
	byteRate := binary.LittleEndian.Uint32(result[28:32])
	expectedByteRate := uint32(sampleRate * channels * bitsPerSample / 8)
	if byteRate != expectedByteRate {
		t.Errorf("offset 28 ByteRate: expected %d, got %d", expectedByteRate, byteRate)
	}

	// BlockAlign: channels × bitsPerSample / 8
	blockAlign := binary.LittleEndian.Uint16(result[32:34])
	expectedBlockAlign := uint16(channels * bitsPerSample / 8)
	if blockAlign != expectedBlockAlign {
		t.Errorf("offset 32 BlockAlign: expected %d, got %d", expectedBlockAlign, blockAlign)
	}

	// BitsPerSample
	bps := binary.LittleEndian.Uint16(result[34:36])
	if bps != uint16(bitsPerSample) {
		t.Errorf("offset 34 BitsPerSample: expected %d, got %d", bitsPerSample, bps)
	}

	// Subchunk2ID: "data"
	if string(result[36:40]) != "data" {
		t.Errorf("offset 36: expected data, got %q", result[36:40])
	}

	// Subchunk2Size: len(pcm)
	sub2Size := binary.LittleEndian.Uint32(result[40:44])
	if sub2Size != uint32(len(pcm)) {
		t.Errorf("offset 40 Subchunk2Size: expected %d, got %d", len(pcm), sub2Size)
	}

	// PCM data follows unchanged
	for i, b := range pcm {
		if result[44+i] != b {
			t.Fatalf("PCM data mismatch at offset %d", 44+i)
		}
	}
}

// TestPCMToWAV_EmptyPCM ensures a zero-length PCM buffer produces a valid minimal WAV.
func TestPCMToWAV_EmptyPCM(t *testing.T) {
	result := PCMToWAV(nil, 24000, 1, 16)
	if len(result) != 44 {
		t.Fatalf("expected 44 bytes for empty PCM, got %d", len(result))
	}
	chunkSize := binary.LittleEndian.Uint32(result[4:8])
	if chunkSize != 36 {
		t.Errorf("ChunkSize for empty PCM: expected 36, got %d", chunkSize)
	}
	sub2Size := binary.LittleEndian.Uint32(result[40:44])
	if sub2Size != 0 {
		t.Errorf("Subchunk2Size for empty PCM: expected 0, got %d", sub2Size)
	}
}

// isValidMP3 returns true if data looks like a valid MP3 stream.
// MP3 can start with an ID3 tag ("ID3") or directly with a sync frame (0xFF 0xEx).
func isValidMP3(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	if len(data) >= 3 && data[0] == 0x49 && data[1] == 0x44 && data[2] == 0x33 {
		return true // ID3 tag header
	}
	return data[0] == 0xFF && (data[1]&0xE0) == 0xE0 // MPEG sync word
}

// TestPCMToMP3_HappyPath encodes 1 second of silence and verifies the output is a valid MP3.
func TestPCMToMP3_HappyPath(t *testing.T) {
	pcm := oneSec24kHzMono16Bit()

	out, err := PCMToMP3(context.Background(), pcm, 24000, 1)
	if err != nil {
		t.Fatalf("PCMToMP3 failed: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected non-empty MP3 output")
	}
	if !isValidMP3(out) {
		prefix := out
		if len(prefix) > 4 {
			prefix = prefix[:4]
		}
		t.Errorf("output does not look like MP3: first bytes %#v", prefix)
	}
}

// TestPCMToMP3_ContextCancelled verifies that a cancelled context causes PCMToMP3 to return an error.
func TestPCMToMP3_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call

	_, err := PCMToMP3(ctx, oneSec24kHzMono16Bit(), 24000, 1)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

// TestPCMToMP3_MissingBinary exercises the error path when the ffmpeg binary cannot be found.
// This test must NOT run in parallel because it mutates the package-level ffmpegBinary variable.
func TestPCMToMP3_MissingBinary(t *testing.T) {
	original := ffmpegBinary
	ffmpegBinary = "/nonexistent/path/to/ffmpeg"
	defer func() { ffmpegBinary = original }()

	_, err := PCMToMP3(context.Background(), oneSec24kHzMono16Bit(), 24000, 1)
	if err == nil {
		t.Fatal("expected error for missing ffmpeg binary, got nil")
	}
	if !strings.Contains(err.Error(), "ffmpeg") {
		t.Errorf("expected error to mention ffmpeg, got: %v", err)
	}
}
