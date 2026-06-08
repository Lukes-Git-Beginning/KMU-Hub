package dachfmt

import (
	"regexp"
	"strings"
)

// bicPattern matches an ISO 9362 BIC/SWIFT code:
//   - 4 letters: institution code
//   - 2 letters: ISO 3166-1 country code
//   - 2 alphanumerics: location code
//   - optional 3 alphanumerics: branch code (XXX = primary office)
var bicPattern = regexp.MustCompile(`^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$`)

// ValidateBIC reports whether raw is a syntactically valid BIC (8 or 11 chars).
// Input is upper-cased and trimmed before checking.
func ValidateBIC(raw string) bool {
	s := strings.ToUpper(strings.TrimSpace(raw))
	return bicPattern.MatchString(s)
}
