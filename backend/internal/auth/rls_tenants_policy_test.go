package auth_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestRLS_Tenants_TenantASeesOwnRow verifies the USING clause
// id = current_tenant_id(): a caller whose app.tenant_id matches the tenants
// row can read it.
func TestRLS_Tenants_TenantASeesOwnRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "tenants", testutil.TenantA, 1)
}

// TestRLS_Tenants_TenantACannotSeeTenantBRow verifies the isolation half of
// the tenant_self_isolation policy: TenantA context cannot see TenantB's row
// (id != current_tenant_id() and not system context).
func TestRLS_Tenants_TenantACannotSeeTenantBRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "tenants", testutil.TenantB, 0)
}

// TestRLS_Tenants_SystemContextSeesAll verifies is_system_context() bypass on
// the tenants table — the system context must read any tenant row.
func TestRLS_Tenants_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	sysCtx := testutil.WithSystemCtx(context.Background())
	testutil.AssertRowCount(t, pool, sysCtx, "tenants", testutil.TenantA, 1)
	testutil.AssertRowCount(t, pool, sysCtx, "tenants", testutil.TenantB, 1)
}

// TestRLS_Tenants_UserContextInsertRejected verifies the WITH CHECK clause:
// is_system_context() — a non-system INSERT into tenants must be rejected.
func TestRLS_Tenants_UserContextInsertRejected(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	ctxUser := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	newID := uuid.New()
	_, err := pool.Exec(ctxUser,
		"INSERT INTO tenants (id, name) VALUES ($1, $2)",
		newID, "Should-Fail")
	if err == nil {
		// Attempt cleanup if the insert unexpectedly succeeded.
		sysCtx := testutil.WithSystemCtx(context.Background())
		_, _ = pool.Exec(sysCtx, "DELETE FROM tenants WHERE id = $1", newID)
		t.Fatal("expected INSERT to be rejected by tenant_self_isolation WITH CHECK")
	}
}

// TestRLS_Tenants_SystemContextInsertSucceeds verifies that a system-context
// INSERT into tenants is admitted by the WITH CHECK is_system_context() clause.
// EnsureTenant already exercises this path; this test makes the contract
// explicit and visible in the test suite.
func TestRLS_Tenants_SystemContextInsertSucceeds(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	newID := uuid.New()
	sysCtx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(sysCtx,
		"INSERT INTO tenants (id, name) VALUES ($1, $2)",
		newID, "System-Inserted-Tenant")
	if err != nil {
		t.Fatalf("system-context INSERT into tenants failed: %v", err)
	}

	// Cleanup.
	_, _ = pool.Exec(sysCtx, "DELETE FROM tenants WHERE id = $1", newID)
}
