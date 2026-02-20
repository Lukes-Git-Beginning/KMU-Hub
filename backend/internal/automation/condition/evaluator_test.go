package condition

import (
	"context"
	"testing"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluator_EmptyConfig(t *testing.T) {
	e := NewEvaluator()
	result, err := e.Evaluate(context.Background(), models.ConditionConfig{}, nil)
	require.NoError(t, err)
	assert.True(t, result, "empty config should always match")
}

func TestEvaluator_EmptyMode(t *testing.T) {
	e := NewEvaluator()
	result, err := e.Evaluate(context.Background(), models.ConditionConfig{Mode: ""}, nil)
	require.NoError(t, err)
	assert.True(t, result, "empty mode should always match")
}

func TestEvaluator_SimpleNilCondition(t *testing.T) {
	e := NewEvaluator()
	config := models.ConditionConfig{Mode: "simple", Simple: nil}
	result, err := e.Evaluate(context.Background(), config, nil)
	require.NoError(t, err)
	assert.True(t, result, "nil simple condition should match")
}

func TestEvaluator_UnknownMode(t *testing.T) {
	e := NewEvaluator()
	config := models.ConditionConfig{Mode: "quantum"}
	_, err := e.Evaluate(context.Background(), config, nil)
	assert.Error(t, err, "unknown mode should return error")
}

// ============================================================================
// Simple mode: AND/OR tree
// ============================================================================

func TestEvaluator_SimpleAND_AllTrue(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"status": "active", "priority": "high"}

	config := models.ConditionConfig{
		Mode: "simple",
		Simple: &models.Condition{
			And: []models.Condition{
				{Field: "status", Operator: "equals", Value: "active"},
				{Field: "priority", Operator: "equals", Value: "high"},
			},
		},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_SimpleAND_OneFalse(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"status": "active", "priority": "low"}

	config := models.ConditionConfig{
		Mode: "simple",
		Simple: &models.Condition{
			And: []models.Condition{
				{Field: "status", Operator: "equals", Value: "active"},
				{Field: "priority", Operator: "equals", Value: "high"},
			},
		},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.False(t, result)
}

func TestEvaluator_SimpleAND_Empty(t *testing.T) {
	e := NewEvaluator()
	config := models.ConditionConfig{
		Mode: "simple",
		Simple: &models.Condition{
			And: []models.Condition{},
		},
	}

	result, err := e.Evaluate(context.Background(), config, nil)
	require.NoError(t, err)
	assert.True(t, result, "empty AND is vacuous truth")
}

func TestEvaluator_SimpleOR_OneTrue(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"status": "closed"}

	config := models.ConditionConfig{
		Mode: "simple",
		Simple: &models.Condition{
			Or: []models.Condition{
				{Field: "status", Operator: "equals", Value: "active"},
				{Field: "status", Operator: "equals", Value: "closed"},
			},
		},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_SimpleOR_AllFalse(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"status": "pending"}

	config := models.ConditionConfig{
		Mode: "simple",
		Simple: &models.Condition{
			Or: []models.Condition{
				{Field: "status", Operator: "equals", Value: "active"},
				{Field: "status", Operator: "equals", Value: "closed"},
			},
		},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.False(t, result)
}

func TestEvaluator_SimpleOR_Empty(t *testing.T) {
	e := NewEvaluator()
	config := models.ConditionConfig{
		Mode: "simple",
		Simple: &models.Condition{
			Or: []models.Condition{},
		},
	}

	result, err := e.Evaluate(context.Background(), config, nil)
	require.NoError(t, err)
	assert.False(t, result, "empty OR returns false")
}

func TestEvaluator_SimpleNestedANDOR(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"status": "active", "priority": "low", "source": "web"}

	// (status == active) AND (priority == high OR source == web)
	config := models.ConditionConfig{
		Mode: "simple",
		Simple: &models.Condition{
			And: []models.Condition{
				{Field: "status", Operator: "equals", Value: "active"},
				{
					Or: []models.Condition{
						{Field: "priority", Operator: "equals", Value: "high"},
						{Field: "source", Operator: "equals", Value: "web"},
					},
				},
			},
		},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

// ============================================================================
// Leaf operators
// ============================================================================

func TestEvaluator_Equals(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"name": "John"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "name", Operator: "equals", Value: "john"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result, "equals should be case-insensitive")
}

func TestEvaluator_EqualsNumber(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"count": "42"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "count", Operator: "equals", Value: "42"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_NotEquals(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"status": "active"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "status", Operator: "not_equals", Value: "closed"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_Contains(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"email": "user@example.com"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "email", Operator: "contains", Value: "example"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_NotContains(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"email": "user@example.com"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "email", Operator: "not_contains", Value: "spam"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_StartsWith(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"name": "KMU Hub GmbH"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "name", Operator: "starts_with", Value: "kmu"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_EndsWith(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"name": "KMU Hub GmbH"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "name", Operator: "ends_with", Value: "gmbh"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_GT(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"value": "15000"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "value", Operator: "gt", Value: "10000"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_GT_Equal(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"value": "10000"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "value", Operator: "gt", Value: "10000"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.False(t, result, "gt should not match equal values")
}

func TestEvaluator_GTE(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"value": "10000"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "value", Operator: "gte", Value: "10000"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_LT(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"value": "5000"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "value", Operator: "lt", Value: "10000"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_LTE(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"value": "10000"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "value", Operator: "lte", Value: "10000"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_In(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"status": "Won"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "status", Operator: "in", Value: "Won,Lost,Negotiation"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_In_NotFound(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"status": "Pending"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "status", Operator: "in", Value: "Won,Lost"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.False(t, result)
}

func TestEvaluator_NotIn(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"status": "Active"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "status", Operator: "not_in", Value: "Won,Lost"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_Exists(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"email": "test@example.com"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "email", Operator: "exists"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_Exists_Missing(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"name": "Test"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "email", Operator: "exists"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.False(t, result)
}

func TestEvaluator_NotExists(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"name": "Test"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "phone", Operator: "not_exists"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_Matches(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"email": "user@example.com"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "email", Operator: "matches", Value: `^[a-z]+@example\.com$`},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_Matches_InvalidRegex(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"email": "test"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "email", Operator: "matches", Value: "[invalid"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.False(t, result, "invalid regex should return false, not error")
}

func TestEvaluator_UnknownOperator(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"x": "y"}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "x", Operator: "between", Value: "1"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.False(t, result, "unknown operator should return false")
}

func TestEvaluator_DottedFieldPath(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{
		"deal": map[string]any{
			"value": "25000",
		},
	}

	config := models.ConditionConfig{
		Mode:   "simple",
		Simple: &models.Condition{Field: "deal.value", Operator: "gt", Value: "10000"},
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

// ============================================================================
// Expression mode
// ============================================================================

func TestEvaluator_ExprBasicComparison(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"value": 15000.0}

	config := models.ConditionConfig{
		Mode:       "expression",
		Expression: "value > 10000",
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_ExprStringComparison(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"stage_name": "Won"}

	config := models.ConditionConfig{
		Mode:       "expression",
		Expression: `stage_name == "Won"`,
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_ExprCombinedConditions(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{
		"value":      15000.0,
		"stage_name": "Negotiation",
	}

	config := models.ConditionConfig{
		Mode:       "expression",
		Expression: `value > 5000 and stage_name != "Lost"`,
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_ExprCombinedFalse(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{
		"value":      3000.0,
		"stage_name": "Lost",
	}

	config := models.ConditionConfig{
		Mode:       "expression",
		Expression: `value > 5000 and stage_name != "Lost"`,
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.False(t, result)
}

func TestEvaluator_ExprStringContains(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"email": "user@example.com"}

	config := models.ConditionConfig{
		Mode:       "expression",
		Expression: `email contains "example"`,
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_ExprEmptyExpression(t *testing.T) {
	e := NewEvaluator()
	config := models.ConditionConfig{
		Mode:       "expression",
		Expression: "",
	}

	result, err := e.Evaluate(context.Background(), config, nil)
	require.NoError(t, err)
	assert.True(t, result, "empty expression should always match")
}

func TestEvaluator_ExprInvalidSyntax(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"x": 1}

	config := models.ConditionConfig{
		Mode:       "expression",
		Expression: "((( invalid",
	}

	_, err := e.Evaluate(context.Background(), config, env)
	assert.Error(t, err, "invalid expression should return error")
}

func TestEvaluator_ExprCache(t *testing.T) {
	e := NewEvaluator()
	expression := "value > 100"
	env := map[string]any{"value": 200.0}

	config := models.ConditionConfig{
		Mode:       "expression",
		Expression: expression,
	}

	// First evaluation - compiles and caches
	result1, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result1)

	// Second evaluation - should use cached program
	result2, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result2)

	// Verify program is in cache
	_, ok := e.exprCache.Load(expression)
	assert.True(t, ok, "expression should be cached after evaluation")

	// Invalidate and verify removal
	e.InvalidateCache(expression)
	_, ok = e.exprCache.Load(expression)
	assert.False(t, ok, "expression should be removed from cache after invalidation")
}

func TestEvaluator_ExprWithNestedFields(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{
		"deal": map[string]any{
			"value":      25000.0,
			"stage_name": "Won",
		},
	}

	config := models.ConditionConfig{
		Mode:       "expression",
		Expression: `deal.value > 10000 and deal.stage_name == "Won"`,
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_ExprMatchesOperator(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"email": "admin@kmu-hub.de"}

	config := models.ConditionConfig{
		Mode:       "expression",
		Expression: `email matches "^admin@"`,
	}

	result, err := e.Evaluate(context.Background(), config, env)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_ValidateExpression_Valid(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"value": 0.0}

	err := e.ValidateExpression("value > 100", env)
	assert.NoError(t, err)
}

func TestEvaluator_ValidateExpression_Invalid(t *testing.T) {
	e := NewEvaluator()
	env := map[string]any{"value": 0.0}

	err := e.ValidateExpression("((( broken", env)
	assert.Error(t, err)
}
