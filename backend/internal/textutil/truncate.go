// Package textutil holds small string helpers shared across services.
package textutil

import "unicode/utf8"

// TruncateBytes cuts s to at most maxBytes bytes without splitting a multi-byte
// rune, so the result is valid UTF-8 whenever s is. Postgres rejects invalid
// UTF-8 in TEXT columns (SQLSTATE 22021), which a plain s[:n] would produce.
func TruncateBytes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	if maxBytes <= 0 {
		return ""
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}
