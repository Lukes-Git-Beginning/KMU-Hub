package invoice

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PaymentStats contains aggregated payment statistics for a date range.
// Used by GetPaymentStats on the invoice repository.
type PaymentStats struct {
	TotalInvoices          int
	TotalPaid              int
	TotalOutstanding       int
	TotalPaidAmount        decimal.Decimal
	TotalOutstandingAmount decimal.Decimal
	AverageDaysToPay       decimal.Decimal
}

// Repository defines the interface for invoice persistence.
type Repository interface {
	Create(ctx context.Context, invoice *models.Invoice) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
	List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.Invoice, int, error)
	Update(ctx context.Context, invoice *models.Invoice) error
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error
	GetOverdue(ctx context.Context, tenantID uuid.UUID) ([]*models.Invoice, error)
	GetByQuoteID(ctx context.Context, tenantID, quoteID uuid.UUID) (*models.Invoice, error)
	// LinkTimeTracking persists a time_tracking_source JSONB payload on the invoice.
	// Used after invoice creation to attach the audit trail from the HR time-tracking
	// module. The operation is best-effort: callers should treat errors as non-fatal.
	LinkTimeTracking(ctx context.Context, invoiceID uuid.UUID, src json.RawMessage) error
	// SetLock sets locked_at and locked_by on a single invoice without triggering
	// a full Update (ADR-0007 / Migration 000132 — replaces snapshot_data lock hack).
	SetLock(ctx context.Context, tenantID, id uuid.UUID, lockedAt time.Time, lockedBy uuid.UUID) error

	// GoBD-completion methods (Sprint 2 / Wave 1.B)

	// InvoiceNumberExists returns true if the given invoice number is already
	// assigned to an invoice belonging to the tenant.
	InvoiceNumberExists(ctx context.Context, tenantID uuid.UUID, invoiceNumber string) (bool, error)
	// CountByFiscalYear counts all non-draft invoices with an invoice_number
	// assigned in the given calendar year (using invoice_date year).
	CountByFiscalYear(ctx context.Context, tenantID uuid.UUID, year int) (int, error)
	// AggregatePaymentStats returns aggregated payment statistics for invoices
	// with invoice_date within [fromDate, toDate].
	AggregatePaymentStats(ctx context.Context, tenantID uuid.UUID, fromDate, toDate time.Time) (PaymentStats, error)
	// ListForGoBDExport returns all non-draft invoices with an invoice_number
	// assigned and invoice_date within [fromDate, toDate], ordered by invoice_number.
	// Used by GenerateGoBDExport to build the GoBD CSV.
	ListForGoBDExport(ctx context.Context, tenantID uuid.UUID, fromDate, toDate time.Time) ([]*models.Invoice, error)
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

// SequenceInfo carries current state of a number sequence row.
type SequenceInfo struct {
	CurrentNumber int
	FiscalYear    int
}

// NumberSequenceRepo provides gap-free invoice number generation.
// Uses SELECT FOR UPDATE on finance_number_sequences for serialization.
type NumberSequenceRepo interface {
	NextNumber(ctx context.Context, tenantID uuid.UUID, documentType string, fiscalYear int, prefix string) (string, error)
	// GetSequenceInfo returns the current sequence state for the given document type
	// and fiscal year, or nil if no sequence exists yet.
	GetSequenceInfo(ctx context.Context, tenantID uuid.UUID, documentType string, fiscalYear int) (*SequenceInfo, error)
}

// CompanySettingsRepo provides access to per-tenant company settings.
type CompanySettingsRepo interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*models.CompanySettings, error)
}

// QuoteReader provides read access to quotes for the CreateFromQuote operation.
type QuoteReader interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Quote, error)
}
