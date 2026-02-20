package trigger

import "github.com/kmuhub/kmuhub/internal/models"

// TriggerDefinition describes a trigger that can fire an automation.
type TriggerDefinition struct {
	// Type is the unique identifier, e.g., "crm.deal.stage_changed".
	Type string
	// Module is the source module, e.g., "crm", "work", "email".
	Module string
	// Name is a human-readable name (German).
	Name string
	// Description is a human-readable description (German).
	Description string
	// EventType maps to the event.Event* constant that fires this trigger.
	EventType string
	// Fields are the available condition fields for this trigger.
	Fields []TriggerField
	// ConfigParams are optional configuration parameters.
	ConfigParams []ConfigParam
	// TimeBased indicates whether this trigger is polled rather than event-driven.
	TimeBased bool
}

// TriggerField describes a field available for conditions on this trigger.
type TriggerField struct {
	Key       string
	Label     string
	Type      string // "string", "number", "boolean", "date"
	Operators []string
}

// ConfigParam describes an optional parameter for trigger or action configuration.
type ConfigParam struct {
	Key      string
	Label    string
	Type     string // "string", "number", "boolean", "select"
	Required bool
	Default  any
}

// TriggerMatch represents a matched automation for a given event.
type TriggerMatch struct {
	Automation *models.Automation
	TriggerDef *TriggerDefinition
}
