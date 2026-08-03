package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// dashboard_defaults is unique on (tenant_id, role) since migration 000274, so
// these tests seed their own tenants rather than sharing testutil.TenantA/
// TenantB: a parallel test saving the same role for a shared tenant would
// collide on that index. Fixed ids keep repeated runs from adding a tenant row
// every time — EnsureTenant is idempotent.
var (
	ddTenantOne = uuid.MustParse("dd500000-0000-0000-0000-000000000001")
	ddTenantTwo = uuid.MustParse("dd500000-0000-0000-0000-000000000002")
)

func seedDashboardTenants(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	testutil.EnsureTenant(t, pool, ddTenantOne, "Dashboard Defaults Tenant One")
	testutil.EnsureTenant(t, pool, ddTenantTwo, "Dashboard Defaults Tenant Two")
}

// TestRLS_DashboardDefaults_ForeignTenantSeesNothing is the standard isolation
// check for the tenant_isolation policy migration 000274 added.
func TestRLS_DashboardDefaults_ForeignTenantSeesNothing(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	seedDashboardTenants(t, pool)

	defaultID := testutil.SeedRow(t, pool, "dashboard_defaults", map[string]any{
		"id":             uuid.New(),
		"tenant_id":      ddTenantOne,
		"role":           "isolation-probe",
		"layout":         `[{"i":"probe","x":0,"y":0,"w":1,"h":1}]`,
		"active_widgets": `["probe"]`,
	})
	defer testutil.CleanupRow(t, pool, "dashboard_defaults", defaultID)

	ctxOne := testutil.WithTenantCtx(context.Background(), ddTenantOne)
	testutil.AssertRowCount(t, pool, ctxOne, "dashboard_defaults", defaultID, 1)

	ctxTwo := testutil.WithTenantCtx(context.Background(), ddTenantTwo)
	testutil.AssertRowCount(t, pool, ctxTwo, "dashboard_defaults", defaultID, 0)
}

// TestDashboardDefaults_TenantsHoldIndependentPresets is the bug 000274 exists
// for. role was globally unique and UpsertDefaultLayout upserted on it, so the
// second tenant's PUT /admin/dashboard/defaults/{role} did not create a row —
// it overwrote the first tenant's preset for everyone.
func TestDashboardDefaults_TenantsHoldIndependentPresets(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	seedDashboardTenants(t, pool)
	repo := NewPostgresDashboardRepository(pool)

	ctxOne := testutil.WithTenantCtx(context.Background(), ddTenantOne)
	one, err := repo.UpsertDefaultLayout(ctxOne, ddTenantOne, &models.DashboardDefault{
		Role:          "manager",
		Layout:        json.RawMessage(`[{"i":"one","x":0,"y":0,"w":1,"h":1}]`),
		ActiveWidgets: json.RawMessage(`["one"]`),
	})
	if err != nil {
		t.Fatalf("upsert for tenant one: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "dashboard_defaults", uuid.MustParse(one.ID))

	ctxTwo := testutil.WithTenantCtx(context.Background(), ddTenantTwo)
	two, err := repo.UpsertDefaultLayout(ctxTwo, ddTenantTwo, &models.DashboardDefault{
		Role:          "manager",
		Layout:        json.RawMessage(`[{"i":"two","x":0,"y":0,"w":2,"h":2}]`),
		ActiveWidgets: json.RawMessage(`["two"]`),
	})
	if err != nil {
		t.Fatalf("upsert for tenant two: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "dashboard_defaults", uuid.MustParse(two.ID))

	if one.ID == two.ID {
		t.Fatal("both tenants share one row — the upsert still conflicts on role alone")
	}

	readBack, err := repo.GetDefaultLayout(ctxOne, ddTenantOne, "manager")
	if err != nil {
		t.Fatalf("read back tenant one: %v", err)
	}
	if !json.Valid(readBack.ActiveWidgets) || string(readBack.ActiveWidgets) != `["one"]` {
		t.Fatalf("tenant one preset: want [\"one\"], got %s — the second tenant's write leaked", readBack.ActiveWidgets)
	}
}

// TestDashboardDefaults_ForeignTenantReadIsNotFound checks the read side. A
// missing tenant filter here would have surfaced another tenant's preset for a
// role the caller never configured.
func TestDashboardDefaults_ForeignTenantReadIsNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	seedDashboardTenants(t, pool)
	repo := NewPostgresDashboardRepository(pool)

	ctxOne := testutil.WithTenantCtx(context.Background(), ddTenantOne)
	own, err := repo.UpsertDefaultLayout(ctxOne, ddTenantOne, &models.DashboardDefault{
		Role:          "member",
		Layout:        json.RawMessage(`[{"i":"own","x":0,"y":0,"w":1,"h":1}]`),
		ActiveWidgets: json.RawMessage(`["own"]`),
	})
	if err != nil {
		t.Fatalf("upsert for tenant one: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "dashboard_defaults", uuid.MustParse(own.ID))

	// Same role, other tenant, no row of its own.
	ctxTwo := testutil.WithTenantCtx(context.Background(), ddTenantTwo)
	if _, err := repo.GetDefaultLayout(ctxTwo, ddTenantTwo, "member"); !errors.Is(err, ErrDashboardNotFound) {
		t.Fatalf("foreign tenant: want ErrDashboardNotFound, got %v", err)
	}
}
