package hr

// Welle 4.3 — role-based access to hr_employee_documents (mig 000127).
//
// Verifies the hr_document_access policy installed by 000127 against the
// three visibility levels:
//   hr_only  -> hr_admin/admin only
//   manager  -> hr_admin + manager
//   employee -> hr_admin + manager + the employee themself

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func withRoles(ctx context.Context, userID uuid.UUID, roles ...string) context.Context {
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID.String())
	ctx = context.WithValue(ctx, middleware.UserRolesKey, roles)
	return ctx
}

func TestHRRoleBased_DocumentAccess_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Tenant A users: employee, manager, hr_admin
	empA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "emp-" + uuid.New().String()[:8] + "@a.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", empA)

	mgrA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "mgr-" + uuid.New().String()[:8] + "@a.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", mgrA)

	hrA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         "hr-" + uuid.New().String()[:8] + "@a.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", hrA)

	// Tenant B user (different tenant, any role — should never see Tenant A docs)
	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantB,
		"email":         "u-" + uuid.New().String()[:8] + "@b.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userB)

	// Categories per visibility level (Tenant A)
	catHROnly := testutil.SeedRow(t, pool, "hr_document_categories", map[string]any{
		"tenant_id":  testutil.TenantA,
		"name":       "Abmahnungen",
		"key":        "abmahnung-" + uuid.New().String()[:6],
		"visibility": "hr_only",
	})
	defer testutil.CleanupRow(t, pool, "hr_document_categories", catHROnly)

	catManager := testutil.SeedRow(t, pool, "hr_document_categories", map[string]any{
		"tenant_id":  testutil.TenantA,
		"name":       "Zeugnisse",
		"key":        "zeugnis-" + uuid.New().String()[:6],
		"visibility": "manager",
	})
	defer testutil.CleanupRow(t, pool, "hr_document_categories", catManager)

	catEmployee := testutil.SeedRow(t, pool, "hr_document_categories", map[string]any{
		"tenant_id":  testutil.TenantA,
		"name":       "Sonstiges",
		"key":        "sonst-" + uuid.New().String()[:6],
		"visibility": "employee",
	})
	defer testutil.CleanupRow(t, pool, "hr_document_categories", catEmployee)

	// Documents for empA in all three categories
	docHROnly := testutil.SeedRow(t, pool, "hr_employee_documents", map[string]any{
		"tenant_id":   testutil.TenantA,
		"employee_id": empA,
		"category_id": catHROnly,
		"file_id":     uuid.New(),
		"uploaded_by": hrA,
	})
	defer testutil.CleanupRow(t, pool, "hr_employee_documents", docHROnly)

	docManager := testutil.SeedRow(t, pool, "hr_employee_documents", map[string]any{
		"tenant_id":   testutil.TenantA,
		"employee_id": empA,
		"category_id": catManager,
		"file_id":     uuid.New(),
		"uploaded_by": hrA,
	})
	defer testutil.CleanupRow(t, pool, "hr_employee_documents", docManager)

	docEmployee := testutil.SeedRow(t, pool, "hr_employee_documents", map[string]any{
		"tenant_id":   testutil.TenantA,
		"employee_id": empA,
		"category_id": catEmployee,
		"file_id":     uuid.New(),
		"uploaded_by": hrA,
	})
	defer testutil.CleanupRow(t, pool, "hr_employee_documents", docEmployee)

	// hr_admin in Tenant A — sees all three
	ctxHR := withRoles(testutil.WithTenantCtx(context.Background(), testutil.TenantA), hrA, "hr_admin")
	testutil.AssertRowCount(t, pool, ctxHR, "hr_employee_documents", docHROnly, 1)
	testutil.AssertRowCount(t, pool, ctxHR, "hr_employee_documents", docManager, 1)
	testutil.AssertRowCount(t, pool, ctxHR, "hr_employee_documents", docEmployee, 1)

	// manager in Tenant A — sees manager + employee visibility, NOT hr_only
	ctxMgr := withRoles(testutil.WithTenantCtx(context.Background(), testutil.TenantA), mgrA, "manager")
	testutil.AssertRowCount(t, pool, ctxMgr, "hr_employee_documents", docHROnly, 0)
	testutil.AssertRowCount(t, pool, ctxMgr, "hr_employee_documents", docManager, 1)
	testutil.AssertRowCount(t, pool, ctxMgr, "hr_employee_documents", docEmployee, 1)

	// employee self in Tenant A (no privileged role) — sees only employee visibility
	ctxSelf := withRoles(testutil.WithTenantCtx(context.Background(), testutil.TenantA), empA, "member")
	testutil.AssertRowCount(t, pool, ctxSelf, "hr_employee_documents", docHROnly, 0)
	testutil.AssertRowCount(t, pool, ctxSelf, "hr_employee_documents", docManager, 0)
	testutil.AssertRowCount(t, pool, ctxSelf, "hr_employee_documents", docEmployee, 1)

	// Tenant B user — even with hr_admin role, cross-tenant blocked
	ctxOtherTenant := withRoles(testutil.WithTenantCtx(context.Background(), testutil.TenantB), userB, "hr_admin")
	testutil.AssertRowCount(t, pool, ctxOtherTenant, "hr_employee_documents", docHROnly, 0)
	testutil.AssertRowCount(t, pool, ctxOtherTenant, "hr_employee_documents", docManager, 0)
	testutil.AssertRowCount(t, pool, ctxOtherTenant, "hr_employee_documents", docEmployee, 0)
}
