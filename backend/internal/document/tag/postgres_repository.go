package tag

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

// GetByID retrieves a tag by its ID with file_count.
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DocumentTag, error) {
	query := `
		SELECT t.id, t.name, t.color, t.created_by, t.created_at,
		       (SELECT COUNT(*) FROM document_file_tags WHERE tag_id = t.id) AS file_count
		FROM document_tags t
		WHERE t.id = $1
	`
	var tag models.DocumentTag
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&tag.ID, &tag.Name, &tag.Color, &tag.CreatedBy, &tag.CreatedAt,
		&tag.FileCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, fmt.Errorf("get tag by id: %w", err)
	}
	return &tag, nil
}

// List returns all tags ordered by name with file_count via subquery.
func (r *PostgresRepository) List(ctx context.Context) ([]*models.DocumentTag, error) {
	query := `
		SELECT t.id, t.name, t.color, t.created_by, t.created_at,
		       (SELECT COUNT(*) FROM document_file_tags WHERE tag_id = t.id) AS file_count
		FROM document_tags t
		ORDER BY t.name
	`
	rows, err := r.pool.Query(ctx, query)
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

// Delete removes a tag by ID. File-tag associations are cascade-deleted via FK.
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM document_tags WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTagNotFound
	}
	return nil
}

// TagFile creates a file-tag association. Idempotent via ON CONFLICT DO NOTHING.
func (r *PostgresRepository) TagFile(ctx context.Context, fileID, tagID uuid.UUID) error {
	query := `
		INSERT INTO document_file_tags (file_id, tag_id)
		VALUES ($1, $2)
		ON CONFLICT (file_id, tag_id) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, query, fileID, tagID)
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
func (r *PostgresRepository) UntagFile(ctx context.Context, fileID, tagID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM document_file_tags WHERE file_id = $1 AND tag_id = $2`,
		fileID, tagID,
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

// ListFileTags returns all tags associated with a given file.
func (r *PostgresRepository) ListFileTags(ctx context.Context, fileID uuid.UUID) ([]*models.DocumentTag, error) {
	query := `
		SELECT t.id, t.name, t.color, t.created_by, t.created_at,
		       (SELECT COUNT(*) FROM document_file_tags WHERE tag_id = t.id) AS file_count
		FROM document_tags t
		INNER JOIN document_file_tags dft ON dft.tag_id = t.id
		WHERE dft.file_id = $1
		ORDER BY t.name
	`
	rows, err := r.pool.Query(ctx, query, fileID)
	if err != nil {
		return nil, fmt.Errorf("list file tags: %w", err)
	}
	defer rows.Close()

	var tags []*models.DocumentTag
	for rows.Next() {
		var tag models.DocumentTag
		if err := rows.Scan(
			&tag.ID, &tag.Name, &tag.Color, &tag.CreatedBy, &tag.CreatedAt,
			&tag.FileCount,
		); err != nil {
			return nil, fmt.Errorf("list file tags scan: %w", err)
		}
		tags = append(tags, &tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list file tags rows: %w", err)
	}

	if tags == nil {
		tags = []*models.DocumentTag{}
	}

	return tags, nil
}

// ListFilesByTag returns files with a given tag, filtered to non-deleted files.
func (r *PostgresRepository) ListFilesByTag(ctx context.Context, tagID uuid.UUID, limit, offset int) ([]*models.DocumentFile, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Count total
	var total int
	countQuery := `
		SELECT COUNT(*)
		FROM document_files f
		INNER JOIN document_file_tags dft ON dft.file_id = f.id
		WHERE dft.tag_id = $1 AND f.is_deleted = FALSE
	`
	if err := r.pool.QueryRow(ctx, countQuery, tagID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("list files by tag count: %w", err)
	}

	if total == 0 {
		return []*models.DocumentFile{}, 0, nil
	}

	// Fetch files
	query := `
		SELECT f.id, f.folder_id, f.filename, f.mime_type, f.file_size, f.storage_key,
		       f.thumbnail_key, f.current_version, f.owner_id, f.is_favorite, f.is_deleted,
		       f.content_text, f.created_at, f.updated_at, f.deleted_at
		FROM document_files f
		INNER JOIN document_file_tags dft ON dft.file_id = f.id
		WHERE dft.tag_id = $1 AND f.is_deleted = FALSE
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, tagID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list files by tag: %w", err)
	}
	defer rows.Close()

	var files []*models.DocumentFile
	for rows.Next() {
		var f models.DocumentFile
		if err := rows.Scan(
			&f.ID, &f.FolderID, &f.Filename, &f.MimeType, &f.FileSize, &f.StorageKey,
			&f.ThumbnailKey, &f.CurrentVersion, &f.OwnerID, &f.IsFavorite, &f.IsDeleted,
			&f.ContentText, &f.CreatedAt, &f.UpdatedAt, &f.DeletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("list files by tag scan: %w", err)
		}
		files = append(files, &f)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list files by tag rows: %w", err)
	}

	return files, total, nil
}
