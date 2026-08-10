package integration

// DB-backed tests for the read/list/update/delete/cleanup paths of
// PostgresRepository — previously only the four Create* methods were exercised
// against real SQL (tenant_write_test.go). Each test mints its own tenant(s)
// rather than sharing testutil.TenantA/TenantB: integration_configs is
// UNIQUE(platform, tenant_id) since migration 000125, and tests running in
// parallel against a shared tenant race on that constraint (see
// tenant_write_test.go's comment on the same issue).

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedIntegrationUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("integ-repo-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
	return userID
}

func TestPostgresRepository_ConfigCRUD(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// Registered before any row-cleanup t.Cleanup calls below: Cleanup funcs run
	// LIFO, and a plain `defer pool.Close()` here would fire when the test
	// function returns — before t.Cleanup-registered row deletions, which run
	// after. Closing the pool first made every row cleanup silently fail
	// ("closed pool"), leaving cross-test fixtures behind.
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Config CRUD Tenant A")
	testutil.EnsureTenant(t, pool, tenantOther, "Config CRUD Tenant Other")
	userA := seedIntegrationUser(t, pool, tenantA)
	userOther := seedIntegrationUser(t, pool, tenantOther)

	slackA := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenantA,
		"platform":              PlatformSlack,
		"credentials_vault_key": "slack/" + uuid.New().String(),
		"created_by":            userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_configs", slackA) })

	teamsA := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenantA,
		"platform":              PlatformTeams,
		"credentials_vault_key": "teams/" + uuid.New().String(),
		"created_by":            userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_configs", teamsA) })

	slackOther := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenantOther,
		"platform":              PlatformSlack,
		"credentials_vault_key": "slack/" + uuid.New().String(),
		"created_by":            userOther,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_configs", slackOther) })

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	t.Run("GetConfigByPlatform is tenant-scoped", func(t *testing.T) {
		cfg, err := repo.GetConfigByPlatform(ctxA, PlatformSlack)
		if err != nil {
			t.Fatalf("GetConfigByPlatform(A, slack): %v", err)
		}
		if cfg.ID != slackA {
			t.Fatalf("GetConfigByPlatform(A, slack): got id %s, want %s", cfg.ID, slackA)
		}

		cfgOther, err := repo.GetConfigByPlatform(ctxOther, PlatformSlack)
		if err != nil {
			t.Fatalf("GetConfigByPlatform(Other, slack): %v", err)
		}
		if cfgOther.ID != slackOther {
			t.Fatalf("GetConfigByPlatform(Other, slack): got id %s, want %s", cfgOther.ID, slackOther)
		}
	})

	t.Run("GetConfigByPlatform not found", func(t *testing.T) {
		// tenantOther has no teams config.
		if _, err := repo.GetConfigByPlatform(ctxOther, PlatformTeams); err != ErrConfigNotFound {
			t.Fatalf("want ErrConfigNotFound, got %v", err)
		}
	})

	t.Run("ListConfigs is tenant-scoped and ordered by platform", func(t *testing.T) {
		configs, err := repo.ListConfigs(ctxA)
		if err != nil {
			t.Fatalf("ListConfigs(A): %v", err)
		}
		if len(configs) != 2 {
			t.Fatalf("ListConfigs(A): got %d configs, want 2", len(configs))
		}
		if configs[0].Platform != PlatformSlack || configs[1].Platform != PlatformTeams {
			t.Fatalf("ListConfigs(A): want [slack, teams] order, got [%s, %s]", configs[0].Platform, configs[1].Platform)
		}

		configsOther, err := repo.ListConfigs(ctxOther)
		if err != nil {
			t.Fatalf("ListConfigs(Other): %v", err)
		}
		if len(configsOther) != 1 || configsOther[0].ID != slackOther {
			t.Fatalf("ListConfigs(Other): got %d configs, want exactly the other tenant's own slack config", len(configsOther))
		}
	})

	t.Run("UpdateConfig", func(t *testing.T) {
		now := time.Now()
		err := repo.UpdateConfig(ctxA, &IntegrationConfig{
			ID:                  teamsA,
			IsActive:            true,
			CredentialsVaultKey: "teams/rotated",
			Metadata:            []byte(`{"tenant_id":"AZURE-ROTATED"}`),
			UpdatedAt:           now,
		})
		if err != nil {
			t.Fatalf("UpdateConfig: %v", err)
		}

		cfg, err := repo.GetConfigByPlatform(ctxA, PlatformTeams)
		if err != nil {
			t.Fatalf("GetConfigByPlatform after update: %v", err)
		}
		if !cfg.IsActive || cfg.CredentialsVaultKey != "teams/rotated" {
			t.Fatalf("UpdateConfig did not persist: is_active=%v credentials_vault_key=%s", cfg.IsActive, cfg.CredentialsVaultKey)
		}

		if err := repo.UpdateConfig(ctxA, &IntegrationConfig{ID: uuid.New(), UpdatedAt: now}); err != ErrConfigNotFound {
			t.Fatalf("UpdateConfig unknown id: want ErrConfigNotFound, got %v", err)
		}
	})

	t.Run("DeleteConfig", func(t *testing.T) {
		if err := repo.DeleteConfig(ctxA, teamsA); err != nil {
			t.Fatalf("DeleteConfig: %v", err)
		}
		if _, err := repo.GetConfigByPlatform(ctxA, PlatformTeams); err != ErrConfigNotFound {
			t.Fatalf("GetConfigByPlatform after delete: want ErrConfigNotFound, got %v", err)
		}
		if err := repo.DeleteConfig(ctxA, teamsA); err != ErrConfigNotFound {
			t.Fatalf("DeleteConfig again: want ErrConfigNotFound, got %v", err)
		}
	})
}

