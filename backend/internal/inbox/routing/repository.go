package routing

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for routing rule persistence.
type Repository interface {
	// Create persists a new routing rule.
	Create(ctx context.Context, rule *models.RoutingRule) error

	// Update updates an existing routing rule.
	Update(ctx context.Context, rule *models.RoutingRule) error

	// Delete deletes a routing rule by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// GetByID retrieves a routing rule by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*models.RoutingRule, error)

	// ListActive returns all active routing rules ordered by priority ASC,
	// optionally filtered by channel (nil = all channels).
	ListActive(ctx context.Context, channel *string) ([]*models.RoutingRule, error)

	// ListAll returns all routing rules regardless of active status.
	ListAll(ctx context.Context) ([]*models.RoutingRule, error)
}
