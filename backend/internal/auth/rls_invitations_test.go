package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestRLS_Invitations_TenantBSeesNoTenantAInvitation is the read half of the
// gap migration 000249 closes: before it, invitations carried no tenant at all
// and every admin saw every tenant's pending invitations, addresses included.
func TestRLS_Invitations_TenantBSeesNoTenantAInvitation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	inviterID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "inviter-rls@test.local",
		"password_hash": "x",
		"first_name":    "Inviter",
		"last_name":     "RLS",
	})
	defer testutil.CleanupRow(t, pool, "users", inviterID)

	invID := testutil.SeedRow(t, pool, "invitations", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"email":      "invited-rls@test.local",
		"role":       "member",
		"token_hash": uuid.NewString(),
		"created_by": inviterID,
		"expires_at": time.Now().Add(24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "invitations", invID)

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "invitations", invID, 0)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	testutil.AssertRowCount(t, pool, ctxA, "invitations", invID, 1)
}

// TestRLS_Invitations_SystemContextSeesAll covers the accept path: it runs
// before a JWT exists, so it reads the invitation under the system context.
// Without the is_system_context() branch no token could ever be redeemed.
func TestRLS_Invitations_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	inviterID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "inviter-sys@test.local",
		"password_hash": "x",
		"first_name":    "Inviter",
		"last_name":     "Sys",
	})
	defer testutil.CleanupRow(t, pool, "users", inviterID)

	invID := testutil.SeedRow(t, pool, "invitations", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"email":      "invited-sys@test.local",
		"role":       "member",
		"token_hash": uuid.NewString(),
		"created_by": inviterID,
		"expires_at": time.Now().Add(24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "invitations", invID)

	sysCtx := testutil.WithSystemCtx(context.Background())
	testutil.AssertRowCount(t, pool, sysCtx, "invitations", invID, 1)
}

// TestRLS_Invitations_TenantBCannotClaimTenantAInvitation is the write half:
// the accept path claims an invitation with an UPDATE, and a foreign tenant
// must not be able to move that row.
func TestRLS_Invitations_TenantBCannotClaimTenantAInvitation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	inviterID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "inviter-claim@test.local",
		"password_hash": "x",
		"first_name":    "Inviter",
		"last_name":     "Claim",
	})
	defer testutil.CleanupRow(t, pool, "users", inviterID)

	invID := testutil.SeedRow(t, pool, "invitations", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"email":      "invited-claim@test.local",
		"role":       "member",
		"token_hash": uuid.NewString(),
		"created_by": inviterID,
		"expires_at": time.Now().Add(24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "invitations", invID)

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	testutil.AssertUpdateRowsAffected(t, pool, ctxB,
		"UPDATE invitations SET accepted_at = NOW() WHERE id = $1 AND accepted_at IS NULL", 0, invID)
}
