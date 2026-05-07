package signature

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for email signature persistence.
type Repository interface {
	Create(ctx context.Context, sig *models.EmailSignature) error
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error)
	GetDefault(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error)
	ListByUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) ([]*models.EmailSignature, error)
	Update(ctx context.Context, sig *models.EmailSignature) error
	Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error
	ClearDefaultForUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) error
	CountByUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (int, error)
}
