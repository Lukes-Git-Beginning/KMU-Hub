package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/sysctx"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestIssuePasswordResetToken_SpendsOnlyTheUsersOwnOpenTokens runs the real
// transaction under RLS: both open tokens of the user are spent and the new one
// stays open, another user's token in the same tenant stays valid, and an insert
// that fails (duplicate hash) rolls the invalidation back, so the user keeps a
// working link.
func TestIssuePasswordResetToken_SpendsOnlyTheUsersOwnOpenTokens(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

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
	issue := func(hash string) error {
		return repo.IssuePasswordResetToken(sysCtx, &models.PasswordResetToken{
			ID:        uuid.New(),
			TenantID:  testutil.TenantA,
			UserID:    owner,
			TokenHash: hash,
			ExpiresAt: time.Now().Add(time.Hour),
			CreatedAt: time.Now(),
		})
	}

	require.Error(t, issue(otherTok), "a duplicate hash must fail the insert")
	require.False(t, used(ownerOld), "a failed insert must not spend the old link")

	fresh := "prt-hash-" + uuid.NewString()
	require.NoError(t, issue(fresh))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM password_reset_tokens WHERE token_hash = $1`, fresh)
	})
	require.True(t, used(ownerOld))
	require.True(t, used(ownerNewer))
	require.False(t, used(fresh), "the newly issued link stays open")
	require.False(t, used(otherTok), "another user's open token stays valid")
}
