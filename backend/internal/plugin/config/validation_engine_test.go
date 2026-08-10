package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

func rule(t *testing.T, ruleType models.ValidationRuleType, field, msg string, cfg any) *models.ValidationRule {
	t.Helper()
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	return &models.ValidationRule{
		Name:         "test-rule",
		FieldName:    field,
		RuleType:     ruleType,
		RuleConfig:   raw,
		ErrorMessage: msg,
		Enabled:      true,
	}
}

func TestValidationEngine_ValidateEntity_SkipsDisabledRules(t *testing.T) {
	e := NewValidationEngine()
	r := rule(t, models.ValidationRuleTypeRegex, "email", "bad email", RegexConfig{Pattern: `^\d+$`, Required: true})
	r.Enabled = false

	errs := e.ValidateEntity([]*models.ValidationRule{r}, map[string]any{"email": "not-a-number"})
	assert.Empty(t, errs)
}

func TestValidationEngine_ValidateEntity_UnknownRuleTypeYieldsNoError(t *testing.T) {
	e := NewValidationEngine()
	r := rule(t, models.ValidationRuleTypeCustom, "field", "msg", map[string]any{})

	errs := e.ValidateEntity([]*models.ValidationRule{r}, map[string]any{})
	assert.Empty(t, errs)
}

func TestValidationEngine_ValidateEntity_AccumulatesAcrossRules(t *testing.T) {
	e := NewValidationEngine()
	r1 := rule(t, models.ValidationRuleTypeRegex, "code", "bad code", RegexConfig{Pattern: `^\d+$`, Required: true})
	r2 := rule(t, models.ValidationRuleTypeRange, "age", "bad age", RangeConfig{Min: floatPtr(0), Required: true})

	errs := e.ValidateEntity([]*models.ValidationRule{r1, r2}, map[string]any{"code": "abc", "age": -1.0})
	require.Len(t, errs, 2)
}

func TestValidationEngine_ValidateEntity_DispatchesFormatAndEnumRuleTypes(t *testing.T) {
	e := NewValidationEngine()
	r1 := rule(t, models.ValidationRuleTypeFormat, "email", "bad email", FormatConfig{Format: "email", Required: true})
	r2 := rule(t, models.ValidationRuleTypeEnum, "status", "bad status", EnumConfig{Values: []string{"open", "closed"}})

	errs := e.ValidateEntity([]*models.ValidationRule{r1, r2}, map[string]any{"email": "not-an-email", "status": "unknown"})
	require.Len(t, errs, 2)
}

func floatPtr(f float64) *float64 { return &f }

// ---- evalRegex ----

func TestEvalRegex(t *testing.T) {
	e := NewValidationEngine()

	t.Run("invalid rule config JSON is a no-op", func(t *testing.T) {
		r := &models.ValidationRule{FieldName: "f", Name: "n", RuleConfig: json.RawMessage(`{not json`)}
		got := e.evalRegex(r, "x", true)
		assert.Nil(t, got)
	})

	t.Run("missing and required", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRegex, "f", "required", RegexConfig{Pattern: `^\d+$`, Required: true})
		got := e.evalRegex(r, nil, false)
		require.NotNil(t, got)
		assert.Equal(t, "required", got.Message)
	})

	t.Run("missing and not required", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRegex, "f", "required", RegexConfig{Pattern: `^\d+$`, Required: false})
		got := e.evalRegex(r, nil, false)
		assert.Nil(t, got)
	})

	t.Run("wrong type", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRegex, "f", "bad type", RegexConfig{Pattern: `^\d+$`})
		got := e.evalRegex(r, 42, true)
		require.NotNil(t, got)
	})

	t.Run("empty string not required", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRegex, "f", "bad", RegexConfig{Pattern: `^\d+$`, Required: false})
		got := e.evalRegex(r, "", true)
		assert.Nil(t, got)
	})

	t.Run("pattern mismatch", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRegex, "f", "must be numeric", RegexConfig{Pattern: `^\d+$`})
		got := e.evalRegex(r, "abc", true)
		require.NotNil(t, got)
		assert.Equal(t, "must be numeric", got.Message)
	})

	t.Run("pattern match", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRegex, "f", "must be numeric", RegexConfig{Pattern: `^\d+$`})
		got := e.evalRegex(r, "123", true)
		assert.Nil(t, got)
	})

	t.Run("invalid regex pattern surfaces as error", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRegex, "f", "bad pattern", RegexConfig{Pattern: `(unclosed`})
		got := e.evalRegex(r, "abc", true)
		require.NotNil(t, got)
	})
}

// ---- evalRange ----

