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
		`INSERT INTO saved_filters (id, tenant_id, name, entity_type, filter_json, is_default, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		filter.ID, filter.TenantID, filter.Name, filter.EntityType, filter.FilterJSON,
		filter.IsDefault, filter.CreatedBy, filter.CreatedAt, filter.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.SavedFilter, error) {
	var f models.SavedFilter
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, entity_type, filter_json, is_default, created_by, created_at, updated_at
		 FROM saved_filters WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	).Scan(&f.ID, &f.TenantID, &f.Name, &f.EntityType, &f.FilterJSON, &f.IsDefault, &f.CreatedBy, &f.CreatedAt, &f.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFilterNotFound
	}
	return &f, err
}

func (r *PostgresRepository) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.SavedFilter, error) {
	conditions := []string{"tenant_id = $1"}
	args := []any{tenantID}
	argNum := 2

	if filter.EntityType != nil {
		conditions = append(conditions, fmt.Sprintf("entity_type = $%d", argNum))
		args = append(args, *filter.EntityType)
		argNum++
	}

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argNum))
		args = append(args, *filter.UserID)
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, entity_type, filter_json, is_default, created_by, created_at, updated_at
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
		if scanErr := rows.Scan(&f.ID, &f.TenantID, &f.Name, &f.EntityType, &f.FilterJSON, &f.IsDefault, &f.CreatedBy, &f.CreatedAt, &f.UpdatedAt); scanErr != nil {
			return nil, scanErr
		}
		filters = append(filters, &f)
	}

	return filters, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, filter *models.SavedFilter) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE saved_filters SET name = $1, filter_json = $2, is_default = $3, updated_at = $4
		 WHERE id = $5 AND tenant_id = $6`,
		filter.Name, filter.FilterJSON, filter.IsDefault, filter.UpdatedAt, filter.ID, filter.TenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFilterNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM saved_filters WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFilterNotFound
	}
	return nil
}

func (r *PostgresRepository) ClearDefault(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, entityType models.EntityType) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE saved_filters SET is_default = FALSE, updated_at = NOW()
		 WHERE tenant_id = $1 AND created_by = $2 AND entity_type = $3 AND is_default = TRUE`,
		tenantID, userID, entityType,
	)
	return err
}
