package auth_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestRLS_UserSessions_TenantBSeesNoTenantASessions verifies the standard
// tenant_isolation policy on user_sessions: sessions belonging to TenantA are
// invisible to a caller context stamped with TenantB.
func TestRLS_UserSessions_TenantBSeesNoTenantASessions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	// Seed a user in TenantA that the session will FK-reference.
	ownerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "sess-owner-a@test.local",
		"password_hash": "x",
		"first_name":    "Sess",
		"last_name":     "Owner",
	})
	defer testutil.CleanupRow(t, pool, "users", ownerID)

	sessionID := testutil.SeedRow(t, pool, "user_sessions", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"user_id":   ownerID,
	})
	defer testutil.CleanupRow(t, pool, "user_sessions", sessionID)

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "user_sessions", sessionID, 0)
}

// TestRLS_UserSessions_TenantASeesOwnSessions verifies that a caller in
// TenantA can see session rows that belong to TenantA.
func TestRLS_UserSessions_TenantASeesOwnSessions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	ownerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "sess-owner-a2@test.local",
		"password_hash": "x",
		"first_name":    "Sess2",
		"last_name":     "Owner",
	})
	defer testutil.CleanupRow(t, pool, "users", ownerID)

	sessionID := testutil.SeedRow(t, pool, "user_sessions", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"user_id":   ownerID,
	})
	defer testutil.CleanupRow(t, pool, "user_sessions", sessionID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "user_sessions", sessionID, 1)
}

// TestRLS_UserSessions_SystemContextSeesAll verifies that system context
// bypasses tenant_isolation and can see sessions across tenants — required by
// pre-JWT auth paths (Login, Logout, RefreshToken) that use sysctx.With.
func TestRLS_UserSessions_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	ownerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "sess-owner-a3@test.local",
		"password_hash": "x",
		"first_name":    "Sess3",
		"last_name":     "Owner",
	})
	defer testutil.CleanupRow(t, pool, "users", ownerID)

	sessionID := testutil.SeedRow(t, pool, "user_sessions", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"user_id":   ownerID,
	})
	defer testutil.CleanupRow(t, pool, "user_sessions", sessionID)

	sysCtx := testutil.WithSystemCtx(context.Background())
	testutil.AssertRowCount(t, pool, sysCtx, "user_sessions", sessionID, 1)
}

// TestRLS_UserSessions_CrossTenantUpdateBlocked verifies the WITH CHECK
// clause: TenantB cannot update a TenantA session (zero rows affected).
func TestRLS_UserSessions_CrossTenantUpdateBlocked(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	ownerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "sess-owner-a4@test.local",
		"password_hash": "x",
		"first_name":    "Sess4",
		"last_name":     "Owner",
	})
	defer testutil.CleanupRow(t, pool, "users", ownerID)

	sessionID := testutil.SeedRow(t, pool, "user_sessions", map[string]any{
		"id":        uuid.New(),
		"tenant_id": testutil.TenantA,
		"user_id":   ownerID,
	})
	defer testutil.CleanupRow(t, pool, "user_sessions", sessionID)

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	testutil.AssertUpdateRowsAffected(t, pool, ctxB,
		"UPDATE user_sessions SET device_name = 'Hacked' WHERE id = $1",
		0, sessionID)
}