func TestEvalRange(t *testing.T) {
	e := NewValidationEngine()

	t.Run("invalid rule config JSON is a no-op", func(t *testing.T) {
		r := &models.ValidationRule{FieldName: "f", Name: "n", RuleConfig: json.RawMessage(`{not json`)}
		got := e.evalRange(r, 1, true)
		assert.Nil(t, got)
	})

	t.Run("missing and required", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRange, "f", "required", RangeConfig{Required: true})
		got := e.evalRange(r, nil, false)
		require.NotNil(t, got)
	})

	t.Run("missing and not required", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRange, "f", "required", RangeConfig{Required: false})
		got := e.evalRange(r, nil, false)
		assert.Nil(t, got)
	})

	t.Run("not a number", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRange, "f", "must be numeric", RangeConfig{})
		got := e.evalRange(r, "abc", true)
		require.NotNil(t, got)
	})

	t.Run("below minimum", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRange, "f", "too small", RangeConfig{Min: floatPtr(10)})
		got := e.evalRange(r, 5.0, true)
		require.NotNil(t, got)
	})

	t.Run("above maximum", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRange, "f", "too large", RangeConfig{Max: floatPtr(10)})
		got := e.evalRange(r, 15.0, true)
		require.NotNil(t, got)
	})

	t.Run("within range", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRange, "f", "err", RangeConfig{Min: floatPtr(0), Max: floatPtr(10)})
		got := e.evalRange(r, 5.0, true)
		assert.Nil(t, got)
	})
}

// ---- evalRequiredIf ----

func TestEvalRequiredIf(t *testing.T) {
	e := NewValidationEngine()

	t.Run("invalid rule config JSON is a no-op", func(t *testing.T) {
		r := &models.ValidationRule{FieldName: "f", Name: "n", RuleConfig: json.RawMessage(`{not json`)}
		got := e.evalRequiredIf(r, nil, false, map[string]any{})
		assert.Nil(t, got)
	})

	t.Run("eq condition met, value missing", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRequiredIf, "f", "required", RequiredIfConfig{DependsOn: "type", Operator: "eq", Value: "company"})
		got := e.evalRequiredIf(r, nil, false, map[string]any{"type": "company"})
		require.NotNil(t, got)
	})

	t.Run("eq condition not met", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRequiredIf, "f", "required", RequiredIfConfig{DependsOn: "type", Operator: "eq", Value: "company"})
		got := e.evalRequiredIf(r, nil, false, map[string]any{"type": "person"})
		assert.Nil(t, got)
	})

	t.Run("neq condition met", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRequiredIf, "f", "required", RequiredIfConfig{DependsOn: "type", Operator: "neq", Value: "company"})
		got := e.evalRequiredIf(r, nil, false, map[string]any{"type": "person"})
		require.NotNil(t, got)
	})

	t.Run("exists condition met", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRequiredIf, "f", "required", RequiredIfConfig{DependsOn: "dep", Operator: "exists"})
		got := e.evalRequiredIf(r, nil, false, map[string]any{"dep": "anything"})
		require.NotNil(t, got)
	})

	t.Run("exists condition not met (missing dep)", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRequiredIf, "f", "required", RequiredIfConfig{DependsOn: "dep", Operator: "exists"})
		got := e.evalRequiredIf(r, nil, false, map[string]any{})
		assert.Nil(t, got)
	})

	t.Run("not_empty with non-empty string", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRequiredIf, "f", "required", RequiredIfConfig{DependsOn: "dep", Operator: "not_empty"})
		got := e.evalRequiredIf(r, nil, false, map[string]any{"dep": "  x  "})
		require.NotNil(t, got)
	})

	t.Run("not_empty with whitespace-only string", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRequiredIf, "f", "required", RequiredIfConfig{DependsOn: "dep", Operator: "not_empty"})
		got := e.evalRequiredIf(r, nil, false, map[string]any{"dep": "   "})
		assert.Nil(t, got)
	})

	t.Run("not_empty with non-string dep falls back to exists check", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRequiredIf, "f", "required", RequiredIfConfig{DependsOn: "dep", Operator: "not_empty"})
		got := e.evalRequiredIf(r, nil, false, map[string]any{"dep": 42})
		require.NotNil(t, got)
	})

	t.Run("condition met but value present", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRequiredIf, "f", "required", RequiredIfConfig{DependsOn: "type", Operator: "eq", Value: "company"})
		got := e.evalRequiredIf(r, "acme gmbh", true, map[string]any{"type": "company"})
		assert.Nil(t, got)
	})

	t.Run("unknown operator never triggers", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeRequiredIf, "f", "required", RequiredIfConfig{DependsOn: "type", Operator: "bogus"})
		got := e.evalRequiredIf(r, nil, false, map[string]any{"type": "company"})
		assert.Nil(t, got)
	})
}

// ---- evalFormat ----

