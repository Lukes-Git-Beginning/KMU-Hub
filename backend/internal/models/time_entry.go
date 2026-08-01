package models

import (
	"time"

	"github.com/google/uuid"
)

// TimeEntry represents a tracked time period on a task
type TimeEntry struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	TaskID          uuid.UUID  `json:"task_id"`
	UserID          uuid.UUID  `json:"user_id"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	DurationSeconds *int       `json:"duration_seconds,omitempty"`
	Description     *string    `json:"description,omitempty"`
	IsManual        bool       `json:"is_manual"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TimeEntryWithUser includes user display name for list views
type TimeEntryWithUser struct {
	TimeEntry
	UserName string `json:"user_name"`
}

// TimeEntrySummary holds aggregated time info for a task
type TimeEntrySummary struct {
	TaskID              uuid.UUID `json:"task_id"`
	TotalDurationSeconds int      `json:"total_duration_seconds"`
	EntryCount          int       `json:"entry_count"`
}

// ActiveTimer represents a currently running timer for a user
type ActiveTimer struct {
	TimeEntry
	TaskTitle string `json:"task_title"`
}

// BillableTimeEntry is a completed time entry with its task/project/employee
// names resolved, for the finance "Stunden -> Rechnung" view across all
// projects (unlike TimeEntryWithUser, which is scoped to a single task).
type BillableTimeEntry struct {
	TimeEntry
	UserName    string `json:"user_name"`
	TaskTitle   string `json:"task_title"`
	ProjectName string `json:"project_name"`
}
