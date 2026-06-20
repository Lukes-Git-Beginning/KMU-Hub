package creditnote

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for credit note persistence.
type Repository interface {
	Create(ctx context.Context, cn *models.CreditNote) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.CreditNote, error)
	Update(ctx context.Context, cn *models.CreditNote) error
	// UpdateInTx performs Update within a caller-owned transaction (used by Send to
	// couple number assignment + status/number update atomically — GoBD).
	UpdateInTx(ctx context.Context, tx pgx.Tx, cn *models.CreditNote) error
	List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.CreditNote, int, error)
	GetByInvoiceID(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]*models.CreditNote, error)
	// ListForDATEVExport returns sent credit notes with created_at within [fromDate,
	// toDate], keyset-paged by (created_at, id) for bounded-memory DATEV streaming.
	// Credit notes have no dedicated issue-date column, so created_at is used as the
	// period key. Pass afterDate/afterID = nil for the first page; for subsequent pages
	// pass the last returned row's created_at and id. Ordered by (created_at, id) ASC.
	ListForDATEVExport(ctx context.Context, tenantID uuid.UUID, fromDate, toDate time.Time, afterDate *time.Time, afterID *uuid.UUID, limit int) ([]*models.CreditNote, error)
}

// ListFilter contains filtering options for listing credit notes.
type ListFilter struct {
	Status         string
	OriginalInvID  *uuid.UUID
	Limit          int
	Offset         int
}

// InvoiceReader provides read access to invoices for validation.
type InvoiceReader interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
}

// NumberSequenceRepo provides gap-free document number generation.
type NumberSequenceRepo interface {
	NextNumber(ctx context.Context, tenantID uuid.UUID, documentType string, fiscalYear int, prefix string) (string, error)
	// NextNumberInTx assigns the next number within a caller-owned transaction so
	// numbering can be rolled back if the coupled document update fails (GoBD).
	NextNumberInTx(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, documentType string, fiscalYear int, prefix string) (string, error)
}
