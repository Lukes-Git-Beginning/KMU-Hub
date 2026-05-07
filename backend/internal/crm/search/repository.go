package search

import (
	"context"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for search persistence
type Repository interface {
	// SearchContacts searches contacts using full-text search, scoped to tenant
	SearchContacts(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error)

	// SearchCompanies searches companies using full-text search, scoped to tenant
	SearchCompanies(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error)

	// SearchDeals searches deals using full-text search, scoped to tenant
	SearchDeals(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error)

	// SearchActivities searches activities using full-text search, scoped to tenant
	SearchActivities(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error)
}
