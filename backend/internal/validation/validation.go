// Package validation provides a shared go-playground/validator instance with
// DACH-specific custom validators and a structured error type that maps cleanly
// onto the HTTP error response contract used across the gateway.
//
// The HTTP layer should not call into go-playground directly; it calls Validate
// and, on failure, ErrorBody to obtain a JSON-serialisable body of the shape
//
//	{"error": "<human aggregate>", "code": "validation_failed",
//	 "details": [{"field": "...", "rule": "...", "param": "..."}]}
//
// Custom format validators (iban, bic, phone_dach, plz_dach, ustid_dach,
// ustid_eu, steuernr) return false for empty input, matching the builtin
// "email" semantics. Combine them with "omitempty" for optional fields.
package validation

import (
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/dachfmt"
)

var (
	once     sync.Once
	instance *validator.Validate
)

// validate returns the lazily-initialised shared validator instance.
func validate() *validator.Validate {
	once.Do(func() {
		v := validator.New(validator.WithRequiredStructEnabled())

		// Report the JSON field name (not the Go field name) in errors so the
		// "field" in details matches the request body the client sent.
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" || name == "" {
				return fld.Name
			}
			return name
		})

		mustRegister(v, "phone_dach", func(fl validator.FieldLevel) bool {
			country := fl.Param()
			if country == "" {
				country = "DE"
			}
			_, err := dachfmt.NormalizePhoneE164(fl.Field().String(), country)
			return err == nil
		})
		mustRegister(v, "iban", func(fl validator.FieldLevel) bool {
			return dachfmt.ValidateIBAN(fl.Field().String())
		})
		mustRegister(v, "bic", func(fl validator.FieldLevel) bool {
			return dachfmt.ValidateBIC(fl.Field().String())
		})
		mustRegister(v, "plz_dach", func(fl validator.FieldLevel) bool {
			if c := fl.Param(); c != "" {
				return dachfmt.ValidatePLZ(fl.Field().String(), c)
			}
			return dachfmt.ValidatePLZAny(fl.Field().String())
		})
		mustRegister(v, "ustid_dach", func(fl validator.FieldLevel) bool {
			return dachfmt.ValidateUStIDDACH(fl.Field().String())
		})
		mustRegister(v, "ustid_eu", func(fl validator.FieldLevel) bool {
			return dachfmt.ValidateUStIDEU(fl.Field().String())
		})
		mustRegister(v, "steuernr", func(fl validator.FieldLevel) bool {
			return dachfmt.ValidateSteuernummer(fl.Field().String())
		})
		// decimal_gt0 accepts a decimal string strictly greater than zero. Used for
		// monetary amounts (e.g. payment amounts) that arrive as JSON strings and
		// must never be negative, zero, or non-numeric.
		mustRegister(v, "decimal_gt0", func(fl validator.FieldLevel) bool {
			d, err := decimal.NewFromString(strings.TrimSpace(fl.Field().String()))
			if err != nil {
				return false
			}
			return d.GreaterThan(decimal.Zero)
		})

		instance = v
	})
	return instance
}

func mustRegister(v *validator.Validate, tag string, fn validator.Func) {
	if err := v.RegisterValidation(tag, fn); err != nil {
		panic("validation: failed to register " + tag + ": " + err.Error())
	}
}

// Validate validates the struct pointed to by s against its `validate` tags.
// It returns nil when valid, a *Errors when one or more fields fail, or the
// raw go-playground error for programmer mistakes (e.g. passing a non-struct).
func Validate(s any) error {
	if err := validate().Struct(s); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return fromValidatorErrors(ve)
		}
		return err
	}
	return nil
}
