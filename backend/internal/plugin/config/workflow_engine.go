package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kmuhub/kmuhub/internal/models"
)

// WorkflowEngine evaluates config-based workflow rules against entity data
type WorkflowEngine struct{}

// NewWorkflowEngine creates a new workflow engine
func NewWorkflowEngine() *WorkflowEngine {
	return &WorkflowEngine{}
}

// Condition represents a single workflow condition
type Condition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"` // eq, neq, gt, lt, gte, lte, contains, starts_with, ends_with, exists, not_exists
	Value    any    `json:"value,omitempty"`
}

// Action represents a single workflow action
type Action struct {
	Type   string         `json:"type"` // create_task, send_notification, set_field, log
	Config map[string]any `json:"config"`
}

// TriggeredAction represents an action that should be executed
type TriggeredAction struct {
	RuleName string
	RuleID   string
	Action   Action
}

// EvaluateWorkflows evaluates workflow rules against entity data and returns triggered actions
func (e *WorkflowEngine) EvaluateWorkflows(rules []*models.WorkflowRule, eventType string, data map[string]any) []TriggeredAction {
	var triggered []TriggeredAction

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if rule.TriggerEvent != eventType {
			continue
		}

		if e.evaluateConditions(rule, data) {
			actions := e.parseActions(rule)
			for _, action := range actions {
				triggered = append(triggered, TriggeredAction{
					RuleName: rule.Name,
					RuleID:   rule.ID.String(),
					Action:   action,
				})
			}
		}
	}

	return triggered
}

func (e *WorkflowEngine) evaluateConditions(rule *models.WorkflowRule, data map[string]any) bool {
	var conditions []Condition
	if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
		slog.Warn("failed to parse workflow conditions",
			"rule_id", rule.ID,
			"error", err,
		)
		return false
	}

	// All conditions must be true (AND logic)
	for _, cond := range conditions {
		if !e.evaluateCondition(cond, data) {
			return false
		}
	}

	return true
}

func (e *WorkflowEngine) evaluateCondition(cond Condition, data map[string]any) bool {
	value, exists := data[cond.Field]

	switch cond.Operator {
	case "exists":
		return exists && value != nil
	case "not_exists":
		return !exists || value == nil
	case "eq":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", cond.Value)
	case "neq":
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", cond.Value)
	case "gt":
		a, aOK := toFloat64(value)
		b, bOK := toFloat64(cond.Value)
		return aOK && bOK && a > b
	case "lt":
		a, aOK := toFloat64(value)
		b, bOK := toFloat64(cond.Value)
		return aOK && bOK && a < b
	case "gte":
		a, aOK := toFloat64(value)
		b, bOK := toFloat64(cond.Value)
		return aOK && bOK && a >= b
	case "lte":
		a, aOK := toFloat64(value)
		b, bOK := toFloat64(cond.Value)
		return aOK && bOK && a <= b
	case "contains":
		str, ok := value.(string)
		sub, ok2 := cond.Value.(string)
		return ok && ok2 && strings.Contains(str, sub)
	case "starts_with":
		str, ok := value.(string)
		prefix, ok2 := cond.Value.(string)
		return ok && ok2 && strings.HasPrefix(str, prefix)
	case "ends_with":
		str, ok := value.(string)
		suffix, ok2 := cond.Value.(string)
		return ok && ok2 && strings.HasSuffix(str, suffix)
	default:
		return false
	}
}

func (e *WorkflowEngine) parseActions(rule *models.WorkflowRule) []Action {
	var actions []Action
	if err := json.Unmarshal(rule.Actions, &actions); err != nil {
		slog.Warn("failed to parse workflow actions",
			"rule_id", rule.ID,
			"error", err,
		)
		return nil
	}
	return actions
}
