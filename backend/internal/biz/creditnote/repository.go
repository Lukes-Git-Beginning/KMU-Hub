package creditnote

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for credit note persistence.
type Repository interface {
	Create(ctx context.Context, cn *models.CreditNote) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.CreditNote, error)
	Update(ctx context.Context, cn *models.CreditNote) error
	List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.CreditNote, int, error)
	GetByInvoiceID(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]*models.CreditNote, error)
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
}
