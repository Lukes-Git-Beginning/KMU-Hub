package auth_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestRLS_Users_TenantBSeesNoTenantAUsers verifies the tenant_id branch of
// the user_isolation policy: a user in TenantB must not see rows belonging to
// TenantA when the only matching clause is tenant_id = current_tenant_id().
func TestRLS_Users_TenantBSeesNoTenantAUsers(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "alice-rls@test.local",
		"password_hash": "x",
		"first_name":    "Alice",
		"last_name":     "RLS",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "users", userID, 0)
}

// TestRLS_Users_TenantASeesOwnUsers verifies that a tenant can see its own
// user rows (the tenant_id = current_tenant_id() branch).
func TestRLS_Users_TenantASeesOwnUsers(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "alice2-rls@test.local",
		"password_hash": "x",
		"first_name":    "Alice2",
		"last_name":     "RLS",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "users", userID, 1)
}

// TestRLS_Users_OwnRowReadableEvenWithDifferentTenant verifies the
// id = current_user_id() OR-clause: a user can read their own row even when
// the caller context carries a different tenant ID.
func TestRLS_Users_OwnRowReadableEvenWithDifferentTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	aliceID := uuid.New()
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            aliceID,
		"tenant_id":     testutil.TenantA,
		"email":         "alice3-rls@test.local",
		"password_hash": "x",
		"first_name":    "Alice3",
		"last_name":     "RLS",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// ctx carries TenantB as tenant but Alice's own ID as user_id —
	// the policy USING clause admits via id = current_user_id().
	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	ctx = context.WithValue(ctx, middleware.UserIDKey, aliceID.String())
	testutil.AssertRowCount(t, pool, ctx, "users", userID, 1)
}

// TestRLS_Users_SystemContextSeesAll verifies that is_system_context() admits
// rows regardless of tenant — used by seed helpers and pre-JWT service paths.
func TestRLS_Users_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "alice4-rls@test.local",
		"password_hash": "x",
		"first_name":    "Alice4",
		"last_name":     "RLS",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	sysCtx := testutil.WithSystemCtx(context.Background())
	testutil.AssertRowCount(t, pool, sysCtx, "users", userID, 1)
}

// TestRLS_Users_CrossTenantUpdateBlocked verifies the WITH CHECK clause:
// a user in TenantB cannot update a row belonging to TenantA (zero rows
// affected — RLS silently filters instead of erroring on UPDATE).
func TestRLS_Users_CrossTenantUpdateBlocked(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "alice5-rls@test.local",
		"password_hash": "x",
		"first_name":    "Alice5",
		"last_name":     "RLS",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	testutil.AssertUpdateRowsAffected(t, pool, ctxB,
		"UPDATE users SET first_name = 'Hacked' WHERE id = $1",
		0, userID)
}
