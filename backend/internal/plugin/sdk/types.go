package sdk

import "encoding/json"

// PluginMeta describes a plugin for the manifest
type PluginMeta struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Permissions []string `json:"permissions"`
}

// HookRequest is the data passed to a plugin hook handler
type HookRequest struct {
	HookType   string          `json:"hook_type"`
	Module     string          `json:"module"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	EntityData json.RawMessage `json:"entity_data"`
}

// HookResponse is the data returned from a plugin hook handler
type HookResponse struct {
	// Modified indicates whether the plugin modified the entity data
	Modified bool `json:"modified"`
	// ModifiedData contains the modified entity data (only if Modified is true)
	ModifiedData json.RawMessage `json:"modified_data,omitempty"`
	// Error message (empty string means success)
	Error string `json:"error,omitempty"`
}

// HookHandler is the function signature for plugin hook handlers
type HookHandler func(req *HookRequest) *HookResponse

// Contact represents a CRM contact (subset of fields available to plugins)
type Contact struct {
	ID           string            `json:"id"`
	FirstName    string            `json:"first_name"`
	LastName     string            `json:"last_name"`
	Email        string            `json:"email,omitempty"`
	Phone        string            `json:"phone,omitempty"`
	CompanyID    string            `json:"company_id,omitempty"`
	CustomFields map[string]string `json:"custom_fields,omitempty"`
}

// Task represents a work task (subset of fields available to plugins)
type Task struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	Priority    string `json:"priority,omitempty"`
	AssigneeID  string `json:"assignee_id,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
}

// CreateTaskInput is the input for creating a task via the host API
type CreateTaskInput struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Priority    string `json:"priority,omitempty"`
	AssigneeID  string `json:"assignee_id,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
}
