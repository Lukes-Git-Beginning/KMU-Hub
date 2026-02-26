package config

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ValidationEngine evaluates config-based validation rules against entity data
type ValidationEngine struct{}

// NewValidationEngine creates a new validation engine
func NewValidationEngine() *ValidationEngine {
	return &ValidationEngine{}
}

// FieldError represents a validation error for a specific field
type FieldError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// ValidateEntity validates entity data against a set of validation rules
func (e *ValidationEngine) ValidateEntity(rules []*models.ValidationRule, data map[string]any) []FieldError {
	var errors []FieldError

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if err := e.evaluateRule(rule, data); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

func (e *ValidationEngine) evaluateRule(rule *models.ValidationRule, data map[string]any) *FieldError {
	value, exists := data[rule.FieldName]

	switch models.ValidationRuleType(rule.RuleType) {
	case models.ValidationRuleTypeRegex:
		return e.evalRegex(rule, value, exists)
	case models.ValidationRuleTypeRange:
		return e.evalRange(rule, value, exists)
	case models.ValidationRuleTypeRequiredIf:
		return e.evalRequiredIf(rule, value, exists, data)
	case models.ValidationRuleTypeFormat:
		return e.evalFormat(rule, value, exists)
	case models.ValidationRuleTypeEnum:
		return e.evalEnum(rule, value, exists)
	default:
		return nil
	}
}

// RegexConfig holds the configuration for regex validation
type RegexConfig struct {
	Pattern  string `json:"pattern"`
	Required bool   `json:"required"`
}

func (e *ValidationEngine) evalRegex(rule *models.ValidationRule, value any, exists bool) *FieldError {
	var cfg RegexConfig
	if err := json.Unmarshal(rule.RuleConfig, &cfg); err != nil {
		return nil
	}

	if !exists || value == nil {
		if cfg.Required {
			return &FieldError{Field: rule.FieldName, Rule: rule.Name, Message: rule.ErrorMessage}
		}
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return &FieldError{Field: rule.FieldName, Rule: rule.Name, Message: rule.ErrorMessage}
	}

	if str == "" && !cfg.Required {
		return nil
	}

	matched, err := regexp.MatchString(cfg.Pattern, str)
	if err != nil || !matched {
		return &FieldError{Field: rule.FieldName, Rule: rule.Name, Message: rule.ErrorMessage}
	}

	return nil
}

// RangeConfig holds the configuration for range validation
type RangeConfig struct {
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	Required bool     `json:"required"`
}

func (e *ValidationEngine) evalRange(rule *models.ValidationRule, value any, exists bool) *FieldError {
	var cfg RangeConfig
	if err := json.Unmarshal(rule.RuleConfig, &cfg); err != nil {
		return nil
	}

	if !exists || value == nil {
		if cfg.Required {
			return &FieldError{Field: rule.FieldName, Rule: rule.Name, Message: rule.ErrorMessage}
		}
		return nil
	}

	num, ok := toFloat64(value)
	if !ok {
		return &FieldError{Field: rule.FieldName, Rule: rule.Name, Message: rule.ErrorMessage}
	}

	if cfg.Min != nil && num < *cfg.Min {
		return &FieldError{Field: rule.FieldName, Rule: rule.Name, Message: rule.ErrorMessage}
	}
	if cfg.Max != nil && num > *cfg.Max {
		return &FieldError{Field: rule.FieldName, Rule: rule.Name, Message: rule.ErrorMessage}
	}

	return nil
}

// RequiredIfConfig holds the configuration for required_if validation
type RequiredIfConfig struct {
	DependsOn string `json:"depends_on"`
	Operator  string `json:"operator"` // eq, neq, exists, not_empty
	Value     any    `json:"value,omitempty"`
}

func (e *ValidationEngine) evalRequiredIf(rule *models.ValidationRule, value any, exists bool, data map[string]any) *FieldError {
	var cfg RequiredIfConfig
	if err := json.Unmarshal(rule.RuleConfig, &cfg); err != nil {
		return nil
	}

	depValue, depExists := data[cfg.DependsOn]
	conditionMet := false

	switch cfg.Operator {
	case "eq":
		conditionMet = fmt.Sprintf("%v", depValue) == fmt.Sprintf("%v", cfg.Value)
	case "neq":
		conditionMet = fmt.Sprintf("%v", depValue) != fmt.Sprintf("%v", cfg.Value)
	case "exists":
		conditionMet = depExists && depValue != nil
	case "not_empty":
		if str, ok := depValue.(string); ok {
			conditionMet = strings.TrimSpace(str) != ""
		} else {
			conditionMet = depExists && depValue != nil
		}
	}

	if conditionMet && (!exists || value == nil || value == "") {
		return &FieldError{Field: rule.FieldName, Rule: rule.Name, Message: rule.ErrorMessage}
	}

	return nil
}

// FormatConfig holds the configuration for format validation
type FormatConfig struct {
	Format   string `json:"format"` // email, url, date, phone, iban
	Required bool   `json:"required"`
}

var formatPatterns = map[string]string{
	"email": `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`,
	"url":   `^https?://[^\s]+$`,
	"date":  `^\d{4}-\d{2}-\d{2}$`,
	"phone": `^[\+]?[\d\s\-\(\)]{6,20}$`,
	"iban":  `^[A-Z]{2}\d{2}[A-Z0-9]{10,30}$`,
}

func (e *ValidationEngine) evalFormat(rule *models.ValidationRule, value any, exists bool) *FieldError {
	var cfg FormatConfig
	if err := json.Unmarshal(rule.RuleConfig, &cfg); err != nil {
		return nil
	}

	if !exists || value == nil {
		if cfg.Required {
			return &FieldError{Field: rule.FieldName, Rule: rule.Name, Message: rule.ErrorMessage}
		}
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return &FieldError{Field: rule.FieldName, Rule: rule.Name, Message: rule.ErrorMessage}
	}

	if str == "" && !cfg.Required {
		return nil
	}

	pattern, found := formatPatterns[cfg.Format]
	if !found {
		return nil
	}

	matched, err := regexp.MatchString(pattern, str)
	if err != nil || !matched {
		return &FieldError{Field: rule.FieldName, Rule: rule.Name, Message: rule.ErrorMessage}
	}

	return nil
}

// EnumConfig holds the configuration for enum validation
type EnumConfig struct {
	Values   []string `json:"values"`
	Required bool     `json:"required"`
}

func (e *ValidationEngine) evalEnum(rule *models.ValidationRule, value any, exists bool) *FieldError {
	var cfg EnumConfig
	if err := json.Unmarshal(rule.RuleConfig, &cfg); err != nil {
		return nil
	}

	if !exists || value == nil {
		if cfg.Required {
			return &FieldError{Field: rule.FieldName, Rule: rule.Name, Message: rule.ErrorMessage}
		}
		return nil
	}

	str := fmt.Sprintf("%v", value)
	if str == "" && !cfg.Required {
		return nil
	}

	for _, v := range cfg.Values {
		if v == str {
			return nil
		}
	}

	return &FieldError{Field: rule.FieldName, Rule: rule.Name, Message: rule.ErrorMessage}
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
