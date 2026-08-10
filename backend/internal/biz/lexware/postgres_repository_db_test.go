package lexware

// Real-schema DB tests for PostgresRepository against lexware_sync_configs,
// lexware_entity_mappings, lexware_field_mappings, lexware_sync_log,
// lexware_webhook_subscriptions.
//
// UpsertSyncConfig, UpsertEntityMapping, UpsertFieldMappings, CreateSyncLog,
// and UpsertWebhookSubscription used to build their INSERTs without a
// tenant_id column, while all five target tables have tenant_id UUID NOT
// NULL with no default and no populating trigger — every call failed on a
// live database with a not-null-violation (see BACKLOG.yml unit
// fix-lexware-tenant-id-missing-on-upsert for the original finding). The
// four models.Lexware* structs now carry a TenantID field and the five
// repository methods write it; TestPostgresRepository_Upsert* /
// TestPostgresRepository_CreateSyncLog_RealSchema below exercise those real
// write paths end to end instead of seeding around them. Read/list/delete
// tests further down still seed their fixtures directly via
// testutil.SeedRow — that is just the cheaper way to set up a read fixture,
// not a workaround.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedLexwareFixture seeds a tenant, a user, and a lexware integration
// config — the common parent row every other lexware_* table FKs into.
func seedLexwareFixture(t *testing.T, pool *pgxpool.Pool) (tenantID, userID, cfgID uuid.UUID) {
	t.Helper()
	tenantID = uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Lexware Repo DB Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	userID = testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         "lw-repo-" + uuid.New().String()[:8] + "@a.local",
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	cfgID = testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenantID,
		"platform":              "lexware",
		"credentials_vault_key": "lexware/" + uuid.New().String(),
		"created_by":            userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_configs", cfgID) })

	return tenantID, userID, cfgID
}

func TestPostgresRepository_GetSyncConfig(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, _, cfgID := seedLexwareFixture(t, pool)

	syncID := testutil.SeedRow(t, pool, "lexware_sync_configs", map[string]any{
		"tenant_id":              tenantID,
		"config_id":              cfgID,
		"contact_sync_enabled":   true,
		"invoice_push_enabled":   false,
		"quote_push_enabled":     false,
		"webhook_enabled":        true,
	})
	defer testutil.CleanupRow(t, pool, "lexware_sync_configs", syncID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	got, err := repo.GetSyncConfig(ctx, cfgID)
	require.NoError(t, err)
	assert.Equal(t, cfgID, got.ConfigID)
	assert.True(t, got.ContactSyncEnabled)
	assert.False(t, got.InvoicePushEnabled)
	assert.True(t, got.WebhookEnabled)
}

func TestPostgresRepository_GetSyncConfig_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	_, err := repo.GetSyncConfig(context.Background(), uuid.New())
	require.ErrorIs(t, err, ErrConfigNotFound)
}

