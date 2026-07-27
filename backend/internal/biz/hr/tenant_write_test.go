package hr

// Nacht 1 fixed the one dead HR write it found (hr_break_entries, d78f9176) but
// left the rest of the module unchecked with the same method: every existing
// HR DB test (tenant_isolation_phase2_test.go, hr_role_based_test.go) seeds
// exclusively through testutil.SeedRow (a raw INSERT under system context) and
// never calls the real employee/leave repositories. This file closes that gap
// for the employee-profile, employee-document and leave-request write surface.
// absence has no write path at all (AbsenceRepository is read-only); compliance
// is pure calculation with no DB access — neither needs a write test.
//
// Two dead-write-shaped bugs found and fixed in the same commit as this file:
//   - leave.PostgresLeaveRequestRepo.Update ran `WHERE id = $8` with no
//     tenant_id predicate, relying solely on RLS. Approve/Reject/Cancel all
//     call it on the live path. Fixed to carry `AND tenant_id = $9` from the
//     already-populated req.TenantID, same shape as every other Update in the
//     repo.
//   - employee.PostgresEmployeeDocRepo.Delete ran `WHERE id = $1` with no
//     tenant predicate at all, and its interface did not even accept a
//     tenantID to check. Unlike the leave case this path has zero callers
//     anywhere in the repo (Service.DeleteEmployeeDocument is dead weight,
//     like the auth session RPCs Iteration 41 found) — fixed anyway since it
//     is the exact landmine class this loop exists to defuse, and the
//     interface change was free (no caller to break).
//
// Each write below is exercised once from a foreign-tenant ctx passing the
// row's *real* tenantID as the explicit value, so only RLS (or, for the two
// fixes above, the new WHERE predicate backed by RLS) can be stopping it —
// then repeated from the owning ctx to confirm it lands.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/biz/hr/employee"
	"github.com/kmuhub/kmuhub/internal/biz/hr/leave"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestEmployeeProfileWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Biz HR Employee Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Biz HR Employee Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         "hr-write-" + uuid.New().String()[:8] + "@tenantown.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := employee.NewPostgresEmployeeRepo(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	profile := &models.EmployeeProfile{
		ID:              uuid.New(),
		TenantID:        tenantOwn,
		UserID:          userID,
		Department:      "Engineering",
		PositionTitle:   "Write Test Developer",
		ContractType:    models.HRContractFullTime,
		WorkDaysPerWeek: 5,
		AnnualLeaveDays: 28,
		StartDate:       now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := repo.Create(ctxOwn, profile); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "hr_employee_profiles", profile.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "hr_employee_profiles", profile.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "hr_employee_profiles", profile.ID, 0)

	// Update already carries an explicit tenant_id predicate; call it from the
	// foreign ctx with the row's real tenantID so only RLS can be stopping it.
	foreign := *profile
	foreign.PositionTitle = "Hacked Title"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign); err != nil {
		t.Fatalf("Update (foreign ctx): unexpected error %v", err)
	}
	got, err := repo.GetByID(ctxOwn, profile.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.PositionTitle != "Write Test Developer" {
		t.Fatalf("a foreign-tenant write reached the employee profile: position_title=%q", got.PositionTitle)
	}

	foreign.PositionTitle = "Renamed Title"
	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, profile.ID)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.PositionTitle != "Renamed Title" {
		t.Fatalf("own-tenant write did not land: position_title=%q", got.PositionTitle)
	}
}

func TestEmployeeDocumentWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Biz HR Document Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Biz HR Document Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         "hr-doc-" + uuid.New().String()[:8] + "@tenantown.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	categoryID := testutil.SeedRow(t, pool, "hr_document_categories", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "Write Test Category",
		"key":        "write-test-" + uuid.New().String()[:8],
		"visibility": "hr_only",
	})
	defer testutil.CleanupRow(t, pool, "hr_document_categories", categoryID)

	repo := employee.NewPostgresEmployeeDocRepo(pool)
	// hr_employee_documents runs the role-aware hr_document_access policy
	// (migration 000127), not the plain tenant_isolation policy — both USING
	// and WITH CHECK require an hr_admin/admin role on top of tenant match.
	ctxOwn := withRoles(testutil.WithTenantCtx(context.Background(), tenantOwn), userID, "hr_admin")
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())

	doc := &models.EmployeeDocument{
		ID:         uuid.New(),
		TenantID:   tenantOwn,
		EmployeeID: userID,
		CategoryID: categoryID,
		FileID:     uuid.New(),
		UploadedBy: userID,
		Notes:      "write test",
		CreatedAt:  time.Now().UTC(),
	}
	if err := repo.Create(ctxOwn, doc); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "hr_employee_documents", doc.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "hr_employee_documents", doc.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "hr_employee_documents", doc.ID, 0)

	// Delete now takes an explicit tenantID and carries a matching predicate
	// (fixed in this commit) — call it from the foreign ctx with the row's
	// real tenantID so only RLS can be stopping it.
	if err := repo.Delete(ctxOther, tenantOwn, doc.ID); err != nil {
		t.Fatalf("Delete (foreign ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "hr_employee_documents", doc.ID, 1)

	if err := repo.Delete(ctxOwn, tenantOwn, doc.ID); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "hr_employee_documents", doc.ID, 0)
}

func TestLeaveRequestWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Biz HR Leave Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Biz HR Leave Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	employeeID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         "hr-leave-" + uuid.New().String()[:8] + "@tenantown.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", employeeID)

	// approved_by REFERENCES users(id) — needs a real row, not a random UUID.
	approverID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         "hr-approver-" + uuid.New().String()[:8] + "@tenantown.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", approverID)

	leaveTypeID := testutil.SeedRow(t, pool, "hr_leave_types", map[string]any{
		"tenant_id": tenantOwn,
		"name":      "Write Test Leave Type",
		"key":       "write-test-" + uuid.New().String()[:8],
	})
	defer testutil.CleanupRow(t, pool, "hr_leave_types", leaveTypeID)

	repo := leave.NewPostgresLeaveRequestRepo(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	req := &models.LeaveRequest{
		ID:          uuid.New(),
		TenantID:    tenantOwn,
		EmployeeID:  employeeID,
		LeaveTypeID: leaveTypeID,
		StartDate:   now,
		EndDate:     now.Add(3 * 24 * time.Hour),
		TotalDays:   decimal.NewFromInt(3),
		Reason:      "Write test",
		Status:      models.LeaveStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.Create(ctxOwn, req); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "hr_leave_requests", req.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "hr_leave_requests", req.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "hr_leave_requests", req.ID, 0)

	// Update now carries an explicit tenant_id predicate (fixed in this
	// commit) — call it from the foreign ctx with the row's real tenantID so
	// only RLS/the new predicate can be stopping it.
	foreign := *req
	foreign.Status = models.LeaveStatusApproved
	foreign.ApprovedBy = &approverID
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign); err != nil {
		t.Fatalf("Update (foreign ctx): unexpected error %v", err)
	}
	got, err := repo.GetByID(ctxOwn, req.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != models.LeaveStatusPending {
		t.Fatalf("a foreign-tenant write reached the leave request: status=%q", got.Status)
	}

	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, req.ID)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.Status != models.LeaveStatusApproved {
		t.Fatalf("own-tenant write did not land: status=%q", got.Status)
	}
}

func TestLeaveBalanceAndSettingsWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Biz HR Balance Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Biz HR Balance Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	employeeID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         "hr-balance-" + uuid.New().String()[:8] + "@tenantown.local",
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", employeeID)

	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	// --- Leave balance ---
	// hr_leave_balances has no dedicated Delete; Upsert is a straight
	// INSERT ... ON CONFLICT (tenant_id, employee_id, year), already scoping
	// on tenant_id as part of the conflict target and insert column. A
	// foreign-ctx call with the row's real tenantID must be rejected by RLS's
	// WITH CHECK outright (there is no existing row for tenantOther to match
	// against, so this is an insert attempt, not a silent no-op).
	balRepo := leave.NewPostgresLeaveBalanceRepo(pool)
	bal := &models.HRLeaveBalance{
		ID:          uuid.New(),
		TenantID:    tenantOwn,
		EmployeeID:  employeeID,
		Year:        2026,
		Entitlement: decimal.NewFromInt(20),
		Remaining:   decimal.NewFromInt(20),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := balRepo.Upsert(ctxOther, bal); err == nil {
		t.Fatalf("Upsert (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, testutil.WithSystemCtx(context.Background()), "hr_leave_balances", bal.ID, 0)

	if err := balRepo.Upsert(ctxOwn, bal); err != nil {
		t.Fatalf("Upsert (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "hr_leave_balances", bal.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "hr_leave_balances", bal.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "hr_leave_balances", bal.ID, 0)

	// --- HR company settings ---
	// Same shape: INSERT ... ON CONFLICT (tenant_id), tenant_id is both the
	// insert column and the conflict target.
	settingsRepo := leave.NewPostgresHRSettingsRepo(pool)
	settings := &models.HRCompanySettings{
		ID:                     uuid.New(),
		TenantID:               tenantOwn,
		AUThresholdDays:        3,
		ShowAbsenceReason:      true,
		DefaultAnnualLeaveDays: 20,
		Timezone:               "Europe/Berlin",
		WorkHoursPerDay:        8,
		MaxDailyHours:          10,
		BreakAfterHours:        6,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if err := settingsRepo.Upsert(ctxOther, settings); err == nil {
		t.Fatalf("Upsert settings (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, testutil.WithSystemCtx(context.Background()), "hr_company_settings", settings.ID, 0)

	if err := settingsRepo.Upsert(ctxOwn, settings); err != nil {
		t.Fatalf("Upsert settings (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "hr_company_settings", settings.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "hr_company_settings", settings.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "hr_company_settings", settings.ID, 0)
}
