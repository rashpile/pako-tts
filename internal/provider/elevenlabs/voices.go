package elevenlabs

// DefaultVoices contains commonly used voice IDs for quick reference.
var DefaultVoices = map[string]string{
	"adam":    "pNInz6obpgDQGcFmaJgB",
	"aria":    "9BWtsMINqrJLrRacOk9x",
	"sarah":   "EXAVITQu4vr4xnSDxMaL",
	"laura":   "FGY2WhTYpPnrIDTdsKH5",
	"charlie": "IKne3meq5aSn9XLyUdCD",
	"george":  "JBFqnCBsd6RMkjVDRZzb",
}

// DefaultVoiceID is the default voice used when none is specified.
const DefaultVoiceID = "pNInz6obpgDQGcFmaJgB" // Adam

// GetVoiceID returns a voice ID by name or the ID itself if not found.
func GetVoiceID(nameOrID string) string {
	if id, ok := DefaultVoices[nameOrID]; ok {
		return id
	}
	return nameOrID
}
