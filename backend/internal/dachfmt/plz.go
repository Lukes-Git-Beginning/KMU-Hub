package dachfmt

import (
	"regexp"
	"strings"
)

// plzPatterns holds the postal-code shape per DACH country.
//   - DE: 5 digits (01067–99998)
//   - AT: 4 digits
//   - CH: 4 digits
var plzPatterns = map[string]*regexp.Regexp{
	"DE": regexp.MustCompile(`^\d{5}$`),
	"AT": regexp.MustCompile(`^\d{4}$`),
	"CH": regexp.MustCompile(`^\d{4}$`),
}

// ValidatePLZ reports whether raw is a valid postal code for the given country
// ("DE", "AT", "CH"). An unknown country returns false.
func ValidatePLZ(raw, country string) bool {
	p, ok := plzPatterns[strings.ToUpper(country)]
	if !ok {
		return false
	}
	return p.MatchString(strings.TrimSpace(raw))
}

// ValidatePLZAny reports whether raw matches any supported DACH postal-code
// shape. Use when the country is not known at validation time.
func ValidatePLZAny(raw string) bool {
	s := strings.TrimSpace(raw)
	for _, p := range plzPatterns {
		if p.MatchString(s) {
			return true
		}
	}
	return false
}
