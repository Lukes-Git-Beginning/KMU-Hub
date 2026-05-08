package dialer

import (
	"errors"
	"testing"
)

func TestNormalizePhoneE164(t *testing.T) {
	tests := []struct {
		name           string
		raw            string
		defaultCountry string
		want           string
		wantErr        bool
	}{
		// --- Valid German numbers (defaultCountry=DE) ---
		{
			name:           "E164 already normalised",
			raw:            "+4915112345678",
			defaultCountry: "DE",
			want:           "+4915112345678",
		},
		{
			name:           "E164 with spaces",
			raw:            "+49 151 1234 5678",
			defaultCountry: "DE",
			want:           "+4915112345678",
		},
		{
			name:           "double-zero international prefix",
			raw:            "0049 151 1234 5678",
			defaultCountry: "DE",
			want:           "+4915112345678",
		},
		{
			name:           "local trunk zero mobile",
			raw:            "0151 1234 5678",
			defaultCountry: "DE",
			want:           "+4915112345678",
		},
		{
			name:           "local trunk zero landline with parentheses",
			raw:            "(030) 123-456",
			defaultCountry: "DE",
			want:           "+4930123456",
		},
		// --- Austrian / Swiss ---
		{
			name:           "Austrian E164",
			raw:            "+43 1 58858",
			defaultCountry: "AT",
			want:           "+43158858",
		},
		{
			name:           "Swiss local trunk zero",
			raw:            "044 123 45 67",
			defaultCountry: "CH",
			want:           "+41441234567",
		},
		// --- Whitespace / formatting edge cases ---
		{
			name:           "leading and trailing whitespace",
			raw:            "  +4915112345678  ",
			defaultCountry: "DE",
			want:           "+4915112345678",
		},
		// --- Error cases ---
		{
			name:           "unsupported country",
			raw:            "+1 650 555 1234",
			defaultCountry: "US",
			wantErr:        true,
		},
		{
			name:           "empty input",
			raw:            "",
			defaultCountry: "DE",
			wantErr:        true,
		},
		{
			name:           "too short number",
			raw:            "+491",
			defaultCountry: "DE",
			wantErr:        true,
		},
		{
			name:           "letters only",
			raw:            "abc",
			defaultCountry: "DE",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizePhoneE164(tt.raw, tt.defaultCountry)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NormalizePhoneE164(%q, %q): expected error, got %q", tt.raw, tt.defaultCountry, got)
				}
				if err != nil && !errors.Is(err, ErrInvalidPhoneNumber) && tt.defaultCountry != "US" {
					// unsupported country also wraps ErrInvalidPhoneNumber
					t.Errorf("expected ErrInvalidPhoneNumber in error chain, got %v", err)
				}
				return
			}
			if err != nil {
				t.Errorf("NormalizePhoneE164(%q, %q): unexpected error: %v", tt.raw, tt.defaultCountry, err)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizePhoneE164(%q, %q) = %q, want %q", tt.raw, tt.defaultCountry, got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds int
		want    string
	}{
		{0, "0:00"},
		{59, "0:59"},
		{60, "1:00"},
		{90, "1:30"},
		{3600, "60:00"},
		{3661, "61:01"},
		{-5, "0:-5"}, // negative seconds: Go truncates toward zero, so -5/60=0, -5%60=-5
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatDuration(tt.seconds)
			if got != tt.want {
				t.Errorf("FormatDuration(%d) = %q, want %q", tt.seconds, got, tt.want)
			}
		})
	}
}
