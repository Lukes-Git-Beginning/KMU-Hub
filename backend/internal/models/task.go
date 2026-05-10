package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Task priority constants
const (
	TaskPriorityUrgent = "urgent"
	TaskPriorityHigh   = "high"
	TaskPriorityMedium = "medium"
	TaskPriorityLow    = "low"
)

// ValidTaskPriorities contains all valid task priority levels
var ValidTaskPriorities = map[string]bool{
	TaskPriorityUrgent: true,
	TaskPriorityHigh:   true,
	TaskPriorityMedium: true,
	TaskPriorityLow:    true,
}

// Dependency type constants
const (
	DepTypeBlocks     = "blocks"
	DepTypeBlockedBy  = "blocked_by"
	DepTypeRelatesTo  = "relates_to"
	DepTypeDuplicates = "duplicates"
)

// ValidDependencyTypes contains all valid task dependency types
var ValidDependencyTypes = map[string]bool{
	DepTypeBlocks:     true,
	DepTypeBlockedBy:  true,
	DepTypeRelatesTo:  true,
	DepTypeDuplicates: true,
}

// Entity link type constants
const (
	LinkTypeContact = "contact"
	LinkTypeCompany = "company"
	LinkTypeDeal    = "deal"
	LinkTypeChannel = "channel"
	LinkTypeMessage = "message"
)

// ValidLinkTypes contains all valid entity link types
var ValidLinkTypes = map[string]bool{
	LinkTypeContact: true,
	LinkTypeCompany: true,
	LinkTypeDeal:    true,
	LinkTypeChannel: true,
	LinkTypeMessage: true,
}

// Task activity action constants
const (
	TaskActionCreated            = "created"
	TaskActionStatusChanged      = "status_changed"
	TaskActionAssigned           = "assigned"
	TaskActionPriorityChanged    = "priority_changed"
	TaskActionDueDateChanged     = "due_date_changed"
	TaskActionTitleChanged       = "title_changed"
	TaskActionDescriptionChanged = "description_changed"
	TaskActionCommented          = "commented"
	TaskActionAttachmentAdded    = "attachment_added"
	TaskActionLinked             = "linked"
	TaskActionUnlinked           = "unlinked"
	TaskActionTimerStarted       = "timer_started"
	TaskActionTimerStopped       = "timer_stopped"
	TaskActionTimeEntryAdded     = "time_entry_added"
)

// Task represents a work item within a project
type Task struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	ProjectID    *uuid.UUID `json:"project_id,omitempty"`
	TaskNumber   int        `json:"task_number"`
	Title        string     `json:"title"`
	Description  *string    `json:"description,omitempty"`
	StatusID     *uuid.UUID `json:"status_id,omitempty"`
	Priority     string     `json:"priority"`
	AssigneeID   *uuid.UUID `json:"assignee_id,omitempty"`
	ParentTaskID *uuid.UUID `json:"parent_task_id,omitempty"`
	Depth        int        `json:"depth"`
	SortOrder    float64    `json:"sort_order"`
	DueDate      *time.Time `json:"due_date,omitempty"`
	CreatedBy    uuid.UUID  `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// TaskWithRelations includes denormalized data for API responses
type TaskWithRelations struct {
	Task
	StatusName     *string        `json:"status_name,omitempty"`
	StatusColor    *string        `json:"status_color,omitempty"`
	AssigneeName   *string        `json:"assignee_name,omitempty"`
	ProjectKey     *string        `json:"project_key,omitempty"`
	SubtaskCount   int            `json:"subtask_count"`
	HasBlockedDeps bool           `json:"has_blocked_deps"`
	CustomFields   map[string]any `json:"custom_fields,omitempty"`
}

// TaskDependency represents a relationship between two tasks
type TaskDependency struct {
	ID             uuid.UUID `json:"id"`
	SourceTaskID   uuid.UUID `json:"source_task_id"`
	TargetTaskID   uuid.UUID `json:"target_task_id"`
	DependencyType string    `json:"dependency_type"` // "blocks", "blocked_by", "relates_to", "duplicates"
	CreatedBy      uuid.UUID `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
}

// TaskComment represents a comment on a task
type TaskComment struct {
	ID               uuid.UUID  `json:"id"`
	TaskID           uuid.UUID  `json:"task_id"`
	AuthorID         uuid.UUID  `json:"author_id"`
	AuthorName       string     `json:"author_name"` // denormalized from users table
	Content          string     `json:"content"`
	QuotedCommentID  *uuid.UUID `json:"quoted_comment_id,omitempty"`
	QuotedPreview    *string    `json:"quoted_content_preview,omitempty"` // first N chars of quoted comment
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// TaskEntityLink connects a task to a CRM entity
type TaskEntityLink struct {
	ID                uuid.UUID `json:"id"`
	TenantID          uuid.UUID `json:"tenant_id"`
	TaskID            uuid.UUID `json:"task_id"`
	EntityType        string    `json:"entity_type"` // "contact", "company", "deal", "channel", "message"
	EntityID          uuid.UUID `json:"entity_id"`
	EntityDisplayName string    `json:"entity_display_name"` // denormalized for display
	CreatedBy         uuid.UUID `json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
}

// TaskActivity records a change event on a task for the activity log
type TaskActivity struct {
	ID        uuid.UUID        `json:"id"`
	TaskID    uuid.UUID        `json:"task_id"`
	ActorID   uuid.UUID        `json:"actor_id"`
	ActorName string           `json:"actor_name"` // denormalized from users table
	Action    string           `json:"action"`
	FieldName *string          `json:"field_name,omitempty"`
	OldValue  *json.RawMessage `json:"old_value,omitempty"`
	NewValue  *json.RawMessage `json:"new_value,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}

// TaskFile represents a file attached to a task
type TaskFile struct {
	ID             uuid.UUID `json:"id"`
	TaskID         uuid.UUID `json:"task_id"`
	Filename       string    `json:"filename"`
	MimeType       string    `json:"mime_type"`
	FileSize       int64     `json:"file_size"`
	StorageKey     string    `json:"storage_key"`
	ThumbnailKey   *string   `json:"thumbnail_key,omitempty"`
	UploadedBy     uuid.UUID `json:"uploaded_by"`
	UploadedByName string    `json:"uploaded_by_name"` // denormalized from users table
	CreatedAt      time.Time `json:"created_at"`
}
