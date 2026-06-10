package einvoice

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the persistence interface for incoming invoices.
type Repository interface {
	// Create inserts a new incoming invoice. Returns ErrDuplicateInvoice on
	// unique-constraint violation (tenant_id, supplier_name, invoice_number).
	Create(ctx context.Context, inv *models.IncomingInvoice) error

	// GetByID retrieves an invoice by ID scoped to the given tenant.
	// Returns ErrNotFound when the row does not exist or belongs to another tenant.
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.IncomingInvoice, error)

	// List retrieves incoming invoices for a tenant with optional filters and pagination.
	List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.IncomingInvoice, int, error)

	// UpdateStatus transitions the invoice status.
	// The transition is validated by the service layer before this is called.
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string, updatedAt time.Time) error
}

// ListFilter defines optional filtering and pagination for List.
type ListFilter struct {
	Status   string     // empty = all statuses
	DateFrom *time.Time // inclusive filter on invoice_date
	DateTo   *time.Time // inclusive filter on invoice_date
	Limit    int        // default 50
	Offset   int
}
