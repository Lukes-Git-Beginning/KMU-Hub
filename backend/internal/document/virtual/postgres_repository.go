package virtual

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
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

func normalizePagination(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// ListChatFiles returns files from chat channels the user is a member of.
// Access control: user must be in channel_memberships for the channel.
func (r *PostgresRepository) ListChatFiles(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.VirtualFile, int, error) {
	limit, offset = normalizePagination(limit, offset)

	countQuery := `
		SELECT COUNT(*)
		FROM chat_files cf
		INNER JOIN channel_memberships cm ON cm.channel_id = cf.channel_id AND cm.user_id = $1
		WHERE cf.is_deleted = FALSE
	`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count chat files: %w", err)
	}

	if total == 0 {
		return []*models.VirtualFile{}, 0, nil
	}

	query := `
		SELECT cf.id, cf.filename, cf.mime_type, cf.file_size, cf.storage_key,
		       'chat' AS source_type, cf.channel_id AS source_id,
		       c.name AS source_name,
		       cf.uploaded_by,
		       COALESCE(NULLIF(TRIM(u.first_name || ' ' || u.last_name), ''), u.email) AS uploaded_by_name,
		       cf.created_at
		FROM chat_files cf
		INNER JOIN channel_memberships cm ON cm.channel_id = cf.channel_id AND cm.user_id = $1
		INNER JOIN channels c ON c.id = cf.channel_id
		LEFT JOIN users u ON u.id = cf.uploaded_by
		WHERE cf.is_deleted = FALSE
		ORDER BY cf.created_at DESC
		LIMIT $2 OFFSET $3
	`
	return r.queryVirtualFiles(ctx, query, total, userID, limit, offset)
}

// ListEmailAttachments returns email attachments from the user's email accounts.
// Access control: user must own the email account.
func (r *PostgresRepository) ListEmailAttachments(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.VirtualFile, int, error) {
	limit, offset = normalizePagination(limit, offset)

	countQuery := `
		SELECT COUNT(*)
		FROM email_attachments ea
		INNER JOIN email_messages em ON em.id = ea.message_id
		INNER JOIN email_accounts eacc ON eacc.id = em.account_id AND eacc.user_id = $1
	`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count email attachments: %w", err)
	}

	if total == 0 {
		return []*models.VirtualFile{}, 0, nil
	}

	query := `
		SELECT ea.id, ea.filename, ea.content_type AS mime_type, ea.size_bytes AS file_size,
		       ea.minio_key AS storage_key,
		       'email' AS source_type, em.id AS source_id,
		       COALESCE(LEFT(em.subject, 80), em.from_email) AS source_name,
		       eacc.user_id AS uploaded_by,
		       COALESCE(eacc.display_name, eacc.email_address) AS uploaded_by_name,
		       ea.created_at
		FROM email_attachments ea
		INNER JOIN email_messages em ON em.id = ea.message_id
		INNER JOIN email_accounts eacc ON eacc.id = em.account_id AND eacc.user_id = $1
		ORDER BY ea.created_at DESC
		LIMIT $2 OFFSET $3
	`
	return r.queryVirtualFiles(ctx, query, total, userID, limit, offset)
}

// ListTaskFiles returns files from tasks the user has access to via
// project membership or direct task assignment.
func (r *PostgresRepository) ListTaskFiles(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.VirtualFile, int, error) {
	limit, offset = normalizePagination(limit, offset)

	countQuery := `
		SELECT COUNT(*)
		FROM task_files tf
		INNER JOIN tasks t ON t.id = tf.task_id
		WHERE (
			t.assignee_id = $1
			OR EXISTS (
				SELECT 1 FROM project_members pm
				WHERE pm.project_id = t.project_id AND pm.user_id = $1
			)
		)
	`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count task files: %w", err)
	}

	if total == 0 {
		return []*models.VirtualFile{}, 0, nil
	}

	query := `
		SELECT tf.id, tf.filename, tf.mime_type, tf.file_size, tf.storage_key,
		       'task' AS source_type, t.id AS source_id,
		       t.title AS source_name,
		       tf.uploaded_by,
		       COALESCE(NULLIF(TRIM(u.first_name || ' ' || u.last_name), ''), u.email) AS uploaded_by_name,
		       tf.created_at
		FROM task_files tf
		INNER JOIN tasks t ON t.id = tf.task_id
		LEFT JOIN users u ON u.id = tf.uploaded_by
		WHERE (
			t.assignee_id = $1
			OR EXISTS (
				SELECT 1 FROM project_members pm
				WHERE pm.project_id = t.project_id AND pm.user_id = $1
			)
		)
		ORDER BY tf.created_at DESC
		LIMIT $2 OFFSET $3
	`
	return r.queryVirtualFiles(ctx, query, total, userID, limit, offset)
}

