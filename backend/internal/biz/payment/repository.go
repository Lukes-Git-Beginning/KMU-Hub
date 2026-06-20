package payment

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ErrDuplicatePayment is returned by Create when a payment with the same
// (tenant_id, idempotency_key) already exists — the DB-level deduplication
// guard (F5). The caller should fetch and return the existing payment.
var ErrDuplicatePayment = errors.New("duplicate payment: idempotency key already used")

// Repository defines the interface for payment persistence.
type Repository interface {
	Create(ctx context.Context, payment *models.Payment) error
	List(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]*models.Payment, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	SumByInvoiceID(ctx context.Context, tenantID, invoiceID uuid.UUID) (decimal.Decimal, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Payment, error)
	// GetByIdempotencyKey returns the payment recorded with the given key for the
	// tenant, or (nil, nil) if none exists. Used to resolve ErrDuplicatePayment.
	GetByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, key string) (*models.Payment, error)

	// In-transaction variants used by the service to couple a payment write with the
	// invoice status transition atomically (no status drift on partial failure).
	CreateInTx(ctx context.Context, tx pgx.Tx, payment *models.Payment) error
	DeleteInTx(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID) error
	SumByInvoiceIDInTx(ctx context.Context, tx pgx.Tx, tenantID, invoiceID uuid.UUID) (decimal.Decimal, error)
}

// InvoiceReader provides read access to invoices for validation and status checks.
type InvoiceReader interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
}

// InvoiceStatusUpdater provides the ability to update invoice status, both
// standalone and within a caller-owned transaction (for atomic payment writes).
type InvoiceStatusUpdater interface {
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error
	UpdateStatusInTx(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID, status string) error
}
