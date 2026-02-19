package folder

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

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL folder repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, folder *models.DocumentFolder) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO document_folders (id, name, parent_id, space_type, space_id, is_system, icon, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		folder.ID, folder.Name, folder.ParentID, folder.SpaceType, folder.SpaceID,
		folder.IsSystem, folder.Icon, folder.CreatedBy, folder.CreatedAt, folder.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DocumentFolder, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT f.id, f.name, f.parent_id, f.space_type, f.space_id, f.is_system, f.icon, f.created_by, f.created_at, f.updated_at,
		        COALESCE((SELECT COUNT(*) FROM document_files df WHERE df.folder_id = f.id AND NOT df.is_deleted), 0) AS file_count
		 FROM document_folders f
		 WHERE f.id = $1`, id,
	)
	return r.scanFolder(row)
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]*models.DocumentFolder, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	if filter.ParentID != nil {
		conditions = append(conditions, fmt.Sprintf("f.parent_id = $%d", argNum))
		args = append(args, *filter.ParentID)
		argNum++
	} else {
		// When no parent specified, show root folders (parent_id IS NULL)
		// only when space filters are provided
		if filter.SpaceType != nil {
			conditions = append(conditions, "f.parent_id IS NULL")
		}
	}

	if filter.SpaceType != nil {
		conditions = append(conditions, fmt.Sprintf("f.space_type = $%d", argNum))
		args = append(args, *filter.SpaceType)
		argNum++
	}

	if filter.SpaceID != nil {
		conditions = append(conditions, fmt.Sprintf("f.space_id = $%d", argNum))
		args = append(args, *filter.SpaceID)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM document_folders f %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Query folders with file count, system folders first, then by name
	query := fmt.Sprintf(`
		SELECT f.id, f.name, f.parent_id, f.space_type, f.space_id, f.is_system, f.icon, f.created_by, f.created_at, f.updated_at,
		       COALESCE((SELECT COUNT(*) FROM document_files df WHERE df.folder_id = f.id AND NOT df.is_deleted), 0) AS file_count
		FROM document_folders f
		%s
		ORDER BY f.is_system DESC, f.name ASC
	`, whereClause)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var folders []*models.DocumentFolder
	for rows.Next() {
		folder, scanErr := r.scanFolderFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		folders = append(folders, folder)
	}

	return folders, total, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, id uuid.UUID, input UpdateInput) error {
	var setClauses []string
	var args []any
	argNum := 1

	if input.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *input.Name)
		argNum++
	}

	if input.ParentID != nil {
		setClauses = append(setClauses, fmt.Sprintf("parent_id = $%d", argNum))
		args = append(args, *input.ParentID)
		argNum++
	}

	if input.Icon != nil {
		setClauses = append(setClauses, fmt.Sprintf("icon = $%d", argNum))
		args = append(args, *input.Icon)
		argNum++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf("UPDATE document_folders SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argNum)
	args = append(args, id)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFolderNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Soft-delete files in this folder first (cascade handles subfolder deletion via FK)
	_, err := r.pool.Exec(ctx,
		`UPDATE document_files SET is_deleted = TRUE, deleted_at = NOW() WHERE folder_id = $1 AND NOT is_deleted`, id)
	if err != nil {
		return err
	}

	// Delete the folder (CASCADE will remove subfolders, and their files will be handled by FK ON DELETE CASCADE)
	tag, err := r.pool.Exec(ctx, `DELETE FROM document_folders WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFolderNotFound
	}
	return nil
}

// GetPath returns the breadcrumb path from root to the given folder using a recursive CTE.
func (r *PostgresRepository) GetPath(ctx context.Context, id uuid.UUID) ([]models.FolderPathSegment, error) {
	rows, err := r.pool.Query(ctx,
		`WITH RECURSIVE path AS (
			SELECT id, name, parent_id, 0 AS depth
			FROM document_folders
			WHERE id = $1
			UNION ALL
			SELECT f.id, f.name, f.parent_id, p.depth + 1
			FROM document_folders f
			JOIN path p ON f.id = p.parent_id
		)
		SELECT id, name FROM path ORDER BY depth DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var segments []models.FolderPathSegment
	for rows.Next() {
		var seg models.FolderPathSegment
		if scanErr := rows.Scan(&seg.ID, &seg.Name); scanErr != nil {
			return nil, scanErr
		}
		segments = append(segments, seg)
	}

	if len(segments) == 0 {
		return nil, ErrFolderNotFound
	}

	return segments, rows.Err()
}

func (r *PostgresRepository) GetChildren(ctx context.Context, parentID uuid.UUID) ([]*models.DocumentFolder, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT f.id, f.name, f.parent_id, f.space_type, f.space_id, f.is_system, f.icon, f.created_by, f.created_at, f.updated_at,
		        COALESCE((SELECT COUNT(*) FROM document_files df WHERE df.folder_id = f.id AND NOT df.is_deleted), 0) AS file_count
		 FROM document_folders f
		 WHERE f.parent_id = $1
		 ORDER BY f.is_system DESC, f.name ASC`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*models.DocumentFolder
	for rows.Next() {
		folder, scanErr := r.scanFolderFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		folders = append(folders, folder)
	}
	return folders, rows.Err()
}

func (r *PostgresRepository) CountFiles(ctx context.Context, folderID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM document_files WHERE folder_id = $1 AND NOT is_deleted`, folderID,
	).Scan(&count)
	return count, err
}

// IsDescendant checks if folderID is a descendant of potentialAncestorID using a recursive CTE.
func (r *PostgresRepository) IsDescendant(ctx context.Context, folderID, potentialAncestorID uuid.UUID) (bool, error) {
	var isDesc bool
	err := r.pool.QueryRow(ctx,
		`WITH RECURSIVE ancestors AS (
			SELECT parent_id
			FROM document_folders
			WHERE id = $1
			UNION ALL
			SELECT f.parent_id
			FROM document_folders f
			JOIN ancestors a ON f.id = a.parent_id
			WHERE a.parent_id IS NOT NULL
		)
		SELECT EXISTS(SELECT 1 FROM ancestors WHERE parent_id = $2)`,
		folderID, potentialAncestorID,
	).Scan(&isDesc)
	return isDesc, err
}

// scanFolder scans a single row into a DocumentFolder.
func (r *PostgresRepository) scanFolder(row pgx.Row) (*models.DocumentFolder, error) {
	var f models.DocumentFolder
	err := row.Scan(
		&f.ID, &f.Name, &f.ParentID, &f.SpaceType, &f.SpaceID,
		&f.IsSystem, &f.Icon, &f.CreatedBy, &f.CreatedAt, &f.UpdatedAt,
		&f.FileCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFolderNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// scanFolderFromRows scans a rows iterator into a DocumentFolder.
func (r *PostgresRepository) scanFolderFromRows(rows pgx.Rows) (*models.DocumentFolder, error) {
	var f models.DocumentFolder
	err := rows.Scan(
		&f.ID, &f.Name, &f.ParentID, &f.SpaceType, &f.SpaceID,
		&f.IsSystem, &f.Icon, &f.CreatedBy, &f.CreatedAt, &f.UpdatedAt,
		&f.FileCount,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}
