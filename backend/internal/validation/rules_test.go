package validation

import (
	"errors"
	"testing"
)

// TestValidate_RuleTable drives every custom rule through the shared validator
// instance, so tag registration and the rule itself are exercised together.
func TestValidate_RuleTable(t *testing.T) {
	cases := []struct {
		rule string
		in   string
		ok   bool
	}{
		{"bic", "DEUTDEFF", true},
		{"bic", "DEUTDEFF500", true},
		{"bic", "nope", false},
		{"bic", "", false},
		{"plz_dach", "10115", true},
		{"plz_dach", "8001", true},
		{"plz_dach", "ab", false},
		{"plz_dach=DE", "10115", true},
		{"plz_dach=DE", "1011", false},
		{"plz_dach=CH", "8001", true},
		{"plz_dach=CH", "80010", false},
		{"ustid_dach", "DE136695976", true},
		{"ustid_dach", "DE136695978", false},
		{"ustid_dach", "FR40303265045", false},
		{"ustid_eu", "FR40303265045", true},
		{"ustid_eu", "ZZ123456789", false},
		{"steuernr", "21/815/08150", true},
		{"steuernr", "12", false},
		{"phone_dach", "0151 12345678", true},
		{"phone_dach=CH", "079 123 45 67", true},
		{"phone_dach=US", "0151 12345678", false}, // unsupported country: the param reaches the normaliser
		{"phone_dach=CH", "abc", false},
		{"phone_dach", "", false},
		{"decimal_gt0", "0.01", true},
		{"decimal_gt0", " 5 ", true},
		{"decimal_gt0", "0", false},
		{"decimal_gt0", "-1", false},
		{"decimal_gt0", "abc", false},
		{"decimal_gte0", "0", true},
		{"decimal_gte0", "12.5", true},
		{"decimal_gte0", "-0.01", false},
		{"decimal_gte0", "x", false},
		{"max_2dp", "10.99", true},
		{"max_2dp", "10", true},
		{"max_2dp", "10.999", false},
		{"max_2dp", "1e-3", false},
		{"max_2dp", "abc", false},
		{"rate_fraction", "0", true},
		{"rate_fraction", "0.19", true},
		{"rate_fraction", "1", true},
		{"rate_fraction", "0.1234", true},
		{"rate_fraction", "0.12345", false},
		{"rate_fraction", "19", false},
		{"rate_fraction", "1.0001", false},
		{"rate_fraction", "-0.1", false},
		{"rate_fraction", "abc", false},
		{"clearable_date", "", true},
		{"clearable_date", "2026-10-26", true},
		{"clearable_date", "26.10.2026", false},
		{"clearable_date", "2026-02-30", false},
		{"clearable_oneof=a b", "", true},
		{"clearable_oneof=a b", "b", true},
		{"clearable_oneof=a b", "a b", false},
		{"clearable_oneof=a b", "c", false},
		{"clearable_phone_dach", "", true},
		{"clearable_phone_dach", "0151 12345678", true},
		{"clearable_phone_dach", "abc", false},
		{"clearable_phone_dach=CH", "079 123 45 67", true},
		{"clearable_phone_dach=US", "0151 12345678", false},
	}
	for _, tc := range cases {
		err := validate().Var(tc.in, tc.rule)
		if (err == nil) != tc.ok {
			t.Errorf("%s(%q): err = %v, want ok=%v", tc.rule, tc.in, err, tc.ok)
		}
	}
}

func TestValidate_NonStructIsProgrammerError(t *testing.T) {
	err := Validate("not a struct")
	if err == nil {
		t.Fatal("want error for non-struct input")
	}
	var ve *Errors
	if errors.As(err, &ve) {
		t.Errorf("non-struct input must surface the raw validator error, got *Errors: %v", err)
	}
}

func TestValidate_FieldNameFallsBackToGoName(t *testing.T) {
	type form struct {
		NoTag   string `validate:"required"`
		Dash    string `json:"-" validate:"required"`
		Options string `json:"opts,omitempty" validate:"required"`
	}
	err := Validate(&form{})
	var ve *Errors
	if !errors.As(err, &ve) {
		t.Fatalf("want *Errors, got %v", err)
	}
	got := map[string]bool{}
	for _, f := range ve.Fields {
		got[f.Field] = true
	}
	for _, want := range []string{"NoTag", "Dash", "opts"} {
		if !got[want] {
			t.Errorf("field %q missing in %v", want, ve.Fields)
		}
	}
}

func TestErrors_Describe(t *testing.T) {
	cases := []struct {
		f    FieldError
		want string
	}{
		{FieldError{Field: "a", Rule: "required"}, "a: is required"},
		{FieldError{Field: "a", Rule: "email"}, "a: must be a valid email"},
		{FieldError{Field: "a", Rule: "uuid"}, "a: must be a valid UUID"},
		{FieldError{Field: "a", Rule: "uuid4"}, "a: must be a valid UUID"},
		{FieldError{Field: "a", Rule: "min", Param: "3"}, "a: must be at least 3"},
		{FieldError{Field: "a", Rule: "max", Param: "9"}, "a: must be at most 9"},
		{FieldError{Field: "a", Rule: "len", Param: "5"}, "a: must be exactly 5 characters"},
		{FieldError{Field: "a", Rule: "gt", Param: "0"}, "a: must be greater than 0"},
		{FieldError{Field: "a", Rule: "gte", Param: "1"}, "a: must be greater than or equal to 1"},
		{FieldError{Field: "a", Rule: "lte", Param: "7"}, "a: must be less than or equal to 7"},
		{FieldError{Field: "a", Rule: "oneof", Param: "x y"}, "a: must be one of [x y]"},
		{FieldError{Field: "a", Rule: "clearable_oneof", Param: "x y"}, "a: must be one of [x y]"},
		{FieldError{Field: "a", Rule: "iban"}, "a: must be a valid IBAN"},
		{FieldError{Field: "a", Rule: "bic"}, "a: must be a valid BIC"},
		{FieldError{Field: "a", Rule: "phone_dach"}, "a: must be a valid phone number"},
		{FieldError{Field: "a", Rule: "clearable_phone_dach"}, "a: must be a valid phone number"},
		{FieldError{Field: "a", Rule: "plz_dach"}, "a: must be a valid postal code"},
		{FieldError{Field: "a", Rule: "ustid_dach"}, "a: must be a valid VAT identification number"},
		{FieldError{Field: "a", Rule: "ustid_eu"}, "a: must be a valid VAT identification number"},
		{FieldError{Field: "a", Rule: "steuernr"}, "a: must be a valid tax number"},
		{FieldError{Field: "a", Rule: "rate_fraction"}, "a: failed rate_fraction"},
		{FieldError{Field: "a", Rule: "unknown", Param: "p"}, "a: failed unknown (p)"},
	}
	for _, tc := range cases {
		if got := (&Errors{Fields: []FieldError{tc.f}}).Error(); got != tc.want {
			t.Errorf("rule %q: got %q, want %q", tc.f.Rule, got, tc.want)
		}
	}
	two := &Errors{Fields: []FieldError{{Field: "a", Rule: "required"}, {Field: "b", Rule: "email"}}}
	if got, want := two.Error(), "a: is required; b: must be a valid email"; got != want {
		t.Errorf("joined = %q, want %q", got, want)
	}
	if got := (&Errors{}).Error(); got != "" {
		t.Errorf("empty Errors.Error() = %q, want empty", got)
	}
}
