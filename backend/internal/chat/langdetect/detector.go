package langdetect

import (
	"github.com/pemistahl/lingua-go"
)

// Detector detects the language of a text and returns a PostgreSQL tsconfig name
type Detector interface {
	Detect(text string) string
}

// LinguaDetector uses lingua-go for language detection
type LinguaDetector struct {
	detector lingua.LanguageDetector
}

// minTextLength is the minimum text length for language detection
// Below this, we fall back to "simple"
const minTextLength = 20

// langToTSConfig maps lingua languages to PostgreSQL tsconfig names
var langToTSConfig = map[lingua.Language]string{
	lingua.German:  "german",
	lingua.English: "english",
	lingua.French:  "french",
	lingua.Italian: "italian",
	lingua.Spanish: "spanish",
}

// NewLinguaDetector creates a new language detector supporting German, English, French, Italian, Spanish
func NewLinguaDetector() *LinguaDetector {
	languages := []lingua.Language{
		lingua.German,
		lingua.English,
		lingua.French,
		lingua.Italian,
		lingua.Spanish,
	}

	detector := lingua.NewLanguageDetectorBuilder().
		FromLanguages(languages...).
		Build()

	return &LinguaDetector{detector: detector}
}

// Detect returns a PostgreSQL tsconfig name for the detected language
// Falls back to "simple" if text is too short or confidence is too low
func (d *LinguaDetector) Detect(text string) string {
	if len(text) < minTextLength {
		return "simple"
	}

	lang, exists := d.detector.DetectLanguageOf(text)
	if !exists {
		return "simple"
	}

	if tsConfig, ok := langToTSConfig[lang]; ok {
		return tsConfig
	}

	return "simple"
}
