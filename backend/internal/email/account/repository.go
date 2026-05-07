package account

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for email account persistence.
type Repository interface {
	Create(ctx context.Context, account *models.EmailAccount) error
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailAccount, error)
	GetByUserIDAndTenant(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (*models.EmailAccount, error)
	Update(ctx context.Context, account *models.EmailAccount) error
	Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error
	// ListActive returns sync-enabled accounts for a specific tenant.
	ListActive(ctx context.Context, tenantID uuid.UUID) ([]*models.EmailAccount, error)
	// ListAllActive returns all sync-enabled accounts across all tenants (for sync engine bootstrap).
	ListAllActive(ctx context.Context) ([]*models.EmailAccount, error)
}
