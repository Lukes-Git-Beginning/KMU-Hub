package file

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL file repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, file *models.DocumentFile) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO document_files (id, tenant_id, folder_id, filename, mime_type, file_size, storage_key, thumbnail_key, current_version, owner_id, is_favorite, is_deleted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		file.ID, file.TenantID, file.FolderID, file.Filename, file.MimeType, file.FileSize,
		file.StorageKey, file.ThumbnailKey, file.CurrentVersion, file.OwnerID,
		file.IsFavorite, file.IsDeleted, file.CreatedAt, file.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.DocumentFile, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT f.id, f.folder_id, f.filename, f.mime_type, f.file_size, f.storage_key, f.thumbnail_key,
		        f.current_version, f.owner_id, f.is_favorite, f.is_deleted, f.content_text,
		        f.created_at, f.updated_at, f.deleted_at
		 FROM document_files f
		 WHERE f.id = $1 AND f.tenant_id = $2`, id, tenantID,
	)
	return r.scanFile(row)
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]*models.DocumentFile, int, error) {
	// tenant_id is always the first condition for isolation
	conditions := []string{fmt.Sprintf("f.tenant_id = $%d", 1)}
	args := []any{filter.TenantID}
	argNum := 2

	if filter.FolderID != nil {
		conditions = append(conditions, fmt.Sprintf("f.folder_id = $%d", argNum))
		args = append(args, *filter.FolderID)
		argNum++
	}

	if filter.OwnerID != nil {
		conditions = append(conditions, fmt.Sprintf("f.owner_id = $%d", argNum))
		args = append(args, *filter.OwnerID)
		argNum++
	}

	if filter.IsFavorite != nil {
		conditions = append(conditions, fmt.Sprintf("f.is_favorite = $%d", argNum))
		args = append(args, *filter.IsFavorite)
		argNum++
	}

	if filter.IsDeleted != nil {
		conditions = append(conditions, fmt.Sprintf("f.is_deleted = $%d", argNum))
		args = append(args, *filter.IsDeleted)
		argNum++
	} else {
		// Default: exclude deleted files
		conditions = append(conditions, "NOT f.is_deleted")
	}

	if len(filter.TagIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf(`f.id IN (
			SELECT file_id FROM document_file_tags WHERE tag_id = ANY($%d)
			GROUP BY file_id HAVING COUNT(DISTINCT tag_id) = $%d
		)`, argNum, argNum+1))
		args = append(args, filter.TagIDs, len(filter.TagIDs))
		argNum += 2
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total using window function alternative: just count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM document_files f %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build ORDER BY
	orderBy := "f.created_at"
	switch filter.SortField {
	case "name":
		orderBy = "LOWER(f.filename)"
	case "size":
		orderBy = "f.file_size"
	case "date":
		orderBy = "f.updated_at"
	case "type":
		orderBy = "f.mime_type"
	}
	direction := "DESC"
	if strings.EqualFold(filter.SortDir, "asc") {
		direction = "ASC"
	}

	// Apply pagination defaults
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT f.id, f.folder_id, f.filename, f.mime_type, f.file_size, f.storage_key, f.thumbnail_key,
		       f.current_version, f.owner_id, f.is_favorite, f.is_deleted, f.content_text,
		       f.created_at, f.updated_at, f.deleted_at
		FROM document_files f
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, direction, argNum, argNum+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var files []*models.DocumentFile
	for rows.Next() {
		file, scanErr := r.scanFileFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		files = append(files, file)
	}

	return files, total, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, input UpdateInput) error {
	var setClauses []string
	var args []any
	argNum := 1

	if input.Filename != nil {
		setClauses = append(setClauses, fmt.Sprintf("filename = $%d", argNum))
		args = append(args, *input.Filename)
		argNum++
	}

	if input.FolderID != nil {
		setClauses = append(setClauses, fmt.Sprintf("folder_id = $%d", argNum))
		args = append(args, *input.FolderID)
		argNum++
	}

	if input.IsFavorite != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_favorite = $%d", argNum))
		args = append(args, *input.IsFavorite)
		argNum++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf("UPDATE document_files SET %s WHERE id = $%d AND tenant_id = $%d AND NOT is_deleted",
		strings.Join(setClauses, ", "), argNum, argNum+1)
	args = append(args, id, tenantID)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFileNotFound
	}
	return nil
}

func (r *PostgresRepository) SoftDelete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE document_files SET is_deleted = TRUE, deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND tenant_id = $2 AND NOT is_deleted`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFileNotFound
	}
	return nil
}