func TestPostgresRepository_MappingCRUD(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// Registered before any row-cleanup t.Cleanup calls below: Cleanup funcs run
	// LIFO, and a plain `defer pool.Close()` here would fire when the test
	// function returns — before t.Cleanup-registered row deletions, which run
	// after. Closing the pool first made every row cleanup silently fail
	// ("closed pool"), leaving cross-test fixtures behind.
	t.Cleanup(func() { pool.Close() })

	tenantA := uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Mapping CRUD Tenant A")
	userA := seedIntegrationUser(t, pool, tenantA)

	activeConfig := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenantA,
		"platform":              PlatformSlack,
		"is_active":             true,
		"credentials_vault_key": "slack/" + uuid.New().String(),
		"created_by":            userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_configs", activeConfig) })

	inactiveConfig := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenantA,
		"platform":              PlatformTeams,
		"is_active":             false,
		"credentials_vault_key": "teams/" + uuid.New().String(),
		"created_by":            userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_configs", inactiveConfig) })

	mappingCRM := testutil.SeedRow(t, pool, "integration_channel_mappings", map[string]any{
		"tenant_id":    tenantA,
		"config_id":    activeConfig,
		"channel_id":   "C" + uuid.New().String()[:8],
		"channel_name": "aa-crm-channel",
		"modules":      `["crm"]`,
		"is_active":    true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_channel_mappings", mappingCRM) })

	mappingHR := testutil.SeedRow(t, pool, "integration_channel_mappings", map[string]any{
		"tenant_id":    tenantA,
		"config_id":    activeConfig,
		"channel_id":   "C" + uuid.New().String()[:8],
		"channel_name": "bb-hr-channel",
		"modules":      `["hr"]`,
		"is_active":    false,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_channel_mappings", mappingHR) })

	// Active mapping whose config is inactive — must never surface from
	// ListActiveMappingsForModule, which joins on c.is_active = true.
	mappingOnInactiveConfig := testutil.SeedRow(t, pool, "integration_channel_mappings", map[string]any{
		"tenant_id":    tenantA,
		"config_id":    inactiveConfig,
		"channel_id":   "C" + uuid.New().String()[:8],
		"channel_name": "cc-inactive-config-channel",
		"modules":      `["crm"]`,
		"is_active":    true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_channel_mappings", mappingOnInactiveConfig) })

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)

	t.Run("GetMapping", func(t *testing.T) {
		m, err := repo.GetMapping(ctxA, mappingCRM)
		if err != nil {
			t.Fatalf("GetMapping: %v", err)
		}
		if len(m.Modules) != 1 || m.Modules[0] != "crm" {
			t.Fatalf("GetMapping: modules = %v, want [crm]", m.Modules)
		}

		if _, err := repo.GetMapping(ctxA, uuid.New()); err != ErrMappingNotFound {
			t.Fatalf("GetMapping unknown id: want ErrMappingNotFound, got %v", err)
		}
	})

	t.Run("ListMappingsByConfig ordered by channel_name", func(t *testing.T) {
		mappings, err := repo.ListMappingsByConfig(ctxA, activeConfig)
		if err != nil {
			t.Fatalf("ListMappingsByConfig: %v", err)
		}
		if len(mappings) != 2 || mappings[0].ID != mappingCRM || mappings[1].ID != mappingHR {
			t.Fatalf("ListMappingsByConfig: want [aa-crm, bb-hr] order, got %d entries", len(mappings))
		}
	})

	t.Run("ListActiveMappingsForModule requires both mapping and config active", func(t *testing.T) {
		crm, err := repo.ListActiveMappingsForModule(ctxA, "crm")
		if err != nil {
			t.Fatalf("ListActiveMappingsForModule(crm): %v", err)
		}
		if len(crm) != 1 || crm[0].ID != mappingCRM {
			t.Fatalf("ListActiveMappingsForModule(crm): want exactly [mappingCRM], got %d entries", len(crm))
		}

		hr, err := repo.ListActiveMappingsForModule(ctxA, "hr")
		if err != nil {
			t.Fatalf("ListActiveMappingsForModule(hr): %v", err)
		}
		if len(hr) != 0 {
			t.Fatalf("ListActiveMappingsForModule(hr): want 0 (mapping is inactive), got %d", len(hr))
		}
	})

	t.Run("UpdateMapping", func(t *testing.T) {
		now := time.Now()
		err := repo.UpdateMapping(ctxA, &ChannelMapping{
			ID:           mappingHR,
			ChannelID:    "C-renamed",
			ChannelName:  "bb-hr-renamed",
			Modules:      []string{"hr", "crm"},
			IsActive:     true,
			PlatformData: []byte(`{"renamed":true}`),
			UpdatedAt:    now,
		})
		if err != nil {
			t.Fatalf("UpdateMapping: %v", err)
		}

		m, err := repo.GetMapping(ctxA, mappingHR)
		if err != nil {
			t.Fatalf("GetMapping after update: %v", err)
		}
		if m.ChannelName != "bb-hr-renamed" || !m.IsActive || len(m.Modules) != 2 {
			t.Fatalf("UpdateMapping did not persist: %+v", m)
		}

		if err := repo.UpdateMapping(ctxA, &ChannelMapping{ID: uuid.New(), UpdatedAt: now}); err != ErrMappingNotFound {
			t.Fatalf("UpdateMapping unknown id: want ErrMappingNotFound, got %v", err)
		}
	})

	t.Run("DeleteMapping", func(t *testing.T) {
		if err := repo.DeleteMapping(ctxA, mappingOnInactiveConfig); err != nil {
			t.Fatalf("DeleteMapping: %v", err)
		}
		if _, err := repo.GetMapping(ctxA, mappingOnInactiveConfig); err != ErrMappingNotFound {
			t.Fatalf("GetMapping after delete: want ErrMappingNotFound, got %v", err)
		}
		if err := repo.DeleteMapping(ctxA, mappingOnInactiveConfig); err != ErrMappingNotFound {
			t.Fatalf("DeleteMapping again: want ErrMappingNotFound, got %v", err)
		}
	})
}

func TestPostgresRepository_AccountLinks(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// Registered before any row-cleanup t.Cleanup calls below: Cleanup funcs run
	// LIFO, and a plain `defer pool.Close()` here would fire when the test
	// function returns — before t.Cleanup-registered row deletions, which run
	// after. Closing the pool first made every row cleanup silently fail
	// ("closed pool"), leaving cross-test fixtures behind.
	t.Cleanup(func() { pool.Close() })

	tenantA := uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Account Links Tenant A")
	userLinked := seedIntegrationUser(t, pool, tenantA)
	userUnlinked := seedIntegrationUser(t, pool, tenantA)

	extUserID := uuid.New().String()[:20]
	linkID := testutil.SeedRow(t, pool, "integration_account_links", map[string]any{
		"tenant_id":             tenantA,
		"platform":              PlatformSlack,
		"external_user_id":      extUserID,
		"kmuhub_user_id":        userLinked,
		"external_display_name": "",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_account_links", linkID) })

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)

	t.Run("GetAccountLink", func(t *testing.T) {
		link, err := repo.GetAccountLink(ctxA, PlatformSlack, extUserID)
		if err != nil {
			t.Fatalf("GetAccountLink: %v", err)
		}
		if link.KMUHubUserID != userLinked {
			t.Fatalf("GetAccountLink: kmuhub_user_id = %s, want %s", link.KMUHubUserID, userLinked)
		}

		if _, err := repo.GetAccountLink(ctxA, PlatformSlack, "unknown-ext-id"); err != ErrAccountLinkNotFound {
			t.Fatalf("GetAccountLink unknown: want ErrAccountLinkNotFound, got %v", err)
		}
	})

	t.Run("GetAccountLinkByKMUHubUser", func(t *testing.T) {
		link, err := repo.GetAccountLinkByKMUHubUser(ctxA, PlatformSlack, userLinked)
		if err != nil {
			t.Fatalf("GetAccountLinkByKMUHubUser: %v", err)
		}
		if link.ID != linkID {
			t.Fatalf("GetAccountLinkByKMUHubUser: id = %s, want %s", link.ID, linkID)
		}

		if _, err := repo.GetAccountLinkByKMUHubUser(ctxA, PlatformSlack, userUnlinked); err != ErrAccountLinkNotFound {
			t.Fatalf("GetAccountLinkByKMUHubUser unlinked user: want ErrAccountLinkNotFound, got %v", err)
		}
	})

	t.Run("DeleteAccountLink", func(t *testing.T) {
		if err := repo.DeleteAccountLink(ctxA, linkID); err != nil {
			t.Fatalf("DeleteAccountLink: %v", err)
		}
		if _, err := repo.GetAccountLink(ctxA, PlatformSlack, extUserID); err != ErrAccountLinkNotFound {
			t.Fatalf("GetAccountLink after delete: want ErrAccountLinkNotFound, got %v", err)
		}
		if err := repo.DeleteAccountLink(ctxA, linkID); err != ErrAccountLinkNotFound {
			t.Fatalf("DeleteAccountLink again: want ErrAccountLinkNotFound, got %v", err)
		}
	})
}

func TestPostgresRepository_LinkTokens(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// Registered before any row-cleanup t.Cleanup calls below: Cleanup funcs run
	// LIFO, and a plain `defer pool.Close()` here would fire when the test
	// function returns — before t.Cleanup-registered row deletions, which run
	// after. Closing the pool first made every row cleanup silently fail
	// ("closed pool"), leaving cross-test fixtures behind.
	t.Cleanup(func() { pool.Close() })

	tenantA, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Link Tokens Tenant A")
	testutil.EnsureTenant(t, pool, tenantOther, "Link Tokens Tenant Other")

	now := time.Now()
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	validUnused := testutil.SeedRow(t, pool, "integration_link_tokens", map[string]any{
		"tenant_id":        tenantA,
		"platform":         PlatformSlack,
		"external_user_id": "U-valid",
		"token_hash":       "hash-valid-unused-" + uuid.New().String(),
		"expires_at":       future,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_link_tokens", validUnused) })

	expiredUnused := testutil.SeedRow(t, pool, "integration_link_tokens", map[string]any{
		"tenant_id":        tenantA,
		"platform":         PlatformSlack,
		"external_user_id": "U-expired",
		"token_hash":       "hash-expired-unused-" + uuid.New().String(),
		"expires_at":       past,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_link_tokens", expiredUnused) })

	expiredUsed := testutil.SeedRow(t, pool, "integration_link_tokens", map[string]any{
		"tenant_id":        tenantA,
		"platform":         PlatformSlack,
		"external_user_id": "U-expired-used",
		"token_hash":       "hash-expired-used-" + uuid.New().String(),
		"expires_at":       past,
		"used":             true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_link_tokens", expiredUsed) })

	otherTenantExpired := testutil.SeedRow(t, pool, "integration_link_tokens", map[string]any{
		"tenant_id":        tenantOther,
		"platform":         PlatformSlack,
		"external_user_id": "U-other-expired",
		"token_hash":       "hash-other-expired-" + uuid.New().String(),
		"expires_at":       past,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_link_tokens", otherTenantExpired) })

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	// GetLinkTokenByHash is looked up by the exact stored hash, so read it
	// back through the same seeded value instead of reconstructing it.
	t.Run("GetLinkTokenByHash", func(t *testing.T) {
		hash := seededTokenHash(t, pool, validUnused)
		tok, err := repo.GetLinkTokenByHash(ctxA, hash)
		if err != nil {
			t.Fatalf("GetLinkTokenByHash valid: %v", err)
		}
		if tok.ID != validUnused {
			t.Fatalf("GetLinkTokenByHash: id = %s, want %s", tok.ID, validUnused)
		}

		usedHash := seededTokenHash(t, pool, expiredUsed)
		if _, err := repo.GetLinkTokenByHash(ctxA, usedHash); err != ErrLinkTokenNotFound {
			t.Fatalf("GetLinkTokenByHash used token: want ErrLinkTokenNotFound, got %v", err)
		}

		if _, err := repo.GetLinkTokenByHash(ctxA, "does-not-exist"); err != ErrLinkTokenNotFound {
			t.Fatalf("GetLinkTokenByHash unknown hash: want ErrLinkTokenNotFound, got %v", err)
		}
	})

	t.Run("MarkLinkTokenUsed", func(t *testing.T) {
		if err := repo.MarkLinkTokenUsed(ctxA, validUnused); err != nil {
			t.Fatalf("MarkLinkTokenUsed: %v", err)
		}
		hash := seededTokenHash(t, pool, validUnused)
		if _, err := repo.GetLinkTokenByHash(ctxA, hash); err != ErrLinkTokenNotFound {
			t.Fatalf("GetLinkTokenByHash after mark-used: want ErrLinkTokenNotFound, got %v", err)
		}

		if err := repo.MarkLinkTokenUsed(ctxA, validUnused); err != ErrLinkTokenNotFound {
			t.Fatalf("MarkLinkTokenUsed already used: want ErrLinkTokenNotFound, got %v", err)
		}
		if err := repo.MarkLinkTokenUsed(ctxA, uuid.New()); err != ErrLinkTokenNotFound {
			t.Fatalf("MarkLinkTokenUsed unknown id: want ErrLinkTokenNotFound, got %v", err)
		}
	})

	t.Run("CleanupExpiredTokens deletes only expired-and-unused rows", func(t *testing.T) {
		// validUnused was marked used by the previous subtest but is not
		// expired — it must survive regardless of its used flag.
		n, err := repo.CleanupExpiredTokens(ctxA)
		if err != nil {
			t.Fatalf("CleanupExpiredTokens: %v", err)
		}
		if n != 1 {
			t.Fatalf("CleanupExpiredTokens: deleted %d rows, want 1 (only expiredUnused)", n)
		}

		testutil.AssertRowCount(t, pool, ctxA, "integration_link_tokens", expiredUnused, 0)
		testutil.AssertRowCount(t, pool, ctxA, "integration_link_tokens", expiredUsed, 1)
		testutil.AssertRowCount(t, pool, ctxA, "integration_link_tokens", validUnused, 1)
		// A second tenant's expired token must be untouched by tenantA's cleanup call.
		testutil.AssertRowCount(t, pool, ctxOther, "integration_link_tokens", otherTenantExpired, 1)
	})
}

// seededTokenHash reads back the token_hash SeedRow stored for id — the tests
// above need the literal value to call GetLinkTokenByHash with.
func seededTokenHash(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) string {
	t.Helper()
	var hash string
	ctx := testutil.WithSystemCtx(context.Background())
	if err := pool.QueryRow(ctx, "SELECT token_hash FROM integration_link_tokens WHERE id = $1", id).Scan(&hash); err != nil {
		t.Fatalf("read back token_hash for %s: %v", id, err)
	}
	return hash
}

func TestPostgresRepository_DeliveryLog(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// Registered before any row-cleanup t.Cleanup calls below: Cleanup funcs run
	// LIFO, and a plain `defer pool.Close()` here would fire when the test
	// function returns — before t.Cleanup-registered row deletions, which run
	// after. Closing the pool first made every row cleanup silently fail
	// ("closed pool"), leaving cross-test fixtures behind.
	t.Cleanup(func() { pool.Close() })

	tenantA := uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "Delivery Log Tenant A")
	userA := seedIntegrationUser(t, pool, tenantA)

	configID := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenantA,
		"platform":              PlatformSlack,
		"credentials_vault_key": "slack/" + uuid.New().String(),
		"created_by":            userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_configs", configID) })

	mappingID := testutil.SeedRow(t, pool, "integration_channel_mappings", map[string]any{
		"tenant_id":    tenantA,
		"config_id":    configID,
		"channel_id":   "C" + uuid.New().String()[:8],
		"channel_name": "general",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_channel_mappings", mappingID) })

	base := time.Now().Add(-time.Hour)
	sent := testutil.SeedRow(t, pool, "integration_delivery_log", map[string]any{
		"tenant_id":           tenantA,
		"notification_id":     uuid.New(),
		"mapping_id":          mappingID,
		"platform":            PlatformSlack,
		"status":              DeliveryStatusSent,
		"platform_message_id": "ts-sent",
		"created_at":          base,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_delivery_log", sent) })

	failed1 := testutil.SeedRow(t, pool, "integration_delivery_log", map[string]any{
		"tenant_id":           tenantA,
		"notification_id":     uuid.New(),
		"mapping_id":          mappingID,
		"platform":            PlatformSlack,
		"status":              DeliveryStatusFailed,
		"platform_message_id": "",
		"error_message":       "webhook unreachable",
		"created_at":          base.Add(10 * time.Minute),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_delivery_log", failed1) })

	rateLimited := testutil.SeedRow(t, pool, "integration_delivery_log", map[string]any{
		"tenant_id":           tenantA,
		"notification_id":     uuid.New(),
		"mapping_id":          mappingID,
		"platform":            PlatformSlack,
		"status":              DeliveryStatusRateLimited,
		"platform_message_id": "",
		"error_message":       "rate limited",
		"created_at":          base.Add(20 * time.Minute),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_delivery_log", rateLimited) })

	failed2 := testutil.SeedRow(t, pool, "integration_delivery_log", map[string]any{
		"tenant_id":           tenantA,
		"notification_id":     uuid.New(),
		"mapping_id":          mappingID,
		"platform":            PlatformSlack,
		"status":              DeliveryStatusFailed,
		"platform_message_id": "",
		"error_message":       "webhook unreachable",
		"created_at":          base.Add(30 * time.Minute),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_delivery_log", failed2) })

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)

	t.Run("GetRecentFailures excludes sent, orders desc, respects limit", func(t *testing.T) {
		entries, err := repo.GetRecentFailures(ctxA, mappingID, 2)
		if err != nil {
			t.Fatalf("GetRecentFailures: %v", err)
		}
		if len(entries) != 2 || entries[0].ID != failed2 || entries[1].ID != rateLimited {
			t.Fatalf("GetRecentFailures: want [failed2, rateLimited] desc order, got %d entries", len(entries))
		}
	})

	t.Run("CleanupOldLogs deletes only entries older than cutoff", func(t *testing.T) {
		cutoff := base.Add(15 * time.Minute)
		n, err := repo.CleanupOldLogs(ctxA, cutoff)
		if err != nil {
			t.Fatalf("CleanupOldLogs: %v", err)
		}
		// sent (base) and failed1 (base+10m) are older than cutoff (base+15m).
		if n != 2 {
			t.Fatalf("CleanupOldLogs: deleted %d rows, want 2", n)
		}
		testutil.AssertRowCount(t, pool, ctxA, "integration_delivery_log", sent, 0)
		testutil.AssertRowCount(t, pool, ctxA, "integration_delivery_log", failed1, 0)
		testutil.AssertRowCount(t, pool, ctxA, "integration_delivery_log", rateLimited, 1)
		testutil.AssertRowCount(t, pool, ctxA, "integration_delivery_log", failed2, 1)
	})
}

func TestPostgresRepository_ResolveTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// Registered before any row-cleanup t.Cleanup calls below: Cleanup funcs run
	// LIFO, and a plain `defer pool.Close()` here would fire when the test
	// function returns — before t.Cleanup-registered row deletions, which run
	// after. Closing the pool first made every row cleanup silently fail
	// ("closed pool"), leaving cross-test fixtures behind.
	t.Cleanup(func() { pool.Close() })

	tenantSlack, tenantTeams, tenantInactive, tenantDupA, tenantDupB :=
		uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, tid := range []uuid.UUID{tenantSlack, tenantTeams, tenantInactive, tenantDupA, tenantDupB} {
		testutil.EnsureTenant(t, pool, tid, "ResolveTenant Fixture "+tid.String()[:8])
	}

	seedConfig := func(tenantID uuid.UUID, platform string, active bool, metadataKey, workspaceID string) uuid.UUID {
		userID := seedIntegrationUser(t, pool, tenantID)
		id := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
			"tenant_id":             tenantID,
			"platform":              platform,
			"is_active":             active,
			"credentials_vault_key": platform + "/" + uuid.New().String(),
			"metadata":              fmt.Sprintf(`{%q:%q}`, metadataKey, workspaceID),
			"created_by":            userID,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_configs", id) })
		return id
	}

	seedConfig(tenantSlack, PlatformSlack, true, "team_id", "TSLACK111")
	seedConfig(tenantTeams, PlatformTeams, true, "tenant_id", "AZURE-XYZ")
	seedConfig(tenantInactive, PlatformSlack, false, "team_id", "TINACTIVE")
	seedConfig(tenantDupA, PlatformSlack, true, "team_id", "TDUPLICATE")
	seedConfig(tenantDupB, PlatformSlack, true, "team_id", "TDUPLICATE")

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	cases := []struct {
		name        string
		platform    string
		workspaceID string
		wantTenant  uuid.UUID
		wantErr     error
	}{
		{"resolves active slack workspace", PlatformSlack, "TSLACK111", tenantSlack, nil},
		{"resolves active teams workspace", PlatformTeams, "AZURE-XYZ", tenantTeams, nil},
		{"unknown workspace", PlatformSlack, "T-UNKNOWN", uuid.Nil, ErrTenantUnresolved},
		{"empty workspace id", PlatformSlack, "", uuid.Nil, ErrTenantUnresolved},
		{"invalid platform", "discord", "whatever", uuid.Nil, ErrInvalidPlatform},
		{"inactive config not matched", PlatformSlack, "TINACTIVE", uuid.Nil, ErrTenantUnresolved},
		{"ambiguous workspace refused", PlatformSlack, "TDUPLICATE", uuid.Nil, ErrTenantAmbiguous},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotTenant, err := repo.ResolveTenant(ctx, tc.platform, tc.workspaceID)
			if err != tc.wantErr {
				t.Fatalf("ResolveTenant(%s, %s): err = %v, want %v", tc.platform, tc.workspaceID, err, tc.wantErr)
			}
			if tc.wantErr == nil && gotTenant != tc.wantTenant {
				t.Fatalf("ResolveTenant(%s, %s): tenant = %s, want %s", tc.platform, tc.workspaceID, gotTenant, tc.wantTenant)
			}
		})
	}
}
