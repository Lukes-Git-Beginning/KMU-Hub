package label

import "context"

// Repository defines the persistence contract for labels.
type Repository interface {
	// Create inserts a new label for the given tenant.
	Create(ctx context.Context, tenantID, name, color string) (*Label, error)

	// GetByID returns a label by its ID, scoped to the tenant.
	GetByID(ctx context.Context, tenantID, id string) (*Label, error)

	// List returns all labels for the given tenant.
	List(ctx context.Context, tenantID string) ([]*Label, error)

	// Update modifies name and color of an existing label.
	Update(ctx context.Context, tenantID, id, name, color string) (*Label, error)

	// Delete removes a label by ID (also cascades to task_labels).
	Delete(ctx context.Context, tenantID, id string) error

	// GetLabelsByTaskIDs returns labels grouped by task ID for efficient batch loading.
	GetLabelsByTaskIDs(ctx context.Context, tenantID string, taskIDs []string) (map[string][]*Label, error)

	// SetTaskLabels replaces all labels for a task atomically (DELETE + INSERT).
	SetTaskLabels(ctx context.Context, tenantID, taskID string, labelIDs []string) error

	// GetTaskLabelIDs returns the IDs of all labels assigned to a task.
	GetTaskLabelIDs(ctx context.Context, tenantID, taskID string) ([]string, error)
}
