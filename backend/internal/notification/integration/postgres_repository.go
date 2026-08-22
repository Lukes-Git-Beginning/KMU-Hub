package integration

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/middleware"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL integration repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// tenantForWrite resolves the tenant every INSERT in this package has to carry.
//
// Reads rely on RLS alone — the policy filters them and an empty result is the
// safe outcome. Writes cannot: integration_channel_mappings,
// integration_account_links, integration_link_tokens and
// integration_delivery_log all gained tenant_id NOT NULL in migration 000115
// and FORCE row level security in 000122, so an INSERT that leaves the column
// out fails at the constraint. Resolving up front turns that into a typed
// refusal before the pool is touched.
func tenantForWrite(ctx context.Context) (uuid.UUID, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return uuid.Nil, ErrTenantMissing
	}
	return tenantID, nil
}

// ResolveTenant maps a platform workspace identity to the tenant that
// configured it, for the unauthenticated inbound webhook paths.
//
// The lookup deliberately runs in the system context: the request carries no
// tenant yet — that is the very question — so RLS would filter away the single
// row that answers it. Everything the handler does afterwards runs under the
// resolved tenant and is filtered normally again.
//
// Two configs claiming the same workspace are refused rather than tie-broken,
// and the schema can produce that case today: migration 000125 replaced the
// pre-tenancy UNIQUE(platform) from 000053 with UNIQUE(platform, tenant_id), so
// two tenants may each register the same workspace id. Refusing is the safe
// half of that — the alternative is a cross-tenant leak. The unsafe half stays:
// a tenant who enters another's workspace id makes both webhooks unresolvable.
// Uniqueness on the workspace id itself is what would close it, and that needs
// the metadata key promoted out of JSONB first.
func (r *PostgresRepository) ResolveTenant(ctx context.Context, platform, workspaceID string) (uuid.UUID, error) {
	if workspaceID == "" {
		return uuid.Nil, ErrTenantUnresolved
	}
	metadataKey := WorkspaceMetadataKey(platform)
	if metadataKey == "" {
		return uuid.Nil, ErrInvalidPlatform
	}

	query := `
		SELECT tenant_id
		FROM integration_configs
		WHERE platform = $1
		  AND is_active = true
		  AND tenant_id IS NOT NULL
		  AND metadata->>$2 = $3
		LIMIT 2`

	rows, err := r.pool.Query(database.WithSystemContext(ctx), query, platform, metadataKey, workspaceID)
	if err != nil {
		return uuid.Nil, err
	}
	defer rows.Close()

	var found []uuid.UUID
	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			return uuid.Nil, err
		}
		found = append(found, tenantID)
	}
	if err := rows.Err(); err != nil {
		return uuid.Nil, err
	}

	switch len(found) {
	case 0:
		return uuid.Nil, ErrTenantUnresolved
	case 1:
		return found[0], nil
	default:
		return uuid.Nil, ErrTenantAmbiguous
	}
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
	tenantID, err := tenantForWrite(ctx)
	if err != nil {
		return err
	}

	modulesJSON, err := json.Marshal(m.Modules)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO integration_channel_mappings (id, tenant_id, config_id, channel_id, channel_name, modules, is_active, platform_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err = r.pool.Exec(ctx, query,
		m.ID, tenantID, m.ConfigID, m.ChannelID, m.ChannelName,
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

func (r *PostgresRepository) HasDeliveryLogs(ctx context.Context, mappingID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM integration_delivery_log WHERE mapping_id = $1)",
		mappingID,
	).Scan(&exists)
	return exists, err
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

// CreateAccountLink upserts the mapping between an external platform user and a
// KMU Hub user. The unique key (platform, external_user_id) is global, so a
// conflicting row from another tenant is invisible to the DO UPDATE and RLS
// rejects the write — one Slack account cannot be re-pointed across tenants.
func (r *PostgresRepository) CreateAccountLink(ctx context.Context, link *AccountLink) error {
	tenantID, err := tenantForWrite(ctx)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO integration_account_links (id, tenant_id, platform, external_user_id, kmuhub_user_id, external_display_name, linked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (platform, external_user_id) DO UPDATE
		SET kmuhub_user_id = EXCLUDED.kmuhub_user_id,
		    external_display_name = EXCLUDED.external_display_name,
		    linked_at = EXCLUDED.linked_at`

	_, err = r.pool.Exec(ctx, query,
		link.ID, tenantID, link.Platform, link.ExternalUserID,
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

// CreateLinkToken stores the hash of a short-lived account linking token.
//
// The only caller today is the inbound webhook path (/kmuhub link), which is
// unauthenticated and therefore has no tenant — those routes stay refused until
// the webhook adapters resolve the tenant from the platform identity. Until
// then this returns ErrTenantMissing rather than inventing one.
func (r *PostgresRepository) CreateLinkToken(ctx context.Context, token *LinkToken) error {
	tenantID, err := tenantForWrite(ctx)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO integration_link_tokens (id, tenant_id, platform, external_user_id, token_hash, expires_at, used, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = r.pool.Exec(ctx, query,
		token.ID, tenantID, token.Platform, token.ExternalUserID,
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
	tenantID, err := tenantForWrite(ctx)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO integration_delivery_log (id, tenant_id, notification_id, mapping_id, platform, status, platform_message_id, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = r.pool.Exec(ctx, query,
		entry.ID, tenantID, entry.NotificationID, entry.MappingID, entry.Platform,
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
