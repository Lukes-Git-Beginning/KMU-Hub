package langdetect

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinguaDetector_Detect(t *testing.T) {
	detector := NewLinguaDetector()

	t.Run("german text", func(t *testing.T) {
		text := "Dies ist ein langer deutscher Text, der zur Erkennung der Sprache verwendet wird"
		result := detector.Detect(text)
		assert.Equal(t, "german", result)
	})

	t.Run("english text", func(t *testing.T) {
		text := "This is a long English text that should be detected as English language"
		result := detector.Detect(text)
		assert.Equal(t, "english", result)
	})

	t.Run("short text falls back to simple", func(t *testing.T) {
		text := "Hello world"
		result := detector.Detect(text)
		assert.Equal(t, "simple", result)
	})

	t.Run("empty text falls back to simple", func(t *testing.T) {
		result := detector.Detect("")
		assert.Equal(t, "simple", result)
	})

	t.Run("french text", func(t *testing.T) {
		text := "Ceci est un long texte en français qui devrait être détecté comme étant en français"
		result := detector.Detect(text)
		assert.Equal(t, "french", result)
	})
}
