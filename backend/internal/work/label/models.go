package label

import "time"

// Label represents a color-coded tag that can be applied to tasks.
type Label struct {
	ID        string
	TenantID  string
	Name      string
	Color     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TaskLabelAssignment represents the association between a task and a label.
type TaskLabelAssignment struct {
	TaskID  string
	LabelID string
}
