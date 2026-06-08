package validation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// FieldError describes a single failed field validation.
type FieldError struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
	Param string `json:"param,omitempty"`
}

// Errors is the structured result of a failed Validate call. It implements the
// error interface; Error() returns a human-readable aggregate of all field
// failures, suitable for the contract-stable "error" field of the HTTP body.
type Errors struct {
	Fields []FieldError
}

func (e *Errors) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		parts = append(parts, fmt.Sprintf("%s: %s", f.Field, describe(f)))
	}
	return strings.Join(parts, "; ")
}

// fromValidatorErrors converts go-playground field errors into our type.
func fromValidatorErrors(ve validator.ValidationErrors) *Errors {
	fields := make([]FieldError, 0, len(ve))
	for _, fe := range ve {
		fields = append(fields, FieldError{
			Field: fe.Field(),
			Rule:  fe.Tag(),
			Param: fe.Param(),
		})
	}
	return &Errors{Fields: fields}
}

// describe renders a human-readable phrase for a single field failure.
func describe(f FieldError) string {
	switch f.Rule {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "uuid", "uuid4":
		return "must be a valid UUID"
	case "min":
		return "must be at least " + f.Param
	case "max":
		return "must be at most " + f.Param
	case "len":
		return "must be exactly " + f.Param + " characters"
	case "gt":
		return "must be greater than " + f.Param
	case "gte":
		return "must be greater than or equal to " + f.Param
	case "lte":
		return "must be less than or equal to " + f.Param
	case "oneof":
		return "must be one of [" + f.Param + "]"
	case "iban":
		return "must be a valid IBAN"
	case "bic":
		return "must be a valid BIC"
	case "phone_dach":
		return "must be a valid phone number"
	case "plz_dach":
		return "must be a valid postal code"
	case "ustid_dach", "ustid_eu":
		return "must be a valid VAT identification number"
	case "steuernr":
		return "must be a valid tax number"
	default:
		if f.Param != "" {
			return "failed " + f.Rule + " (" + f.Param + ")"
		}
		return "failed " + f.Rule
	}
}

type errorBody struct {
	Error   string       `json:"error"`
	Code    string       `json:"code"`
	Details []FieldError `json:"details"`
}

// Code is the stable machine-readable code emitted for every validation
// failure. The frontend can map this plus details[].rule to localised copy.
const Code = "validation_failed"

// ErrorBody builds a JSON-serialisable body for a validation error. It returns
// (body, true) when err is a *Errors, and (nil, false) otherwise so callers can
// fall back to a generic response.
func ErrorBody(err error) (any, bool) {
	var ve *Errors
	if errors.As(err, &ve) {
		return errorBody{Error: ve.Error(), Code: Code, Details: ve.Fields}, true
	}
	return nil, false
}
