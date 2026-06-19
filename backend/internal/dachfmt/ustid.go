package dachfmt

import (
	"regexp"
	"strings"
)

// euVATPatterns holds the official VIES structural pattern per EU member-state
// VAT prefix (Greece uses "EL", not "GR"). The pattern matches the full
// identifier including the two-letter prefix, upper-cased and stripped of
// spaces, dots and hyphens.
var euVATPatterns = map[string]*regexp.Regexp{
	"AT": regexp.MustCompile(`^ATU\d{8}$`),
	"BE": regexp.MustCompile(`^BE[01]\d{9}$`),
	"BG": regexp.MustCompile(`^BG\d{9,10}$`),
	"CY": regexp.MustCompile(`^CY\d{8}[A-Z]$`),
	"CZ": regexp.MustCompile(`^CZ\d{8,10}$`),
	"DE": regexp.MustCompile(`^DE\d{9}$`),
	"DK": regexp.MustCompile(`^DK\d{8}$`),
	"EE": regexp.MustCompile(`^EE\d{9}$`),
	"EL": regexp.MustCompile(`^EL\d{9}$`),
	"ES": regexp.MustCompile(`^ES[A-Z0-9]\d{7}[A-Z0-9]$`),
	"FI": regexp.MustCompile(`^FI\d{8}$`),
	"FR": regexp.MustCompile(`^FR[A-Z0-9]{2}\d{9}$`),
	"HR": regexp.MustCompile(`^HR\d{11}$`),
	"HU": regexp.MustCompile(`^HU\d{8}$`),
	"IE": regexp.MustCompile(`^IE(\d{7}[A-Z]{1,2}|\d[A-Z]\d{5}[A-Z])$`),
	"IT": regexp.MustCompile(`^IT\d{11}$`),
	"LT": regexp.MustCompile(`^LT(\d{9}|\d{12})$`),
	"LU": regexp.MustCompile(`^LU\d{8}$`),
	"LV": regexp.MustCompile(`^LV\d{11}$`),
	"MT": regexp.MustCompile(`^MT\d{8}$`),
	"NL": regexp.MustCompile(`^NL\d{9}B\d{2}$`),
	"PL": regexp.MustCompile(`^PL\d{10}$`),
	"PT": regexp.MustCompile(`^PT\d{9}$`),
	"RO": regexp.MustCompile(`^RO\d{2,10}$`),
	"SE": regexp.MustCompile(`^SE\d{12}$`),
	"SI": regexp.MustCompile(`^SI\d{8}$`),
	"SK": regexp.MustCompile(`^SK\d{10}$`),
}

// chVATPattern matches a Swiss UID/VAT number (CHE + 9 digits) with an optional
// MWST/TVA/IVA suffix. Formatting dots/hyphens are stripped before matching, so
// "CHE-100.155.212 MWST" → "CHE100155212MWST".
var chVATPattern = regexp.MustCompile(`^CHE\d{9}(MWST|TVA|IVA)?$`)

// normalizeVAT upper-cases raw and removes spaces, dots and hyphens.
func normalizeVAT(raw string) string {
	r := strings.NewReplacer(" ", "", ".", "", "-", "", "/", "")
	return r.Replace(strings.ToUpper(raw))
}

// ValidateUStIDDACH reports whether raw is a valid VAT identification number for
// Germany (DE+9 digits), Austria (ATU+8 digits) or Switzerland (CHE+9 digits,
// optional MWST/TVA/IVA suffix). Spaces, dots and hyphens are ignored.
func ValidateUStIDDACH(raw string) bool {
	s := normalizeVAT(raw)
	switch {
	case strings.HasPrefix(s, "CHE"):
		return chVATPattern.MatchString(s) && checkDigitCH(s[3:12])
	case strings.HasPrefix(s, "DE"):
		return euVATPatterns["DE"].MatchString(s) && checkDigitDE(s[2:11])
	case strings.HasPrefix(s, "AT"):
		return euVATPatterns["AT"].MatchString(s) && checkDigitAT(s[3:11])
	}
	return false
}

// ValidateUStIDEU reports whether raw is a valid VAT identification number for
// any EU member state (VIES format) or Switzerland. Spaces, dots and hyphens
// are ignored; the two-letter country prefix is required. The structural format
// is always checked; the check digit is additionally verified for the DACH
// countries (DE/AT/CH) whose algorithm is implemented, while the rest of the EU
// stays structural-only so a legitimate number is never rejected on an
// unverified checksum.
func ValidateUStIDEU(raw string) bool {
	s := normalizeVAT(raw)
	if strings.HasPrefix(s, "CHE") {
		return chVATPattern.MatchString(s) && checkDigitCH(s[3:12])
	}
	if len(s) < 2 {
		return false
	}
	p, ok := euVATPatterns[s[:2]]
	if !ok {
		return false
	}
	if !p.MatchString(s) {
		return false
	}
	switch s[:2] {
	case "DE":
		return checkDigitDE(s[2:11])
	case "AT":
		return checkDigitAT(s[3:11])
	}
	return true
}
