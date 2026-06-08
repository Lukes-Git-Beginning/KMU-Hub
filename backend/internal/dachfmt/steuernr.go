package dachfmt

import (
	"regexp"
	"strings"
)

// bundeslandSteuernummerLen holds the length of the länderspezifische (state)
// German Steuernummer per Bundesland. Most states use 11 digits; Baden-
// Württemberg and Berlin use 10. The federal unified scheme ("Vereinheitlichtes
// Bundesschema", 13 digits) is accepted for every state in addition to this.
var bundeslandSteuernummerLen = map[string]int{
	"BW": 10, "BE": 10,
	"BY": 11, "BB": 11, "HB": 11, "HH": 11, "HE": 11, "MV": 11,
	"NI": 11, "NW": 11, "RP": 11, "SL": 11, "SN": 11, "ST": 11,
	"SH": 11, "TH": 11,
}

// stripSteuernummer removes the separators commonly printed in a Steuernummer
// ("21/815/08150", "151/815/08156") leaving only the digits.
func stripSteuernummer(raw string) string {
	r := strings.NewReplacer("/", "", " ", "", ".", "", "-", "")
	return r.Replace(strings.TrimSpace(raw))
}

var digitsOnly = regexp.MustCompile(`^\d+$`)

// isUnifiedScheme reports whether digits is a valid 13-digit unified federal
// Steuernummer. The unified scheme always carries a '0' as its fifth digit
// (the separator between the four-digit Bundesfinanzamtsnummer and the rest).
func isUnifiedScheme(digits string) bool {
	return len(digits) == 13 && digits[4] == '0'
}

// ValidateSteuernummer reports whether raw is a structurally valid German
// Steuernummer: either the länderspezifische form (10 or 11 digits) or the
// 13-digit unified federal scheme (fifth digit '0'). Separators are ignored.
//
// Note: the Steuernummer has no nationwide check digit — validation is a
// structural length/scheme check, deliberately lenient to avoid rejecting
// legitimate numbers in a finance context.
func ValidateSteuernummer(raw string) bool {
	d := stripSteuernummer(raw)
	if !digitsOnly.MatchString(d) {
		return false
	}
	switch len(d) {
	case 10, 11:
		return true
	case 13:
		return isUnifiedScheme(d)
	default:
		return false
	}
}

// ValidateSteuernummerForBundesland validates raw against the expected
// länderspezifische length for the given Bundesland (ISO-style code, e.g. "BW",
// "BY", "NW"), additionally always accepting the 13-digit unified scheme. An
// unknown Bundesland returns false.
func ValidateSteuernummerForBundesland(raw, bundesland string) bool {
	d := stripSteuernummer(raw)
	if !digitsOnly.MatchString(d) {
		return false
	}
	if isUnifiedScheme(d) {
		return true
	}
	want, ok := bundeslandSteuernummerLen[strings.ToUpper(bundesland)]
	if !ok {
		return false
	}
	return len(d) == want
}
