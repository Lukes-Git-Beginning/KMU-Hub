package presence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConfigRepository defines the interface for presence configuration persistence.
// The tenant travels as an explicit parameter rather than relying on the RLS
// session GUC alone: a caller reaching these queries under a system context
// would otherwise read or overwrite an arbitrary tenant's row, because the
// tenant_isolation policy admits every row for app.role='system'.
type ConfigRepository interface {
	GetConfig(ctx context.Context, tenantID uuid.UUID) (*PresenceConfig, error)
	UpdateConfig(ctx context.Context, tenantID uuid.UUID, awayTimeoutSeconds int, updatedBy uuid.UUID) error
}

// PostgresConfigRepository implements ConfigRepository using PostgreSQL
type PostgresConfigRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresConfigRepository creates a new PostgreSQL config repository
func NewPostgresConfigRepository(pool *pgxpool.Pool) *PostgresConfigRepository {
	return &PostgresConfigRepository{pool: pool}
}

// GetConfig returns the tenant's presence configuration, or ErrConfigNotFound
// when the tenant has never written one. Migration 000274 makes tenant_id
// unique, so there is at most one row per tenant.
func (r *PostgresConfigRepository) GetConfig(ctx context.Context, tenantID uuid.UUID) (*PresenceConfig, error) {
	var cfg PresenceConfig
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, away_timeout_seconds, updated_at, updated_by
		 FROM presence_config WHERE tenant_id = $1`, tenantID,
	).Scan(&cfg.ID, &cfg.TenantID, &cfg.AwayTimeoutSeconds, &cfg.UpdatedAt, &cfg.UpdatedBy)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("get presence config: %w", err)
	}
	return &cfg, nil
}

// UpdateConfig writes the tenant's presence configuration. It upserts rather
// than updates: tenants created after 000274 have no row until their first
// write, and an UPDATE would have reported success while changing nothing.
func (r *PostgresConfigRepository) UpdateConfig(ctx context.Context, tenantID uuid.UUID, awayTimeoutSeconds int, updatedBy uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO presence_config (tenant_id, away_timeout_seconds, updated_by, updated_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (tenant_id) DO UPDATE
		 SET away_timeout_seconds = EXCLUDED.away_timeout_seconds,
		     updated_by = EXCLUDED.updated_by,
		     updated_at = NOW()`,
		tenantID, awayTimeoutSeconds, updatedBy,
	)
	if err != nil {
		return fmt.Errorf("update presence config: %w", err)
	}
	return nil
}
