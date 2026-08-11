package password

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestPostgresRepository_PasswordHistory exercises AddPasswordHistory and
// GetPasswordHistory against the real database. AddPasswordHistory resolves
// tenant_id server-side via a subquery on users(id) rather than trusting a
// caller-supplied value -- verified directly against the table, not via the
// repository's own read path, so a broken subquery can't hide behind a
// matching read.
func TestPostgresRepository_PasswordHistory(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Password History Repo Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("pw-hist-repo-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	hashes := []string{"hash-oldest", "hash-middle", "hash-newest"}
	for _, h := range hashes {
		if err := repo.AddPasswordHistory(ctx, userID, h); err != nil {
			t.Fatalf("AddPasswordHistory(%s): %v", h, err)
		}
		// created_at has no explicit ordering column beyond the timestamp;
		// separate inserts enough to guarantee distinct values.
		time.Sleep(5 * time.Millisecond)
	}

	var tenantCount int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM password_history WHERE user_id = $1 AND tenant_id = $2`,
		userID, tenantID,
	).Scan(&tenantCount); err != nil {
		t.Fatalf("verify tenant_id resolution: %v", err)
	}
	if tenantCount != 3 {
		t.Fatalf("expected 3 password_history rows resolved to tenant %s, got %d", tenantID, tenantCount)
	}

	got, err := repo.GetPasswordHistory(ctx, userID, 2)
	if err != nil {
		t.Fatalf("GetPasswordHistory(limit=2): %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries (limit), got %d: %v", len(got), got)
	}
	if got[0] != "hash-newest" || got[1] != "hash-middle" {
		t.Fatalf("expected newest-first order [hash-newest, hash-middle], got %v", got)
	}

	all, err := repo.GetPasswordHistory(ctx, userID, 10)
	if err != nil {
		t.Fatalf("GetPasswordHistory(limit=10): %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected all 3 entries, got %d: %v", len(all), all)
	}
	if all[0] != "hash-newest" || all[2] != "hash-oldest" {
		t.Fatalf("expected full newest-first order, got %v", all)
	}

	// Error path: password_history.user_id is NOT NULL REFERENCES users(id) --
	// an unknown user must fail the FK constraint, not silently insert.
	if err := repo.AddPasswordHistory(ctx, uuid.New(), "orphan-hash"); err == nil {
		t.Fatal("expected an FK violation error for AddPasswordHistory with an unknown user_id")
	}
}

// TestPostgresRepository_GetPolicy_DefaultFallback exercises the branch where
// no password_policies row exists for a tenant -- GetPolicy must fall back to
// defaultPolicy() rather than erroring, and the returned policy is not backed
// by any row (there is nothing to update against without calling UpdatePolicy
// first, by contract of the service layer).
func TestPostgresRepository_GetPolicy_DefaultFallback(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Password Policy Default Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	got, err := repo.GetPolicy(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}

	if got.MinLength != 12 || got.MinEntropy != 50.0 || got.PreventReuseCount != 5 {
		t.Fatalf("unexpected default policy values: %+v", got)
	}
	if got.RequireUppercase || got.RequireLowercase || got.RequireDigit || got.RequireSpecial {
		t.Fatalf("default policy should have no complexity requirements: %+v", got)
	}
	if got.MaxAgeDays != nil {
		t.Fatalf("default policy should have no max age, got %v", *got.MaxAgeDays)
	}
	if got.TenantID != tenantID {
		t.Fatalf("default policy TenantID = %s, want %s", got.TenantID, tenantID)
	}
}
