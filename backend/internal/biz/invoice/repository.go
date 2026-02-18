package invoice

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for invoice persistence.
type Repository interface {
	Create(ctx context.Context, invoice *models.Invoice) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
	List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.Invoice, int, error)
	Update(ctx context.Context, invoice *models.Invoice) error
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error
	GetOverdue(ctx context.Context, tenantID uuid.UUID) ([]*models.Invoice, error)
	GetByQuoteID(ctx context.Context, tenantID, quoteID uuid.UUID) (*models.Invoice, error)
}

// ListFilter contains filtering options for listing invoices.
type ListFilter struct {
	Status   string
	DateFrom *time.Time
	DateTo   *time.Time
	Overdue  bool // If true, only returns sent invoices past due_date
	Limit    int
	Offset   int
}

// NumberSequenceRepo provides gap-free invoice number generation.
// Uses SELECT FOR UPDATE on finance_number_sequences for serialization.
type NumberSequenceRepo interface {
	NextNumber(ctx context.Context, tenantID uuid.UUID, documentType string, fiscalYear int, prefix string) (string, error)
}

// CompanySettingsRepo provides access to per-tenant company settings.
type CompanySettingsRepo interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*models.CompanySettings, error)
}

// QuoteReader provides read access to quotes for the CreateFromQuote operation.
type QuoteReader interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Quote, error)
}
