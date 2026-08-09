// postgres_repository_db_test.go covers all four repositories in this
// package (LeaveRequest, LeaveBalance, LeaveType, HRSettings) against the
// real migration schema (testutil.SkipIfNoDB, the pattern
// internal/biz/hr/absence/postgres_repository_db_test.go already uses).
// Before this commit postgres_repository.go had 0% DB coverage; the only
// DB tests were the build-tagged integration_test.go, which nightly.yml
// runs but the loop's day-to-day gate does not. This closes that gap and
// also fixes the same CONCAT_WS blank-name regression already found and
// fixed in absence/employee: GetByID/List lacked the TRIM() around
// CONCAT_WS, so an employee with first_name='' and last_name='' produced
// a single-space employee_name instead of falling back to email.
package leave

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// newLeaveFixture seeds a fresh tenant per test — hr_leave_types has a
// (tenant_id, key) uniqueness constraint and hr_company_settings a
// (tenant_id) one, so sharing a fixture across tests would collide.
func newLeaveFixture(t *testing.T) (*pgxpool.Pool, uuid.UUID, context.Context) {
	t.Helper()
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Leave DB Test Tenant")

	return pool, tenantID, testutil.WithTenantCtx(context.Background(), tenantID)
}

func seedLeaveUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, firstName, lastName, email string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "email": email,
		"password_hash": "hash", "first_name": firstName, "last_name": lastName,
		"is_active": true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
	return id
}

type leaveTypeOpts struct {
	name               string
	deductsFromBalance bool
	requiresApproval   bool
	requiresAUDocument bool
	sortOrder          int
}

