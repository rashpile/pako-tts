package transcode

import "encoding/binary"

// PCMToWAV wraps raw PCM data in a RIFF/WAVE container with a canonical 44-byte header.
// Parameters must match the PCM stream: sampleRate in Hz, channels (1=mono), bitsPerSample (typically 16).
func PCMToWAV(pcm []byte, sampleRate, channels, bitsPerSample int) []byte {
	dataSize := uint32(len(pcm))
	byteRate := uint32(sampleRate * channels * bitsPerSample / 8)
	blockAlign := uint16(channels * bitsPerSample / 8)

	var header [44]byte
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:], 36+dataSize)
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:], 16) // Subchunk1Size: always 16 for PCM
	binary.LittleEndian.PutUint16(header[20:], 1)  // AudioFormat: 1 = PCM
	binary.LittleEndian.PutUint16(header[22:], uint16(channels))
	binary.LittleEndian.PutUint32(header[24:], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:], byteRate)
	binary.LittleEndian.PutUint16(header[32:], blockAlign)
	binary.LittleEndian.PutUint16(header[34:], uint16(bitsPerSample))
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:], dataSize)

	result := make([]byte, 44+len(pcm))
	copy(result[0:44], header[:])
	copy(result[44:], pcm)
	return result
}
