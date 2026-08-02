package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestRLS_RefreshTokens_TenantBSeesNoTenantATokens verifies the standard
// tenant_isolation policy added in migration 000272.
func TestRLS_RefreshTokens_TenantBSeesNoTenantATokens(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	ownerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "rt-owner-a@test.local",
		"password_hash": "x",
		"first_name":    "RT",
		"last_name":     "OwnerA",
	})
	defer testutil.CleanupRow(t, pool, "users", ownerID)

	tokenID := testutil.SeedRow(t, pool, "refresh_tokens", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"user_id":    ownerID,
		"token_hash": "rt-hash-tenant-b-visibility",
		"expires_at": time.Now().Add(time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "refresh_tokens", tokenID)

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	testutil.AssertRowCount(t, pool, ctxB, "refresh_tokens", tokenID, 0)
}

// TestRLS_RefreshTokens_SystemContextSeesAll verifies that the pre-JWT paths
// (RefreshToken, Logout in auth/service.go) can still resolve a token by hash:
// both wrap the whole call in sysctx.With before ctx carries any tenant.
func TestRLS_RefreshTokens_SystemContextSeesAll(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	ownerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "rt-owner-a2@test.local",
		"password_hash": "x",
		"first_name":    "RT2",
		"last_name":     "OwnerA",
	})
	defer testutil.CleanupRow(t, pool, "users", ownerID)

	tokenID := testutil.SeedRow(t, pool, "refresh_tokens", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"user_id":    ownerID,
		"token_hash": "rt-hash-system-context",
		"expires_at": time.Now().Add(time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "refresh_tokens", tokenID)

	repo := auth.NewPostgresRepository(pool)
	sysCtx := testutil.WithSystemCtx(context.Background())
	token, err := repo.GetRefreshTokenByHash(sysCtx, "rt-hash-system-context")
	if err != nil {
		t.Fatalf("GetRefreshTokenByHash under system context: %v", err)
	}
	if token.ID != tokenID {
		t.Fatalf("got token %s, want %s", token.ID, tokenID)
	}
}

// TestRLS_RefreshTokens_CrossTenantRevokeBlocked mirrors the user_sessions
// WITH-CHECK test: TenantB must not be able to revoke a TenantA token by id.
func TestRLS_RefreshTokens_CrossTenantRevokeBlocked(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	ownerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "rt-owner-a3@test.local",
		"password_hash": "x",
		"first_name":    "RT3",
		"last_name":     "OwnerA",
	})
	defer testutil.CleanupRow(t, pool, "users", ownerID)

	tokenID := testutil.SeedRow(t, pool, "refresh_tokens", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"user_id":    ownerID,
		"token_hash": "rt-hash-cross-tenant-revoke",
		"expires_at": time.Now().Add(time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "refresh_tokens", tokenID)

	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)
	testutil.AssertUpdateRowsAffected(t, pool, ctxB,
		"UPDATE refresh_tokens SET revoked = true WHERE id = $1",
		0, tokenID)
}

// TestRefreshTokens_CrossTenantStampedWrite_Rejected exercises the real write
// path (auth.PostgresRepository.StoreRefreshToken, used by createTokenPair at
// issuance) rather than SeedRow: a token stamped for TenantB but written on
// TenantA's connection must fail the policy's WITH CHECK, the same guarantee
// events' CreateEvent gets from not running under system context.
func TestRefreshTokens_CrossTenantStampedWrite_Rejected(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	ownerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "rt-owner-a4@test.local",
		"password_hash": "x",
		"first_name":    "RT4",
		"last_name":     "OwnerA",
	})
	defer testutil.CleanupRow(t, pool, "users", ownerID)

	repo := auth.NewPostgresRepository(pool)
	crossToken := &models.RefreshToken{
		ID:        uuid.New(),
		TenantID:  testutil.TenantB,
		UserID:    ownerID,
		TokenHash: "rt-hash-cross-stamp",
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	err := repo.StoreRefreshToken(ctxA, crossToken)
	if err == nil {
		testutil.CleanupRow(t, pool, "refresh_tokens", crossToken.ID)
		t.Fatal("StoreRefreshToken with a foreign tenant stamp succeeded; the RLS policy is not in force")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Fatalf("expected SQLSTATE 42501, got: %v", err)
	}
}
