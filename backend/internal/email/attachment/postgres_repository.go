package attachment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using pgxpool.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, att *models.EmailAttachment) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_attachments (id, message_id, filename, content_type,
			size_bytes, minio_key, content_id, is_inline, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		att.ID, att.MessageID, att.Filename, att.ContentType,
		att.SizeBytes, att.MinIOKey, att.ContentID, att.IsInline, att.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert attachment: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByMessage(ctx context.Context, messageID uuid.UUID) ([]*models.EmailAttachment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, message_id, filename, content_type, size_bytes, minio_key,
			content_id, is_inline, created_at
		 FROM email_attachments WHERE message_id = $1
		 ORDER BY created_at ASC`, messageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []*models.EmailAttachment
	for rows.Next() {
		var att models.EmailAttachment
		if scanErr := rows.Scan(&att.ID, &att.MessageID, &att.Filename, &att.ContentType,
			&att.SizeBytes, &att.MinIOKey, &att.ContentID, &att.IsInline,
			&att.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		attachments = append(attachments, &att)
	}
	return attachments, rows.Err()
}

func (r *PostgresRepository) GetMinIOKeyByID(ctx context.Context, id uuid.UUID) (string, error) {
	var minioKey string
	err := r.pool.QueryRow(ctx,
		`SELECT minio_key FROM email_attachments WHERE id = $1`, id,
	).Scan(&minioKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrAttachmentNotFound
		}
		return "", err
	}
	return minioKey, nil
}
