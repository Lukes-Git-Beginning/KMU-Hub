package dunning

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for dunning record persistence.
type Repository interface {
	Create(ctx context.Context, record *models.DunningRecord) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.DunningRecord, error)
	List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.DunningRecord, int, error)
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string, sentAt *time.Time) error
	GetByInvoiceID(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]*models.DunningRecord, error)
	// GetByInvoiceIDs batch-loads dunning records for multiple invoices in one
	// query, grouped by invoice ID, to avoid an N+1 in dunning detection.
	GetByInvoiceIDs(ctx context.Context, tenantID uuid.UUID, invoiceIDs []uuid.UUID) (map[uuid.UUID][]*models.DunningRecord, error)
	GetHighestLevelByInvoiceID(ctx context.Context, tenantID, invoiceID uuid.UUID) (*models.DunningRecord, error)
}

// ConfigRepository defines the interface for dunning configuration persistence.
type ConfigRepository interface {
	Get(ctx context.Context, tenantID uuid.UUID) (*models.DunningConfig, error)
	Upsert(ctx context.Context, config *models.DunningConfig) error
}

// ListFilter contains filtering options for listing dunning records.
type ListFilter struct {
	InvoiceID *uuid.UUID
	Status    string
	Level     int // 0 = all levels
	Limit     int
	Offset    int
}

// InvoiceReader provides read access to invoices for dunning detection and for
// the open-items (Offene Posten) list, which is the read side of the same
// receivables domain: the same invoices, seen as outstanding amounts rather than
// as documents.
type InvoiceReader interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
	GetOverdue(ctx context.Context, tenantID uuid.UUID) ([]*models.Invoice, error)
	// ListOpenItems returns one page of unpaid receivables plus the total row
	// count, most overdue first. An invoice is an open item when it is sent or
	// overdue and its gross total is not yet fully covered by recorded payments.
	ListOpenItems(ctx context.Context, tenantID uuid.UUID, filter models.OpenItemFilter) ([]*models.OpenItem, int, error)
	// SummarizeOpenItems aggregates every open item of the tenant — not just the
	// requested page — grouped by currency and aging bucket index. The service
	// folds those groups into the per-currency totals.
	SummarizeOpenItems(ctx context.Context, tenantID uuid.UUID, asOf time.Time) ([]*models.OpenItemBucketTotal, error)
}

// The open-items filter lives in the models package (models.OpenItemFilter):
// the invoice repository implements this interface structurally, and a filter
// type declared here would force that package to import dunning.