func seedLeaveTypeRow(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, opts leaveTypeOpts) uuid.UUID {
	t.Helper()
	name := opts.name
	if name == "" {
		name = "Urlaub"
	}
	id := testutil.SeedRow(t, pool, "hr_leave_types", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "name": name,
		"key": "key-" + uuid.NewString()[:8], "color": "#3d8abf",
		"deducts_from_balance": opts.deductsFromBalance, "requires_approval": opts.requiresApproval,
		"requires_au_document": opts.requiresAUDocument, "is_system": false,
		"sort_order": opts.sortOrder,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_leave_types", id) })
	return id
}

func decimalFromInt(n int) decimal.Decimal {
	return decimal.NewFromInt(int64(n))
}

func newLeaveRequestFixture(tenantID, employeeID, leaveTypeID uuid.UUID, start, end time.Time) *models.LeaveRequest {
	now := time.Now().UTC().Truncate(time.Second)
	days := int(end.Sub(start).Hours()/24) + 1
	return &models.LeaveRequest{
		ID:          uuid.New(),
		TenantID:    tenantID,
		EmployeeID:  employeeID,
		LeaveTypeID: leaveTypeID,
		StartDate:   start,
		EndDate:     end,
		TotalDays:   decimal.NewFromInt(int64(days)),
		Reason:      "DB test",
		Status:      models.LeaveStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// ============================================================================
// LeaveRequestRepository — Create/GetByID round-trip and CONCAT_WS regression
// ============================================================================

func TestPostgresLeaveRequestRepo_CreateGetByID_RoundTrip(t *testing.T) {
	pool, tenantID, ctx := newLeaveFixture(t)
	repo := NewPostgresLeaveRequestRepo(pool)

	empID := seedLeaveUser(t, pool, tenantID, "Frida", "Nowak", "frida-"+uuid.NewString()+"@example.test")
	typeID := seedLeaveTypeRow(t, pool, tenantID, leaveTypeOpts{deductsFromBalance: true, requiresApproval: true})
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)

	req := newLeaveRequestFixture(tenantID, empID, typeID, start, end)
	req.Reason = "Sommerurlaub"
	if err := repo.Create(ctx, req); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_leave_requests", req.ID) })

	got, err := repo.GetByID(ctx, req.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != req.ID {
		t.Errorf("ID: got %s, want %s", got.ID, req.ID)
	}
	if got.EmployeeName != "Frida Nowak" {
		t.Errorf("EmployeeName: got %q, want %q", got.EmployeeName, "Frida Nowak")
	}
	if got.Reason != "Sommerurlaub" {
		t.Errorf("Reason: got %q, want %q", got.Reason, "Sommerurlaub")
	}
	if got.Status != models.LeaveStatusPending {
		t.Errorf("Status: got %q, want %q", got.Status, models.LeaveStatusPending)
	}
	if !got.TotalDays.Equal(req.TotalDays) {
		t.Errorf("TotalDays: got %s, want %s", got.TotalDays, req.TotalDays)
	}
}

// TestPostgresLeaveRequestRepo_GetByID_BlankNameFallsBackToEmail is the
// regression this unit exists for. CONCAT_WS(' ', a, b) only skips NULL
// arguments, not empty ones — with two empty strings it still applies the
// separator and returns a single space, which NULLIF(x, '') does not
// recognize as blank. Fails against the pre-fix query and passes only with
// TRIM() around the CONCAT_WS before the NULLIF.
func TestPostgresLeaveRequestRepo_GetByID_BlankNameFallsBackToEmail(t *testing.T) {
	pool, tenantID, ctx := newLeaveFixture(t)
	repo := NewPostgresLeaveRequestRepo(pool)

	email := "blank-" + uuid.NewString() + "@example.test"
	empID := seedLeaveUser(t, pool, tenantID, "", "", email)
	typeID := seedLeaveTypeRow(t, pool, tenantID, leaveTypeOpts{deductsFromBalance: true, requiresApproval: true})
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)

	req := newLeaveRequestFixture(tenantID, empID, typeID, start, end)
	if err := repo.Create(ctx, req); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_leave_requests", req.ID) })

	got, err := repo.GetByID(ctx, req.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.EmployeeName != email {
		t.Errorf("EmployeeName: got %q, want the email fallback %q", got.EmployeeName, email)
	}
}

func TestPostgresLeaveRequestRepo_GetByID_NotFound(t *testing.T) {
	pool, _, ctx := newLeaveFixture(t)
	repo := NewPostgresLeaveRequestRepo(pool)

	_, err := repo.GetByID(ctx, uuid.New())
	if err != ErrLeaveRequestNotFound {
		t.Errorf("GetByID: got err %v, want %v", err, ErrLeaveRequestNotFound)
	}
}

// ============================================================================
// LeaveRequestRepository — List filters and pagination
// ============================================================================

func TestPostgresLeaveRequestRepo_List_FiltersAndPagination(t *testing.T) {
	pool, tenantID, ctx := newLeaveFixture(t)
	repo := NewPostgresLeaveRequestRepo(pool)
	typeID := seedLeaveTypeRow(t, pool, tenantID, leaveTypeOpts{deductsFromBalance: true, requiresApproval: true})

	empA := seedLeaveUser(t, pool, tenantID, "Anna", "List", "anna-list-"+uuid.NewString()+"@example.test")
	empB := seedLeaveUser(t, pool, tenantID, "Ben", "List", "ben-list-"+uuid.NewString()+"@example.test")

	mkReq := func(empID uuid.UUID, start, end time.Time, status models.LeaveRequestStatus) *models.LeaveRequest {
		req := newLeaveRequestFixture(tenantID, empID, typeID, start, end)
		req.Status = status
		if err := repo.Create(ctx, req); err != nil {
			t.Fatalf("Create: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_leave_requests", req.ID) })
		return req
	}

	reqAPending := mkReq(empA, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), models.LeaveStatusPending)
	reqAApproved := mkReq(empA, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC), models.LeaveStatusApproved)
	reqB := mkReq(empB, time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC), models.LeaveStatusPending)

	t.Run("employee_filter", func(t *testing.T) {
		results, total, err := repo.List(ctx, LeaveRequestFilter{TenantID: tenantID, EmployeeID: &empA, Page: 1, PerPage: 20})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 2 {
			t.Errorf("total: got %d, want 2", total)
		}
		for _, r := range results {
			if r.EmployeeID != empA {
				t.Errorf("List with EmployeeID filter returned request for a different employee: %s", r.EmployeeID)
			}
		}
	})

	t.Run("status_filter", func(t *testing.T) {
		pending := string(models.LeaveStatusPending)
		results, total, err := repo.List(ctx, LeaveRequestFilter{TenantID: tenantID, Status: &pending, Page: 1, PerPage: 20})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		ids := map[uuid.UUID]bool{}
		for _, r := range results {
			ids[r.ID] = true
			if r.Status != models.LeaveStatusPending {
				t.Errorf("List with status filter returned a %s request", r.Status)
			}
		}
		if !ids[reqAPending.ID] || !ids[reqB.ID] {
			t.Errorf("status filter missing expected pending requests")
		}
		if ids[reqAApproved.ID] {
			t.Errorf("status filter leaked an approved request")
		}
		_ = total
	})

	t.Run("date_range_filter", func(t *testing.T) {
		from := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
		results, _, err := repo.List(ctx, LeaveRequestFilter{TenantID: tenantID, StartDateFrom: &from, StartDateTo: &to, Page: 1, PerPage: 20})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		var found bool
		for _, r := range results {
			if r.ID == reqB.ID {
				found = true
			}
			if r.ID == reqAPending.ID || r.ID == reqAApproved.ID {
				t.Errorf("date range filter leaked request %s outside the window", r.ID)
			}
		}
		if !found {
			t.Errorf("date range filter missing request %s inside the window", reqB.ID)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		page1, total, err := repo.List(ctx, LeaveRequestFilter{TenantID: tenantID, Page: 1, PerPage: 2})
		if err != nil {
			t.Fatalf("List page 1: %v", err)
		}
		if total != 3 {
			t.Errorf("total: got %d, want 3", total)
		}
		if len(page1) != 2 {
			t.Fatalf("page 1: got %d results, want 2", len(page1))
		}
		page2, _, err := repo.List(ctx, LeaveRequestFilter{TenantID: tenantID, Page: 2, PerPage: 2})
		if err != nil {
			t.Fatalf("List page 2: %v", err)
		}
		if len(page2) != 1 {
			t.Fatalf("page 2: got %d results, want 1", len(page2))
		}
		if page1[0].ID == page2[0].ID {
			t.Errorf("page 1 and page 2 returned the same request %s", page1[0].ID)
		}
	})
}