func TestPostgresRepository_UpdateLastSyncTime(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, _, cfgID := seedLexwareFixture(t, pool)

	syncID := testutil.SeedRow(t, pool, "lexware_sync_configs", map[string]any{
		"tenant_id": tenantID,
		"config_id": cfgID,
	})
	defer testutil.CleanupRow(t, pool, "lexware_sync_configs", syncID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC().Truncate(time.Second)

	require.NoError(t, repo.UpdateLastSyncTime(ctx, cfgID, "contact_full", now))

	got, err := repo.GetSyncConfig(ctx, cfgID)
	require.NoError(t, err)
	require.NotNil(t, got.LastContactSyncAt)
	assert.WithinDuration(t, now, *got.LastContactSyncAt, time.Second)
}

func TestPostgresRepository_UpdateLastSyncTime_UnknownSyncType(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	err := repo.UpdateLastSyncTime(context.Background(), uuid.New(), "invoice_full", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown sync type")
}

func TestPostgresRepository_EntityMappings(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, _, cfgID := seedLexwareFixture(t, pool)
	kmuhubID := uuid.New()

	mapID := testutil.SeedRow(t, pool, "lexware_entity_mappings", map[string]any{
		"tenant_id":       tenantID,
		"config_id":       cfgID,
		"entity_type":     "contact",
		"kmuhub_id":       kmuhubID,
		"lexware_id":      "lex-contact-42",
		"lexware_version": 3,
		"sync_direction":  "both",
	})
	defer testutil.CleanupRow(t, pool, "lexware_entity_mappings", mapID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	byKMUHub, err := repo.GetEntityMapping(ctx, cfgID, "contact", kmuhubID)
	require.NoError(t, err)
	assert.Equal(t, "lex-contact-42", byKMUHub.LexwareID)
	assert.Equal(t, 3, byKMUHub.LexwareVersion)

	byLexware, err := repo.GetEntityMappingByLexwareID(ctx, cfgID, "contact", "lex-contact-42")
	require.NoError(t, err)
	assert.Equal(t, kmuhubID, byLexware.KmuhubID)

	list, err := repo.ListEntityMappings(ctx, cfgID, "contact")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, mapID, list[0].ID)

	// A different entity type must not surface this mapping.
	empty, err := repo.ListEntityMappings(ctx, cfgID, "invoice")
	require.NoError(t, err)
	assert.Empty(t, empty)

	require.NoError(t, repo.DeleteEntityMapping(ctx, mapID))
	_, err = repo.GetEntityMapping(ctx, cfgID, "contact", kmuhubID)
	require.ErrorIs(t, err, ErrMappingNotFound)
}

func TestPostgresRepository_GetEntityMapping_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	_, err := repo.GetEntityMapping(context.Background(), uuid.New(), "contact", uuid.New())
	require.ErrorIs(t, err, ErrMappingNotFound)

	_, err = repo.GetEntityMappingByLexwareID(context.Background(), uuid.New(), "contact", "nope")
	require.ErrorIs(t, err, ErrMappingNotFound)
}

func TestPostgresRepository_GetFieldMappings(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, _, cfgID := seedLexwareFixture(t, pool)

	fmID := testutil.SeedRow(t, pool, "lexware_field_mappings", map[string]any{
		"tenant_id":   tenantID,
		"config_id":   cfgID,
		"entity_type": "contact",
		"mappings":    `[{"kmuhub_field":"first_name","lexware_field":"person.firstName","direction":"both","required":true}]`,
	})
	defer testutil.CleanupRow(t, pool, "lexware_field_mappings", fmID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	got, err := repo.GetFieldMappings(ctx, cfgID, "contact")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Len(t, got.Mappings, 1)
	assert.Equal(t, "first_name", got.Mappings[0].KmuhubField)
	assert.True(t, got.Mappings[0].Required)
}

func TestPostgresRepository_GetFieldMappings_NotFoundReturnsNilNil(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	got, err := repo.GetFieldMappings(context.Background(), uuid.New(), "contact")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestPostgresRepository_SyncLog(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, _, cfgID := seedLexwareFixture(t, pool)

	logID := testutil.SeedRow(t, pool, "lexware_sync_log", map[string]any{
		"tenant_id":  tenantID,
		"config_id":  cfgID,
		"sync_type":  "contact_full",
		"status":     "running",
		"metadata":   `{"seed":"initial"}`,
	})
	defer testutil.CleanupRow(t, pool, "lexware_sync_log", logID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	logs, err := repo.ListSyncLogs(ctx, cfgID, 0)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, "running", logs[0].Status)

	latest, err := repo.GetLatestSyncLog(ctx, cfgID, "contact_full")
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, logID, latest.ID)
	assert.Equal(t, "initial", latest.Metadata["seed"])

	completed := time.Now().UTC()
	update := &models.LexwareSyncLog{
		ID:             logID,
		Status:         "completed",
		ItemsProcessed: 5,
		ItemsCreated:   2,
		ItemsUpdated:   3,
		CompletedAt:    &completed,
		Metadata:       map[string]any{"seed": "updated"},
	}
	require.NoError(t, repo.UpdateSyncLog(ctx, update))

	latest, err = repo.GetLatestSyncLog(ctx, cfgID, "contact_full")
	require.NoError(t, err)
	assert.Equal(t, "completed", latest.Status)
	assert.Equal(t, 5, latest.ItemsProcessed)
	assert.Equal(t, "updated", latest.Metadata["seed"])
}

func TestPostgresRepository_GetLatestSyncLog_NotFoundReturnsNilNil(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	repo := NewPostgresRepository(pool)
	got, err := repo.GetLatestSyncLog(context.Background(), uuid.New(), "contact_full")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestPostgresRepository_ListSyncLogs_DefaultLimit(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, _, cfgID := seedLexwareFixture(t, pool)

	logID := testutil.SeedRow(t, pool, "lexware_sync_log", map[string]any{
		"tenant_id": tenantID,
		"config_id": cfgID,
		"sync_type": "contact_delta",
		"status":    "completed",
	})
	defer testutil.CleanupRow(t, pool, "lexware_sync_log", logID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	// limit <= 0 falls back to 50 internally — assert it does not error and
	// still returns the seeded row, proving the fallback branch ran.
	logs, err := repo.ListSyncLogs(ctx, cfgID, -1)
	require.NoError(t, err)
	require.Len(t, logs, 1)
}

func TestPostgresRepository_WebhookSubscriptions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, _, cfgID := seedLexwareFixture(t, pool)

	subID := testutil.SeedRow(t, pool, "lexware_webhook_subscriptions", map[string]any{
		"tenant_id":       tenantID,
		"config_id":       cfgID,
		"subscription_id": "sub-1",
		"event_type":      "contact.created",
		"callback_url":    "https://app.zentria.tech/webhooks/lexware",
	})
	defer testutil.CleanupRow(t, pool, "lexware_webhook_subscriptions", subID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	subs, err := repo.ListWebhookSubscriptions(ctx, cfgID)
	require.NoError(t, err)
	require.Len(t, subs, 1)
	assert.Equal(t, "sub-1", subs[0].SubscriptionID)
	assert.True(t, subs[0].IsActive)

	require.NoError(t, repo.DeleteWebhookSubscription(ctx, subID))

	subs, err = repo.ListWebhookSubscriptions(ctx, cfgID)
	require.NoError(t, err)
	assert.Empty(t, subs)
}

func TestPostgresRepository_UpsertSyncConfig_RealSchema(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, _, cfgID := seedLexwareFixture(t, pool)
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	config := &models.LexwareSyncConfig{
		TenantID:               tenantID,
		ConfigID:               cfgID,
		ContactSyncEnabled:     true,
		ContactSyncIntervalMin: 15,
		InvoicePushEnabled:     true,
		QuotePushEnabled:       true,
		WebhookEnabled:         true,
	}
	require.NoError(t, repo.UpsertSyncConfig(ctx, config))
	require.NotEqual(t, uuid.Nil, config.ID)
	defer testutil.CleanupRow(t, pool, "lexware_sync_configs", config.ID)

	got, err := repo.GetSyncConfig(ctx, cfgID)
	require.NoError(t, err)
	assert.True(t, got.ContactSyncEnabled)
	assert.Equal(t, 15, got.ContactSyncIntervalMin)

	// ON CONFLICT (config_id) DO UPDATE must update the existing row, not
	// insert a second one.
	config.ContactSyncEnabled = false
	require.NoError(t, repo.UpsertSyncConfig(ctx, config))

	updated, err := repo.GetSyncConfig(ctx, cfgID)
	require.NoError(t, err)
	assert.Equal(t, config.ID, updated.ID, "upsert must update the existing row, not insert a second one")
	assert.False(t, updated.ContactSyncEnabled)
}

func TestPostgresRepository_UpsertEntityMapping_RealSchema(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, _, cfgID := seedLexwareFixture(t, pool)
	kmuhubID := uuid.New()
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	mapping := &models.LexwareEntityMapping{
		TenantID:       tenantID,
		ConfigID:       cfgID,
		EntityType:     "contact",
		KmuhubID:       kmuhubID,
		LexwareID:      "lex-real-1",
		LexwareVersion: 1,
		LastSyncedAt:   time.Now().UTC(),
		SyncDirection:  "both",
	}
	require.NoError(t, repo.UpsertEntityMapping(ctx, mapping))
	require.NotEqual(t, uuid.Nil, mapping.ID)
	defer testutil.CleanupRow(t, pool, "lexware_entity_mappings", mapping.ID)

	got, err := repo.GetEntityMapping(ctx, cfgID, "contact", kmuhubID)
	require.NoError(t, err)
	assert.Equal(t, "lex-real-1", got.LexwareID)

	// ON CONFLICT (config_id, entity_type, kmuhub_id) DO UPDATE must update
	// the existing row, not insert a second one.
	mapping.LexwareVersion = 2
	require.NoError(t, repo.UpsertEntityMapping(ctx, mapping))

	updated, err := repo.GetEntityMapping(ctx, cfgID, "contact", kmuhubID)
	require.NoError(t, err)
	assert.Equal(t, mapping.ID, updated.ID, "upsert must update the existing row, not insert a second one")
	assert.Equal(t, 2, updated.LexwareVersion)
}

func TestPostgresRepository_UpsertFieldMappings_RealSchema(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, _, cfgID := seedLexwareFixture(t, pool)
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	fm := &models.LexwareFieldMapping{
		TenantID:   tenantID,
		ConfigID:   cfgID,
		EntityType: "contact",
		Mappings: []models.LexwareFieldMappingEntry{
			{KmuhubField: "first_name", LexwareField: "person.firstName", Direction: "both", Required: true},
		},
	}
	require.NoError(t, repo.UpsertFieldMappings(ctx, fm))
	require.NotEqual(t, uuid.Nil, fm.ID)
	defer testutil.CleanupRow(t, pool, "lexware_field_mappings", fm.ID)

	got, err := repo.GetFieldMappings(ctx, cfgID, "contact")
	require.NoError(t, err)
	require.Len(t, got.Mappings, 1)
	assert.Equal(t, "first_name", got.Mappings[0].KmuhubField)

	// ON CONFLICT (config_id, entity_type) DO UPDATE must update the
	// existing row, not insert a second one.
	fm.Mappings = append(fm.Mappings, models.LexwareFieldMappingEntry{
		KmuhubField: "last_name", LexwareField: "person.lastName", Direction: "both",
	})
	require.NoError(t, repo.UpsertFieldMappings(ctx, fm))

	updated, err := repo.GetFieldMappings(ctx, cfgID, "contact")
	require.NoError(t, err)
	assert.Equal(t, fm.ID, updated.ID, "upsert must update the existing row, not insert a second one")
	assert.Len(t, updated.Mappings, 2)
}

func TestPostgresRepository_CreateSyncLog_RealSchema(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, _, cfgID := seedLexwareFixture(t, pool)
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	log := &models.LexwareSyncLog{
		TenantID:  tenantID,
		ConfigID:  cfgID,
		SyncType:  "contact_full",
		Status:    "running",
		StartedAt: time.Now().UTC(),
		Metadata:  map[string]any{"origin": "real-schema-test"},
	}
	require.NoError(t, repo.CreateSyncLog(ctx, log))
	require.NotEqual(t, uuid.Nil, log.ID)
	defer testutil.CleanupRow(t, pool, "lexware_sync_log", log.ID)

	got, err := repo.GetLatestSyncLog(ctx, cfgID, "contact_full")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, log.ID, got.ID)
	assert.Equal(t, "real-schema-test", got.Metadata["origin"])
}

func TestPostgresRepository_UpsertWebhookSubscription_RealSchema(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, _, cfgID := seedLexwareFixture(t, pool)
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	require.NoError(t, repo.UpsertWebhookSubscription(ctx, tenantID, cfgID, "sub-real-1", "contact.created", "https://app.zentria.tech/webhooks/lexware"))

	subs, err := repo.ListWebhookSubscriptions(ctx, cfgID)
	require.NoError(t, err)
	require.Len(t, subs, 1)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "lexware_webhook_subscriptions", subs[0].ID) })
	assert.Equal(t, "sub-real-1", subs[0].SubscriptionID)
	assert.True(t, subs[0].IsActive)

	// ON CONFLICT (config_id, event_type) DO UPDATE must update the existing
	// row (new subscription_id from a re-registration), not insert a second one.
	require.NoError(t, repo.UpsertWebhookSubscription(ctx, tenantID, cfgID, "sub-real-2", "contact.created", "https://app.zentria.tech/webhooks/lexware"))

	subs, err = repo.ListWebhookSubscriptions(ctx, cfgID)
	require.NoError(t, err)
	require.Len(t, subs, 1, "upsert must update the existing row, not insert a second one")
	assert.Equal(t, "sub-real-2", subs[0].SubscriptionID)
}
