package hr

// Cross-tenant isolation tests for HR tables (Migration 000123).
//
// Standard tables (tenant_isolation policy):
//   hr_company_settings, hr_leave_requests, hr_leave_balances,
//   hr_work_time_entries, hr_employee_documents, hr_employee_profiles
//
// Custom-policy tables (system-seed rows visible to all tenants):
//   hr_leave_types, hr_document_categories
//
// System-seed rows (tenant_id = '00000000-0000-0000-0000-000000000000') must be
// visible to BOTH tenants; tenant-specific rows must be invisible to other tenants.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

var zeroTenant = uuid.MustParse("00000000-0000-0000-0000-000000000000")

func TestTenantIsolation_HR_Standard(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA — employee_id FK in hr_leave_requests/balances/work_time_entries all reference users(id).
	employeeID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("hr-emp-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", employeeID)

	// hr_company_settings — one per tenant.
	hrSettingsID := testutil.SeedRow(t, pool, "hr_company_settings", map[string]any{
		"tenant_id": testutil.TenantA,
	})
	defer testutil.CleanupRow(t, pool, "hr_company_settings", hrSettingsID)

	// hr_leave_types — seed a per-tenant type (not a system seed).
	leaveTypeID := testutil.SeedRow(t, pool, "hr_leave_types", map[string]any{
		"tenant_id": testutil.TenantA,
		"name":      "Betriebsurlaub " + uuid.New().String()[:6],
		"key":       "betrieb_" + uuid.New().String()[:6],
	})
	defer testutil.CleanupRow(t, pool, "hr_leave_types", leaveTypeID)

	// hr_leave_requests — employee_id FK to users(id) NOT NULL.
	leaveReqID := testutil.SeedRow(t, pool, "hr_leave_requests", map[string]any{
		"tenant_id":     testutil.TenantA,
		"employee_id":   employeeID,
		"leave_type_id": leaveTypeID,
		"start_date":    time.Now().UTC().Format("2006-01-02"),
		"end_date":      time.Now().UTC().Add(3 * 24 * time.Hour).Format("2006-01-02"),
		"total_days":    "3.0",
	})
	defer testutil.CleanupRow(t, pool, "hr_leave_requests", leaveReqID)

	// hr_leave_balances — employee_id FK to users(id) NOT NULL.
	leaveBal := testutil.SeedRow(t, pool, "hr_leave_balances", map[string]any{
		"tenant_id":   testutil.TenantA,
		"employee_id": employeeID,
		"year":        2026,
		"entitlement": "20.0",
		"remaining":   "20.0",
	})
	defer testutil.CleanupRow(t, pool, "hr_leave_balances", leaveBal)

	// hr_work_time_entries — employee_id FK to users(id) NOT NULL.
	wteID := testutil.SeedRow(t, pool, "hr_work_time_entries", map[string]any{
		"tenant_id":   testutil.TenantA,
		"employee_id": employeeID,
		"clock_in":    time.Now().UTC(),
	})
	defer testutil.CleanupRow(t, pool, "hr_work_time_entries", wteID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	standardTests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"hr_company_settings", "hr_company_settings", hrSettingsID},
		{"hr_leave_types_per_tenant", "hr_leave_types", leaveTypeID},
		{"hr_leave_requests", "hr_leave_requests", leaveReqID},
		{"hr_leave_balances", "hr_leave_balances", leaveBal},
		{"hr_work_time_entries", "hr_work_time_entries", wteID},
	}

	for _, tc := range standardTests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}

// TestTenantIsolation_HR_SystemSeedRows verifies that system-seed rows (zero-UUID tenant)
// in hr_leave_types and hr_document_categories are visible to ALL tenants.
// This validates the custom USING policy in mig 000123.
func TestTenantIsolation_HR_SystemSeedRows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a hr_leave_type with zero-UUID tenant (system seed pattern).
	sysLeaveTypeID := testutil.SeedRow(t, pool, "hr_leave_types", map[string]any{
		"tenant_id": zeroTenant,
		"name":      "System Leave " + uuid.New().String()[:6],
		"key":       "sys_" + uuid.New().String()[:6],
	})
	defer testutil.CleanupRow(t, pool, "hr_leave_types", sysLeaveTypeID)

	// Seed a hr_document_category with zero-UUID tenant.
	sysDocCatID := testutil.SeedRow(t, pool, "hr_document_categories", map[string]any{
		"tenant_id":  zeroTenant,
		"name":       "System Category " + uuid.New().String()[:6],
		"key":        "syscat_" + uuid.New().String()[:6],
		"visibility": "hr_only",
	})
	defer testutil.CleanupRow(t, pool, "hr_document_categories", sysDocCatID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	// Both tenants must see system-seed rows (zero-UUID tenant_id permitted by USING clause).
	t.Run("hr_leave_types_system_seed_visible_to_A", func(t *testing.T) {
		testutil.AssertRowCount(t, pool, ctxA, "hr_leave_types", sysLeaveTypeID, 1)
	})
	t.Run("hr_leave_types_system_seed_visible_to_B", func(t *testing.T) {
		testutil.AssertRowCount(t, pool, ctxB, "hr_leave_types", sysLeaveTypeID, 1)
	})
	t.Run("hr_document_categories_system_seed_visible_to_A", func(t *testing.T) {
		testutil.AssertRowCount(t, pool, ctxA, "hr_document_categories", sysDocCatID, 1)
	})
	t.Run("hr_document_categories_system_seed_visible_to_B", func(t *testing.T) {
		testutil.AssertRowCount(t, pool, ctxB, "hr_document_categories", sysDocCatID, 1)
	})
}

// TestTenantIsolation_HR_DocCategories_PerTenant verifies that per-tenant hr_document_categories
// are NOT visible to other tenants.
func TestTenantIsolation_HR_DocCategories_PerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	catID := testutil.SeedRow(t, pool, "hr_document_categories", map[string]any{
		"tenant_id":  testutil.TenantA,
		"name":       "Tenant-A Category " + uuid.New().String()[:6],
		"key":        "ta_" + uuid.New().String()[:6],
		"visibility": "hr_only",
	})
	defer testutil.CleanupRow(t, pool, "hr_document_categories", catID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	testutil.AssertRowCount(t, pool, ctxA, "hr_document_categories", catID, 1)
	testutil.AssertRowCount(t, pool, ctxB, "hr_document_categories", catID, 0)
}
