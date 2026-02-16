package account

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for email account persistence.
type Repository interface {
	Create(ctx context.Context, account *models.EmailAccount) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.EmailAccount, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*models.EmailAccount, error)
	Update(ctx context.Context, account *models.EmailAccount) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListActive(ctx context.Context) ([]*models.EmailAccount, error)
}
