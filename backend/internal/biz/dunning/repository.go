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

// InvoiceReader provides read access to invoices for dunning detection.
type InvoiceReader interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
	GetOverdue(ctx context.Context, tenantID uuid.UUID) ([]*models.Invoice, error)
}
