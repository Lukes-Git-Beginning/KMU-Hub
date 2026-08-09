package lexware

// Real-schema DB tests for PostgresRepository's read/list/delete/update
// paths (lexware_sync_configs, lexware_entity_mappings, lexware_field_mappings,
// lexware_sync_log, lexware_webhook_subscriptions).
//
// Deliberately NOT covered here: UpsertSyncConfig, UpsertEntityMapping,
// UpsertFieldMappings, CreateSyncLog, UpsertWebhookSubscription. Their INSERT
// statements omit the tenant_id column entirely, and all five target tables
// have tenant_id UUID NOT NULL with no default and no populating trigger
// (verified via `\d <table>` against the local DB and a migration grep for
// any tenant_id-setting trigger — there is none). Every call to these five
// methods fails on a live database with a not-null-violation; the Go structs
// they take don't even carry a TenantID field to fix it locally. See
// JOURNAL.md iteration entry and BACKLOG.yml unit
// fix-lexware-tenant-id-missing-on-upsert for the real-schema evidence and
// scope of the fix. Fixtures for the tests below are seeded directly via
// testutil.SeedRow (which does carry tenant_id), bypassing the broken
// write path so the read/list/delete/update paths can still be verified
// against the real schema.

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
