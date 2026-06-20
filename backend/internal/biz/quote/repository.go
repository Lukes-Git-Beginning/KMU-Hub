package quote

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for quote persistence.
type Repository interface {
	Create(ctx context.Context, quote *models.Quote) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Quote, error)
	List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.Quote, int, error)
	Update(ctx context.Context, quote *models.Quote) error
	// UpdateInTx performs the update within the caller's transaction, used by Send to
	// assign the quote number and persist the sent state atomically (GoBD: no gap).
	UpdateInTx(ctx context.Context, tx pgx.Tx, quote *models.Quote) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error
	GetByDealID(ctx context.Context, tenantID, dealID uuid.UUID) ([]*models.Quote, error)
}

// ListFilter contains filtering options for listing quotes.
type ListFilter struct {
	Status   string
	DealID   *uuid.UUID
	DateFrom *time.Time
	DateTo   *time.Time
	Limit    int
	Offset   int
}

// NumberSequenceRepo provides gap-free document number generation.
type NumberSequenceRepo interface {
	NextNumber(ctx context.Context, tenantID uuid.UUID, documentType string, fiscalYear int, prefix string) (string, error)
	// NextNumberInTx assigns the next number within the caller's transaction so the
	// number and the document status update commit (or roll back) atomically.
	NextNumberInTx(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, documentType string, fiscalYear int, prefix string) (string, error)
}

// CompanySettingsRepo provides access to per-tenant company settings.
type CompanySettingsRepo interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*models.CompanySettings, error)
}

// DealValueUpdater updates the CRM deal value. Implemented by a thin wrapper
// around the CRM gRPC client. Nil-safe: if nil, sync is skipped (standalone mode).
type DealValueUpdater interface {
	UpdateDealValue(ctx context.Context, dealID uuid.UUID, grossTotal decimal.Decimal) error
}
