package lexware

// Real-schema DB tests for PostgresIntegrationConfigRepo (integration_configs,
// platform='lexware'). Unlike the lexware_* tables below, integration_configs
// carries tenant_id all the way through Upsert — the write path here is
// sound, so these tests exercise the real INSERT/UPDATE, not a seeded
// fixture.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPostgresIntegrationConfigRepo_UpsertAndGetByPlatform(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Lexware ConfigRepo Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         "lw-cfgrepo-" + uuid.New().String()[:8] + "@a.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresIntegrationConfigRepo(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	cfg := &IntegrationConfig{
		TenantID:            tenant,
		Platform:            "lexware",
		IsActive:            true,
		CredentialsVaultKey: "lexware/" + uuid.New().String(),
		Metadata:            map[string]any{"note": "created by test"},
		CreatedBy:           userID,
	}
	require.NoError(t, repo.Upsert(ctx, cfg))
	require.NotEqual(t, uuid.Nil, cfg.ID)
	defer testutil.CleanupRow(t, pool, "integration_configs", cfg.ID)

	got, err := repo.GetByPlatform(ctx, "lexware")
	require.NoError(t, err)
	assert.Equal(t, cfg.ID, got.ID)
	assert.Equal(t, tenant, got.TenantID)
	assert.True(t, got.IsActive)
	assert.Equal(t, cfg.CredentialsVaultKey, got.CredentialsVaultKey)
	require.NotNil(t, got.Metadata)
	assert.Equal(t, "created by test", got.Metadata["note"])

	// Upsert again with the same ID: ON CONFLICT (platform, tenant_id) must
	// update in place, not create a second row.
	cfg.IsActive = false
	cfg.Metadata = map[string]any{"note": "deactivated by test"}
	require.NoError(t, repo.Upsert(ctx, cfg))

	updated, err := repo.GetByPlatform(ctx, "lexware")
	require.NoError(t, err)
	assert.Equal(t, cfg.ID, updated.ID, "upsert must update the existing row, not insert a second one")
	assert.False(t, updated.IsActive)
	assert.Equal(t, "deactivated by test", updated.Metadata["note"])
}

func TestPostgresIntegrationConfigRepo_GetByPlatform_NotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Lexware ConfigRepo NotFound Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	repo := NewPostgresIntegrationConfigRepo(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	_, err := repo.GetByPlatform(ctx, "lexware")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPostgresIntegrationConfigRepo_Deactivate(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Lexware ConfigRepo Deactivate Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         "lw-cfgrepo-deact-" + uuid.New().String()[:8] + "@a.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	cfgID := testutil.SeedRow(t, pool, "integration_configs", map[string]any{
		"tenant_id":             tenant,
		"platform":              "lexware",
		"is_active":             true,
		"credentials_vault_key": "lexware/" + uuid.New().String(),
		"created_by":            userID,
	})
	defer testutil.CleanupRow(t, pool, "integration_configs", cfgID)

	repo := NewPostgresIntegrationConfigRepo(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	require.NoError(t, repo.Deactivate(ctx, cfgID))

	got, err := repo.GetByPlatform(ctx, "lexware")
	require.NoError(t, err)
	assert.False(t, got.IsActive)
}
