package validation

import (
	"errors"
	"strings"
	"testing"
)

type sample struct {
	Email    string  `json:"email" validate:"required,email"`
	Password string  `json:"password" validate:"required,min=8"`
	Role     string  `json:"role" validate:"omitempty,oneof=admin manager member"`
	Company  *string `json:"company_id" validate:"omitempty,uuid4"`
	IBAN     string  `json:"iban" validate:"omitempty,iban"`
	Phone    string  `json:"phone" validate:"omitempty,phone_dach"`
}

func TestValidate_Valid(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	s := sample{Email: "a@b.de", Password: "supersecret", Role: "admin", Company: &id, IBAN: "DE89370400440532013000", Phone: "0151 12345678"}
	if err := Validate(&s); err != nil {
		t.Fatalf("Validate: unexpected error %v", err)
	}
}

func TestValidate_CollectsFieldErrors(t *testing.T) {
	s := sample{Email: "not-an-email", Password: "short", Role: "ceo", IBAN: "DE00", Phone: "xx"}
	err := Validate(&s)
	if err == nil {
		t.Fatal("Validate: want error, got nil")
	}
	var ve *Errors
	if !errors.As(err, &ve) {
		t.Fatalf("Validate: want *Errors, got %T", err)
	}
	got := map[string]string{}
	for _, f := range ve.Fields {
		got[f.Field] = f.Rule
	}
	for field, rule := range map[string]string{
		"email":    "email",
		"password": "min",
		"role":     "oneof",
		"iban":     "iban",
		"phone":    "phone_dach",
	} {
		if got[field] != rule {
			t.Errorf("field %q: want rule %q, got %q", field, rule, got[field])
		}
	}
}

func TestValidate_AggregateMessage(t *testing.T) {
	s := sample{Email: "", Password: "short"}
	err := Validate(&s)
	msg := err.Error()
	if !strings.Contains(msg, "email: is required") {
		t.Errorf("aggregate message missing email clause: %q", msg)
	}
	if !strings.Contains(msg, "password: must be at least 8") {
		t.Errorf("aggregate message missing password clause: %q", msg)
	}
}

func TestErrorBody_Shape(t *testing.T) {
	s := sample{Email: "bad", Password: "x"}
	err := Validate(&s)
	body, ok := ErrorBody(err)
	if !ok {
		t.Fatal("ErrorBody: want ok=true for *Errors")
	}
	eb, valid := body.(errorBody)
	if !valid {
		t.Fatalf("ErrorBody: unexpected type %T", body)
	}
	if eb.Code != Code {
		t.Errorf("code = %q, want %q", eb.Code, Code)
	}
	if eb.Error == "" {
		t.Error("error string must be non-empty")
	}
	if len(eb.Details) == 0 {
		t.Error("details must be populated")
	}
}

func TestErrorBody_NonValidationError(t *testing.T) {
	if _, ok := ErrorBody(errors.New("boom")); ok {
		t.Error("ErrorBody: want ok=false for non-validation error")
	}
}