// ListAll returns files from all sources combined, optionally filtered by sourceType.
// Uses UNION ALL of chat, email, and task file queries.
func (r *PostgresRepository) ListAll(ctx context.Context, userID uuid.UUID, sourceType string, limit, offset int) ([]*models.VirtualFile, int, error) {
	// If a specific source type is requested, delegate to the appropriate method
	switch sourceType {
	case models.VirtualFileSourceChat:
		return r.ListChatFiles(ctx, userID, limit, offset)
	case models.VirtualFileSourceEmail:
		return r.ListEmailAttachments(ctx, userID, limit, offset)
	case models.VirtualFileSourceTask:
		return r.ListTaskFiles(ctx, userID, limit, offset)
	case "":
		// Fall through to union query
	default:
		slog.Warn("unknown virtual file source type",
			"source_type", sourceType,
		)
		return []*models.VirtualFile{}, 0, nil
	}

	limit, offset = normalizePagination(limit, offset)

	// Count total from all sources
	countQuery := `
		SELECT (
			SELECT COUNT(*)
			FROM chat_files cf
			INNER JOIN channel_memberships cm ON cm.channel_id = cf.channel_id AND cm.user_id = $1
			WHERE cf.is_deleted = FALSE
		) + (
			SELECT COUNT(*)
			FROM email_attachments ea
			INNER JOIN email_messages em ON em.id = ea.message_id
			INNER JOIN email_accounts eacc ON eacc.id = em.account_id AND eacc.user_id = $1
		) + (
			SELECT COUNT(*)
			FROM task_files tf
			INNER JOIN tasks t ON t.id = tf.task_id
			WHERE t.assignee_id = $1
			   OR EXISTS (
			       SELECT 1 FROM project_members pm
			       WHERE pm.project_id = t.project_id AND pm.user_id = $1
			   )
		) AS total
	`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count all virtual files: %w", err)
	}

	if total == 0 {
		return []*models.VirtualFile{}, 0, nil
	}

	// UNION ALL of all three sources, ordered by created_at DESC
	query := `
		SELECT * FROM (
			SELECT cf.id, cf.filename, cf.mime_type, cf.file_size, cf.storage_key,
			       'chat'::text AS source_type, cf.channel_id AS source_id,
			       c.name AS source_name,
			       cf.uploaded_by,
			       COALESCE(NULLIF(TRIM(u.first_name || ' ' || u.last_name), ''), u.email) AS uploaded_by_name,
			       cf.created_at
			FROM chat_files cf
			INNER JOIN channel_memberships cm ON cm.channel_id = cf.channel_id AND cm.user_id = $1
			INNER JOIN channels c ON c.id = cf.channel_id
			LEFT JOIN users u ON u.id = cf.uploaded_by
			WHERE cf.is_deleted = FALSE

			UNION ALL

			SELECT ea.id, ea.filename, ea.content_type, ea.size_bytes,
			       ea.minio_key,
			       'email'::text AS source_type, em.id AS source_id,
			       COALESCE(LEFT(em.subject, 80), em.from_email) AS source_name,
			       eacc.user_id AS uploaded_by,
			       COALESCE(eacc.display_name, eacc.email_address) AS uploaded_by_name,
			       ea.created_at
			FROM email_attachments ea
			INNER JOIN email_messages em ON em.id = ea.message_id
			INNER JOIN email_accounts eacc ON eacc.id = em.account_id AND eacc.user_id = $1

			UNION ALL

			SELECT tf.id, tf.filename, tf.mime_type, tf.file_size, tf.storage_key,
			       'task'::text AS source_type, t.id AS source_id,
			       t.title AS source_name,
			       tf.uploaded_by,
			       COALESCE(NULLIF(TRIM(u.first_name || ' ' || u.last_name), ''), u.email) AS uploaded_by_name,
			       tf.created_at
			FROM task_files tf
			INNER JOIN tasks t ON t.id = tf.task_id
			LEFT JOIN users u ON u.id = tf.uploaded_by
			WHERE t.assignee_id = $1
			   OR EXISTS (
			       SELECT 1 FROM project_members pm
			       WHERE pm.project_id = t.project_id AND pm.user_id = $1
			   )
		) AS virtual_files
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	return r.queryVirtualFiles(ctx, query, total, userID, limit, offset)
}

// queryVirtualFiles is a helper that executes a query returning VirtualFile rows.
func (r *PostgresRepository) queryVirtualFiles(ctx context.Context, query string, total int, userID uuid.UUID, limit, offset int) ([]*models.VirtualFile, int, error) {
	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query virtual files: %w", err)
	}
	defer rows.Close()

	var files []*models.VirtualFile
	for rows.Next() {
		var vf models.VirtualFile
		if err := rows.Scan(
			&vf.ID, &vf.Filename, &vf.MimeType, &vf.FileSize, &vf.StorageKey,
			&vf.SourceType, &vf.SourceID, &vf.SourceName,
			&vf.UploadedBy, &vf.UploadedByName, &vf.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan virtual file: %w", err)
		}
		files = append(files, &vf)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("virtual files rows: %w", err)
	}

	if files == nil {
		files = []*models.VirtualFile{}
	}

	return files, total, nil
}
