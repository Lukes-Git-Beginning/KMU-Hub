package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaValidator_Validate_EmptySchema(t *testing.T) {
	v := NewSchemaValidator()

	cases := []struct {
		name   string
		schema string
	}{
		{"nil bytes", ""},
		{"empty object", "{}"},
		{"literal null", "null"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var schemaJSON json.RawMessage
			if tc.schema != "" {
				schemaJSON = json.RawMessage(tc.schema)
			}
			result := v.Validate(schemaJSON, json.RawMessage(`{"anything":"goes"}`))
			require.NotNil(t, result)
			assert.True(t, result.Valid)
			assert.Empty(t, result.Errors)
		})
	}
}

func TestSchemaValidator_Validate_InvalidSchemaJSON(t *testing.T) {
	v := NewSchemaValidator()
	result := v.Validate(json.RawMessage(`{not json`), json.RawMessage(`{}`))
	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "invalid schema")
}

func TestSchemaValidator_Validate_InvalidSettingsJSON(t *testing.T) {
	v := NewSchemaValidator()
	schema := `{"type":"object","properties":{"name":{"type":"string"}}}`
	result := v.Validate(json.RawMessage(schema), json.RawMessage(`{not json`))
	assert.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "invalid settings JSON")
}

func TestSchemaValidator_Validate_RequiredField(t *testing.T) {
	v := NewSchemaValidator()
	schema := `{"type":"object","properties":{"apiKey":{"type":"string"}},"required":["apiKey"]}`

	t.Run("missing required field", func(t *testing.T) {
		result := v.Validate(json.RawMessage(schema), json.RawMessage(`{}`))
		assert.False(t, result.Valid)
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "required field 'apiKey' is missing")
	})

	t.Run("present required field", func(t *testing.T) {
		result := v.Validate(json.RawMessage(schema), json.RawMessage(`{"apiKey":"abc"}`))
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})
}

func TestSchemaValidator_Validate_AdditionalPropertiesAllowed(t *testing.T) {
	v := NewSchemaValidator()
	schema := `{"type":"object","properties":{"name":{"type":"string"}}}`
	result := v.Validate(json.RawMessage(schema), json.RawMessage(`{"name":"ok","extra":123}`))
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestSchemaValidator_Validate_MultiplePropertyErrorsAccumulate(t *testing.T) {
	v := NewSchemaValidator()
	schema := `{
		"type":"object",
		"properties":{
			"name":{"type":"string","minLength":3},
			"age":{"type":"number","minimum":0}
		},
		"required":["name","age"]
	}`
	result := v.Validate(json.RawMessage(schema), json.RawMessage(`{"name":"ab","age":-5}`))
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 2)
}

func TestValidateProperty_String(t *testing.T) {
	minLen := 2
	maxLen := 5
	prop := SchemaProperty{Type: "string", MinLength: &minLen, MaxLength: &maxLen}

	t.Run("wrong type", func(t *testing.T) {
		errs := validateProperty("f", 42, prop)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0], "must be a string")
	})

	t.Run("too short", func(t *testing.T) {
		errs := validateProperty("f", "a", prop)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0], "at least 2 characters")
	})

	t.Run("too long", func(t *testing.T) {
		errs := validateProperty("f", "abcdef", prop)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0], "at most 5 characters")
	})

	t.Run("within bounds", func(t *testing.T) {
		errs := validateProperty("f", "abc", prop)
		assert.Empty(t, errs)
	})
}

func TestValidateProperty_Number(t *testing.T) {
	minV := 1.0
	maxV := 10.0
	prop := SchemaProperty{Type: "number", Minimum: &minV, Maximum: &maxV}

	t.Run("wrong type", func(t *testing.T) {
		errs := validateProperty("f", "not-a-number", prop)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0], "must be a number")
	})

	t.Run("below minimum", func(t *testing.T) {
		errs := validateProperty("f", 0.5, prop)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0], "must be >=")
	})

	t.Run("above maximum", func(t *testing.T) {
		errs := validateProperty("f", 11.0, prop)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0], "must be <=")
	})

	t.Run("integer type uses same numeric branch", func(t *testing.T) {
		intProp := SchemaProperty{Type: "integer", Minimum: &minV, Maximum: &maxV}
		errs := validateProperty("f", 5.0, intProp)
		assert.Empty(t, errs)
	})

	t.Run("within bounds", func(t *testing.T) {
		errs := validateProperty("f", 5.0, prop)
		assert.Empty(t, errs)
	})
}

func TestValidateProperty_Boolean(t *testing.T) {
	prop := SchemaProperty{Type: "boolean"}

	t.Run("wrong type", func(t *testing.T) {
		errs := validateProperty("f", "true", prop)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0], "must be a boolean")
	})

	t.Run("valid bool", func(t *testing.T) {
		errs := validateProperty("f", true, prop)
		assert.Empty(t, errs)
	})
}

func TestValidateProperty_UnknownType_NoTypeCheck(t *testing.T) {
	// A type outside string/number/integer/boolean falls through the switch
	// with no type-specific validation - only enum (if configured) still applies.
	prop := SchemaProperty{Type: "array"}
	errs := validateProperty("f", []any{1, 2, 3}, prop)
	assert.Empty(t, errs)
}

func TestValidateProperty_Enum(t *testing.T) {
	prop := SchemaProperty{Type: "string", Enum: []any{"a", "b", "c"}}

	t.Run("value in enum", func(t *testing.T) {
		errs := validateProperty("f", "b", prop)
		assert.Empty(t, errs)
	})

	t.Run("value not in enum", func(t *testing.T) {
		errs := validateProperty("f", "z", prop)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0], "must be one of")
	})
}
