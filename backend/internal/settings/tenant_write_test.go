package settings_test

// internal/settings had zero DB-backed tests before this file: service_test.go
// exercises the service against an in-memory fakeRepo, so PostgresRepository's
// SQL never ran against a real, RLS-enforced table. Night 1 added tenant-wide
// module activation here (Migration 000250) on exactly that untested path — the
// same blind spot that left auth/invitations (41bf1080) and hr breaks
// (d78f9176) writing rows that failed on every real call for months.
//
// Every write method on Repository is called for real, once, and checked two
// ways: the row is visible under the caller's own tenant context, and a
// neighbouring tenant sees nothing — even when the query still names the
// writer's tenant_id explicitly, so the assertion is proof of the RLS policy,
// not just of the WHERE clause. None of these five tables (tenant_module_leads,
// user_module_grants, tenant_settings, user_settings,
// tenant_module_activations) have an `id` column — their primary keys are
// composite — so testutil.AssertRowCount does not fit; assertRowCountWhere
// below generalizes the same pattern auth/rls_provisioning_test.go already
// uses for tenant_module_activations.

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/settings"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestSettingsWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	// Dedicated tenants: tenant_module_activations backfills every EXISTING
	// tenant at migration time (000250), so reusing testutil.TenantA/TenantB
	// would start this test with rows already in place instead of the empty
	// slate the assertions below assume.
	//
	// tenant_module_leads/user_module_grants/tenant_settings/user_settings/
	// tenant_module_activations all reference tenants(id) ON DELETE CASCADE, so
	// deleting the tenant row cleans them up (same as
	// auth/rls_provisioning_test.go relies on for tenant_module_activations).
	// users.tenant_id has no such cascade (fk_users_tenant), so the tenant
	// delete below fails until both seeded users are gone first — hence the
	// reverse registration order (defers run LIFO: users, then tenants).
	tenantWrite, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantWrite, "Settings Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Settings Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantWrite)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userWrite := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantWrite,
		"email":         fmt.Sprintf("settings-write-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userWrite)

	adminWrite := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantWrite,
		"email":         fmt.Sprintf("settings-admin-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", adminWrite)

	repo := settings.NewPostgresRepository(pool)
	ctxWrite := testutil.WithTenantCtx(context.Background(), tenantWrite)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	const moduleID = "crm"

	// ------------------------------------------------------------------
	// tenant_module_leads — GrantModuleLead / RevokeModuleLead
	// ------------------------------------------------------------------
	if _, err := repo.GrantModuleLead(ctxWrite, tenantWrite, userWrite, moduleID, &adminWrite); err != nil {
		t.Fatalf("GrantModuleLead: %v", err)
	}
	if _, err := repo.GrantModuleLead(ctxWrite, tenantWrite, adminWrite, moduleID, &adminWrite); err != nil {
		t.Fatalf("GrantModuleLead (second user): %v", err)
	}

	assertRowCountWhere(t, pool, ctxWrite, "tenant_module_leads",
		"tenant_id = $1 AND user_id = $2 AND module_id = $3", 1, tenantWrite, userWrite, moduleID)
	assertRowCountWhere(t, pool, ctxOther, "tenant_module_leads",
		"tenant_id = $1 AND user_id = $2 AND module_id = $3", 0, tenantWrite, userWrite, moduleID)

	if err := repo.RevokeModuleLead(ctxWrite, tenantWrite, adminWrite, moduleID); err != nil {
		t.Fatalf("RevokeModuleLead: %v", err)
	}
	assertRowCountWhere(t, pool, ctxWrite, "tenant_module_leads",
		"tenant_id = $1 AND user_id = $2 AND module_id = $3", 0, tenantWrite, adminWrite, moduleID)
	// The sibling lead survives the revoke of the other — proves the DELETE
	// carries the (tenant_id, user_id, module_id) predicate, not just user_id.
	assertRowCountWhere(t, pool, ctxWrite, "tenant_module_leads",
		"tenant_id = $1 AND user_id = $2 AND module_id = $3", 1, tenantWrite, userWrite, moduleID)

	// ------------------------------------------------------------------
	// user_module_grants — GrantModuleAccess / RevokeModuleAccess / BulkRevokeModuleAccess
	// ------------------------------------------------------------------
	const moduleFinance = "finance"
	if _, err := repo.GrantModuleAccess(ctxWrite, tenantWrite, userWrite, moduleID, &adminWrite); err != nil {
		t.Fatalf("GrantModuleAccess: %v", err)
	}
	if _, err := repo.GrantModuleAccess(ctxWrite, tenantWrite, userWrite, moduleFinance, &adminWrite); err != nil {
		t.Fatalf("GrantModuleAccess (second module): %v", err)
	}
	if _, err := repo.GrantModuleAccess(ctxWrite, tenantWrite, adminWrite, moduleID, &adminWrite); err != nil {
		t.Fatalf("GrantModuleAccess (second user): %v", err)
	}

	assertRowCountWhere(t, pool, ctxWrite, "user_module_grants",
		"tenant_id = $1 AND user_id = $2 AND module_id = $3", 1, tenantWrite, userWrite, moduleID)
	assertRowCountWhere(t, pool, ctxOther, "user_module_grants",
		"tenant_id = $1 AND user_id = $2 AND module_id = $3", 0, tenantWrite, userWrite, moduleID)

	if err := repo.RevokeModuleAccess(ctxWrite, tenantWrite, userWrite, moduleID); err != nil {
		t.Fatalf("RevokeModuleAccess: %v", err)
	}
	assertRowCountWhere(t, pool, ctxWrite, "user_module_grants",
		"tenant_id = $1 AND user_id = $2 AND module_id = $3", 0, tenantWrite, userWrite, moduleID)

	affected, err := repo.BulkRevokeModuleAccess(ctxWrite, tenantWrite, []settings.ModuleGrantRef{
		{UserID: userWrite, ModuleID: moduleFinance},
		{UserID: adminWrite, ModuleID: moduleID},
	})
	if err != nil {
		t.Fatalf("BulkRevokeModuleAccess: %v", err)
	}
	if affected != 2 {
		t.Fatalf("BulkRevokeModuleAccess: expected 2 rows affected, got %d", affected)
	}
	assertRowCountWhere(t, pool, ctxWrite, "user_module_grants",
		"tenant_id = $1", 0, tenantWrite)

	// ------------------------------------------------------------------
	// tenant_settings — PutTenantSettings
	// ------------------------------------------------------------------
	tsEntries, err := repo.PutTenantSettings(ctxWrite, tenantWrite, moduleID, &adminWrite, []*settings.SettingEntry{
		{Key: "workStartHour", Value: json.RawMessage(`8`)},
	})
	if err != nil {
		t.Fatalf("PutTenantSettings: %v", err)
	}
	if len(tsEntries) != 1 || tsEntries[0].Key != "workStartHour" {
		t.Fatalf("PutTenantSettings: unexpected result %+v", tsEntries)
	}
	assertRowCountWhere(t, pool, ctxWrite, "tenant_settings",
		"tenant_id = $1 AND module_id = $2 AND key = $3", 1, tenantWrite, moduleID, "workStartHour")
	assertRowCountWhere(t, pool, ctxOther, "tenant_settings",
		"tenant_id = $1 AND module_id = $2 AND key = $3", 0, tenantWrite, moduleID, "workStartHour")

	got, err := repo.GetTenantSettings(ctxOther, tenantWrite, moduleID)
	if err != nil {
		t.Fatalf("GetTenantSettings (foreign tenant): %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("GetTenantSettings (foreign tenant): expected no entries, got %+v", got)
	}

	// ------------------------------------------------------------------
	// user_settings — PutUserSettings
	// ------------------------------------------------------------------
	usEntries, err := repo.PutUserSettings(ctxWrite, tenantWrite, userWrite, moduleID, []*settings.SettingEntry{
		{Key: "theme", Value: json.RawMessage(`"dark"`)},
	})
	if err != nil {
		t.Fatalf("PutUserSettings: %v", err)
	}
	if len(usEntries) != 1 || usEntries[0].Key != "theme" {
		t.Fatalf("PutUserSettings: unexpected result %+v", usEntries)
	}
	assertRowCountWhere(t, pool, ctxWrite, "user_settings",
		"tenant_id = $1 AND user_id = $2 AND module_id = $3 AND key = $4", 1, tenantWrite, userWrite, moduleID, "theme")
	assertRowCountWhere(t, pool, ctxOther, "user_settings",
		"tenant_id = $1 AND user_id = $2 AND module_id = $3 AND key = $4", 0, tenantWrite, userWrite, moduleID, "theme")

	// ------------------------------------------------------------------
	// tenant_module_activations — SetModuleActivation
	// ------------------------------------------------------------------
	if err := repo.SetModuleActivation(ctxWrite, tenantWrite, moduleID, true, &adminWrite); err != nil {
		t.Fatalf("SetModuleActivation: %v", err)
	}

	assertRowCountWhere(t, pool, ctxWrite, "tenant_module_activations",
		"tenant_id = $1 AND module_id = $2 AND active = TRUE", 1, tenantWrite, moduleID)
	assertRowCountWhere(t, pool, ctxOther, "tenant_module_activations",
		"tenant_id = $1 AND module_id = $2", 0, tenantWrite, moduleID)

	activations, err := repo.ListModuleActivations(ctxOther, tenantWrite)
	if err != nil {
		t.Fatalf("ListModuleActivations (foreign tenant): %v", err)
	}
	if len(activations) != 0 {
		t.Fatalf("ListModuleActivations (foreign tenant): expected empty, got %+v", activations)
	}
}

// assertRowCountWhere runs `SELECT count(*) FROM <table> WHERE <where>` under
// the given ctx (which determines RLS visibility via the app.tenant_id GUC)
// and fails if the count differs. Generalizes assertModuleCount from
// auth/rls_provisioning_test.go for tables keyed by a composite primary key
// instead of a single `id` column.
func assertRowCountWhere(t *testing.T, pool *pgxpool.Pool, ctx context.Context, table, where string, expected int, args ...any) {
	t.Helper()
	var count int
	q := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, where)
	if err := pool.QueryRow(ctx, q, args...).Scan(&count); err != nil {
		t.Fatalf("count %s (%s): %v", table, where, err)
	}
	if count != expected {
		t.Fatalf("table %s (%s): expected %d row(s), got %d", table, where, expected, count)
	}
}
