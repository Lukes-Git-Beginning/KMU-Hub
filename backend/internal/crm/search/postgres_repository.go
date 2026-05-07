package search

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL full-text search
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) SearchContacts(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id::text,
			'contact' as entity_type,
			first_name || ' ' || last_name as title,
			COALESCE(email, '') as subtitle,
			ts_rank(search_vector, plainto_tsquery('german', $2)) as score
		FROM contacts
		WHERE tenant_id = $1
		  AND search_vector @@ plainto_tsquery('german', $2)
		ORDER BY score DESC
		LIMIT $3
	`, tenantID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSearchResults(rows)
}

func (r *PostgresRepository) SearchCompanies(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id::text,
			'company' as entity_type,
			name as title,
			COALESCE(industry, '') as subtitle,
			ts_rank(search_vector, plainto_tsquery('german', $2)) as score
		FROM companies
		WHERE tenant_id = $1
		  AND search_vector @@ plainto_tsquery('german', $2)
		ORDER BY score DESC
		LIMIT $3
	`, tenantID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSearchResults(rows)
}

func (r *PostgresRepository) SearchDeals(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id::text,
			'deal' as entity_type,
			name as title,
			COALESCE(value::text || ' ' || currency, '') as subtitle,
			ts_rank(search_vector, plainto_tsquery('german', $2)) as score
		FROM deals
		WHERE tenant_id = $1
		  AND search_vector @@ plainto_tsquery('german', $2)
		ORDER BY score DESC
		LIMIT $3
	`, tenantID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSearchResults(rows)
}

func (r *PostgresRepository) SearchActivities(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id::text,
			'activity' as entity_type,
			subject as title,
			activity_type::text as subtitle,
			ts_rank(search_vector, plainto_tsquery('german', $2)) as score
		FROM activities
		WHERE tenant_id = $1
		  AND search_vector @@ plainto_tsquery('german', $2)
		ORDER BY score DESC
		LIMIT $3
	`, tenantID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSearchResults(rows)
}

// scanSearchResults scans rows into SearchResult slice
func scanSearchResults(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.SearchResult, error) {
	var results []*models.SearchResult
	for rows.Next() {
		var r models.SearchResult
		if err := rows.Scan(&r.ID, &r.EntityType, &r.Title, &r.Subtitle, &r.Score); err != nil {
			return nil, err
		}
		results = append(results, &r)
	}
	return results, rows.Err()
}
