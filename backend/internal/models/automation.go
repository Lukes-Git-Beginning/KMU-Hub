package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Automation represents a workflow automation rule that triggers actions
// based on events and conditions.
type Automation struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Scope           string          `json:"scope"`
	OwnerID         uuid.UUID       `json:"owner_id"`
	TriggerType     string          `json:"trigger_type"`
	TriggerConfig   json.RawMessage `json:"trigger_config"`
	Conditions      json.RawMessage `json:"conditions"`
	Actions         json.RawMessage `json:"actions"`
	IsActive        bool            `json:"is_active"`
	MaxSteps        int             `json:"max_steps"`
	TemplateID      *string         `json:"template_id,omitempty"`
	LastTriggeredAt *time.Time      `json:"last_triggered_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// AutomationExecution represents a single execution of an automation workflow.
type AutomationExecution struct {
	ID              uuid.UUID       `json:"id"`
	AutomationID    uuid.UUID       `json:"automation_id"`
	ChainID         uuid.UUID       `json:"chain_id"`
	TriggerEvent    json.RawMessage `json:"trigger_event"`
	ConditionResult bool            `json:"condition_result"`
	Status          string          `json:"status"`
	Steps           json.RawMessage `json:"steps"`
	ErrorMessage    *string         `json:"error_message,omitempty"`
	StartedAt       time.Time       `json:"started_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	DurationMs      *int            `json:"duration_ms,omitempty"`
}

// AutomationTemplate represents a pre-built automation template that
// users can instantiate to create their own automations.
type AutomationTemplate struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Category      string          `json:"category"`
	Complexity    string          `json:"complexity"`
	TriggerType   string          `json:"trigger_type"`
	TriggerConfig json.RawMessage `json:"trigger_config"`
	Conditions    json.RawMessage `json:"conditions"`
	Actions       json.RawMessage `json:"actions"`
	CreatedAt     time.Time       `json:"created_at"`
}

// ConditionConfig describes how conditions should be evaluated for
// an automation. Supports two modes:
//   - "simple": AND/OR tree using the existing models.Condition type
//   - "expression": expr-lang expression string for advanced conditions
type ConditionConfig struct {
	Mode       string     `json:"mode"`
	Simple     *Condition `json:"simple,omitempty"`
	Expression string     `json:"expression,omitempty"`
}

// ActionConfig describes a single action to execute within an automation.
type ActionConfig struct {
	Type    string          `json:"type"`
	Config  json.RawMessage `json:"config"`
	OnError string          `json:"on_error"`
}

// TriggerConfig describes the trigger configuration for an automation.
type TriggerConfig struct {
	EventType string          `json:"event_type"`
	Config    json.RawMessage `json:"config,omitempty"`
}

// ExecutionStep represents a single action step within an automation execution.
type ExecutionStep struct {
	ActionType string          `json:"action_type"`
	Input      json.RawMessage `json:"input,omitempty"`
	Output     json.RawMessage `json:"output,omitempty"`
	Error      *string         `json:"error,omitempty"`
	DurationMs int             `json:"duration_ms"`
}

// Automation scope constants.
const (
	AutomationScopePersonal     = "personal"
	AutomationScopeTeam         = "team"
	AutomationScopeOrganization = "organization"
)

// Automation execution status constants.
const (
	ExecutionStatusRunning   = "running"
	ExecutionStatusCompleted = "completed"
	ExecutionStatusFailed    = "failed"
	ExecutionStatusSkipped   = "skipped"
	ExecutionStatusAborted   = "aborted"
)

// Template complexity constants.
const (
	ComplexitySimple   = "einfach"
	ComplexityMedium   = "mittel"
	ComplexityAdvanced = "fortgeschritten"
)

// Condition evaluation mode constants.
const (
	ConditionModeSimple     = "simple"
	ConditionModeExpression = "expression"
)

// Action on-error behavior constants.
const (
	OnErrorContinue = "continue"
	OnErrorAbort    = "abort"
)