// ============================================================================
// LeaveRequestRepository — Update and tenant scoping
// ============================================================================

func TestPostgresLeaveRequestRepo_Update_RoundTrip(t *testing.T) {
	pool, tenantID, ctx := newLeaveFixture(t)
	repo := NewPostgresLeaveRequestRepo(pool)
	empID := seedLeaveUser(t, pool, tenantID, "Update", "Test", "update-"+uuid.NewString()+"@example.test")
	typeID := seedLeaveTypeRow(t, pool, tenantID, leaveTypeOpts{deductsFromBalance: true, requiresApproval: true})
	// Seed the approver before creating the request so t.Cleanup (LIFO)
	// deletes the referencing hr_leave_requests row before the referenced
	// users row — otherwise the approved_by FK rejects the user delete.
	approverID := seedLeaveUser(t, pool, tenantID, "Manager", "Approver", "approver-"+uuid.NewString()+"@example.test")

	req := newLeaveRequestFixture(tenantID, empID, typeID,
		time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC))
	if err := repo.Create(ctx, req); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_leave_requests", req.ID) })

	now := time.Now().UTC().Truncate(time.Second)
	req.Status = models.LeaveStatusApproved
	req.ApprovedBy = &approverID
	req.ApprovalComment = "Genehmigt via DB test"
	req.ApprovedAt = &now
	req.UpdatedAt = now

	if err := repo.Update(ctx, req); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(ctx, req.ID)
	if err != nil {
		t.Fatalf("GetByID after Update: %v", err)
	}
	if got.Status != models.LeaveStatusApproved {
		t.Errorf("Status: got %q, want approved", got.Status)
	}
	if got.ApprovedBy == nil || *got.ApprovedBy != approverID {
		t.Errorf("ApprovedBy: got %v, want %s", got.ApprovedBy, approverID)
	}
	if got.ApprovalComment != "Genehmigt via DB test" {
		t.Errorf("ApprovalComment: got %q, want %q", got.ApprovalComment, "Genehmigt via DB test")
	}
	if got.ApprovedAt == nil {
		t.Errorf("ApprovedAt: got nil, want a timestamp")
	}
}

