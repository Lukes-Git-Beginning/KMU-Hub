package datev

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
func (r *PostgresIntegrationConfigRepo) GetByPlatform(ctx context.Context, platform string) (*IntegrationConfig, error) {
	query := `
		SELECT id, platform, is_active, credentials_vault_key, metadata, created_by
		FROM integration_configs
		WHERE platform = $1`

	cfg := &IntegrationConfig{}
	var metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, platform).Scan(
		&cfg.ID, &cfg.Platform, &cfg.IsActive, &cfg.CredentialsVaultKey,
		&metadataJSON, &cfg.CreatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("datev: integration config not found for platform %s", platform)
		}
		return nil, fmt.Errorf("datev: get config by platform: %w", err)
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
		return fmt.Errorf("datev: marshal config metadata: %w", err)
	}

	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}

	now := time.Now()
	query := `
		INSERT INTO integration_configs (id, platform, is_active, credentials_vault_key, metadata, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (platform) DO UPDATE SET
			is_active = $3, credentials_vault_key = $4, metadata = $5, updated_at = $8`

	_, err = r.pool.Exec(ctx, query,
		config.ID, config.Platform, config.IsActive, config.CredentialsVaultKey,
		metadataJSON, config.CreatedBy, now, now,
	)
	if err != nil {
		return fmt.Errorf("datev: upsert config: %w", err)
	}

	return nil
}

// Deactivate marks an integration config as inactive.
func (r *PostgresIntegrationConfigRepo) Deactivate(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE integration_configs SET is_active = false, updated_at = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("datev: deactivate config: %w", err)
	}
	return nil
}
