package tag

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Create inserts a new tag into the database.
func (r *PostgresRepository) Create(ctx context.Context, tag *models.DocumentTag) error {
	query := `
		INSERT INTO document_tags (id, tenant_id, name, color, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		tag.ID, tag.TenantID, tag.Name, tag.Color, tag.CreatedBy, tag.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrTagNameConflict
		}
		return fmt.Errorf("create tag: %w", err)
	}
	return nil
}

// List returns all tags for the tenant ordered by name with file_count via subquery.
func (r *PostgresRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*models.DocumentTag, error) {
	query := `
		SELECT t.id, t.name, t.color, t.created_by, t.created_at,
		       (SELECT COUNT(*) FROM document_file_tags WHERE tag_id = t.id) AS file_count
		FROM document_tags t
		WHERE t.tenant_id = $1
		ORDER BY t.name
	`
	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var tags []*models.DocumentTag
	for rows.Next() {
		var tag models.DocumentTag
		if err := rows.Scan(
			&tag.ID, &tag.Name, &tag.Color, &tag.CreatedBy, &tag.CreatedAt,
			&tag.FileCount,
		); err != nil {
			return nil, fmt.Errorf("list tags scan: %w", err)
		}
		tags = append(tags, &tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tags rows: %w", err)
	}

	if tags == nil {
		tags = []*models.DocumentTag{}
	}

	return tags, nil
}

// Delete removes a tag scoped to the tenant. File-tag associations are
// cascade-deleted via FK.
func (r *PostgresRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx,
		`DELETE FROM document_tags WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTagNotFound
	}
	return nil
}

// TagFile creates a file-tag association. Idempotent via ON CONFLICT DO NOTHING.
// document_file_tags.tenant_id has been NOT NULL since migration 000114 — this
// insert used to omit the column entirely, so every TagFile call failed the
// constraint in production; the feature was dead, not degraded.
//
// The document_tags(id) FK on document_file_tags is checked by a trigger that
// runs with the table owner's privileges, not the caller's RLS session — it
// would happily accept a tag_id belonging to a different tenant. The explicit
// existence check below is what actually enforces tenant ownership of the tag
// before the association is created.
func (r *PostgresRepository) TagFile(ctx context.Context, tenantID, fileID, tagID uuid.UUID) error {
	var ownedByTenant bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM document_tags WHERE id = $1 AND tenant_id = $2)`,
		tagID, tenantID,
	).Scan(&ownedByTenant); err != nil {
		return fmt.Errorf("check tag ownership: %w", err)
	}
	if !ownedByTenant {
		return ErrTagNotFound
	}

	query := `
		INSERT INTO document_file_tags (tenant_id, file_id, tag_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (file_id, tag_id) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, query, tenantID, fileID, tagID)
	if err != nil {
		return fmt.Errorf("tag file: %w", err)
	}

	slog.Info("file tagged",
		"file_id", fileID,
		"tag_id", tagID,
	)
	return nil
}

// UntagFile removes a file-tag association.
func (r *PostgresRepository) UntagFile(ctx context.Context, tenantID, fileID, tagID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM document_file_tags WHERE file_id = $1 AND tag_id = $2 AND tenant_id = $3`,
		fileID, tagID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("untag file: %w", err)
	}

	slog.Info("file untagged",
		"file_id", fileID,
		"tag_id", tagID,
	)
	return nil
}
