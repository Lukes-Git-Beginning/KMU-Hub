package action

import (
	"context"
	"encoding/json"
)

// ActionExecutor is the interface that all action implementations must satisfy.
type ActionExecutor interface {
	// Type returns the unique action type identifier, e.g., "crm.update_deal_field".
	Type() string

	// Execute runs the action with the given config and environment.
	// The env map contains outputs from previous actions (chained via prev_* keys).
	Execute(ctx context.Context, config json.RawMessage, env map[string]any) (ActionResult, error)
}

// ActionResult holds the output of a single action execution.
type ActionResult struct {
	Success bool           `json:"success"`
	Output  map[string]any `json:"output,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// ActionDefinition describes an action for the catalog API.
type ActionDefinition struct {
	Type         string
	Module       string
	Name         string // German
	Description  string // German
	Params       []ConfigParam
	OutputFields []OutputField
}

// OutputField describes a field produced by an action.
type OutputField struct {
	Key   string
	Label string
	Type  string // "string", "number", "boolean"
}

// ConfigParam describes a configuration parameter for an action.
type ConfigParam struct {
	Key      string
	Label    string
	Type     string // "string", "number", "boolean", "select", "template"
	Required bool
	Default  any
}
