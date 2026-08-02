package expense

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository persists expenses (Migration 000257). Every method takes the
// tenant explicitly and scopes on it -- RLS is the second line, not the first,
// so a missing tenant predicate shows up as a wrong result here rather than as
// a phantom 404 in production.
type Repository interface {
	Create(ctx context.Context, e *models.Expense) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Expense, error)
	Update(ctx context.Context, e *models.Expense) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.Expense, int, error)
}

// ListFilter narrows the expense list. A zero value lists the whole tenant.
type ListFilter struct {
	// Status restricts to one of pending/approved/rejected when non-empty.
	Status string
	// SubmittedBy restricts to one submitter. It backs the "own" data scope and
	// is set from the caller's identity, never from the request body.
	SubmittedBy *uuid.UUID
	Limit       int
	Offset      int
}
