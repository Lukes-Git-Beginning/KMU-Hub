package dachfmt

import (
	"errors"
	"testing"
)

func TestNormalizePhoneE164(t *testing.T) {
	tests := []struct {
		raw, country, want string
		wantErr            bool
	}{
		{"+49 151 1234 5678", "DE", "+4915112345678", false},
		{"0049 151 1234 5678", "DE", "+4915112345678", false},
		{"0151 1234 5678", "DE", "+4915112345678", false},
		{"(030) 123-456", "DE", "+4930123456", false},
		{"0660 1234567", "AT", "+436601234567", false},
		{"079 123 45 67", "CH", "+41791234567", false},
		{"", "DE", "", true},
		{"abc", "DE", "", true},
		{"0151", "DE", "", true}, // too short after normalisation
		{"0151123456", "US", "", true},
	}
	for _, tt := range tests {
		got, err := NormalizePhoneE164(tt.raw, tt.country)
		if tt.wantErr {
			if err == nil {
				t.Errorf("NormalizePhoneE164(%q,%q): want error, got %q", tt.raw, tt.country, got)
			} else if !errors.Is(err, ErrInvalidPhoneNumber) {
				t.Errorf("NormalizePhoneE164(%q,%q): want ErrInvalidPhoneNumber, got %v", tt.raw, tt.country, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizePhoneE164(%q,%q): unexpected error %v", tt.raw, tt.country, err)
		}
		if got != tt.want {
			t.Errorf("NormalizePhoneE164(%q,%q) = %q, want %q", tt.raw, tt.country, got, tt.want)
		}
	}
}

func TestValidateIBAN(t *testing.T) {
	valid := []string{
		"DE89370400440532013000",
		"DE89 3704 0044 0532 0130 00", // with spaces
		"AT611904300234573201",
		"CH9300762011623852957",
		"GB82WEST12345698765432",
		"FR1420041010050500013M02606",
		"NL91ABNA0417164300",
	}
	for _, s := range valid {
		if !ValidateIBAN(s) {
			t.Errorf("ValidateIBAN(%q) = false, want true", s)
		}
	}
	invalid := []string{
		"DE89370400440532013001",  // bad check digit
		"DE8937040044053201300",   // wrong length
		"XX89370400440532013000",  // unknown country
		"DE8937040044053201300A0", // non-digit check digits
		"",
		"DE",
	}
	for _, s := range invalid {
		if ValidateIBAN(s) {
			t.Errorf("ValidateIBAN(%q) = true, want false", s)
		}
	}
}

func TestNormalizeAndFormatIBAN(t *testing.T) {
	// Both spellings of one IBAN must normalise to the same string: that is
	// what keeps a grouped entry and a compact one from becoming two accounts.
	const canonical = "DE89370400440532013000"
	for _, raw := range []string{canonical, "DE89 3704 0044 0532 0130 00", "de89370400440532013000"} {
		if got := NormalizeIBAN(raw); got != canonical {
			t.Errorf("NormalizeIBAN(%q) = %q, want %q", raw, got, canonical)
		}
	}

	const grouped = "DE89 3704 0044 0532 0130 00"
	if got := FormatIBAN(canonical); got != grouped {
		t.Errorf("FormatIBAN(%q) = %q, want %q", canonical, got, grouped)
	}
	// Already grouped input regroups rather than doubling the spaces.
	if got := FormatIBAN(grouped); got != grouped {
		t.Errorf("FormatIBAN(%q) = %q, want it unchanged", grouped, got)
	}
	// Anything that is not IBAN-shaped comes back untouched instead of mangled.
	for _, raw := range []string{"", "not an iban", "1234"} {
		if got := FormatIBAN(raw); got != raw {
			t.Errorf("FormatIBAN(%q) = %q, want it unchanged", raw, got)
		}
	}
}

func TestValidateBIC(t *testing.T) {
	valid := []string{"DEUTDEFF", "DEUTDEFF500", "DEUTDEFFXXX", "deutdeff", "RZTIAT22263"}
	for _, s := range valid {
		if !ValidateBIC(s) {
			t.Errorf("ValidateBIC(%q) = false, want true", s)
		}
	}
	invalid := []string{"DEUTDEF", "DEUTDEFF5", "DEUTDEFF5000", "1234DEFF", ""}
	for _, s := range invalid {
		if ValidateBIC(s) {
			t.Errorf("ValidateBIC(%q) = true, want false", s)
		}
	}
}

func TestValidatePLZ(t *testing.T) {
	cases := []struct {
		raw, country string
		want         bool
	}{
		{"10115", "DE", true},
		{"1011", "DE", false},
		{"1010", "AT", true},
		{"10115", "AT", false},
		{"8001", "CH", true},
		{"abcde", "DE", false},
		{"12345", "XX", false},
	}
	for _, c := range cases {
		if got := ValidatePLZ(c.raw, c.country); got != c.want {
			t.Errorf("ValidatePLZ(%q,%q) = %v, want %v", c.raw, c.country, got, c.want)
		}
	}
	if !ValidatePLZAny("10115") || !ValidatePLZAny("1010") {
		t.Error("ValidatePLZAny should accept valid DE/AT codes")
	}
	if ValidatePLZAny("123") {
		t.Error("ValidatePLZAny should reject 3-digit code")
	}
}

func TestValidateUStIDDACH(t *testing.T) {
	// Authentic check-digit-valid numbers (test vectors from python-stdnum).
	valid := []string{"DE136695976", "ATU13585627", "CHE-100.155.212", "CHE-100.155.212 MWST", "CHE100155212TVA"}
	for _, s := range valid {
		if !ValidateUStIDDACH(s) {
			t.Errorf("ValidateUStIDDACH(%q) = false, want true", s)
		}
	}
	// Wrong length / wrong country plus structurally-valid-but-wrong-check-digit.
	invalid := []string{"DE12345678", "ATU1234567", "FR12345678901", "CHE12345678", "",
		"DE136695978", "ATU13585626", "CHE-100.155.213"}
	for _, s := range invalid {
		if ValidateUStIDDACH(s) {
			t.Errorf("ValidateUStIDDACH(%q) = true, want false", s)
		}
	}
}

func TestValidateUStIDEU(t *testing.T) {
	// DACH numbers are check-digit-valid; the non-DACH ones are structural-only.
	valid := []string{"DE136695976", "ATU13585627", "FR40303265045", "NL123456789B01", "IT12345678901", "CHE-100.155.212"}
	for _, s := range valid {
		if !ValidateUStIDEU(s) {
			t.Errorf("ValidateUStIDEU(%q) = false, want true", s)
		}
	}
	// Unknown prefix, wrong length, bad NL structure, and a DACH number whose
	// check digit is wrong (must be rejected because DE/AT/CH are gated).
	invalid := []string{"ZZ123456789", "DE12345678", "NL123456789X01", "", "DE136695978", "ATU13585626"}
	for _, s := range invalid {
		if ValidateUStIDEU(s) {
			t.Errorf("ValidateUStIDEU(%q) = true, want false", s)
		}
	}
}

func TestValidateSteuernummer(t *testing.T) {
	valid := []string{"2181508150", "21/815/08150", "1234567890", "12345678901", "1121081508150"}
	for _, s := range valid {
		if !ValidateSteuernummer(s) {
			t.Errorf("ValidateSteuernummer(%q) = false, want true", s)
		}
	}
	invalid := []string{"123456789", "1121181508150", "abcdefghij", ""}
	for _, s := range invalid {
		if ValidateSteuernummer(s) {
			t.Errorf("ValidateSteuernummer(%q) = true, want false", s)
		}
	}
}

func TestValidateSteuernummerForBundesland(t *testing.T) {
	if !ValidateSteuernummerForBundesland("9396508150", "BW") { // 10 digits
		t.Error("BW 10-digit Steuernummer should be valid")
	}
	if ValidateSteuernummerForBundesland("93965081507", "BW") { // 11 digits, BW wants 10
		t.Error("BW with 11 digits should be invalid")
	}
	if !ValidateSteuernummerForBundesland("12345678901", "BY") { // 11 digits
		t.Error("BY 11-digit Steuernummer should be valid")
	}
	if !ValidateSteuernummerForBundesland("1121081508150", "BY") { // unified scheme always ok
		t.Error("unified 13-digit scheme should be valid for any state")
	}
	if ValidateSteuernummerForBundesland("12345678901", "XX") {
		t.Error("unknown Bundesland should be invalid")
	}
}
