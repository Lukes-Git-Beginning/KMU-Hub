package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/plugin/repository"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// plugin_permissions carries no tenant_id of its own — migration 000272 put it
// under enable_tenant_rls_via_join() through installation_id, the same shape
// 000270 chose for the CRM custom-field-value tables. seedInstallation creates
// the manifest (a shared catalog row, no tenant of its own) and the tenant-owned
// installation the permission grant joins through.
func seedInstallation(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, slug string) uuid.UUID {
	t.Helper()

	manifestID := testutil.SeedRow(t, pool, "plugin_manifests", map[string]any{
		"id":   uuid.New(),
		"slug": slug,
		"name": slug,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "plugin_manifests", manifestID) })

	installationID := testutil.SeedRow(t, pool, "plugin_installations", map[string]any{
		"id":           uuid.New(),
		"tenant_id":    tenantID,
		"manifest_id":  manifestID,
		"installed_by": uuid.New(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "plugin_installations", installationID) })

	return installationID
}

// TestRLS_PluginPermissions_TenantBSeesNoTenantAGrants verifies that the join
// through plugin_installations blocks cross-tenant reads.
func TestRLS_PluginPermissions_TenantBSeesNoTenantAGrants(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	// t.Cleanup, not defer: seedInstallation registers its own cleanups via
	// t.Cleanup, which all run after this function returns. A plain defer here
	// would close the pool before those fire.
	t.Cleanup(func() { pool.Close() })

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	installationID := seedInstallation(t, pool, testutil.TenantA, "rls-test-plugin-a")

	permID := testutil.SeedRow(t, pool, "plugin_permissions", map[string]any{
		"id":              uuid.New(),
		"installation_id": installationID,
		"permission":      "contacts:read",
		"granted_by":      uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "plugin_permissions", permID)

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "plugin_permissions", permID, 0)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "plugin_permissions", permID, 1)
}

// TestPluginPermissions_Grant_CrossTenantInstallation_Rejected exercises the
// real write path (repository.PermissionRepository.Grant) rather than SeedRow:
// granting a permission against another tenant's installation must fail the
// join policy's WITH CHECK, not silently attach to a foreign installation.
func TestPluginPermissions_Grant_CrossTenantInstallation_Rejected(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	// t.Cleanup, not defer: seedInstallation registers its own cleanups via
	// t.Cleanup, which all run after this function returns. A plain defer here
	// would close the pool before those fire.
	t.Cleanup(func() { pool.Close() })

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	installationID := seedInstallation(t, pool, testutil.TenantB, "rls-test-plugin-b")

	repo := repository.NewPermissionRepository(pool)
	perm := &models.PluginPermission{
		ID:             uuid.New(),
		InstallationID: installationID,
		Permission:     "contacts:read",
		GrantedBy:      uuid.New(),
	}

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	err := repo.Grant(ctxA, perm)
	if err == nil {
		testutil.CleanupRow(t, pool, "plugin_permissions", perm.ID)
		t.Fatal("Grant against a foreign tenant's installation succeeded; the RLS policy is not in force")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Fatalf("expected SQLSTATE 42501, got: %v", err)
	}
}
