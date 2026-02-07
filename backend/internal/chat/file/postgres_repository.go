package file

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL file repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateFile(ctx context.Context, file *models.ChatFile) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO chat_files (id, message_id, channel_id, filename, mime_type, file_size, storage_key, thumbnail_key, uploaded_by, is_deleted, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		file.ID, file.MessageID, file.ChannelID, file.Filename, file.MimeType,
		file.FileSize, file.StorageKey, file.ThumbnailKey, file.UploadedBy,
		file.IsDeleted, file.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetFileByID(ctx context.Context, id uuid.UUID) (*models.ChatFile, error) {
	var f models.ChatFile
	err := r.pool.QueryRow(ctx,
		`SELECT id, message_id, channel_id, filename, mime_type, file_size, storage_key, thumbnail_key,
		        uploaded_by, is_deleted, created_at, deleted_at
		 FROM chat_files WHERE id = $1`, id,
	).Scan(&f.ID, &f.MessageID, &f.ChannelID, &f.Filename, &f.MimeType, &f.FileSize,
		&f.StorageKey, &f.ThumbnailKey, &f.UploadedBy, &f.IsDeleted, &f.CreatedAt, &f.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFileNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *PostgresRepository) ListChannelFiles(ctx context.Context, channelID uuid.UUID, limit, offset int) ([]*models.ChatFileWithUploader, int, error) {
	// Count total
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM chat_files WHERE channel_id = $1 AND is_deleted = FALSE`, channelID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT cf.id, cf.message_id, cf.channel_id, cf.filename, cf.mime_type, cf.file_size,
		       cf.storage_key, cf.thumbnail_key, cf.uploaded_by, cf.is_deleted, cf.created_at, cf.deleted_at,
		       u.first_name, u.last_name
		FROM chat_files cf
		JOIN users u ON cf.uploaded_by = u.id
		WHERE cf.channel_id = $1 AND cf.is_deleted = FALSE
		ORDER BY cf.created_at DESC
		LIMIT $2 OFFSET $3
	`, channelID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var files []*models.ChatFileWithUploader
	for rows.Next() {
		var f models.ChatFileWithUploader
		if scanErr := rows.Scan(
			&f.ID, &f.MessageID, &f.ChannelID, &f.Filename, &f.MimeType, &f.FileSize,
			&f.StorageKey, &f.ThumbnailKey, &f.UploadedBy, &f.IsDeleted, &f.CreatedAt, &f.DeletedAt,
			&f.UploaderFirstName, &f.UploaderLastName,
		); scanErr != nil {
			return nil, 0, scanErr
		}
		files = append(files, &f)
	}

	return files, total, rows.Err()
}

func (r *PostgresRepository) GetFilesByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]models.ChatFileWithUploader, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT cf.id, cf.message_id, cf.channel_id, cf.filename, cf.mime_type, cf.file_size,
		       cf.storage_key, cf.thumbnail_key, cf.uploaded_by, cf.is_deleted, cf.created_at, cf.deleted_at,
		       u.first_name, u.last_name
		FROM chat_files cf
		JOIN users u ON cf.uploaded_by = u.id
		WHERE cf.message_id = ANY($1) AND cf.is_deleted = FALSE
		ORDER BY cf.created_at
	`, messageIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]models.ChatFileWithUploader)
	for rows.Next() {
		var f models.ChatFileWithUploader
		if scanErr := rows.Scan(
			&f.ID, &f.MessageID, &f.ChannelID, &f.Filename, &f.MimeType, &f.FileSize,
			&f.StorageKey, &f.ThumbnailKey, &f.UploadedBy, &f.IsDeleted, &f.CreatedAt, &f.DeletedAt,
			&f.UploaderFirstName, &f.UploaderLastName,
		); scanErr != nil {
			return nil, scanErr
		}
		if f.MessageID != nil {
			result[*f.MessageID] = append(result[*f.MessageID], f)
		}
	}

	return result, rows.Err()
}

func (r *PostgresRepository) DeleteFile(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE chat_files SET is_deleted = TRUE, deleted_at = $1 WHERE id = $2`,
		now, id,
	)
	return err
}

// Channel checks

func (r *PostgresRepository) IsChannelMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM channel_memberships WHERE channel_id = $1 AND user_id = $2)`,
		channelID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) IsChannelArchived(ctx context.Context, channelID uuid.UUID) (bool, error) {
	var archived bool
	err := r.pool.QueryRow(ctx,
		`SELECT is_archived FROM channels WHERE id = $1`, channelID,
	).Scan(&archived)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrFileNotFound
	}
	return archived, err
}

func (r *PostgresRepository) GetChannelRole(ctx context.Context, channelID, userID uuid.UUID) (models.ChannelRole, error) {
	var role models.ChannelRole
	err := r.pool.QueryRow(ctx,
		`SELECT role FROM channel_memberships WHERE channel_id = $1 AND user_id = $2`,
		channelID, userID,
	).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotChannelMember
	}
	return role, err
}

// Quota

func (r *PostgresRepository) GetStorageQuota(ctx context.Context) (*models.StorageQuota, error) {
	var q models.StorageQuota
	err := r.pool.QueryRow(ctx,
		`SELECT id, max_bytes, used_bytes, updated_at FROM storage_quotas LIMIT 1`,
	).Scan(&q.ID, &q.MaxBytes, &q.UsedBytes, &q.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *PostgresRepository) IncrementUsedBytes(ctx context.Context, bytes int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE storage_quotas SET used_bytes = used_bytes + $1, updated_at = NOW()`, bytes,
	)
	return err
}

func (r *PostgresRepository) DecrementUsedBytes(ctx context.Context, bytes int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE storage_quotas SET used_bytes = GREATEST(used_bytes - $1, 0), updated_at = NOW()`, bytes,
	)
	return err
}

// User info

func (r *PostgresRepository) GetUserInfo(ctx context.Context, userID uuid.UUID) (firstName, lastName string, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT first_name, last_name FROM users WHERE id = $1`, userID,
	).Scan(&firstName, &lastName)
	return
}
