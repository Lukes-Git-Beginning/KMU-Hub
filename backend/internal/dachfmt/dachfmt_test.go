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
	valid := []string{"DE123456789", "ATU12345678", "CHE-123.456.789", "CHE-123.456.789 MWST", "CHE123456789TVA"}
	for _, s := range valid {
		if !ValidateUStIDDACH(s) {
			t.Errorf("ValidateUStIDDACH(%q) = false, want true", s)
		}
	}
	invalid := []string{"DE12345678", "ATU1234567", "FR12345678901", "CHE12345678", ""}
	for _, s := range invalid {
		if ValidateUStIDDACH(s) {
			t.Errorf("ValidateUStIDDACH(%q) = true, want false", s)
		}
	}
}

func TestValidateUStIDEU(t *testing.T) {
	valid := []string{"DE123456789", "ATU12345678", "FR40303265045", "NL123456789B01", "IT12345678901", "CHE-123.456.789"}
	for _, s := range valid {
		if !ValidateUStIDEU(s) {
			t.Errorf("ValidateUStIDEU(%q) = false, want true", s)
		}
	}
	invalid := []string{"ZZ123456789", "DE12345678", "NL123456789X01", ""}
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
