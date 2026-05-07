package search

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/models"
)

const (
	defaultLimit = 20
	maxLimit     = 100
	minQueryLen  = 2
	maxQueryLen  = 200
)

// Service handles search business logic
type Service struct {
	repo Repository
}

// NewService creates a new search service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SearchInput contains search parameters
type SearchInput struct {
	Query       string
	EntityTypes []models.SearchEntityType // Empty = search all
	Limit       int
	TenantID    uuid.UUID
}

// Search performs a full-text search across CRM entities, scoped to a tenant
func (s *Service) Search(ctx context.Context, input SearchInput) ([]*models.SearchResult, error) {
	// Validate query
	query := strings.TrimSpace(input.Query)
	if len(query) < minQueryLen {
		return nil, ErrQueryTooShort
	}
	if len(query) > maxQueryLen {
		return nil, ErrQueryTooLong
	}

	// Validate and normalize limit
	limit := input.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	// Validate entity types
	entityTypes := input.EntityTypes
	if len(entityTypes) == 0 {
		// Search all entities
		entityTypes = []models.SearchEntityType{
			models.SearchEntityContact,
			models.SearchEntityCompany,
			models.SearchEntityDeal,
			models.SearchEntityActivity,
		}
	} else {
		for _, et := range entityTypes {
			if !et.IsValid() {
				return nil, ErrInvalidEntity
			}
		}
	}

	// Calculate limit per entity type to ensure fair distribution
	// Each entity type gets equal share, but we'll still sort by score at the end
	limitPerType := limit
	if len(entityTypes) > 1 {
		limitPerType = (limit / len(entityTypes)) + 1
		if limitPerType < 5 {
			limitPerType = 5
		}
	}

	// Collect results from all requested entity types
	var allResults []*models.SearchResult

	for _, et := range entityTypes {
		var results []*models.SearchResult
		var err error

		switch et {
		case models.SearchEntityContact:
			results, err = s.repo.SearchContacts(ctx, input.TenantID, query, limitPerType)
		case models.SearchEntityCompany:
			results, err = s.repo.SearchCompanies(ctx, input.TenantID, query, limitPerType)
		case models.SearchEntityDeal:
			results, err = s.repo.SearchDeals(ctx, input.TenantID, query, limitPerType)
		case models.SearchEntityActivity:
			results, err = s.repo.SearchActivities(ctx, input.TenantID, query, limitPerType)
		}

		if err != nil {
			slog.Warn("search failed for entity type",
				"entity_type", et,
				"error", err,
			)
			// Continue with other entity types even if one fails
			continue
		}

		allResults = append(allResults, results...)
	}

	// Sort all results by score descending
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	// Apply final limit
	if len(allResults) > limit {
		allResults = allResults[:limit]
	}

	slog.Info("search completed",
		"query", query,
		"entity_types", entityTypes,
		"result_count", len(allResults),
	)

	return allResults, nil
}