func TestPostgresLeaveRequestRepo_Update_WrongTenantIsNoop(t *testing.T) {
	pool, tenantA, ctxA := newLeaveFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Leave DB Test Tenant B")
	repo := NewPostgresLeaveRequestRepo(pool)

	empID := seedLeaveUser(t, pool, tenantA, "Cross", "Tenant", "cross-"+uuid.NewString()+"@example.test")
	typeID := seedLeaveTypeRow(t, pool, tenantA, leaveTypeOpts{deductsFromBalance: true, requiresApproval: true})
	req := newLeaveRequestFixture(tenantA, empID, typeID,
		time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC))
	if err := repo.Create(ctxA, req); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_leave_requests", req.ID) })

	// Attempt to update the row while claiming it belongs to tenant B — the
	// WHERE id = $8 AND tenant_id = $9 clause must reject this silently
	// (no error, no rows affected) rather than letting tenant B mutate
	// tenant A's request.
	forged := *req
	forged.TenantID = tenantB
	forged.Status = models.LeaveStatusApproved
	if err := repo.Update(context.Background(), &forged); err != nil {
		t.Fatalf("Update with mismatched tenant_id returned an error instead of affecting 0 rows: %v", err)
	}

	got, err := repo.GetByID(ctxA, req.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != models.LeaveStatusPending {
		t.Errorf("cross-tenant Update mutated the row: status is now %q, want pending unchanged", got.Status)
	}
}

// ============================================================================
// LeaveRequestRepository — FindOverlaps
// ============================================================================

func TestPostgresLeaveRequestRepo_FindOverlaps(t *testing.T) {
	pool, tenantID, ctx := newLeaveFixture(t)
	repo := NewPostgresLeaveRequestRepo(pool)
	empID := seedLeaveUser(t, pool, tenantID, "Overlap", "Test", "overlap-"+uuid.NewString()+"@example.test")
	typeID := seedLeaveTypeRow(t, pool, tenantID, leaveTypeOpts{deductsFromBalance: true, requiresApproval: true})

	base := newLeaveRequestFixture(tenantID, empID, typeID,
		time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC))
	base.Status = models.LeaveStatusApproved
	if err := repo.Create(ctx, base); err != nil {
		t.Fatalf("Create base: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_leave_requests", base.ID) })

	rejected := newLeaveRequestFixture(tenantID, empID, typeID,
		time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC))
	rejected.Status = models.LeaveStatusRejected
	if err := repo.Create(ctx, rejected); err != nil {
		t.Fatalf("Create rejected: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_leave_requests", rejected.ID) })

	// Overlapping window, should find `base` but not the rejected request
	// (FindOverlaps only considers pending/approved).
	overlaps, err := repo.FindOverlaps(ctx, empID,
		time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatalf("FindOverlaps: %v", err)
	}
	if len(overlaps) != 1 || overlaps[0].ID != base.ID {
		t.Fatalf("FindOverlaps: got %d results, want [base]", len(overlaps))
	}

	// excludeID excludes the overlapping request itself (self-exclusion during approve).
	excluded, err := repo.FindOverlaps(ctx, empID,
		time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC), &base.ID)
	if err != nil {
		t.Fatalf("FindOverlaps with excludeID: %v", err)
	}
	if len(excluded) != 0 {
		t.Errorf("FindOverlaps with excludeID=base.ID: got %d results, want 0", len(excluded))
	}

	// Non-overlapping window finds nothing.
	none, err := repo.FindOverlaps(ctx, empID,
		time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatalf("FindOverlaps non-overlapping: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("FindOverlaps non-overlapping window: got %d results, want 0", len(none))
	}
}

// ============================================================================
// LeaveBalanceRepository
// ============================================================================

func TestPostgresLeaveBalanceRepo_GetByEmployeeYear_NotFoundReturnsNilNil(t *testing.T) {
	pool, tenantID, ctx := newLeaveFixture(t)
	repo := NewPostgresLeaveBalanceRepo(pool)
	empID := seedLeaveUser(t, pool, tenantID, "NoBalance", "Test", "nobalance-"+uuid.NewString()+"@example.test")

	balance, err := repo.GetByEmployeeYear(ctx, tenantID, empID, 2026)
	if err != nil {
		t.Fatalf("GetByEmployeeYear: unexpected error %v", err)
	}
	if balance != nil {
		t.Errorf("GetByEmployeeYear: got %+v, want nil (no error, no sentinel — service.go relies on this)", balance)
	}
}

