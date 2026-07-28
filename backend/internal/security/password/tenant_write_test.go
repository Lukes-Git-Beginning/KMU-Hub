package password

// wp-security: closes the write-surface gap for the password policy. UpdatePolicy
// took a client-controlled row ID with no tenant_id predicate at all (the worst
// case in this audit: the ID came straight from the request, not a
// server-resolved lookup); GetPolicy ran with no tenant filter, relying purely
// on RLS to return exactly one row. Both now carry an explicit tenant_id
// predicate, fixed in the same commit as this test.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPasswordPolicyWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Password Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Password Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("wp-security-password-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
	// Deferred before the policy rows below so it runs after their cleanup
	// (LIFO) -- password_policies.updated_by references users, so the user
	// row must outlive any policy row that points at it.
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	// Seed one policy row per tenant directly -- the repository never inserts
	// a row itself (GetPolicy falls back to an in-memory default), so a write
	// test against UpdatePolicy needs a real row to update.
	policyOwn := testutil.SeedRow(t, pool, "password_policies", map[string]any{
		"tenant_id":           tenantOwn,
		"min_length":          12,
		"min_entropy":         50.0,
		"prevent_reuse_count": 5,
	})
	defer testutil.CleanupRow(t, pool, "password_policies", policyOwn)
	policyOther := testutil.SeedRow(t, pool, "password_policies", map[string]any{
		"tenant_id":           tenantOther,
		"min_length":          12,
		"min_entropy":         50.0,
		"prevent_reuse_count": 5,
	})
	defer testutil.CleanupRow(t, pool, "password_policies", policyOther)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	// GetPolicy is explicitly tenant-scoped: each tenant must see only its own row.
	got, err := repo.GetPolicy(ctxOwn, tenantOwn)
	if err != nil {
		t.Fatalf("GetPolicy (own ctx): %v", err)
	}
	if got.ID != policyOwn {
		t.Fatalf("GetPolicy (own ctx): got wrong policy %s, want %s", got.ID, policyOwn)
	}

	gotOther, err := repo.GetPolicy(ctxOther, tenantOther)
	if err != nil {
		t.Fatalf("GetPolicy (other ctx): %v", err)
	}
	if gotOther.ID != policyOther {
		t.Fatalf("GetPolicy (other ctx): got wrong policy %s, want %s", gotOther.ID, policyOther)
	}

	// UpdatePolicy carries an explicit tenant_id predicate — an attacker who
	// somehow supplies the victim's policy ID from their own tenant session
	// must not be able to mutate it.
	attack := &models.PasswordPolicy{
		ID:                policyOwn,
		MinLength:         4,
		MinEntropy:        0,
		PreventReuseCount: 0,
		UpdatedAt:         time.Now().UTC(),
	}
	if err := repo.UpdatePolicy(ctxOther, tenantOther, attack); err != nil {
		t.Fatalf("UpdatePolicy (foreign ctx): unexpected error: %v", err)
	}
	got, err = repo.GetPolicy(ctxOwn, tenantOwn)
	if err != nil {
		t.Fatalf("GetPolicy after foreign update: %v", err)
	}
	if got.MinLength == 4 {
		t.Fatalf("a foreign-tenant write reached the password policy: min_length=%d", got.MinLength)
	}

	legit := &models.PasswordPolicy{
		ID:                policyOwn,
		MinLength:         16,
		MinEntropy:        60,
		PreventReuseCount: 8,
		UpdatedAt:         time.Now().UTC(),
	}
	if err := repo.UpdatePolicy(ctxOwn, tenantOwn, legit); err != nil {
		t.Fatalf("UpdatePolicy (own ctx): %v", err)
	}
	got, err = repo.GetPolicy(ctxOwn, tenantOwn)
	if err != nil {
		t.Fatalf("GetPolicy after own update: %v", err)
	}
	if got.MinLength != 16 {
		t.Fatalf("own-tenant write did not land: min_length=%d", got.MinLength)
	}

	// Service.UpdatePolicy resolves the row ID server-side, scoped to the
	// caller's tenant -- a caller-supplied ID pointing at another tenant's
	// row must never be trusted.
	svc := NewService(repo)
	spoofed := &models.PasswordPolicy{ID: policyOther, MinLength: 99}
	if err := svc.UpdatePolicy(ctxOwn, spoofed, userOwn); err != nil {
		t.Fatalf("Service.UpdatePolicy (own ctx): %v", err)
	}
	gotOther, err = repo.GetPolicy(ctxOther, tenantOther)
	if err != nil {
		t.Fatalf("GetPolicy (other tenant) after spoofed service update: %v", err)
	}
	if gotOther.MinLength == 99 {
		t.Fatalf("Service.UpdatePolicy let a caller-supplied ID target another tenant's policy")
	}
	got, err = repo.GetPolicy(ctxOwn, tenantOwn)
	if err != nil {
		t.Fatalf("GetPolicy (own tenant) after spoofed service update: %v", err)
	}
	if got.MinLength != 99 {
		t.Fatalf("Service.UpdatePolicy did not redirect the write to the caller's own policy row")
	}
}
