package integration

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL integration repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// ============================================================================
// Config CRUD
// ============================================================================

func (r *PostgresRepository) CreateConfig(ctx context.Context, cfg *IntegrationConfig) error {
	query := `
		INSERT INTO integration_configs (id, tenant_id, platform, is_active, credentials_vault_key, metadata, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.pool.Exec(ctx, query,
		cfg.ID, cfg.TenantID, cfg.Platform, cfg.IsActive, cfg.CredentialsVaultKey,
		cfg.Metadata, cfg.CreatedBy, cfg.CreatedAt, cfg.UpdatedAt,
	)
	return err
}

// GetConfigByPlatform — tenant scoping is enforced via RLS (mig 000125).
func (r *PostgresRepository) GetConfigByPlatform(ctx context.Context, platform string) (*IntegrationConfig, error) {
	query := `
		SELECT id, tenant_id, platform, is_active, credentials_vault_key, metadata, created_by, created_at, updated_at
		FROM integration_configs
		WHERE platform = $1`

	cfg := &IntegrationConfig{}
	err := r.pool.QueryRow(ctx, query, platform).Scan(
		&cfg.ID, &cfg.TenantID, &cfg.Platform, &cfg.IsActive, &cfg.CredentialsVaultKey,
		&cfg.Metadata, &cfg.CreatedBy, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrConfigNotFound
	}
	return cfg, err
}

func (r *PostgresRepository) UpdateConfig(ctx context.Context, cfg *IntegrationConfig) error {
	query := `
		UPDATE integration_configs
		SET is_active = $2, credentials_vault_key = $3, metadata = $4, updated_at = $5
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query,
		cfg.ID, cfg.IsActive, cfg.CredentialsVaultKey, cfg.Metadata, cfg.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrConfigNotFound
	}
	return nil
}

// ListConfigs — tenant scoping is enforced via RLS (mig 000125).
func (r *PostgresRepository) ListConfigs(ctx context.Context) ([]*IntegrationConfig, error) {
	query := `
		SELECT id, tenant_id, platform, is_active, credentials_vault_key, metadata, created_by, created_at, updated_at
		FROM integration_configs
		ORDER BY platform ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*IntegrationConfig
	for rows.Next() {
		cfg := &IntegrationConfig{}
		err := rows.Scan(
			&cfg.ID, &cfg.TenantID, &cfg.Platform, &cfg.IsActive, &cfg.CredentialsVaultKey,
			&cfg.Metadata, &cfg.CreatedBy, &cfg.CreatedAt, &cfg.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}
	return configs, rows.Err()
}

func (r *PostgresRepository) DeleteConfig(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM integration_configs WHERE id = $1",
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrConfigNotFound
	}
	return nil
}

// ============================================================================
// Channel Mapping CRUD
// ============================================================================

func (r *PostgresRepository) CreateMapping(ctx context.Context, m *ChannelMapping) error {
	modulesJSON, err := json.Marshal(m.Modules)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO integration_channel_mappings (id, config_id, channel_id, channel_name, modules, is_active, platform_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = r.pool.Exec(ctx, query,
		m.ID, m.ConfigID, m.ChannelID, m.ChannelName,
		modulesJSON, m.IsActive, m.PlatformData, m.CreatedAt, m.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetMapping(ctx context.Context, id uuid.UUID) (*ChannelMapping, error) {
	query := `
		SELECT id, config_id, channel_id, channel_name, modules, is_active, platform_data, created_at, updated_at
		FROM integration_channel_mappings
		WHERE id = $1`

	m := &ChannelMapping{}
	var modulesJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.ConfigID, &m.ChannelID, &m.ChannelName,
		&modulesJSON, &m.IsActive, &m.PlatformData, &m.CreatedAt, &m.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrMappingNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(modulesJSON, &m.Modules); err != nil {
		return nil, err
	}
	return m, nil
}

func (r *PostgresRepository) ListMappingsByConfig(ctx context.Context, configID uuid.UUID) ([]*ChannelMapping, error) {
	query := `
		SELECT id, config_id, channel_id, channel_name, modules, is_active, platform_data, created_at, updated_at
		FROM integration_channel_mappings
		WHERE config_id = $1
		ORDER BY channel_name ASC`

	rows, err := r.pool.Query(ctx, query, configID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMappings(rows)
}

func (r *PostgresRepository) ListActiveMappingsForModule(ctx context.Context, moduleID string) ([]*ChannelMapping, error) {
	moduleFilter, err := json.Marshal([]string{moduleID})
	if err != nil {
		return nil, err
	}

	query := `
		SELECT m.id, m.config_id, m.channel_id, m.channel_name, m.modules, m.is_active, m.platform_data, m.created_at, m.updated_at
		FROM integration_channel_mappings m
		JOIN integration_configs c ON c.id = m.config_id
		WHERE m.is_active = true
		  AND c.is_active = true
		  AND m.modules @> $1::jsonb`

	rows, err := r.pool.Query(ctx, query, moduleFilter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMappings(rows)
}

func (r *PostgresRepository) UpdateMapping(ctx context.Context, m *ChannelMapping) error {
	modulesJSON, err := json.Marshal(m.Modules)
	if err != nil {
		return err
	}

	query := `
		UPDATE integration_channel_mappings
		SET channel_id = $2, channel_name = $3, modules = $4, is_active = $5, platform_data = $6, updated_at = $7
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query,
		m.ID, m.ChannelID, m.ChannelName, modulesJSON, m.IsActive, m.PlatformData, m.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMappingNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteMapping(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM integration_channel_mappings WHERE id = $1",
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMappingNotFound
	}
	return nil
}

