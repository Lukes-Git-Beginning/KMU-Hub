package config

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

func workflowRule(t *testing.T, name, event, conditions, actions string) *models.WorkflowRule {
	t.Helper()
	return &models.WorkflowRule{
		ID:           uuid.New(),
		Name:         name,
		TriggerEvent: event,
		Conditions:   json.RawMessage(conditions),
		Actions:      json.RawMessage(actions),
		Enabled:      true,
	}
}

// ---- EvaluateWorkflows ----

func TestEvaluateWorkflows_SkipsDisabledRules(t *testing.T) {
	e := NewWorkflowEngine()
	r := workflowRule(t, "r1", "deal.created", `[]`, `[{"type":"log","config":{}}]`)
	r.Enabled = false

	triggered := e.EvaluateWorkflows([]*models.WorkflowRule{r}, "deal.created", map[string]any{})
	assert.Empty(t, triggered)
}

func TestEvaluateWorkflows_SkipsNonMatchingTriggerEvent(t *testing.T) {
	e := NewWorkflowEngine()
	r := workflowRule(t, "r1", "deal.created", `[]`, `[{"type":"log","config":{}}]`)

	triggered := e.EvaluateWorkflows([]*models.WorkflowRule{r}, "deal.updated", map[string]any{})
	assert.Empty(t, triggered)
}

func TestEvaluateWorkflows_InvalidConditionsJSONNeverTriggers(t *testing.T) {
	e := NewWorkflowEngine()
	r := workflowRule(t, "r1", "deal.created", `{not json`, `[{"type":"log","config":{}}]`)

	triggered := e.EvaluateWorkflows([]*models.WorkflowRule{r}, "deal.created", map[string]any{})
	assert.Empty(t, triggered)
}

func TestEvaluateWorkflows_ConditionsMetBuildsTriggeredActions(t *testing.T) {
	e := NewWorkflowEngine()
	r := workflowRule(t, "escalate", "deal.created",
		`[{"field":"amount","operator":"gt","value":1000}]`,
		`[{"type":"send_notification","config":{"channel":"email"}},{"type":"create_task","config":{}}]`,
	)

	triggered := e.EvaluateWorkflows([]*models.WorkflowRule{r}, "deal.created", map[string]any{"amount": 5000.0})
	require.Len(t, triggered, 2)
	assert.Equal(t, "escalate", triggered[0].RuleName)
	assert.Equal(t, r.ID.String(), triggered[0].RuleID)
	assert.Equal(t, "send_notification", triggered[0].Action.Type)
	assert.Equal(t, "create_task", triggered[1].Action.Type)
}

func TestEvaluateWorkflows_ConditionsNotMetYieldsNoActions(t *testing.T) {
	e := NewWorkflowEngine()
	r := workflowRule(t, "escalate", "deal.created",
		`[{"field":"amount","operator":"gt","value":1000}]`,
		`[{"type":"send_notification","config":{}}]`,
	)

	triggered := e.EvaluateWorkflows([]*models.WorkflowRule{r}, "deal.created", map[string]any{"amount": 10.0})
	assert.Empty(t, triggered)
}

func TestEvaluateWorkflows_UnknownActionTypeDoesNotPanicAndPassesThrough(t *testing.T) {
	e := NewWorkflowEngine()
	r := workflowRule(t, "r1", "deal.created", `[]`,
		`[{"type":"some_future_action_type","config":{"foo":"bar"}}]`,
	)

	require.NotPanics(t, func() {
		triggered := e.EvaluateWorkflows([]*models.WorkflowRule{r}, "deal.created", map[string]any{})
		require.Len(t, triggered, 1)
		assert.Equal(t, "some_future_action_type", triggered[0].Action.Type)
		assert.Equal(t, "bar", triggered[0].Action.Config["foo"])
	})
}

// ---- evaluateConditions ----

