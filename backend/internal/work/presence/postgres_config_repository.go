package presence

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConfigRepository defines the interface for presence configuration persistence
type ConfigRepository interface {
	GetConfig(ctx context.Context) (*PresenceConfig, error)
	UpdateConfig(ctx context.Context, awayTimeoutSeconds int, updatedBy uuid.UUID) error
}

// PostgresConfigRepository implements ConfigRepository using PostgreSQL
type PostgresConfigRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresConfigRepository creates a new PostgreSQL config repository
func NewPostgresConfigRepository(pool *pgxpool.Pool) *PostgresConfigRepository {
	return &PostgresConfigRepository{pool: pool}
}

func (r *PostgresConfigRepository) GetConfig(ctx context.Context) (*PresenceConfig, error) {
	var cfg PresenceConfig
	err := r.pool.QueryRow(ctx,
		`SELECT id, away_timeout_seconds, updated_at, updated_by
		 FROM presence_config LIMIT 1`,
	).Scan(&cfg.ID, &cfg.AwayTimeoutSeconds, &cfg.UpdatedAt, &cfg.UpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("get presence config: %w", err)
	}
	return &cfg, nil
}

func (r *PostgresConfigRepository) UpdateConfig(ctx context.Context, awayTimeoutSeconds int, updatedBy uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE presence_config SET away_timeout_seconds=$1, updated_by=$2, updated_at=NOW()`,
		awayTimeoutSeconds, updatedBy,
	)
	if err != nil {
		return fmt.Errorf("update presence config: %w", err)
	}
	return nil
}
