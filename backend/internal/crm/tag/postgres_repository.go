package tag

import (
	"context"
	"errors"

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

func (r *PostgresRepository) Create(ctx context.Context, tag *models.Tag) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tags (id, tenant_id, name, color, entity_type, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		tag.ID, tag.TenantID, tag.Name, tag.Color, tag.EntityType, tag.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Tag, error) {
	return r.scanTag(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, color, entity_type, created_at
		 FROM tags WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	))
}

func (r *PostgresRepository) GetByEntityAndName(ctx context.Context, tenantID uuid.UUID, entityType models.EntityType, name string) (*models.Tag, error) {
	return r.scanTag(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, color, entity_type, created_at
		 FROM tags WHERE tenant_id = $1 AND entity_type = $2 AND LOWER(name) = LOWER($3)`,
		tenantID, entityType, name,
	))
}

func (r *PostgresRepository) List(ctx context.Context, tenantID uuid.UUID, entityType *models.EntityType, offset, limit int) ([]*models.Tag, int, error) {
	var total int
	var countErr error

	if entityType != nil {
		countErr = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM tags WHERE tenant_id = $1 AND entity_type = $2`, tenantID, *entityType,
		).Scan(&total)
	} else {
		countErr = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM tags WHERE tenant_id = $1`, tenantID,
		).Scan(&total)
	}
	if countErr != nil {
		return nil, 0, countErr
	}

	var rows pgx.Rows
	var err error

	if entityType != nil {
		rows, err = r.pool.Query(ctx,
			`SELECT id, tenant_id, name, color, entity_type, created_at
			 FROM tags
			 WHERE tenant_id = $1 AND entity_type = $2
			 ORDER BY name ASC
			 LIMIT $3 OFFSET $4`, tenantID, *entityType, limit, offset,
		)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT id, tenant_id, name, color, entity_type, created_at
			 FROM tags
			 WHERE tenant_id = $1
			 ORDER BY entity_type ASC, name ASC
			 LIMIT $2 OFFSET $3`, tenantID, limit, offset,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		tag, scanErr := r.scanTagFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		tags = append(tags, tag)
	}

	return tags, total, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, tag *models.Tag) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tags SET name = $1, color = $2 WHERE id = $3 AND tenant_id = $4`,
		tag.Name, tag.Color, tag.ID, tag.TenantID,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM tags WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	)
	return err
}

func (r *PostgresRepository) IsInUse(ctx context.Context, id, tenantID uuid.UUID) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT (
			SELECT COUNT(*) FROM contact_tags WHERE tag_id = $1 AND tenant_id = $2
		) + (
			SELECT COUNT(*) FROM company_tags WHERE tag_id = $1 AND tenant_id = $2
		) + (
			SELECT COUNT(*) FROM deal_tags WHERE tag_id = $1 AND tenant_id = $2
		) + (
			SELECT COUNT(*) FROM activity_tags WHERE tag_id = $1 AND tenant_id = $2
		)
	`, id, tenantID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Helper to scan a single row into Tag
func (r *PostgresRepository) scanTag(row pgx.Row) (*models.Tag, error) {
	var tag models.Tag
	err := row.Scan(
		&tag.ID, &tag.TenantID, &tag.Name, &tag.Color, &tag.EntityType, &tag.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTagNotFound
	}
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// Helper to scan rows iterator
func (r *PostgresRepository) scanTagFromRows(rows pgx.Rows) (*models.Tag, error) {
	var tag models.Tag
	err := rows.Scan(
		&tag.ID, &tag.TenantID, &tag.Name, &tag.Color, &tag.EntityType, &tag.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &tag, nil
}
