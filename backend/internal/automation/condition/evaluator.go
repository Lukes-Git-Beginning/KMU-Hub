package condition

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Evaluator provides dual-mode condition evaluation:
//   - "simple" mode: AND/OR tree with leaf comparisons (generalized from inbox routing)
//   - "expression" mode: expr-lang expressions with typed environments
//
// Expression programs are cached in a sync.Map for repeated evaluations.
type Evaluator struct {
	exprCache sync.Map // map[string]*vm.Program
}

// NewEvaluator creates a new condition evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// Evaluate checks whether the given condition config matches the environment.
// Returns true if conditions are satisfied, false otherwise.
//
// Behavior by mode:
//   - "" (empty): always true (no condition = always fire)
//   - "simple": evaluates the AND/OR condition tree
//   - "expression": compiles and runs the expr-lang expression
func (e *Evaluator) Evaluate(_ context.Context, config models.ConditionConfig, env map[string]any) (bool, error) {
	switch config.Mode {
	case "":
		// No condition configured = always fire
		return true, nil

	case models.ConditionModeSimple:
		if config.Simple == nil {
			return true, nil
		}
		return evaluateSimple(config.Simple, env), nil

	case models.ConditionModeExpression:
		if config.Expression == "" {
			return true, nil
		}
		return e.evaluateExpr(config.Expression, env)

	default:
		return false, fmt.Errorf("unknown condition mode: %q", config.Mode)
	}
}

// ValidateExpression checks whether the expression compiles without executing it.
// Useful for frontend validation of user-entered expressions.
func (e *Evaluator) ValidateExpression(expression string, env map[string]any) error {
	_, err := expr.Compile(expression, expr.Env(env), expr.AsBool())
	return err
}

// InvalidateCache removes a cached compiled program for the given expression.
func (e *Evaluator) InvalidateCache(expression string) {
	e.exprCache.Delete(expression)
}

// evaluateSimple evaluates a simple AND/OR condition tree against the environment.
// Generalized from the inbox/routing/evaluator.go pattern.
func evaluateSimple(cond *models.Condition, env map[string]any) bool {
	// AND branch: all sub-conditions must be true (empty AND = vacuous truth)
	if cond.And != nil {
		for i := range cond.And {
			if !evaluateSimple(&cond.And[i], env) {
				return false
			}
		}
		return true
	}

	// OR branch: any sub-condition must be true (empty OR = false)
	if cond.Or != nil {
		for i := range cond.Or {
			if evaluateSimple(&cond.Or[i], env) {
				return true
			}
		}
		return false
	}

	// Leaf condition
	return evaluateLeaf(cond, env)
}

// evaluateLeaf evaluates a single leaf condition against the environment.
func evaluateLeaf(cond *models.Condition, env map[string]any) bool {
	fieldValue, fieldExists := getFieldValue(cond.Field, env)
	condValue := fmt.Sprintf("%v", cond.Value)

	switch cond.Operator {
	case OpEquals:
		return strings.EqualFold(fieldValue, condValue)

	case OpNotEquals:
		return !strings.EqualFold(fieldValue, condValue)

	case OpContains:
		return strings.Contains(strings.ToLower(fieldValue), strings.ToLower(condValue))

	case OpNotContain:
		return !strings.Contains(strings.ToLower(fieldValue), strings.ToLower(condValue))

	case OpStartsWith:
		return strings.HasPrefix(strings.ToLower(fieldValue), strings.ToLower(condValue))

	case OpEndsWith:
		return strings.HasSuffix(strings.ToLower(fieldValue), strings.ToLower(condValue))

	case OpGT:
		return compareNumeric(fieldValue, condValue) > 0

	case OpGTE:
		return compareNumeric(fieldValue, condValue) >= 0

	case OpLT:
		return compareNumeric(fieldValue, condValue) < 0

	case OpLTE:
		return compareNumeric(fieldValue, condValue) <= 0

	case OpIn:
		values := strings.Split(condValue, ",")
		lowerField := strings.ToLower(fieldValue)
		for _, v := range values {
			if strings.EqualFold(strings.TrimSpace(v), lowerField) {
				return true
			}
		}
		return false

	case OpNotIn:
		values := strings.Split(condValue, ",")
		lowerField := strings.ToLower(fieldValue)
		for _, v := range values {
			if strings.EqualFold(strings.TrimSpace(v), lowerField) {
				return false
			}
		}
		return true

	case OpExists:
		return fieldExists && fieldValue != ""

	case OpNotExists:
		return !fieldExists || fieldValue == ""

	case OpMatches:
		re, err := regexp.Compile(condValue)
		if err != nil {
			return false
		}
		return re.MatchString(fieldValue)

	default:
		return false
	}
}

// getFieldValue retrieves a field value from the environment map.
// Supports dotted paths like "deal.value" for nested maps.
func getFieldValue(field string, env map[string]any) (string, bool) {
	parts := strings.Split(field, ".")
	current := env

	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return "", false
		}

		// Last part: return the value
		if i == len(parts)-1 {
			return fmt.Sprintf("%v", val), true
		}

		// Intermediate part: must be a map
		next, ok := val.(map[string]any)
		if !ok {
			return "", false
		}
		current = next
	}

	return "", false
}

// compareNumeric compares two string-encoded numbers.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Non-numeric values are treated as 0.
func compareNumeric(a, b string) int {
	fa, errA := strconv.ParseFloat(a, 64)
	fb, errB := strconv.ParseFloat(b, 64)

	if errA != nil {
		fa = 0
	}
	if errB != nil {
		fb = 0
	}

	if fa < fb {
		return -1
	}
	if fa > fb {
		return 1
	}
	return 0
}

// evaluateExpr compiles and evaluates an expr-lang expression.
// Compiled programs are cached in a sync.Map for performance.
func (e *Evaluator) evaluateExpr(expression string, env map[string]any) (bool, error) {
	// Check cache
	cached, ok := e.exprCache.Load(expression)
	if !ok {
		// Compile and cache
		program, err := expr.Compile(expression, expr.Env(env), expr.AsBool())
		if err != nil {
			return false, fmt.Errorf("compile expression: %w", err)
		}
		e.exprCache.Store(expression, program)
		cached = program
	}

	program := cached.(*vm.Program)
	output, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("run expression: %w", err)
	}

	result, ok := output.(bool)
	if !ok {
		return false, fmt.Errorf("expression result is %T, expected bool", output)
	}

	return result, nil
}