// TestPostgresLeaveBalanceRepo_Upsert_InsertThenUpdateRoundtrip proves the
// ON CONFLICT target is (tenant_id, employee_id, year), not id — a second
// Upsert with a fresh ID must land on the same row rather than erroring or
// creating a duplicate. This is exactly what deductBalance/restoreBalance in
// service.go rely on when they re-fetch-then-Upsert.
func TestPostgresLeaveBalanceRepo_Upsert_InsertThenUpdateRoundtrip(t *testing.T) {
	pool, tenantID, ctx := newLeaveFixture(t)
	repo := NewPostgresLeaveBalanceRepo(pool)
	empID := seedLeaveUser(t, pool, tenantID, "Balance", "Test", "balance-"+uuid.NewString()+"@example.test")

	now := time.Now().UTC().Truncate(time.Second)
	first := &models.HRLeaveBalance{
		ID: uuid.New(), TenantID: tenantID, EmployeeID: empID, Year: 2026,
		Entitlement: decimalFromInt(30), CarriedOver: decimalFromInt(0),
		Used: decimalFromInt(0), Remaining: decimalFromInt(30),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Upsert(ctx, first); err != nil {
		t.Fatalf("Upsert (insert): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_leave_balances", first.ID) })

	// Second Upsert with a DIFFERENT id but the same (tenant, employee, year).
	second := &models.HRLeaveBalance{
		ID: uuid.New(), TenantID: tenantID, EmployeeID: empID, Year: 2026,
		Entitlement: decimalFromInt(30), CarriedOver: decimalFromInt(0),
		Used: decimalFromInt(5), Remaining: decimalFromInt(25),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Upsert(ctx, second); err != nil {
		t.Fatalf("Upsert (update): %v", err)
	}

	got, err := repo.GetByEmployeeYear(ctx, tenantID, empID, 2026)
	if err != nil {
		t.Fatalf("GetByEmployeeYear: %v", err)
	}
	if got == nil {
		t.Fatal("GetByEmployeeYear: got nil after two Upserts")
	}
	if got.ID != first.ID {
		t.Errorf("ID: got %s, want the original row id %s — ON CONFLICT target is not (tenant_id, employee_id, year)", got.ID, first.ID)
	}
	if !got.Used.Equal(decimalFromInt(5)) {
		t.Errorf("Used: got %s, want 5 (second Upsert's value)", got.Used)
	}
	if !got.Remaining.Equal(decimalFromInt(25)) {
		t.Errorf("Remaining: got %s, want 25 (second Upsert's value)", got.Remaining)
	}
}

// ============================================================================
// LeaveTypeRepository
// ============================================================================

func TestPostgresLeaveTypeRepo_ListByTenant_OrderingAndTenantScope(t *testing.T) {
	pool, tenantA, ctxA := newLeaveFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Leave DB Test Tenant B")
	repo := NewPostgresLeaveTypeRepo(pool)

	seedLeaveTypeRow(t, pool, tenantA, leaveTypeOpts{name: "Zweiter", sortOrder: 2})
	seedLeaveTypeRow(t, pool, tenantA, leaveTypeOpts{name: "Erster", sortOrder: 1})
	seedLeaveTypeRow(t, pool, tenantB, leaveTypeOpts{name: "Fremd", sortOrder: 0})

	results, err := repo.ListByTenant(ctxA, tenantA)
	if err != nil {
		t.Fatalf("ListByTenant: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("ListByTenant: got %d results, want 2", len(results))
	}
	if results[0].Name != "Erster" || results[1].Name != "Zweiter" {
		t.Errorf("ListByTenant ordering: got [%s, %s], want [Erster, Zweiter]", results[0].Name, results[1].Name)
	}
	for _, r := range results {
		if r.TenantID != tenantA {
			t.Errorf("ListByTenant leaked a row from tenant %s", r.TenantID)
		}
	}
}

func TestPostgresLeaveTypeRepo_GetByID_NotFound(t *testing.T) {
	pool, _, ctx := newLeaveFixture(t)
	repo := NewPostgresLeaveTypeRepo(pool)

	_, err := repo.GetByID(ctx, uuid.New())
	if err != ErrLeaveTypeNotFound {
		t.Errorf("GetByID: got err %v, want %v", err, ErrLeaveTypeNotFound)
	}
}

func TestPostgresLeaveTypeRepo_GetByKey_NotFound(t *testing.T) {
	pool, tenantID, ctx := newLeaveFixture(t)
	repo := NewPostgresLeaveTypeRepo(pool)

	_, err := repo.GetByKey(ctx, tenantID, "does-not-exist")
	if err != ErrLeaveTypeNotFound {
		t.Errorf("GetByKey: got err %v, want %v", err, ErrLeaveTypeNotFound)
	}
}

func TestPostgresLeaveTypeRepo_GetByKey_TenantScoped(t *testing.T) {
	pool, tenantA, ctxA := newLeaveFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Leave DB Test Tenant B")
	repo := NewPostgresLeaveTypeRepo(pool)

	idA := testutil.SeedRow(t, pool, "hr_leave_types", map[string]any{
		"id": uuid.New(), "tenant_id": tenantA, "name": "Urlaub A",
		"key": "shared-key", "color": "#3d8abf",
		"deducts_from_balance": true, "requires_approval": true,
		"requires_au_document": false, "is_system": false, "sort_order": 1,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_leave_types", idA) })
	idB := testutil.SeedRow(t, pool, "hr_leave_types", map[string]any{
		"id": uuid.New(), "tenant_id": tenantB, "name": "Urlaub B",
		"key": "shared-key", "color": "#3d8abf",
		"deducts_from_balance": true, "requires_approval": true,
		"requires_au_document": false, "is_system": false, "sort_order": 1,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_leave_types", idB) })

	got, err := repo.GetByKey(ctxA, tenantA, "shared-key")
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if got.ID != idA {
		t.Errorf("GetByKey: got tenant %s's row, want tenant A's row %s", got.ID, idA)
	}
}

// ============================================================================
// HRSettingsRepository
// ============================================================================

func TestPostgresHRSettingsRepo_GetByTenant_NotFound(t *testing.T) {
	pool, tenantID, ctx := newLeaveFixture(t)
	repo := NewPostgresHRSettingsRepo(pool)

	_, err := repo.GetByTenant(ctx, tenantID)
	if err != ErrSettingsNotFound {
		t.Errorf("GetByTenant: got err %v, want %v", err, ErrSettingsNotFound)
	}
}

func TestPostgresHRSettingsRepo_Upsert_RoundTrip(t *testing.T) {
	pool, tenantID, ctx := newLeaveFixture(t)
	repo := NewPostgresHRSettingsRepo(pool)

	now := time.Now().UTC().Truncate(time.Second)
	settings := &models.HRCompanySettings{
		ID: uuid.New(), TenantID: tenantID, AUThresholdDays: 5,
		ShowAbsenceReason: true, DefaultAnnualLeaveDays: 25, Timezone: "Europe/Berlin",
		WorkHoursPerDay: 8, MaxDailyHours: 10, BreakAfterHours: 6,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Upsert(ctx, settings); err != nil {
		t.Fatalf("Upsert (insert): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_company_settings", settings.ID) })

	got, err := repo.GetByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetByTenant: %v", err)
	}
	if got.AUThresholdDays != 5 {
		t.Errorf("AUThresholdDays: got %d, want 5", got.AUThresholdDays)
	}

	// Update via a second Upsert — unique(tenant_id) forces ON CONFLICT.
	settings.AUThresholdDays = 10
	settings.UpdatedAt = now.Add(time.Minute)
	if err := repo.Upsert(ctx, settings); err != nil {
		t.Fatalf("Upsert (update): %v", err)
	}

	got2, err := repo.GetByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetByTenant after update: %v", err)
	}
	if got2.AUThresholdDays != 10 {
		t.Errorf("AUThresholdDays after update: got %d, want 10", got2.AUThresholdDays)
	}
	if got2.ID != settings.ID {
		t.Errorf("ID after update: got %s, want %s (should be same row, not a duplicate)", got2.ID, settings.ID)
	}
}
