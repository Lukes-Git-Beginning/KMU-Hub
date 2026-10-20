package presence

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserDirectory answers which users belong to a tenant. Presence lives in
// Redis under presence:<userID> with no tenant component and no RLS beneath
// it, so the read paths filter their targets through the directory before
// they touch the store.
//
// The tenant travels as an explicit parameter for the same reason as in
// ConfigRepository: under a system context the users policy admits every
// row, and the filter must still hold.
type UserDirectory interface {
	VisibleUserIDs(ctx context.Context, tenantID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]struct{}, error)
}

// PostgresUserDirectory implements UserDirectory against the users table.
type PostgresUserDirectory struct {
	pool *pgxpool.Pool
}

// NewPostgresUserDirectory creates a new PostgreSQL user directory
func NewPostgresUserDirectory(pool *pgxpool.Pool) *PostgresUserDirectory {
	return &PostgresUserDirectory{pool: pool}
}

// VisibleUserIDs returns the subset of userIDs that belong to tenantID.
func (d *PostgresUserDirectory) VisibleUserIDs(ctx context.Context, tenantID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]struct{}, error) {
	visible := make(map[uuid.UUID]struct{}, len(userIDs))
	if len(userIDs) == 0 {
		return visible, nil
	}

	rows, err := d.pool.Query(ctx,
		`SELECT id FROM users WHERE tenant_id = $1 AND id = ANY($2)`,
		tenantID, userIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("resolve presence targets: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan presence target: %w", err)
		}
		visible[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate presence targets: %w", err)
	}
	return visible, nil
}
