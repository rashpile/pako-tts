package gemini

import "github.com/pako-tts/server/internal/domain"

const (
	providerName    = "gemini"
	defaultModelID  = "gemini-3.1-flash-tts-preview"
	defaultVoiceName = "Kore"
)

// supportedLanguages is the set of ISO 639-1 codes advertised by the model.
// It is the single source of truth — defaultModel.Languages is assigned from it.
var supportedLanguages = []string{
	"af", "sq", "am", "ar", "hy", "az", "eu", "be", "bn", "bs",
	"bg", "ca", "zh", "hr", "cs", "da", "nl", "en", "et", "fi",
	"fr", "gl", "ka", "de", "el", "gu", "he", "hi", "hu", "is",
	"id", "it", "ja", "kn", "kk", "km", "ko", "lo", "lv", "lt",
	"mk", "ms", "ml", "mt", "mr", "mn", "ne", "nb", "fa", "pl",
	"pt", "ro", "ru", "sr", "si", "sk", "sl", "es", "su", "sw",
	"sv", "ta", "te", "th", "tr", "uk", "ur", "uz", "vi", "cy",
	"yo", "zu",
}

// isoToName maps ISO 639-1 codes to the human-readable language name used in
// the spoken language directive injected into the Gemini prompt.
var isoToName = map[string]string{
	"af": "Afrikaans",
	"sq": "Albanian",
	"am": "Amharic",
	"ar": "Arabic",
	"hy": "Armenian",
	"az": "Azerbaijani",
	"eu": "Basque",
	"be": "Belarusian",
	"bn": "Bengali",
	"bs": "Bosnian",
	"bg": "Bulgarian",
	"ca": "Catalan",
	"zh": "Chinese (Mandarin)",
	"hr": "Croatian",
	"cs": "Czech",
	"da": "Danish",
	"nl": "Dutch",
	"en": "English",
	"et": "Estonian",
	"fi": "Finnish",
	"fr": "French",
	"gl": "Galician",
	"ka": "Georgian",
	"de": "German",
	"el": "Greek",
	"gu": "Gujarati",
	"he": "Hebrew",
	"hi": "Hindi",
	"hu": "Hungarian",
	"is": "Icelandic",
	"id": "Indonesian",
	"it": "Italian",
	"ja": "Japanese",
	"kn": "Kannada",
	"kk": "Kazakh",
	"km": "Khmer",
	"ko": "Korean",
	"lo": "Lao",
	"lv": "Latvian",
	"lt": "Lithuanian",
	"mk": "Macedonian",
	"ms": "Malay",
	"ml": "Malayalam",
	"mt": "Maltese",
	"mr": "Marathi",
	"mn": "Mongolian",
	"ne": "Nepali",
	"nb": "Norwegian",
	"fa": "Persian",
	"pl": "Polish",
	"pt": "Portuguese",
	"ro": "Romanian",
	"ru": "Russian",
	"sr": "Serbian",
	"si": "Sinhala",
	"sk": "Slovak",
	"sl": "Slovenian",
	"es": "Spanish",
	"su": "Sundanese",
	"sw": "Swahili",
	"sv": "Swedish",
	"ta": "Tamil",
	"te": "Telugu",
	"th": "Thai",
	"tr": "Turkish",
	"uk": "Ukrainian",
	"ur": "Urdu",
	"uz": "Uzbek",
	"vi": "Vietnamese",
	"cy": "Welsh",
	"yo": "Yoruba",
	"zu": "Zulu",
}

// defaultModel is the single Gemini TTS model exposed by ListModels.
var defaultModel = domain.Model{
	ModelID:     defaultModelID,
	Name:        "Gemini 3.1 Flash TTS",
	Provider:    providerName,
	Description: "Google Gemini Flash TTS model with multilingual support and free-text style instructions",
	Languages:   supportedLanguages,
}

// prebuiltVoices lists the 30 prebuilt Gemini voices.
// Language is empty because these voices are language-agnostic; the spoken
// language is controlled via the LanguageCode field on each request.
var prebuiltVoices = []domain.Voice{
	{VoiceID: "Zephyr", Name: "Zephyr", Provider: providerName},
	{VoiceID: "Puck", Name: "Puck", Provider: providerName},
	{VoiceID: "Charon", Name: "Charon", Provider: providerName},
	{VoiceID: "Kore", Name: "Kore", Provider: providerName},
	{VoiceID: "Fenrir", Name: "Fenrir", Provider: providerName},
	{VoiceID: "Leda", Name: "Leda", Provider: providerName},
	{VoiceID: "Orus", Name: "Orus", Provider: providerName},
	{VoiceID: "Aoede", Name: "Aoede", Provider: providerName},
	{VoiceID: "Callirrhoe", Name: "Callirrhoe", Provider: providerName},
	{VoiceID: "Autonoe", Name: "Autonoe", Provider: providerName},
	{VoiceID: "Enceladus", Name: "Enceladus", Provider: providerName},
	{VoiceID: "Iapetus", Name: "Iapetus", Provider: providerName},
	{VoiceID: "Umbriel", Name: "Umbriel", Provider: providerName},
	{VoiceID: "Algieba", Name: "Algieba", Provider: providerName},
	{VoiceID: "Despina", Name: "Despina", Provider: providerName},
	{VoiceID: "Erinome", Name: "Erinome", Provider: providerName},
	{VoiceID: "Algenib", Name: "Algenib", Provider: providerName},
	{VoiceID: "Rasalgethi", Name: "Rasalgethi", Provider: providerName},
	{VoiceID: "Laomedeia", Name: "Laomedeia", Provider: providerName},
	{VoiceID: "Achernar", Name: "Achernar", Provider: providerName},
	{VoiceID: "Alnilam", Name: "Alnilam", Provider: providerName},
	{VoiceID: "Schedar", Name: "Schedar", Provider: providerName},
	{VoiceID: "Gacrux", Name: "Gacrux", Provider: providerName},
	{VoiceID: "Pulcherrima", Name: "Pulcherrima", Provider: providerName},
	{VoiceID: "Achird", Name: "Achird", Provider: providerName},
	{VoiceID: "Zubenelgenubi", Name: "Zubenelgenubi", Provider: providerName},
	{VoiceID: "Vindemiatrix", Name: "Vindemiatrix", Provider: providerName},
	{VoiceID: "Sadachbia", Name: "Sadachbia", Provider: providerName},
	{VoiceID: "Sadaltager", Name: "Sadaltager", Provider: providerName},
	{VoiceID: "Sulafat", Name: "Sulafat", Provider: providerName},
}
