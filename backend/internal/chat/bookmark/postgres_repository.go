package bookmark

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL bookmark repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Exists(ctx context.Context, tenantID, userID, messageID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM message_bookmarks WHERE tenant_id = $1 AND user_id = $2 AND message_id = $3)`,
		tenantID, userID, messageID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) Add(ctx context.Context, tenantID, userID, messageID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO message_bookmarks (tenant_id, user_id, message_id) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, message_id) DO NOTHING`,
		tenantID, userID, messageID,
	)
	return err
}

func (r *PostgresRepository) Remove(ctx context.Context, tenantID, userID, messageID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM message_bookmarks WHERE tenant_id = $1 AND user_id = $2 AND message_id = $3`,
		tenantID, userID, messageID,
	)
	return err
}

func (r *PostgresRepository) ListMessageIDs(ctx context.Context, tenantID, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT message_id FROM message_bookmarks WHERE tenant_id = $1 AND user_id = $2 ORDER BY created_at DESC`,
		tenantID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
