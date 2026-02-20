package condition

import "github.com/kmuhub/kmuhub/internal/models"

// ConditionConfig is a convenience alias for the models.ConditionConfig type.
// It describes the dual evaluation mode: simple AND/OR tree or expr-lang expression.
type ConditionConfig = models.ConditionConfig

// Condition is a convenience alias for the models.Condition type.
// It represents the AND/OR tree structure used in simple mode.
type Condition = models.Condition

// Supported leaf condition operators for simple mode evaluation.
const (
	OpEquals     = "equals"
	OpNotEquals  = "not_equals"
	OpContains   = "contains"
	OpNotContain = "not_contains"
	OpStartsWith = "starts_with"
	OpEndsWith   = "ends_with"
	OpGT         = "gt"
	OpGTE        = "gte"
	OpLT         = "lt"
	OpLTE        = "lte"
	OpIn         = "in"
	OpNotIn      = "not_in"
	OpExists     = "exists"
	OpNotExists  = "not_exists"
	OpMatches    = "matches"
)
