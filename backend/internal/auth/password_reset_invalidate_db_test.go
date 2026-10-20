package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/sysctx"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestInvalidateOpenPasswordResetTokens_SpendsOnlyTheUsersOwnOpenTokens runs the
// real UPDATE under RLS: both open tokens of the user are spent, another user's
// token in the same tenant stays open, and a call carrying the wrong tenant
// touches nothing (the reset path runs under sysctx, so the tenant_id in the
// WHERE clause is the only guard).
func TestInvalidateOpenPasswordResetTokens_SpendsOnlyTheUsersOwnOpenTokens(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	newUser := func(label string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"id":            uuid.New(),
			"tenant_id":     testutil.TenantA,
			"email":         "prt-" + label + "-" + uuid.NewString() + "@test.local",
			"password_hash": "x",
			"first_name":    "PRT",
			"last_name":     label,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	newToken := func(userID uuid.UUID) string {
		hash := "prt-hash-" + uuid.NewString()
		id := testutil.SeedRow(t, pool, "password_reset_tokens", map[string]any{
			"id":         uuid.New(),
			"tenant_id":  testutil.TenantA,
			"user_id":    userID,
			"token_hash": hash,
			"expires_at": time.Now().Add(time.Hour),
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "password_reset_tokens", id) })
		return hash
	}

	owner, other := newUser("owner"), newUser("other")
	ownerOld, ownerNewer, otherTok := newToken(owner), newToken(owner), newToken(other)

	repo := auth.NewPostgresRepository(pool)
	sysCtx := sysctx.With(context.Background())
	used := func(hash string) bool {
		tok, err := repo.GetPasswordResetToken(sysCtx, hash)
		require.NoError(t, err)
		return tok.UsedAt != nil
	}

	require.NoError(t, repo.InvalidateOpenPasswordResetTokens(sysCtx, testutil.TenantB, owner))
	require.False(t, used(ownerOld), "a foreign tenant id must not spend the token")

	require.NoError(t, repo.InvalidateOpenPasswordResetTokens(sysCtx, testutil.TenantA, owner))
	require.True(t, used(ownerOld))
	require.True(t, used(ownerNewer))
	require.False(t, used(otherTok), "another user's open token stays valid")
}
