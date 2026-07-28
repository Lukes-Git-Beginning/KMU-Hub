package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestRLS_ProvisionedTenant_IsIsolatedFromExistingData provisions a tenant the
// way the route does — through the real repository, in system context — and
// then checks isolation in both directions from the outside.
//
// This is the check the unit exists for: provisioning is the one write path
// that creates a tenant, so a mistake here does not corrupt one record, it
// decides whether every record the tenant ever writes is separated from
// everybody else's.
func TestRLS_ProvisionedTenant_IsIsolatedFromExistingData(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	// The platform operator lives in an existing tenant — provisioning writes
	// entirely outside their own.
	operatorID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "operator-provisioning@test.local",
		"password_hash": "x",
		"first_name":    "Platform",
		"last_name":     "Operator",
	})
	defer testutil.CleanupRow(t, pool, "users", operatorID)

	// A row belonging to TenantA that the new tenant must never see.
	foreignInvitation := testutil.SeedRow(t, pool, "invitations", map[string]any{
		"id":         uuid.New(),
		"tenant_id":  testutil.TenantA,
		"email":      "foreign-provisioning@test.local",
		"role":       "member",
		"token_hash": uuid.NewString(),
		"created_by": operatorID,
		"expires_at": time.Now().Add(24 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "invitations", foreignInvitation)

	repo := auth.NewPostgresRepository(pool)
	svc := auth.NewService(repo, auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour))

	res, err := svc.ProvisionTenant(testutil.WithSystemCtx(context.Background()), auth.ProvisionTenantInput{
		Name:       "RLS Provisioning Test GmbH",
		AdminEmail: "admin-provisioning@test.local",
		Modules:    []string{"crm", "tasks"},
		CreatedBy:  operatorID,
	})
	if err != nil {
		t.Fatalf("provision tenant: %v", err)
	}
	newTenant := res.Tenant.ID

	// Cleanup cascades from the tenant row: module activations and the
	// invitation both reference tenants(id) ON DELETE CASCADE.
	defer testutil.CleanupRow(t, pool, "tenants", newTenant)

	ctxNew := testutil.WithTenantCtx(context.Background(), newTenant)
	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)

	// 1) The fresh tenant sees none of TenantA's data.
	testutil.AssertRowCount(t, pool, ctxNew, "invitations", foreignInvitation, 0)

	// 2) TenantA sees none of the fresh tenant's data.
	testutil.AssertRowCount(t, pool, ctxA, "invitations", res.Invitation.ID, 0)
	testutil.AssertRowCount(t, pool, ctxA, "tenants", newTenant, 0)

	// 3) The fresh tenant does see its own — isolation, not a blank tenant.
	testutil.AssertRowCount(t, pool, ctxNew, "invitations", res.Invitation.ID, 1)
	testutil.AssertRowCount(t, pool, ctxNew, "tenants", newTenant, 1)

	// 4) Module activations landed and are scoped. AssertRowCount keys on id,
	//    which tenant_module_activations does not have, so count directly.
	assertModuleCount(t, pool, ctxNew, newTenant, 2)
	assertModuleCount(t, pool, ctxA, newTenant, 0)

	// 5) A write from the neighbouring tenant reaches nothing.
	testutil.AssertUpdateRowsAffected(t, pool, ctxA,
		"UPDATE tenant_module_activations SET active = FALSE WHERE tenant_id = $1", 0, newTenant)
}

// assertModuleCount counts activations directly: tenant_module_activations is
// keyed by (tenant_id, module_id), so testutil.AssertRowCount — which keys on
// an id column — does not fit.
func assertModuleCount(t *testing.T, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID, expected int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tenant_module_activations WHERE tenant_id = $1`, tenantID,
	).Scan(&count); err != nil {
		t.Fatalf("count module activations: %v", err)
	}
	if count != expected {
		t.Errorf("tenant_module_activations for %s: got %d, want %d", tenantID, count, expected)
	}
}
