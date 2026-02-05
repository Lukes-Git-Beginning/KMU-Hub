package search

import (
	"context"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Repository defines the interface for search persistence
type Repository interface {
	// SearchContacts searches contacts using full-text search
	SearchContacts(ctx context.Context, query string, limit int) ([]*models.SearchResult, error)

	// SearchCompanies searches companies using full-text search
	SearchCompanies(ctx context.Context, query string, limit int) ([]*models.SearchResult, error)

	// SearchDeals searches deals using full-text search
	SearchDeals(ctx context.Context, query string, limit int) ([]*models.SearchResult, error)

	// SearchActivities searches activities using full-text search
	SearchActivities(ctx context.Context, query string, limit int) ([]*models.SearchResult, error)
}
