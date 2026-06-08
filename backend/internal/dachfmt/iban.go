package dachfmt

import (
	"regexp"
	"strings"
)

// ibanCountryLengths holds the exact IBAN length per ISO 3166-1 country code.
// Focused on DACH plus the most common SEPA countries a DACH KMU transacts with.
var ibanCountryLengths = map[string]int{
	"DE": 22, "AT": 20, "CH": 21, "LI": 21, "LU": 20,
	"FR": 27, "IT": 27, "ES": 24, "NL": 18, "BE": 16,
	"DK": 18, "FI": 18, "GB": 22, "IE": 22, "PL": 28,
	"PT": 25, "SE": 24, "NO": 15, "CZ": 24, "SK": 24,
	"HU": 28, "SI": 19, "HR": 21, "RO": 24, "BG": 22,
	"GR": 27, "EE": 20, "LV": 21, "LT": 20, "MT": 31,
	"CY": 28,
}

// ibanShape matches the coarse structure: 2 letters (country), 2 digits
// (check), then up to 30 alphanumerics (BBAN).
var ibanShape = regexp.MustCompile(`^[A-Z]{2}\d{2}[A-Z0-9]{1,30}$`)

// ValidateIBAN reports whether raw is a structurally valid IBAN with a correct
// ISO 7064 mod-97-10 check. Whitespace is ignored and letters are upper-cased,
// so "DE89 3704 0044 0532 0130 00" validates. The country code must be known
// and the total length must match that country's fixed IBAN length.
func ValidateIBAN(raw string) bool {
	s := strings.ToUpper(strings.ReplaceAll(raw, " ", ""))
	if !ibanShape.MatchString(s) {
		return false
	}
	want, ok := ibanCountryLengths[s[:2]]
	if !ok || len(s) != want {
		return false
	}
	return mod97(s) == 1
}

// mod97 computes the ISO 7064 mod-97 remainder of an IBAN. The first four
// characters are moved to the end, letters are expanded to two-digit numbers
// (A=10 … Z=35), and the remainder is accumulated digit-group-wise to avoid
// big-integer arithmetic.
func mod97(iban string) int {
	rearranged := iban[4:] + iban[:4]
	remainder := 0
	for _, c := range rearranged {
		switch {
		case c >= '0' && c <= '9':
			remainder = (remainder*10 + int(c-'0')) % 97
		case c >= 'A' && c <= 'Z':
			// Letter expands to a two-digit value (e.g. D=13 → "13").
			v := int(c-'A') + 10
			remainder = (remainder*100 + v) % 97
		default:
			return -1
		}
	}
	return remainder
}
