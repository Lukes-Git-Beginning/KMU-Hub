package caldav

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// AppPasswordRepository defines the interface for app-specific password persistence.
type AppPasswordRepository interface {
	// Create persists a new app-specific password.
	Create(ctx context.Context, pw *models.AppSpecificPassword) error

	// ListByUser retrieves all passwords (active + revoked) for a user.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.AppSpecificPassword, error)

	// Revoke marks an app-specific password as revoked.
	Revoke(ctx context.Context, id, userID uuid.UUID) error

	// FindActiveByUser retrieves active (non-revoked) passwords for a user.
	FindActiveByUser(ctx context.Context, userID uuid.UUID) ([]*models.AppSpecificPassword, error)

	// UpdateLastUsed updates the last_used_at timestamp for a password.
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}
