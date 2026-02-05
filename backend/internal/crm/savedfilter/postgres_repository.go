package savedfilter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, filter *models.SavedFilter) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO saved_filters (id, name, entity_type, filter_json, is_default, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		filter.ID, filter.Name, filter.EntityType, filter.FilterJSON,
		filter.IsDefault, filter.CreatedBy, filter.CreatedAt, filter.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SavedFilter, error) {
	var f models.SavedFilter
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, entity_type, filter_json, is_default, created_by, created_at, updated_at
		 FROM saved_filters WHERE id = $1`, id,
	).Scan(&f.ID, &f.Name, &f.EntityType, &f.FilterJSON, &f.IsDefault, &f.CreatedBy, &f.CreatedAt, &f.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFilterNotFound
	}
	return &f, err
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]*models.SavedFilter, error) {
	var conditions []string
	var args []any
	argNum := 1

	if filter.EntityType != nil {
		conditions = append(conditions, fmt.Sprintf("entity_type = $%d", argNum))
		args = append(args, *filter.EntityType)
		argNum++
	}

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argNum))
		args = append(args, *filter.UserID)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, name, entity_type, filter_json, is_default, created_by, created_at, updated_at
		FROM saved_filters %s
		ORDER BY is_default DESC, name ASC
	`, whereClause)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var filters []*models.SavedFilter
	for rows.Next() {
		var f models.SavedFilter
		if scanErr := rows.Scan(&f.ID, &f.Name, &f.EntityType, &f.FilterJSON, &f.IsDefault, &f.CreatedBy, &f.CreatedAt, &f.UpdatedAt); scanErr != nil {
			return nil, scanErr
		}
		filters = append(filters, &f)
	}

	return filters, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, filter *models.SavedFilter) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE saved_filters SET name = $1, filter_json = $2, is_default = $3, updated_at = $4
		 WHERE id = $5`,
		filter.Name, filter.FilterJSON, filter.IsDefault, filter.UpdatedAt, filter.ID,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM saved_filters WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) ClearDefault(ctx context.Context, userID uuid.UUID, entityType models.EntityType) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE saved_filters SET is_default = FALSE, updated_at = NOW()
		 WHERE created_by = $1 AND entity_type = $2 AND is_default = TRUE`,
		userID, entityType,
	)
	return err
}
