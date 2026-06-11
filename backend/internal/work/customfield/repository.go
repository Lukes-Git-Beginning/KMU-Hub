package customfield

import "context"

// Repository defines the persistence contract for custom field definitions.
type Repository interface {
	// Create inserts a new custom field definition for the given tenant.
	Create(ctx context.Context, tenantID, name, fieldType string, options []string, position int) (*Definition, error)

	// GetByID returns a definition by its ID, scoped to the tenant.
	GetByID(ctx context.Context, tenantID, id string) (*Definition, error)

	// List returns all definitions for the given tenant, ordered by position.
	List(ctx context.Context, tenantID string) ([]*Definition, error)

	// Update modifies an existing definition.
	Update(ctx context.Context, tenantID, id, name, fieldType string, options []string, position int) (*Definition, error)

	// Delete removes a definition by ID.
	Delete(ctx context.Context, tenantID, id string) error
}
