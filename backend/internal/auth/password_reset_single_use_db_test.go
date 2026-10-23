package auth_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/sysctx"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedResetToken creates a user and one open reset token for it and returns the
// user id plus the plain token a client would hold.
func seedResetToken(t *testing.T, pool *pgxpool.Pool, expiresIn time.Duration) (uuid.UUID, string) {
	t.Helper()
	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "prt-single-" + uuid.NewString() + "@test.local",
		"password_hash": "old-hash",
		"first_name":    "PRT",
		"last_name":     "SingleUse",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	plain := "plain-" + uuid.NewString()
	tokenID := testutil.SeedRow(t, pool, "password_reset_tokens", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"user_id":    userID,
		"token_hash": auth.HashToken(plain),
		"expires_at": time.Now().Add(expiresIn),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "password_reset_tokens", tokenID) })
	return userID, plain
}

// TestResetPassword_TokenIsSingleUseUnderConcurrency runs the real service
// against Postgres: of N simultaneous requests carrying the same token exactly
// one sets a password, the rest are rejected as already used.
func TestResetPassword_TokenIsSingleUseUnderConcurrency(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	_, plain := seedResetToken(t, pool, time.Hour)

	repo := auth.NewPostgresRepository(pool)
	svc := auth.NewService(repo, auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, time.Hour))

	const n = 8
	errs := make([]error, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = svc.ResetPassword(context.Background(), plain, "Brand-New-Pass-99!")
		}()
	}
	close(start)
	wg.Wait()

	wins := 0
	for _, err := range errs {
		if err == nil {
			wins++
		} else {
			assert.ErrorIs(t, err, auth.ErrResetTokenUsed)
		}
	}
	assert.Equal(t, 1, wins, "exactly one concurrent reset may succeed")
}

// TestResetPasswordWithToken_FailedPasswordWriteDoesNotBurnTheToken: a NUL byte
// is not storable in a text column, so the users UPDATE fails after the token
// UPDATE already ran; the rollback must leave the token redeemable.
func TestResetPasswordWithToken_FailedPasswordWriteDoesNotBurnTheToken(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	userID, plain := seedResetToken(t, pool, time.Hour)
	repo := auth.NewPostgresRepository(pool)
	sysCtx := sysctx.With(context.Background())

	tok, err := repo.GetPasswordResetToken(sysCtx, auth.HashToken(plain))
	require.NoError(t, err)

	err = repo.ResetPasswordWithToken(sysCtx, tok.ID, tok.TenantID, userID, "bad\x00hash")
	require.Error(t, err)

	tok, err = repo.GetPasswordResetToken(sysCtx, auth.HashToken(plain))
	require.NoError(t, err)
	assert.Nil(t, tok.UsedAt, "a failed password write must not spend the token")

	require.NoError(t, repo.ResetPasswordWithToken(sysCtx, tok.ID, tok.TenantID, userID, "good-hash"))
	assert.ErrorIs(t, repo.ResetPasswordWithToken(sysCtx, tok.ID, tok.TenantID, userID, "other-hash"), auth.ErrResetTokenUsed)

	u, err := repo.GetUserByID(sysCtx, userID)
	require.NoError(t, err)
	assert.Equal(t, "good-hash", u.PasswordHash)
}

// TestResetPasswordWithToken_ExpiredTokenIsRejected: the expiry check lives in
// the UPDATE itself, so a token that ran out between the service's read and the
// write is refused with ErrResetTokenExpired and the password stays untouched.
func TestResetPasswordWithToken_ExpiredTokenIsRejected(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	userID, plain := seedResetToken(t, pool, -time.Minute)
	repo := auth.NewPostgresRepository(pool)
	sysCtx := sysctx.With(context.Background())

	tok, err := repo.GetPasswordResetToken(sysCtx, auth.HashToken(plain))
	require.NoError(t, err)

	assert.ErrorIs(t, repo.ResetPasswordWithToken(sysCtx, tok.ID, tok.TenantID, userID, "new-hash"), auth.ErrResetTokenExpired)
	// A foreign tenant id must not match the token at all.
	assert.ErrorIs(t, repo.ResetPasswordWithToken(sysCtx, tok.ID, testutil.TenantB, userID, "new-hash"), auth.ErrResetTokenInvalid)

	u, err := repo.GetUserByID(sysCtx, userID)
	require.NoError(t, err)
	assert.Equal(t, "old-hash", u.PasswordHash)
}