func TestEvalFormat(t *testing.T) {
	e := NewValidationEngine()

	t.Run("invalid rule config JSON is a no-op", func(t *testing.T) {
		r := &models.ValidationRule{FieldName: "f", Name: "n", RuleConfig: json.RawMessage(`{not json`)}
		got := e.evalFormat(r, "x", true)
		assert.Nil(t, got)
	})

	t.Run("missing and required", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeFormat, "f", "required", FormatConfig{Format: "email", Required: true})
		got := e.evalFormat(r, nil, false)
		require.NotNil(t, got)
	})

	t.Run("missing and not required", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeFormat, "f", "required", FormatConfig{Format: "email", Required: false})
		got := e.evalFormat(r, nil, false)
		assert.Nil(t, got)
	})

	t.Run("wrong type", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeFormat, "f", "bad", FormatConfig{Format: "email"})
		got := e.evalFormat(r, 42, true)
		require.NotNil(t, got)
	})

	t.Run("empty string not required", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeFormat, "f", "bad", FormatConfig{Format: "email", Required: false})
		got := e.evalFormat(r, "", true)
		assert.Nil(t, got)
	})

	t.Run("unknown format is a no-op", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeFormat, "f", "bad", FormatConfig{Format: "not-a-real-format"})
		got := e.evalFormat(r, "anything", true)
		assert.Nil(t, got)
	})

	formatCases := []struct {
		format  string
		valid   string
		invalid string
	}{
		{"email", "user@example.com", "not-an-email"},
		{"url", "https://example.com/path", "ftp missing scheme"},
		{"date", "2026-08-10", "10.08.2026"},
		{"phone", "+49 30 1234567", "call me maybe"},
		{"iban", "DE89370400440532013000", "not-an-iban"},
	}
	for _, fc := range formatCases {
		t.Run("format "+fc.format+" valid", func(t *testing.T) {
			r := rule(t, models.ValidationRuleTypeFormat, "f", "bad "+fc.format, FormatConfig{Format: fc.format})
			got := e.evalFormat(r, fc.valid, true)
			assert.Nil(t, got, "expected %q to satisfy format %s", fc.valid, fc.format)
		})
		t.Run("format "+fc.format+" invalid", func(t *testing.T) {
			r := rule(t, models.ValidationRuleTypeFormat, "f", "bad "+fc.format, FormatConfig{Format: fc.format})
			got := e.evalFormat(r, fc.invalid, true)
			require.NotNil(t, got, "expected %q to violate format %s", fc.invalid, fc.format)
		})
	}
}

// ---- evalEnum ----

func TestEvalEnum(t *testing.T) {
	e := NewValidationEngine()

	t.Run("invalid rule config JSON is a no-op", func(t *testing.T) {
		r := &models.ValidationRule{FieldName: "f", Name: "n", RuleConfig: json.RawMessage(`{not json`)}
		got := e.evalEnum(r, "x", true)
		assert.Nil(t, got)
	})

	t.Run("missing and required", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeEnum, "f", "required", EnumConfig{Values: []string{"a", "b"}, Required: true})
		got := e.evalEnum(r, nil, false)
		require.NotNil(t, got)
	})

	t.Run("missing and not required", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeEnum, "f", "required", EnumConfig{Values: []string{"a", "b"}, Required: false})
		got := e.evalEnum(r, nil, false)
		assert.Nil(t, got)
	})

	t.Run("empty string not required", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeEnum, "f", "bad", EnumConfig{Values: []string{"a", "b"}, Required: false})
		got := e.evalEnum(r, "", true)
		assert.Nil(t, got)
	})

	t.Run("value in list", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeEnum, "f", "bad", EnumConfig{Values: []string{"a", "b"}})
		got := e.evalEnum(r, "b", true)
		assert.Nil(t, got)
	})

	t.Run("value not in list", func(t *testing.T) {
		r := rule(t, models.ValidationRuleTypeEnum, "f", "must be a or b", EnumConfig{Values: []string{"a", "b"}})
		got := e.evalEnum(r, "z", true)
		require.NotNil(t, got)
		assert.Equal(t, "must be a or b", got.Message)
	})
}

// ---- toFloat64 ----

func TestToFloat64(t *testing.T) {
	cases := []struct {
		name    string
		in      any
		want    float64
		wantOK  bool
	}{
		{"float64", float64(3.5), 3.5, true},
		{"float32", float32(2.5), 2.5, true},
		{"int", int(7), 7, true},
		{"int64", int64(9), 9, true},
		{"json.Number valid", json.Number("12.5"), 12.5, true},
		{"json.Number invalid", json.Number("not-a-number"), 0, false},
		{"unsupported string", "5", 0, false},
		{"unsupported nil", nil, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := toFloat64(tc.in)
			assert.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}
