package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PluginType defines the type of plugin
type PluginType string

const (
	PluginTypeConfig PluginType = "config"
	PluginTypeWASM   PluginType = "wasm"
)

// InstallationStatus defines the lifecycle state of a plugin installation
type InstallationStatus string

const (
	InstallationStatusPendingApproval InstallationStatus = "pending_approval"
	InstallationStatusActive          InstallationStatus = "active"
	InstallationStatusDisabled        InstallationStatus = "disabled"
	InstallationStatusError           InstallationStatus = "error"
	InstallationStatusUninstalled     InstallationStatus = "uninstalled"
)

// ExecutionStatus defines the result of a hook execution
type ExecutionStatus string

const (
	ExecutionStatusSuccess ExecutionStatus = "success"
	ExecutionStatusError   ExecutionStatus = "error"
	ExecutionStatusTimeout ExecutionStatus = "timeout"
)

// ValidationRuleType defines supported validation rule types
type ValidationRuleType string

const (
	ValidationRuleTypeRegex      ValidationRuleType = "regex"
	ValidationRuleTypeRange      ValidationRuleType = "range"
	ValidationRuleTypeRequiredIf ValidationRuleType = "required_if"
	ValidationRuleTypeFormat     ValidationRuleType = "format"
	ValidationRuleTypeCustom     ValidationRuleType = "custom"
	ValidationRuleTypeEnum       ValidationRuleType = "enum"
)

// HookRegistration defines a hook a plugin wants to receive
type HookRegistration struct {
	HookType   string `json:"hook_type"`
	Module     string `json:"module"`
	EntityType string `json:"entity_type"`
	Priority   int    `json:"priority"`
}

// PluginManifest represents a plugin definition
type PluginManifest struct {
	ID                uuid.UUID          `json:"id"`
	Slug              string             `json:"slug"`
	Name              string             `json:"name"`
	Description       string             `json:"description"`
	Version           string             `json:"version"`
	Author            string             `json:"author"`
	SettingsSchema    json.RawMessage    `json:"settings_schema"`
	Permissions       []string           `json:"permissions"`
	HookRegistrations []HookRegistration `json:"hook_registrations"`
	WASMBinary        []byte             `json:"-"`
	WASMBinaryHash    string             `json:"wasm_binary_hash,omitempty"`
	PluginType        PluginType         `json:"plugin_type"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

// PluginInstallation represents a per-tenant plugin installation
type PluginInstallation struct {
	ID           uuid.UUID          `json:"id"`
	TenantID     uuid.UUID          `json:"tenant_id"`
	ManifestID   uuid.UUID          `json:"manifest_id"`
	Status       InstallationStatus `json:"status"`
	Settings     json.RawMessage    `json:"settings"`
	ErrorMessage *string            `json:"error_message,omitempty"`
	InstalledBy  uuid.UUID          `json:"installed_by"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// PluginInstallationWithManifest includes manifest data for API responses
type PluginInstallationWithManifest struct {
	PluginInstallation
	ManifestSlug    string          `json:"manifest_slug"`
	ManifestName    string          `json:"manifest_name"`
	ManifestVersion string          `json:"manifest_version"`
	PluginType      PluginType      `json:"plugin_type"`
	Permissions     []string        `json:"required_permissions"`
	Granted         []string        `json:"granted_permissions"`
	SettingsSchema  json.RawMessage `json:"settings_schema"`
}

// PluginPermission represents a granted permission for an installation
type PluginPermission struct {
	ID             uuid.UUID `json:"id"`
	InstallationID uuid.UUID `json:"installation_id"`
	Permission     string    `json:"permission"`
	GrantedBy      uuid.UUID `json:"granted_by"`
	GrantedAt      time.Time `json:"granted_at"`
}

// PluginKVEntry represents a key-value entry in the plugin store
type PluginKVEntry struct {
	ID             uuid.UUID       `json:"id"`
	InstallationID uuid.UUID       `json:"installation_id"`
	Key            string          `json:"key"`
	Value          json.RawMessage `json:"value"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// PluginExecutionLog represents an audit log entry for hook execution
type PluginExecutionLog struct {
	ID             uuid.UUID       `json:"id"`
	InstallationID uuid.UUID       `json:"installation_id"`
	HookType       string          `json:"hook_type"`
	Module         string          `json:"module"`
	EntityType     string          `json:"entity_type"`
	EntityID       *uuid.UUID      `json:"entity_id,omitempty"`
	DurationMs     int             `json:"duration_ms"`
	Status         ExecutionStatus `json:"status"`
	ErrorMessage   *string         `json:"error_message,omitempty"`
	RequestData    json.RawMessage `json:"request_data,omitempty"`
	ResponseData   json.RawMessage `json:"response_data,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// ValidationRule represents a config-based field validation rule
type ValidationRule struct {
	ID             uuid.UUID          `json:"id"`
	TenantID       uuid.UUID          `json:"tenant_id"`
	InstallationID *uuid.UUID         `json:"installation_id,omitempty"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	EntityType     string             `json:"entity_type"`
	FieldName      string             `json:"field_name"`
	RuleType       ValidationRuleType `json:"rule_type"`
	RuleConfig     json.RawMessage    `json:"rule_config"`
	ErrorMessage   string             `json:"error_message"`
	Priority       int                `json:"priority"`
	Enabled        bool               `json:"enabled"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

// WorkflowRule represents a config-based automation rule
type WorkflowRule struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	InstallationID *uuid.UUID      `json:"installation_id,omitempty"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	TriggerEvent   string          `json:"trigger_event"`
	Conditions     json.RawMessage `json:"conditions"`
	Actions        json.RawMessage `json:"actions"`
	Priority       int             `json:"priority"`
	Enabled        bool            `json:"enabled"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// IndustryTemplate represents a preconfigured industry configuration
type IndustryTemplate struct {
	ID              uuid.UUID       `json:"id"`
	Slug            string          `json:"slug"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Industry        string          `json:"industry"`
	Icon            string          `json:"icon"`
	CustomFields    json.RawMessage `json:"custom_fields"`
	ValidationRules json.RawMessage `json:"validation_rules"`
	WorkflowRules   json.RawMessage `json:"workflow_rules"`
	DefaultSettings json.RawMessage `json:"default_settings"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}