// Versioning

func (r *PostgresRepository) CreateVersion(ctx context.Context, version *models.DocumentFileVersion) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO document_file_versions (id, tenant_id, file_id, version_number, version_label, storage_key, file_size, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		version.ID, version.TenantID, version.FileID, version.VersionNumber, version.VersionLabel,
		version.StorageKey, version.FileSize, version.CreatedBy, version.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) ListVersions(ctx context.Context, fileID uuid.UUID) ([]*models.DocumentFileVersion, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, file_id, version_number, version_label, storage_key, file_size, created_by, created_at
		 FROM document_file_versions
		 WHERE file_id = $1
		 ORDER BY version_number DESC`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*models.DocumentFileVersion
	for rows.Next() {
		var v models.DocumentFileVersion
		if scanErr := rows.Scan(&v.ID, &v.TenantID, &v.FileID, &v.VersionNumber, &v.VersionLabel,
			&v.StorageKey, &v.FileSize, &v.CreatedBy, &v.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		versions = append(versions, &v)
	}
	return versions, rows.Err()
}

func (r *PostgresRepository) GetVersion(ctx context.Context, fileID uuid.UUID, versionNumber int) (*models.DocumentFileVersion, error) {
	var v models.DocumentFileVersion
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, file_id, version_number, version_label, storage_key, file_size, created_by, created_at
		 FROM document_file_versions
		 WHERE file_id = $1 AND version_number = $2`, fileID, versionNumber,
	).Scan(&v.ID, &v.TenantID, &v.FileID, &v.VersionNumber, &v.VersionLabel,
		&v.StorageKey, &v.FileSize, &v.CreatedBy, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrVersionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *PostgresRepository) GetVersionByID(ctx context.Context, fileID, versionID, tenantID uuid.UUID) (*models.DocumentFileVersion, error) {
	var v models.DocumentFileVersion
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, file_id, version_number, version_label, storage_key, file_size, created_by, created_at
		 FROM document_file_versions
		 WHERE id = $1 AND file_id = $2 AND tenant_id = $3`, versionID, fileID, tenantID,
	).Scan(&v.ID, &v.TenantID, &v.FileID, &v.VersionNumber, &v.VersionLabel,
		&v.StorageKey, &v.FileSize, &v.CreatedBy, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrVersionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *PostgresRepository) UpdateCurrentVersion(ctx context.Context, fileID uuid.UUID, versionNumber int) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE document_files SET current_version = $1, updated_at = NOW() WHERE id = $2`,
		versionNumber, fileID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFileNotFound
	}
	return nil
}