// scanMappings scans rows into ChannelMapping slices.
func scanMappings(rows pgx.Rows) ([]*ChannelMapping, error) {
	var mappings []*ChannelMapping
	for rows.Next() {
		m := &ChannelMapping{}
		var modulesJSON []byte
		err := rows.Scan(
			&m.ID, &m.ConfigID, &m.ChannelID, &m.ChannelName,
			&modulesJSON, &m.IsActive, &m.PlatformData, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(modulesJSON, &m.Modules); err != nil {
			return nil, err
		}
		mappings = append(mappings, m)
	}
	return mappings, rows.Err()
}

// ============================================================================
// Account Links
// ============================================================================

func (r *PostgresRepository) CreateAccountLink(ctx context.Context, link *AccountLink) error {
	query := `
		INSERT INTO integration_account_links (id, platform, external_user_id, kmuhub_user_id, external_display_name, linked_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (platform, external_user_id) DO UPDATE
		SET kmuhub_user_id = EXCLUDED.kmuhub_user_id,
		    external_display_name = EXCLUDED.external_display_name,
		    linked_at = EXCLUDED.linked_at`

	_, err := r.pool.Exec(ctx, query,
		link.ID, link.Platform, link.ExternalUserID,
		link.KMUHubUserID, link.ExternalDisplayName, link.LinkedAt,
	)
	return err
}

func (r *PostgresRepository) GetAccountLink(ctx context.Context, platform, externalUserID string) (*AccountLink, error) {
	query := `
		SELECT id, platform, external_user_id, kmuhub_user_id, external_display_name, linked_at
		FROM integration_account_links
		WHERE platform = $1 AND external_user_id = $2`

	link := &AccountLink{}
	err := r.pool.QueryRow(ctx, query, platform, externalUserID).Scan(
		&link.ID, &link.Platform, &link.ExternalUserID,
		&link.KMUHubUserID, &link.ExternalDisplayName, &link.LinkedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrAccountLinkNotFound
	}
	return link, err
}

func (r *PostgresRepository) GetAccountLinkByKMUHubUser(ctx context.Context, platform string, kmuhubUserID uuid.UUID) (*AccountLink, error) {
	query := `
		SELECT id, platform, external_user_id, kmuhub_user_id, external_display_name, linked_at
		FROM integration_account_links
		WHERE platform = $1 AND kmuhub_user_id = $2`

	link := &AccountLink{}
	err := r.pool.QueryRow(ctx, query, platform, kmuhubUserID).Scan(
		&link.ID, &link.Platform, &link.ExternalUserID,
		&link.KMUHubUserID, &link.ExternalDisplayName, &link.LinkedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrAccountLinkNotFound
	}
	return link, err
}

func (r *PostgresRepository) DeleteAccountLink(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM integration_account_links WHERE id = $1",
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountLinkNotFound
	}
	return nil
}

// ============================================================================
// Link Tokens
// ============================================================================

func (r *PostgresRepository) CreateLinkToken(ctx context.Context, token *LinkToken) error {
	query := `
		INSERT INTO integration_link_tokens (id, platform, external_user_id, token_hash, expires_at, used, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query,
		token.ID, token.Platform, token.ExternalUserID,
		token.TokenHash, token.ExpiresAt, token.Used, token.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetLinkTokenByHash(ctx context.Context, tokenHash string) (*LinkToken, error) {
	query := `
		SELECT id, platform, external_user_id, token_hash, expires_at, used, created_at
		FROM integration_link_tokens
		WHERE token_hash = $1 AND NOT used`

	token := &LinkToken{}
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID, &token.Platform, &token.ExternalUserID,
		&token.TokenHash, &token.ExpiresAt, &token.Used, &token.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrLinkTokenNotFound
	}
	return token, err
}

func (r *PostgresRepository) MarkLinkTokenUsed(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE integration_link_tokens SET used = true WHERE id = $1 AND NOT used",
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrLinkTokenNotFound
	}
	return nil
}

func (r *PostgresRepository) CleanupExpiredTokens(ctx context.Context) (int, error) {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM integration_link_tokens WHERE expires_at < NOW() AND NOT used",
	)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ============================================================================
// Delivery Log
// ============================================================================

func (r *PostgresRepository) LogDelivery(ctx context.Context, entry *DeliveryLogEntry) error {
	query := `
		INSERT INTO integration_delivery_log (id, notification_id, mapping_id, platform, status, platform_message_id, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, query,
		entry.ID, entry.NotificationID, entry.MappingID, entry.Platform,
		entry.Status, entry.PlatformMessageID, entry.ErrorMessage, entry.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetRecentFailures(ctx context.Context, mappingID uuid.UUID, limit int) ([]*DeliveryLogEntry, error) {
	query := `
		SELECT id, notification_id, mapping_id, platform, status, platform_message_id, error_message, created_at
		FROM integration_delivery_log
		WHERE mapping_id = $1 AND status != 'sent'
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, mappingID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*DeliveryLogEntry
	for rows.Next() {
		e := &DeliveryLogEntry{}
		err := rows.Scan(
			&e.ID, &e.NotificationID, &e.MappingID, &e.Platform,
			&e.Status, &e.PlatformMessageID, &e.ErrorMessage, &e.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *PostgresRepository) CleanupOldLogs(ctx context.Context, olderThan time.Time) (int, error) {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM integration_delivery_log WHERE created_at < $1",
		olderThan,
	)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}
