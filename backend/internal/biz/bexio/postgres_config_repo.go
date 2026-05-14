package bexio

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresIntegrationConfigRepo implements IntegrationConfigRepo using PostgreSQL.
type PostgresIntegrationConfigRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresIntegrationConfigRepo creates a new PostgresIntegrationConfigRepo.
func NewPostgresIntegrationConfigRepo(pool *pgxpool.Pool) *PostgresIntegrationConfigRepo {
	return &PostgresIntegrationConfigRepo{pool: pool}
}

// GetByPlatform returns the integration config for a given platform.
// Tenant scoping is enforced via RLS (mig 000125): the row is only visible
// to callers running with current_tenant_id() matching the row's tenant_id.
func (r *PostgresIntegrationConfigRepo) GetByPlatform(ctx context.Context, platform string) (*IntegrationConfig, error) {
	query := `
		SELECT id, tenant_id, platform, is_active, credentials_vault_key, metadata, created_by, created_at, updated_at
		FROM integration_configs
		WHERE platform = $1`

	cfg := &IntegrationConfig{}
	var metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, platform).Scan(
		&cfg.ID, &cfg.TenantID, &cfg.Platform, &cfg.IsActive, &cfg.CredentialsVaultKey,
		&metadataJSON, &cfg.CreatedBy, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("bexio: integration config not found for platform %s", platform)
		}
		return nil, fmt.Errorf("bexio: get config by platform: %w", err)
	}

	if metadataJSON != nil {
		_ = json.Unmarshal(metadataJSON, &cfg.Metadata)
	}

	return cfg, nil
}

// Upsert creates or updates an integration config.
func (r *PostgresIntegrationConfigRepo) Upsert(ctx context.Context, config *IntegrationConfig) error {
	metadataJSON, err := json.Marshal(config.Metadata)
	if err != nil {
		return fmt.Errorf("bexio: marshal config metadata: %w", err)
	}

	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}

	now := time.Now()
	query := `
		INSERT INTO integration_configs (id, tenant_id, platform, is_active, credentials_vault_key, metadata, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (platform, tenant_id) DO UPDATE SET
			is_active = $4, credentials_vault_key = $5, metadata = $6, updated_at = $9`

	_, err = r.pool.Exec(ctx, query,
		config.ID, config.TenantID, config.Platform, config.IsActive, config.CredentialsVaultKey,
		metadataJSON, config.CreatedBy, now, now,
	)
	if err != nil {
		return fmt.Errorf("bexio: upsert config: %w", err)
	}

	return nil
}

// Deactivate marks an integration config as inactive.
func (r *PostgresIntegrationConfigRepo) Deactivate(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE integration_configs SET is_active = false, updated_at = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("bexio: deactivate config: %w", err)
	}
	return nil
}
