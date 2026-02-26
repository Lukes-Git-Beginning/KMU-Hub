package config

import (
	"encoding/json"
	"fmt"
)

// SchemaValidator validates JSON data against a JSON Schema.
// Uses a lightweight approach — validates type, required, enum, pattern, min/max constraints.
type SchemaValidator struct{}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{}
}

// SchemaProperty represents a single property in a JSON Schema
type SchemaProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Default     any      `json:"default,omitempty"`
	Enum        []any    `json:"enum,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
	Minimum     *float64 `json:"minimum,omitempty"`
	Maximum     *float64 `json:"maximum,omitempty"`
	MinLength   *int     `json:"minLength,omitempty"`
	MaxLength   *int     `json:"maxLength,omitempty"`
}

// Schema represents a simplified JSON Schema (object type)
type Schema struct {
	Type       string                    `json:"type"`
	Properties map[string]SchemaProperty `json:"properties"`
	Required   []string                  `json:"required,omitempty"`
}

// ValidationResult holds the result of validation
type ValidationResult struct {
	Valid  bool
	Errors []string
}

// Validate validates settings JSON against a JSON Schema
func (v *SchemaValidator) Validate(schemaJSON, settingsJSON json.RawMessage) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Empty schema means no validation needed
	if len(schemaJSON) == 0 || string(schemaJSON) == "{}" || string(schemaJSON) == "null" {
		return result
	}

	var schema Schema
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid schema: %v", err))
		return result
	}

	var settings map[string]any
	if err := json.Unmarshal(settingsJSON, &settings); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid settings JSON: %v", err))
		return result
	}

	// Check required fields
	for _, req := range schema.Required {
		if _, ok := settings[req]; !ok {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("required field '%s' is missing", req))
		}
	}

	// Validate each provided field against schema
	for name, value := range settings {
		prop, exists := schema.Properties[name]
		if !exists {
			continue // Allow additional properties
		}

		if errs := validateProperty(name, value, prop); len(errs) > 0 {
			result.Valid = false
			result.Errors = append(result.Errors, errs...)
		}
	}

	return result
}

func validateProperty(name string, value any, prop SchemaProperty) []string {
	var errs []string

	// Type check
	switch prop.Type {
	case "string":
		str, ok := value.(string)
		if !ok {
			errs = append(errs, fmt.Sprintf("field '%s' must be a string", name))
			return errs
		}
		if prop.MinLength != nil && len(str) < *prop.MinLength {
			errs = append(errs, fmt.Sprintf("field '%s' must be at least %d characters", name, *prop.MinLength))
		}
		if prop.MaxLength != nil && len(str) > *prop.MaxLength {
			errs = append(errs, fmt.Sprintf("field '%s' must be at most %d characters", name, *prop.MaxLength))
		}
	case "number", "integer":
		num, ok := value.(float64)
		if !ok {
			errs = append(errs, fmt.Sprintf("field '%s' must be a number", name))
			return errs
		}
		if prop.Minimum != nil && num < *prop.Minimum {
			errs = append(errs, fmt.Sprintf("field '%s' must be >= %v", name, *prop.Minimum))
		}
		if prop.Maximum != nil && num > *prop.Maximum {
			errs = append(errs, fmt.Sprintf("field '%s' must be <= %v", name, *prop.Maximum))
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			errs = append(errs, fmt.Sprintf("field '%s' must be a boolean", name))
		}
	}

	// Enum check
	if len(prop.Enum) > 0 {
		found := false
		for _, e := range prop.Enum {
			if fmt.Sprintf("%v", e) == fmt.Sprintf("%v", value) {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Sprintf("field '%s' must be one of: %v", name, prop.Enum))
		}
	}

	return errs
}
