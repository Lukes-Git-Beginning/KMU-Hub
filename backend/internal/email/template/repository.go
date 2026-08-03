package template

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for email template persistence.
//
// GetByID and ListVisible both apply the same visibility filter (shared, or
// owned by userID, or isAdmin) directly in SQL — a foreign personal template
// is therefore indistinguishable from a nonexistent one at every call site,
// including Update/Delete, which resolve the row through GetByID first.
type Repository interface {
	Create(ctx context.Context, tpl *models.EmailTemplate) error
	GetByID(ctx context.Context, id, tenantID, userID uuid.UUID, isAdmin bool) (*models.EmailTemplate, error)
	ListVisible(ctx context.Context, tenantID, userID uuid.UUID, isAdmin bool) ([]*models.EmailTemplate, error)
	Update(ctx context.Context, tpl *models.EmailTemplate) error
	Delete(ctx context.Context, id, tenantID uuid.UUID) error
}
