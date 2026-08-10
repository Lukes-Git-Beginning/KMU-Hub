package bexio

// Real-schema DB tests for PostgresRepository against bexio_sync_configs,
// bexio_entity_mappings, bexio_field_mappings, bexio_sync_log.
//
// UpsertSyncConfig, UpsertEntityMapping, UpsertFieldMappings, and
// CreateSyncLog used to build their INSERTs without a tenant_id column,
// while all four target tables have tenant_id UUID NOT NULL with no
// default and no populating trigger — every call failed on a live database
// with a not-null-violation (see BACKLOG.yml unit
// fix-bexio-tenant-id-missing-on-upsert for the original finding, mirroring
// fix-lexware-tenant-id-missing-on-upsert). The four models.Bexio* structs
// now carry a TenantID field and the four repository methods write it; the
// tests below exercise those real write paths end to end.
//
// Unlike lexware, bexio's ON CONFLICT targets (config_id) /
// (config_id, entity_type, kmuhub_id) / (config_id, entity_type) were
// already checked against migration 000055_add_bexio_integration.up.sql and
// match the real unique constraints — no second bug found here.

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

// seedBexioFixture seeds a tenant, a user, and a bexio integration config —
// the common parent row every other bexio_* table FKs into.
func seedBexioFixture(t *testing.T, pool *pgxpool.Pool) (tenantID, userID, cfgID uuid.UUID) {
	t.Helper()
	tenantID = uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Bexio Repo DB Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	userID = testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         "bx-repo-" + uuid.New().String()[:8] + "@a.local",
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	cfgID = testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenantID,
		"platform":              "bexio",
		"credentials_vault_key": "bexio/" + uuid.New().String(),
		"created_by":            userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "integration_configs", cfgID) })

	return tenantID, userID, cfgID
}

func TestPostgresRepository_UpsertSyncConfig_RealSchema(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, _, cfgID := seedBexioFixture(t, pool)
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	config := &models.BexioSyncConfig{
		TenantID:               tenantID,
		ConfigID:               cfgID,
		ContactSyncEnabled:     true,
		ContactSyncIntervalMin: 15,
		InvoicePushEnabled:     true,
		QuotePushEnabled:       true,
		PaymentPollEnabled:     true,
		PaymentPollIntervalMin: 5,
	}
	require.NoError(t, repo.UpsertSyncConfig(ctx, config))
	require.NotEqual(t, uuid.Nil, config.ID)
	defer testutil.CleanupRow(t, pool, "bexio_sync_configs", config.ID)

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

	tenantID, _, cfgID := seedBexioFixture(t, pool)
	kmuhubID := uuid.New()
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	mapping := &models.BexioEntityMapping{
		TenantID:      tenantID,
		ConfigID:      cfgID,
		EntityType:    "contact",
		KmuhubID:      kmuhubID,
		BexioID:       100,
		LastSyncedAt:  time.Now().UTC(),
		SyncDirection: "both",
	}
	require.NoError(t, repo.UpsertEntityMapping(ctx, mapping))
	require.NotEqual(t, uuid.Nil, mapping.ID)
	defer testutil.CleanupRow(t, pool, "bexio_entity_mappings", mapping.ID)

	got, err := repo.GetEntityMapping(ctx, cfgID, "contact", kmuhubID)
	require.NoError(t, err)
	assert.Equal(t, 100, got.BexioID)

	// ON CONFLICT (config_id, entity_type, kmuhub_id) DO UPDATE must update
	// the existing row, not insert a second one.
	mapping.BexioID = 200
	require.NoError(t, repo.UpsertEntityMapping(ctx, mapping))

	updated, err := repo.GetEntityMapping(ctx, cfgID, "contact", kmuhubID)
	require.NoError(t, err)
	assert.Equal(t, mapping.ID, updated.ID, "upsert must update the existing row, not insert a second one")
	assert.Equal(t, 200, updated.BexioID)
}

func TestPostgresRepository_UpsertFieldMappings_RealSchema(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenantID, _, cfgID := seedBexioFixture(t, pool)
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	fm := &models.BexioFieldMapping{
		TenantID:   tenantID,
		ConfigID:   cfgID,
		EntityType: "contact",
		Mappings: []models.BexioFieldMappingEntry{
			{KmuhubField: "first_name", BexioField: "name_1", Direction: "both", Required: true},
		},
	}
	require.NoError(t, repo.UpsertFieldMappings(ctx, fm))
	require.NotEqual(t, uuid.Nil, fm.ID)
	defer testutil.CleanupRow(t, pool, "bexio_field_mappings", fm.ID)

	got, err := repo.GetFieldMappings(ctx, cfgID, "contact")
	require.NoError(t, err)
	require.Len(t, got.Mappings, 1)
	assert.Equal(t, "first_name", got.Mappings[0].KmuhubField)

	// ON CONFLICT (config_id, entity_type) DO UPDATE must update the
	// existing row, not insert a second one.
	fm.Mappings = append(fm.Mappings, models.BexioFieldMappingEntry{
		KmuhubField: "last_name", BexioField: "name_2", Direction: "both",
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

	tenantID, _, cfgID := seedBexioFixture(t, pool)
	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	log := &models.BexioSyncLog{
		TenantID:  tenantID,
		ConfigID:  cfgID,
		SyncType:  "contact_full",
		Status:    "running",
		StartedAt: time.Now().UTC(),
		Metadata:  map[string]any{"origin": "real-schema-test"},
	}
	require.NoError(t, repo.CreateSyncLog(ctx, log))
	require.NotEqual(t, uuid.Nil, log.ID)
	defer testutil.CleanupRow(t, pool, "bexio_sync_log", log.ID)

	got, err := repo.GetLatestSyncLog(ctx, cfgID, "contact_full")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, log.ID, got.ID)
	assert.Equal(t, "real-schema-test", got.Metadata["origin"])
}
