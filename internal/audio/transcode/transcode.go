package transcode

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
)

// ffmpegBinary is the name or path of the ffmpeg executable.
// Tests may override this to exercise the missing-binary error path without mutating PATH.
var ffmpegBinary = "ffmpeg"

// PCMToMP3 converts raw 16-bit signed little-endian PCM to MP3 at 128 kbps via ffmpeg.
// sampleRate is in Hz; channels is the number of audio channels (1 = mono).
// The context controls the lifetime of the ffmpeg subprocess.
func PCMToMP3(ctx context.Context, pcm []byte, sampleRate, channels int) ([]byte, error) {
	cmd := exec.CommandContext(ctx, ffmpegBinary,
		"-f", "s16le",
		"-ar", strconv.Itoa(sampleRate),
		"-ac", strconv.Itoa(channels),
		"-i", "pipe:0",
		"-f", "mp3",
		"-b:a", "128k",
		"pipe:1",
	)
	cmd.Stdin = bytes.NewReader(pcm)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, stderr.String())
	}
	return out, nil
}
