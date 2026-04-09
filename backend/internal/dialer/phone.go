package dialer

import (
	"fmt"
	"regexp"
	"strings"
)

// countryCodes maps ISO 3166-1 alpha-2 country codes to their E.164 prefix.
var countryCodes = map[string]string{
	"DE": "+49",
	"AT": "+43",
	"CH": "+41",
}

// stripNonDialable removes all characters that are not digits or a '+' sign.
var stripNonDialable = regexp.MustCompile(`[^\d+]`)

// e164Pattern validates the final normalised value: '+' followed by 7–15 digits.
var e164Pattern = regexp.MustCompile(`^\+\d{7,15}$`)

// NormalizePhoneE164 normalises raw to E.164 format using DACH country codes.
//
// defaultCountry must be "DE", "AT", or "CH" and is applied when raw contains
// a local (0-prefixed) number without an explicit international prefix.
//
// Handled input forms (examples for defaultCountry="DE"):
//
//	+49 151 1234 5678   → +4915112345678
//	0049 151 1234 5678  → +4915112345678
//	0151 1234 5678      → +4915112345678
//	151 1234 5678       → +4915112345678
//	(030) 123-456       → +4930123456
func NormalizePhoneE164(raw string, defaultCountry string) (string, error) {
	countryPrefix, ok := countryCodes[strings.ToUpper(defaultCountry)]
	if !ok {
		return "", fmt.Errorf("%w: unsupported country %q", ErrInvalidPhoneNumber, defaultCountry)
	}

	// 1. Remove formatting characters (spaces, dashes, dots, parentheses, slashes).
	//    Keep digits and any leading '+'.
	cleaned := stripNonDialable.ReplaceAllString(raw, "")

	// After stripping, only a '+' at position 0 is meaningful; remove any
	// others that crept in from e.g. extension notation "+49 (0)30…".
	if len(cleaned) > 1 {
		cleaned = cleaned[:1] + strings.ReplaceAll(cleaned[1:], "+", "")
	}

	switch {
	// Already carries an E.164 prefix (e.g. +49…, +43…, +41…).
	case strings.HasPrefix(cleaned, "+"):
		// No transformation needed — fall through to validation.

	// International prefix written as 00CC (e.g. 0049…, 0041…).
	case strings.HasPrefix(cleaned, "00"):
		cleaned = "+" + cleaned[2:]

	// Local number with leading trunk zero (e.g. 0151…, 030…).
	case strings.HasPrefix(cleaned, "0"):
		// Remove the trunk zero and prepend the E.164 country prefix.
		cleaned = countryPrefix + cleaned[1:]

	// Subscriber number without any prefix — assume local, no trunk zero.
	default:
		cleaned = countryPrefix + cleaned
	}

	if !e164Pattern.MatchString(cleaned) {
		return "", fmt.Errorf("%w: %q normalised to %q", ErrInvalidPhoneNumber, raw, cleaned)
	}

	return cleaned, nil
}

// FormatDuration formats a call duration given in seconds as "M:SS" for
// display in CRM activity subjects and UI labels (e.g. 90 → "1:30").
func FormatDuration(seconds int) string {
	m := seconds / 60
	s := seconds % 60
	return fmt.Sprintf("%d:%02d", m, s)
}
