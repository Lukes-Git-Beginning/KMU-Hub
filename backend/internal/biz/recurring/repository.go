package recurring

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository persists recurring invoice schedules and their generation ledger.
// Every method takes the tenant explicitly and every query is tenant-scoped —
// RLS is the second line of defence, not the first.
type Repository interface {
	Create(ctx context.Context, rec *models.RecurringInvoice) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.RecurringInvoice, error)
	List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.RecurringInvoice, int, error)
	Update(ctx context.Context, rec *models.RecurringInvoice) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error

	// ClaimPeriod reserves the given period for the schedule. The reservation is a
	// row in finance_recurring_runs guarded by a UNIQUE (tenant_id, recurring_id,
	// period_date) constraint, so two concurrent generate calls cannot both win:
	// the loser gets claimed=false plus the existing run, which carries the
	// invoice the winner emitted.
	ClaimPeriod(ctx context.Context, tenantID, recurringID uuid.UUID, period time.Time) (run *Run, claimed bool, err error)
	// AttachInvoice records the emitted invoice on a claimed run.
	AttachInvoice(ctx context.Context, tenantID, runID, invoiceID uuid.UUID) error
	// ReleasePeriod removes a claim whose invoice creation failed, so a retry can
	// claim the same period again instead of being locked out forever.
	ReleasePeriod(ctx context.Context, tenantID, runID uuid.UUID) error
}

// ListFilter narrows a schedule listing.
type ListFilter struct {
	Status string
	Limit  int
	Offset int
}

// Run is one entry of the generation ledger: the schedule emitted (or is about to
// emit) an invoice for exactly this period.
type Run struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	RecurringID uuid.UUID
	PeriodDate  time.Time
	InvoiceID   *uuid.UUID
	CreatedAt   time.Time
}