func (r *PostgresRepository) UpdateSearchContent(ctx context.Context, id uuid.UUID, contentText string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE document_files SET content_text = $1, updated_at = NOW() WHERE id = $2`,
		contentText, id)
	return err
}

// Entity links

func (r *PostgresRepository) CreateEntityLink(ctx context.Context, link *models.DocumentEntityLink) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO document_entity_links (id, tenant_id, file_id, entity_type, entity_id, linked_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		link.ID, link.TenantID, link.FileID, link.EntityType, link.EntityID, link.LinkedBy, link.CreatedAt,
	)
	return err
}

// DeleteEntityLink removes a link by its own ID. tenantID is required even
// though RLS also enforces it: it turns a cross-tenant delete attempt into
// the same "not found" a bad ID gets, instead of a bare RLS-filtered no-op
// that would be indistinguishable from a database error to the caller.
func (r *PostgresRepository) DeleteEntityLink(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM document_entity_links WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrFileNotFound
	}
	return nil
}

func (r *PostgresRepository) ListEntityLinks(ctx context.Context, fileID uuid.UUID, tenantID uuid.UUID) ([]*models.DocumentEntityLink, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, file_id, entity_type, entity_id, linked_by, created_at
		 FROM document_entity_links
		 WHERE file_id = $1 AND tenant_id = $2
		 ORDER BY created_at DESC`, fileID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*models.DocumentEntityLink
	for rows.Next() {
		var l models.DocumentEntityLink
		if scanErr := rows.Scan(&l.ID, &l.TenantID, &l.FileID, &l.EntityType, &l.EntityID, &l.LinkedBy, &l.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		links = append(links, &l)
	}
	return links, rows.Err()
}

func (r *PostgresRepository) ListFilesByEntity(ctx context.Context, entityType string, entityID uuid.UUID, tenantID uuid.UUID) ([]*models.DocumentFile, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT f.id, f.folder_id, f.filename, f.mime_type, f.file_size, f.storage_key, f.thumbnail_key,
		        f.current_version, f.owner_id, f.is_favorite, f.is_deleted, f.content_text,
		        f.created_at, f.updated_at, f.deleted_at
		 FROM document_files f
		 JOIN document_entity_links del ON f.id = del.file_id
		 WHERE del.entity_type = $1 AND del.entity_id = $2 AND del.tenant_id = $3 AND f.tenant_id = $3 AND NOT f.is_deleted
		 ORDER BY f.created_at DESC`, entityType, entityID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*models.DocumentFile
	for rows.Next() {
		file, scanErr := r.scanFileFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

// Activity (append-only audit trail)

func (r *PostgresRepository) CreateActivity(ctx context.Context, activity *models.DocumentFileActivity) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO document_file_activity (id, tenant_id, file_id, action, actor_id, detail, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		activity.ID, activity.TenantID, activity.FileID, activity.Action, activity.ActorID, activity.Detail, activity.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) ListActivity(ctx context.Context, fileID uuid.UUID, tenantID uuid.UUID) ([]*models.DocumentFileActivity, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT a.id, a.tenant_id, a.file_id, a.action, a.actor_id,
		        COALESCE(u.first_name || ' ' || u.last_name, '') AS actor_name,
		        a.detail, a.created_at
		 FROM document_file_activity a
		 LEFT JOIN users u ON u.id = a.actor_id
		 WHERE a.file_id = $1 AND a.tenant_id = $2
		 ORDER BY a.created_at DESC`, fileID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []*models.DocumentFileActivity
	for rows.Next() {
		var a models.DocumentFileActivity
		if scanErr := rows.Scan(&a.ID, &a.TenantID, &a.FileID, &a.Action, &a.ActorID,
			&a.ActorName, &a.Detail, &a.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		activities = append(activities, &a)
	}
	return activities, rows.Err()
}

// Comments

func (r *PostgresRepository) CreateComment(ctx context.Context, comment *models.DocumentFileComment) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO document_file_comments (id, tenant_id, file_id, author_id, content, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		comment.ID, comment.TenantID, comment.FileID, comment.AuthorID, comment.Content, comment.CreatedAt, comment.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetCommentByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.DocumentFileComment, error) {
	var c models.DocumentFileComment
	err := r.pool.QueryRow(ctx,
		`SELECT c.id, c.tenant_id, c.file_id, c.author_id,
		        COALESCE(u.first_name || ' ' || u.last_name, '') AS author_name,
		        c.content, c.created_at, c.updated_at
		 FROM document_file_comments c
		 LEFT JOIN users u ON u.id = c.author_id
		 WHERE c.id = $1 AND c.tenant_id = $2`, id, tenantID,
	).Scan(&c.ID, &c.TenantID, &c.FileID, &c.AuthorID, &c.AuthorName, &c.Content, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCommentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *PostgresRepository) ListComments(ctx context.Context, fileID uuid.UUID, tenantID uuid.UUID) ([]*models.DocumentFileComment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.tenant_id, c.file_id, c.author_id,
		        COALESCE(u.first_name || ' ' || u.last_name, '') AS author_name,
		        c.content, c.created_at, c.updated_at
		 FROM document_file_comments c
		 LEFT JOIN users u ON u.id = c.author_id
		 WHERE c.file_id = $1 AND c.tenant_id = $2
		 ORDER BY c.created_at ASC`, fileID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*models.DocumentFileComment
	for rows.Next() {
		var c models.DocumentFileComment
		if scanErr := rows.Scan(&c.ID, &c.TenantID, &c.FileID, &c.AuthorID, &c.AuthorName, &c.Content, &c.CreatedAt, &c.UpdatedAt); scanErr != nil {
			return nil, scanErr
		}
		comments = append(comments, &c)
	}
	return comments, rows.Err()
}

func (r *PostgresRepository) UpdateComment(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, content string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE document_file_comments SET content = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`,
		content, id, tenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCommentNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteComment(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM document_file_comments WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCommentNotFound
	}
	return nil
}

// Share links

func (r *PostgresRepository) CreateShareLink(ctx context.Context, link *models.DocumentShareLink) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO document_share_links
		    (id, tenant_id, file_id, token, password_hash, expires_at, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		link.ID, link.TenantID, link.FileID, link.Token, link.PasswordHash, link.ExpiresAt, link.CreatedBy, link.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) ListShareLinks(ctx context.Context, fileID uuid.UUID, tenantID uuid.UUID) ([]*models.DocumentShareLink, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+shareLinkColumns+`
		 FROM document_share_links
		 WHERE file_id = $1 AND tenant_id = $2
		 ORDER BY created_at DESC`, fileID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*models.DocumentShareLink
	for rows.Next() {
		l, scanErr := scanShareLink(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

// RevokeShareLink soft-revokes a link by its own ID. tenantID is required
// even though RLS also enforces it, same reasoning as DeleteEntityLink: a
// cross-tenant attempt becomes the same "not found" a bad ID gets.
func (r *PostgresRepository) RevokeShareLink(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, revokedAt time.Time) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE document_share_links SET revoked_at = $1 WHERE id = $2 AND tenant_id = $3 AND revoked_at IS NULL`,
		revokedAt, id, tenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrShareLinkNotFound
	}
	return nil
}

// GetShareLinkByToken resolves a link — and with it its tenant — from the
// secret alone. Runs in the system context (see Repository.GetShareLinkByToken
// doc comment): RLS would otherwise filter away the one row that answers
// which tenant the caller may see.
func (r *PostgresRepository) GetShareLinkByToken(ctx context.Context, token string) (*models.DocumentShareLink, error) {
	if token == "" {
		return nil, ErrShareLinkInvalid
	}
	row := r.pool.QueryRow(database.WithSystemContext(ctx),
		`SELECT `+shareLinkColumns+` FROM document_share_links WHERE token = $1`, token)
	l, err := scanShareLink(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrShareLinkInvalid
	}
	return l, err
}

func (r *PostgresRepository) IncrementShareLinkView(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE document_share_links SET view_count = view_count + 1 WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	return err
}

const shareLinkColumns = `id, tenant_id, file_id, token, password_hash, expires_at, revoked_at, view_count, created_by, created_at`

type shareLinkScanner interface {
	Scan(dest ...any) error
}

func scanShareLink(row shareLinkScanner) (*models.DocumentShareLink, error) {
	var l models.DocumentShareLink
	err := row.Scan(&l.ID, &l.TenantID, &l.FileID, &l.Token, &l.PasswordHash,
		&l.ExpiresAt, &l.RevokedAt, &l.ViewCount, &l.CreatedBy, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// scanFile scans a single row into a DocumentFile.
func (r *PostgresRepository) scanFile(row pgx.Row) (*models.DocumentFile, error) {
	var f models.DocumentFile
	err := row.Scan(
		&f.ID, &f.FolderID, &f.Filename, &f.MimeType, &f.FileSize, &f.StorageKey, &f.ThumbnailKey,
		&f.CurrentVersion, &f.OwnerID, &f.IsFavorite, &f.IsDeleted, &f.ContentText,
		&f.CreatedAt, &f.UpdatedAt, &f.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFileNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// scanFileFromRows scans a rows iterator into a DocumentFile.
func (r *PostgresRepository) scanFileFromRows(rows pgx.Rows) (*models.DocumentFile, error) {
	var f models.DocumentFile
	err := rows.Scan(
		&f.ID, &f.FolderID, &f.Filename, &f.MimeType, &f.FileSize, &f.StorageKey, &f.ThumbnailKey,
		&f.CurrentVersion, &f.OwnerID, &f.IsFavorite, &f.IsDeleted, &f.ContentText,
		&f.CreatedAt, &f.UpdatedAt, &f.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}
