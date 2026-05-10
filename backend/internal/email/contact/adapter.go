package contact

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// TenantedContactService is the subset of the CRM contact service that this
// package needs. All methods are tenant-scoped.
type TenantedContactService interface {
	GetByEmail(ctx context.Context, email string, tenantID uuid.UUID) (*models.Contact, error)
	CreateForImport(ctx context.Context, c *models.Contact) error
	UpdateForImport(ctx context.Context, c *models.Contact) error
	ListByIDs(ctx context.Context, ids []uuid.UUID, tenantID uuid.UUID) ([]*models.Contact, error)
	ListVisible(ctx context.Context, userID uuid.UUID, isAdmin bool, tenantID uuid.UUID) ([]*models.Contact, error)
}

// TenantScopedAdapter wraps a TenantedContactService and binds it to a fixed
// tenantID, exposing the old ContactProvider interface. Use this when
// constructing ImportService / ExportService inside a request handler that
// already knows the tenant.
type TenantScopedAdapter struct {
	svc      TenantedContactService
	tenantID uuid.UUID
}

// NewTenantScopedAdapter creates an adapter that satisfies ContactProvider.
func NewTenantScopedAdapter(svc TenantedContactService, tenantID uuid.UUID) *TenantScopedAdapter {
	return &TenantScopedAdapter{svc: svc, tenantID: tenantID}
}

func (a *TenantScopedAdapter) GetByEmail(ctx context.Context, email string) (*models.Contact, error) {
	return a.svc.GetByEmail(ctx, email, a.tenantID)
}

func (a *TenantScopedAdapter) CreateForImport(ctx context.Context, c *models.Contact) error {
	if c.TenantID == uuid.Nil {
		c.TenantID = a.tenantID
	}
	return a.svc.CreateForImport(ctx, c)
}

func (a *TenantScopedAdapter) UpdateForImport(ctx context.Context, c *models.Contact) error {
	if c.TenantID == uuid.Nil {
		c.TenantID = a.tenantID
	}
	return a.svc.UpdateForImport(ctx, c)
}

func (a *TenantScopedAdapter) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.Contact, error) {
	return a.svc.ListByIDs(ctx, ids, a.tenantID)
}

func (a *TenantScopedAdapter) ListVisible(ctx context.Context, userID uuid.UUID, isAdmin bool) ([]*models.Contact, error) {
	return a.svc.ListVisible(ctx, userID, isAdmin, a.tenantID)
}
