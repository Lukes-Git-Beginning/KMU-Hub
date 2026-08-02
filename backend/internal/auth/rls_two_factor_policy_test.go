package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/sysctx"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// two_factor_policy is unique on (tenant_id, role_name) since migration 000273,
// so these tests cannot share testutil.TenantA/TenantB with the rest of the
// suite: a parallel test seeding "admin" for the same tenant would collide on
// the index. Fixed ids rather than uuid.New() keep repeated runs from adding a
// new tenant row every time — EnsureTenant is idempotent.
var (
	tfpTenantOne = uuid.MustParse("2fa00000-0000-0000-0000-000000000001")
	tfpTenantTwo = uuid.MustParse("2fa00000-0000-0000-0000-000000000002")
)

func seedTwoFactorTenants(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	testutil.EnsureTenant(t, pool, tfpTenantOne, "2FA Policy Tenant One")
	testutil.EnsureTenant(t, pool, tfpTenantTwo, "2FA Policy Tenant Two")
}

// TestRLS_TwoFactorPolicy_ForeignTenantSeesNothing is the standard isolation
// check for the tenant_isolation policy migration 000273 added.
func TestRLS_TwoFactorPolicy_ForeignTenantSeesNothing(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	seedTwoFactorTenants(t, pool)

	policyID := testutil.SeedRow(t, pool, "two_factor_policy", map[string]any{
		"id":                uuid.New(),
		"tenant_id":         tfpTenantOne,
		"role_name":         "isolation-probe",
		"enforced":          true,
		"grace_period_days": 3,
	})
	defer testutil.CleanupRow(t, pool, "two_factor_policy", policyID)

	ctxOne := testutil.WithTenantCtx(context.Background(), tfpTenantOne)
	testutil.AssertRowCount(t, pool, ctxOne, "two_factor_policy", policyID, 1)

	ctxTwo := testutil.WithTenantCtx(context.Background(), tfpTenantTwo)
	testutil.AssertRowCount(t, pool, ctxTwo, "two_factor_policy", policyID, 0)
}

// TestTwoFactorPolicy_TenantsHoldIndependentPolicies is the bug migration 000273
// exists for. role_name used to be globally unique, so the second tenant's
// upsert did not create a row — it overwrote the first tenant's, silently
// switching off their 2FA enforcement. Both rows must now coexist untouched.
func TestTwoFactorPolicy_TenantsHoldIndependentPolicies(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	seedTwoFactorTenants(t, pool)
	repo := auth.NewPostgresRepository(pool)

	strict := &models.TwoFactorPolicy{
		ID: uuid.New(), TenantID: tfpTenantOne, RoleName: "admin",
		Enforced: true, GracePeriodDays: 7, UpdatedAt: time.Now(),
	}
	ctxOne := testutil.WithTenantCtx(context.Background(), tfpTenantOne)
	if err := repo.UpsertTwoFactorPolicy(ctxOne, strict); err != nil {
		t.Fatalf("upsert for tenant one: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "two_factor_policy", strict.ID)

	lax := &models.TwoFactorPolicy{
		ID: uuid.New(), TenantID: tfpTenantTwo, RoleName: "admin",
		Enforced: false, GracePeriodDays: 30, UpdatedAt: time.Now(),
	}
	ctxTwo := testutil.WithTenantCtx(context.Background(), tfpTenantTwo)
	if err := repo.UpsertTwoFactorPolicy(ctxTwo, lax); err != nil {
		t.Fatalf("upsert for tenant two: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "two_factor_policy", lax.ID)

	got, err := repo.GetTwoFactorPolicy(ctxOne, tfpTenantOne, "admin")
	if err != nil {
		t.Fatalf("read back tenant one: %v", err)
	}
	if got == nil {
		t.Fatal("tenant one lost its policy to tenant two's write")
		return
	}
	if !got.Enforced || got.GracePeriodDays != 7 {
		t.Fatalf("tenant one's policy was overwritten: enforced=%v grace=%d", got.Enforced, got.GracePeriodDays)
	}
}

// TestTwoFactorPolicy_CrossTenantStampedWrite_Rejected exercises the real write
// path with a foreign tenant stamp: the policy's WITH CHECK must reject it, so
// a caller cannot write into another tenant by forging the column.
func TestTwoFactorPolicy_CrossTenantStampedWrite_Rejected(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	seedTwoFactorTenants(t, pool)
	repo := auth.NewPostgresRepository(pool)

	cross := &models.TwoFactorPolicy{
		ID: uuid.New(), TenantID: tfpTenantTwo, RoleName: "cross-stamp-probe",
		Enforced: false, GracePeriodDays: 99, UpdatedAt: time.Now(),
	}

	ctxOne := testutil.WithTenantCtx(context.Background(), tfpTenantOne)
	err := repo.UpsertTwoFactorPolicy(ctxOne, cross)
	if err == nil {
		testutil.CleanupRow(t, pool, "two_factor_policy", cross.ID)
		t.Fatal("upsert with a foreign tenant stamp succeeded; the RLS policy is not in force")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Fatalf("expected SQLSTATE 42501, got: %v", err)
	}
}

// TestTwoFactorPolicy_SystemContextReadStaysTenantScoped covers the one path
// RLS cannot: Login wraps everything in sysctx.With because the user lookup
// happens before a tenant exists on the context, and under the system context
// the policy predicate admits every row. Only the explicit tenant_id filter in
// the query keeps Check2FAEnforcement from judging a login against a stranger's
// 2FA rules.
func TestTwoFactorPolicy_SystemContextReadStaysTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	seedTwoFactorTenants(t, pool)
	repo := auth.NewPostgresRepository(pool)

	foreign := testutil.SeedRow(t, pool, "two_factor_policy", map[string]any{
		"id":                uuid.New(),
		"tenant_id":         tfpTenantTwo,
		"role_name":         "sysctx-probe",
		"enforced":          true,
		"grace_period_days": 1,
	})
	defer testutil.CleanupRow(t, pool, "two_factor_policy", foreign)

	sysCtx := sysctx.With(context.Background())

	got, err := repo.GetTwoFactorPolicy(sysCtx, tfpTenantOne, "sysctx-probe")
	if err != nil {
		t.Fatalf("read under system context: %v", err)
	}
	if got != nil {
		t.Fatalf("tenant one read tenant two's policy under the system context: %+v", got)
	}

	list, err := repo.ListTwoFactorPolicies(sysCtx, tfpTenantOne)
	if err != nil {
		t.Fatalf("list under system context: %v", err)
	}
	for _, p := range list {
		if p.TenantID != tfpTenantOne {
			t.Fatalf("list under system context leaked tenant %s", p.TenantID)
		}
	}

	// The same read for the owning tenant still resolves — the filter scopes,
	// it does not blind the login path.
	own, err := repo.GetTwoFactorPolicy(sysCtx, tfpTenantTwo, "sysctx-probe")
	if err != nil {
		t.Fatalf("read own policy under system context: %v", err)
	}
	if own == nil || own.ID != foreign {
		t.Fatal("the owning tenant could not read its own policy under the system context")
	}
}
