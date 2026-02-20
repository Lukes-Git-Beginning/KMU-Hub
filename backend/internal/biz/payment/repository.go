package payment

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for payment persistence.
type Repository interface {
	Create(ctx context.Context, payment *models.Payment) error
	List(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]*models.Payment, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	SumByInvoiceID(ctx context.Context, tenantID, invoiceID uuid.UUID) (decimal.Decimal, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Payment, error)
}

// InvoiceReader provides read access to invoices for validation and status checks.
type InvoiceReader interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
}

// InvoiceStatusUpdater provides the ability to update invoice status.
type InvoiceStatusUpdater interface {
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error
}