func TestEvaluateConditions_AllMustBeTrue(t *testing.T) {
	e := NewWorkflowEngine()
	r := workflowRule(t, "r1", "evt", `[{"field":"a","operator":"eq","value":"1"},{"field":"b","operator":"eq","value":"2"}]`, `[]`)

	assert.True(t, e.evaluateConditions(r, map[string]any{"a": "1", "b": "2"}))
	assert.False(t, e.evaluateConditions(r, map[string]any{"a": "1", "b": "wrong"}))
}

func TestEvaluateConditions_EmptyConditionsAlwaysTrue(t *testing.T) {
	e := NewWorkflowEngine()
	r := workflowRule(t, "r1", "evt", `[]`, `[]`)
	assert.True(t, e.evaluateConditions(r, map[string]any{}))
}

// ---- evaluateCondition operators ----

func TestEvaluateCondition_Operators(t *testing.T) {
	e := NewWorkflowEngine()
	data := map[string]any{
		"status": "open",
		"amount": 50.0,
		"name":   "Acme Corp",
	}

	cases := []struct {
		name string
		cond Condition
		want bool
	}{
		{"exists true", Condition{Field: "status", Operator: "exists"}, true},
		{"exists false for missing field", Condition{Field: "missing", Operator: "exists"}, false},
		{"not_exists true for missing field", Condition{Field: "missing", Operator: "not_exists"}, true},
		{"not_exists false for present field", Condition{Field: "status", Operator: "not_exists"}, false},
		{"eq match", Condition{Field: "status", Operator: "eq", Value: "open"}, true},
		{"eq mismatch", Condition{Field: "status", Operator: "eq", Value: "closed"}, false},
		{"neq match", Condition{Field: "status", Operator: "neq", Value: "closed"}, true},
		{"neq mismatch", Condition{Field: "status", Operator: "neq", Value: "open"}, false},
		{"gt true", Condition{Field: "amount", Operator: "gt", Value: 10.0}, true},
		{"gt false", Condition{Field: "amount", Operator: "gt", Value: 100.0}, false},
		{"gt non-numeric operand", Condition{Field: "name", Operator: "gt", Value: 10.0}, false},
		{"lt true", Condition{Field: "amount", Operator: "lt", Value: 100.0}, true},
		{"lt false", Condition{Field: "amount", Operator: "lt", Value: 10.0}, false},
		{"gte equal", Condition{Field: "amount", Operator: "gte", Value: 50.0}, true},
		{"gte false", Condition{Field: "amount", Operator: "gte", Value: 51.0}, false},
		{"lte equal", Condition{Field: "amount", Operator: "lte", Value: 50.0}, true},
		{"lte false", Condition{Field: "amount", Operator: "lte", Value: 49.0}, false},
		{"contains true", Condition{Field: "name", Operator: "contains", Value: "Corp"}, true},
		{"contains false", Condition{Field: "name", Operator: "contains", Value: "GmbH"}, false},
		{"contains non-string value operand", Condition{Field: "name", Operator: "contains", Value: 5}, false},
		{"starts_with true", Condition{Field: "name", Operator: "starts_with", Value: "Acme"}, true},
		{"starts_with false", Condition{Field: "name", Operator: "starts_with", Value: "Corp"}, false},
		{"ends_with true", Condition{Field: "name", Operator: "ends_with", Value: "Corp"}, true},
		{"ends_with false", Condition{Field: "name", Operator: "ends_with", Value: "Acme"}, false},
		{"unknown operator", Condition{Field: "status", Operator: "bogus"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, e.evaluateCondition(tc.cond, data))
		})
	}
}

// ---- parseActions ----

func TestParseActions_InvalidJSONReturnsNil(t *testing.T) {
	e := NewWorkflowEngine()
	r := workflowRule(t, "r1", "evt", `[]`, `{not json`)
	actions := e.parseActions(r)
	assert.Nil(t, actions)
}

func TestParseActions_ValidJSON(t *testing.T) {
	e := NewWorkflowEngine()
	r := workflowRule(t, "r1", "evt", `[]`, `[{"type":"log","config":{"level":"info"}}]`)
	actions := e.parseActions(r)
	require.Len(t, actions, 1)
	assert.Equal(t, "log", actions[0].Type)
	assert.Equal(t, "info", actions[0].Config["level"])
}
