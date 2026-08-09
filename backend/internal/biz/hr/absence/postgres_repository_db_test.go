package absence

// postgres_repository_db_test.go covers PostgresAbsenceRepo.GetAbsenceCalendar
// against the real migration schema (testutil.SkipIfNoDB, the pattern
// internal/biz/hr/employee/offboard_db_test.go already uses): the CONCAT_WS
// name resolution with its email fallback (same c2cc98ad regression as
// employee, still unfixed here until this commit — see TRIM in
// postgres_repository.go), the date-range overlap semantics the query
// actually implements (inclusive '[]' on both sides, not the half-open
// interval booking conflict checks elsewhere use), the approved-only status
// filter, the department filter, half-day round-tripping and cross-tenant
// isolation. The build-tagged integration_test.go in this package only runs
// in nightly.yml, not in the loop's day-to-day gate — this file closes that
// same visibility gap for this package.

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

// newAbsenceFixture seeds a fresh tenant per test — the calendar filter's
// tenant scoping and the leave-type uniqueness constraint (tenant_id, key)
// are both properties of the tenant population, so sharing a fixture across
// tests would let one test's rows leak into another's date-range window.
func newAbsenceFixture(t *testing.T) (*pgxpool.Pool, uuid.UUID, context.Context) {
	t.Helper()
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Absence DB Test Tenant")

	return pool, tenantID, testutil.WithTenantCtx(context.Background(), tenantID)
}

func seedAbsenceUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, firstName, lastName, email string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "email": email,
		"password_hash": "hash", "first_name": firstName, "last_name": lastName,
		"is_active": true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
	return id
}

// seedAbsenceProfile is only needed for the department filter: the calendar
// query LEFT JOINs hr_employee_profiles, so an employee without a profile
// row still appears in results with an empty department, not excluded.
func seedAbsenceProfile(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID, department string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "hr_employee_profiles", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "user_id": userID,
		"department": department, "position_title": "Engineer",
		"contract_type": "full_time", "work_days_per_week": 5,
		"annual_leave_days": 25, "start_date": "2020-01-01",
		"address_country": "DE",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_profiles", id) })
	return id
}

func seedAbsenceLeaveType(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "hr_leave_types", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "name": "Urlaub",
		"key": "urlaub-" + uuid.NewString()[:8], "color": "#3d8abf",
		"deducts_from_balance": true, "requires_approval": true,
		"requires_au_document": false, "is_system": false, "sort_order": 1,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_leave_types", id) })
	return id
}

type absenceLeaveRequestOpts struct {
	status                               string
	isHalfDayStart, isHalfDayEnd         bool
	halfDayPeriodStart, halfDayPeriodEnd *string
}

func seedAbsenceLeaveRequest(
	t *testing.T, pool *pgxpool.Pool,
	tenantID, employeeID, leaveTypeID uuid.UUID,
	start, end time.Time,
	opts absenceLeaveRequestOpts,
) uuid.UUID {
	t.Helper()
	status := opts.status
	if status == "" {
		status = "approved"
	}
	cols := map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "employee_id": employeeID,
		"leave_type_id": leaveTypeID, "start_date": start, "end_date": end,
		"is_half_day_start": opts.isHalfDayStart, "is_half_day_end": opts.isHalfDayEnd,
		"total_days": decimal.NewFromFloat(1.0), "status": status,
		"au_document_required": false,
	}
	if opts.halfDayPeriodStart != nil {
		cols["half_day_period_start"] = *opts.halfDayPeriodStart
	}
	if opts.halfDayPeriodEnd != nil {
		cols["half_day_period_end"] = *opts.halfDayPeriodEnd
	}
	id := testutil.SeedRow(t, pool, "hr_leave_requests", cols)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_leave_requests", id) })
	return id
}

func findAbsenceEntry(entries []*AbsenceEntry, employeeID uuid.UUID) *AbsenceEntry {
	for _, e := range entries {
		if e.EmployeeID == employeeID {
			return e
		}
	}
	return nil
}

// ============================================================================
// Name resolution — CONCAT_WS + email fallback (the display_name incident,
// c2cc98ad; the TRIM regression, same class as the employee package's fix)
// ============================================================================

func TestRepository_GetAbsenceCalendar_NamedEmployeeResolvesFullName(t *testing.T) {
	pool, tenantID, ctx := newAbsenceFixture(t)
	repo := NewPostgresAbsenceRepo(pool)

	empID := seedAbsenceUser(t, pool, tenantID, "Greta", "Hoffmann", "greta-"+uuid.NewString()+"@example.test")
	typeID := seedAbsenceLeaveType(t, pool, tenantID)
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)
	seedAbsenceLeaveRequest(t, pool, tenantID, empID, typeID, start, end, absenceLeaveRequestOpts{})

	entries, err := repo.GetAbsenceCalendar(ctx, AbsenceFilter{
		TenantID: tenantID, StartDate: start, EndDate: end,
	})
	if err != nil {
		t.Fatalf("GetAbsenceCalendar: %v", err)
	}
	got := findAbsenceEntry(entries, empID)
	if got == nil {
		t.Fatalf("seeded employee %s not present in result", empID)
	}
	if got.EmployeeName != "Greta Hoffmann" {
		t.Errorf("EmployeeName: got %q, want %q", got.EmployeeName, "Greta Hoffmann")
	}
}

// TestRepository_GetAbsenceCalendar_BlankNameFallsBackToEmail is the
// regression this unit exists for. CONCAT_WS(' ', a, b) only skips NULL
// arguments, not empty ones — with two empty strings it still applies the
// separator and returns a single space, which NULLIF(x, ”) does not
// recognize as blank. Every real user with no name set (first_name/last_name
// default to ”, never NULL) therefore fell through to a one-space display
// name instead of the email fallback. This fails against the pre-fix query
// and passes only with a TRIM() around the CONCAT_WS before the NULLIF.
func TestRepository_GetAbsenceCalendar_BlankNameFallsBackToEmail(t *testing.T) {
	pool, tenantID, ctx := newAbsenceFixture(t)
	repo := NewPostgresAbsenceRepo(pool)

	email := "blank-" + uuid.NewString() + "@example.test"
	empID := seedAbsenceUser(t, pool, tenantID, "", "", email)
	typeID := seedAbsenceLeaveType(t, pool, tenantID)
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)
	seedAbsenceLeaveRequest(t, pool, tenantID, empID, typeID, start, end, absenceLeaveRequestOpts{})

	entries, err := repo.GetAbsenceCalendar(ctx, AbsenceFilter{
		TenantID: tenantID, StartDate: start, EndDate: end,
	})
	if err != nil {
		t.Fatalf("GetAbsenceCalendar: %v", err)
	}
	got := findAbsenceEntry(entries, empID)
	if got == nil {
		t.Fatalf("seeded employee %s not present in result", empID)
	}
	if got.EmployeeName != email {
		t.Errorf("EmployeeName: got %q, want the email fallback %q", got.EmployeeName, email)
	}
}

// ============================================================================
// Status filter — only approved requests are visible on the calendar
// ============================================================================

func TestRepository_GetAbsenceCalendar_OnlyApprovedStatusIncluded(t *testing.T) {
	pool, tenantID, ctx := newAbsenceFixture(t)
	repo := NewPostgresAbsenceRepo(pool)
	typeID := seedAbsenceLeaveType(t, pool, tenantID)
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)

	statuses := []string{"pending", "approved", "rejected", "cancelled"}
	employees := make(map[string]uuid.UUID, len(statuses))
	for _, status := range statuses {
		empID := seedAbsenceUser(t, pool, tenantID, "Status", status, status+"-"+uuid.NewString()+"@example.test")
		employees[status] = empID
		seedAbsenceLeaveRequest(t, pool, tenantID, empID, typeID, start, end, absenceLeaveRequestOpts{status: status})
	}

	entries, err := repo.GetAbsenceCalendar(ctx, AbsenceFilter{
		TenantID: tenantID, StartDate: start, EndDate: end,
	})
	if err != nil {
		t.Fatalf("GetAbsenceCalendar: %v", err)
	}

	for _, status := range statuses {
		found := findAbsenceEntry(entries, employees[status]) != nil
		wantFound := status == "approved"
		if found != wantFound {
			t.Errorf("status %q: present=%v, want %v", status, found, wantFound)
		}
	}
}

// ============================================================================
// Date-range overlap — the query uses daterange(...,'[]') && daterange(...,'[]'),
// inclusive on both sides. That is deliberately NOT the half-open interval
// vehicle/machine bookings use elsewhere in the repo to avoid double-booking
// a shared resource: this query decides what is VISIBLE in a calendar window,
// and an absence that ends exactly on the day a requested window starts must
// still show up on that day. Boundary touch is therefore an overlap here.
// ============================================================================

func TestRepository_GetAbsenceCalendar_DateRangeOverlap(t *testing.T) {
	pool, tenantID, ctx := newAbsenceFixture(t)
	repo := NewPostgresAbsenceRepo(pool)
	typeID := seedAbsenceLeaveType(t, pool, tenantID)

	// A fixed absence: 2026-07-10 .. 2026-07-12.
	absStart := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	absEnd := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name         string
		filterStart  time.Time
		filterEnd    time.Time
		wantIncluded bool
	}{
		{
			name:         "fully_inside_filter_window_is_included",
			filterStart:  time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			filterEnd:    time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
			wantIncluded: true,
		},
		{
			name:         "filter_window_touches_absence_end_is_included",
			filterStart:  absEnd,
			filterEnd:    absEnd.AddDate(0, 0, 5),
			wantIncluded: true,
		},
		{
			name:         "filter_window_touches_absence_start_is_included",
			filterStart:  absStart.AddDate(0, 0, -5),
			filterEnd:    absStart,
			wantIncluded: true,
		},
		{
			name:         "filter_window_strictly_before_absence_is_excluded",
			filterStart:  absStart.AddDate(0, 0, -10),
			filterEnd:    absStart.AddDate(0, 0, -1),
			wantIncluded: false,
		},
		{
			name:         "filter_window_strictly_after_absence_is_excluded",
			filterStart:  absEnd.AddDate(0, 0, 1),
			filterEnd:    absEnd.AddDate(0, 0, 10),
			wantIncluded: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			empID := seedAbsenceUser(t, pool, tenantID, "Overlap", tc.name, tc.name+"-"+uuid.NewString()+"@example.test")
			seedAbsenceLeaveRequest(t, pool, tenantID, empID, typeID, absStart, absEnd, absenceLeaveRequestOpts{})

			entries, err := repo.GetAbsenceCalendar(ctx, AbsenceFilter{
				TenantID: tenantID, StartDate: tc.filterStart, EndDate: tc.filterEnd,
			})
			if err != nil {
				t.Fatalf("GetAbsenceCalendar: %v", err)
			}
			got := findAbsenceEntry(entries, empID) != nil
			if got != tc.wantIncluded {
				t.Errorf("included=%v, want %v (filter %s..%s, absence %s..%s)",
					got, tc.wantIncluded,
					tc.filterStart.Format("2006-01-02"), tc.filterEnd.Format("2006-01-02"),
					absStart.Format("2006-01-02"), absEnd.Format("2006-01-02"))
			}
		})
	}
}

// TestRepository_GetAbsenceCalendar_YearBoundary covers an absence and a
// calendar window that both cross a New Year's Eve — dates are the one place
// naive string/lexical comparisons quietly break at the turn of the year.
func TestRepository_GetAbsenceCalendar_YearBoundary(t *testing.T) {
	pool, tenantID, ctx := newAbsenceFixture(t)
	repo := NewPostgresAbsenceRepo(pool)
	typeID := seedAbsenceLeaveType(t, pool, tenantID)

	empID := seedAbsenceUser(t, pool, tenantID, "Silvester", "Test", "silvester-"+uuid.NewString()+"@example.test")
	absStart := time.Date(2026, 12, 29, 0, 0, 0, 0, time.UTC)
	absEnd := time.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC)
	seedAbsenceLeaveRequest(t, pool, tenantID, empID, typeID, absStart, absEnd, absenceLeaveRequestOpts{})

	entries, err := repo.GetAbsenceCalendar(ctx, AbsenceFilter{
		TenantID:  tenantID,
		StartDate: time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2027, 1, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("GetAbsenceCalendar: %v", err)
	}
	got := findAbsenceEntry(entries, empID)
	if got == nil {
		t.Fatalf("year-boundary absence not found in a window spanning the same New Year's Eve")
	}
	if !got.StartDate.Equal(absStart) || !got.EndDate.Equal(absEnd) {
		t.Errorf("dates: got %s..%s, want %s..%s",
			got.StartDate.Format("2006-01-02"), got.EndDate.Format("2006-01-02"),
			absStart.Format("2006-01-02"), absEnd.Format("2006-01-02"))
	}

	// A window entirely in the new year, one day after the absence ends,
	// must NOT see it — proves the year rollover doesn't make the comparison
	// fail open.
	entriesAfter, err := repo.GetAbsenceCalendar(ctx, AbsenceFilter{
		TenantID:  tenantID,
		StartDate: absEnd.AddDate(0, 0, 1),
		EndDate:   time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("GetAbsenceCalendar (after): %v", err)
	}
	if findAbsenceEntry(entriesAfter, empID) != nil {
		t.Errorf("absence ending %s leaked into a window starting %s", absEnd.Format("2006-01-02"), absEnd.AddDate(0, 0, 1).Format("2006-01-02"))
	}
}

// ============================================================================
// Department filter
// ============================================================================

func TestRepository_GetAbsenceCalendar_DepartmentFilter(t *testing.T) {
	pool, tenantID, ctx := newAbsenceFixture(t)
	repo := NewPostgresAbsenceRepo(pool)
	typeID := seedAbsenceLeaveType(t, pool, tenantID)
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)

	engID := seedAbsenceUser(t, pool, tenantID, "Eng", "Ineer", "eng-"+uuid.NewString()+"@example.test")
	seedAbsenceProfile(t, pool, tenantID, engID, "Engineering")
	seedAbsenceLeaveRequest(t, pool, tenantID, engID, typeID, start, end, absenceLeaveRequestOpts{})

	mktID := seedAbsenceUser(t, pool, tenantID, "Mkt", "Ing", "mkt-"+uuid.NewString()+"@example.test")
	seedAbsenceProfile(t, pool, tenantID, mktID, "Marketing")
	seedAbsenceLeaveRequest(t, pool, tenantID, mktID, typeID, start, end, absenceLeaveRequestOpts{})

	entries, err := repo.GetAbsenceCalendar(ctx, AbsenceFilter{
		TenantID: tenantID, StartDate: start, EndDate: end, Department: "Engineering",
	})
	if err != nil {
		t.Fatalf("GetAbsenceCalendar: %v", err)
	}
	if findAbsenceEntry(entries, engID) == nil {
		t.Errorf("Engineering employee missing from Engineering-filtered result")
	}
	if findAbsenceEntry(entries, mktID) != nil {
		t.Errorf("Marketing employee leaked into Engineering-filtered result")
	}
}

// ============================================================================
// Half-day fields round-trip
// ============================================================================

func TestRepository_GetAbsenceCalendar_HalfDayFieldsRoundTrip(t *testing.T) {
	pool, tenantID, ctx := newAbsenceFixture(t)
	repo := NewPostgresAbsenceRepo(pool)
	typeID := seedAbsenceLeaveType(t, pool, tenantID)

	empID := seedAbsenceUser(t, pool, tenantID, "Half", "Day", "half-"+uuid.NewString()+"@example.test")
	morning := string(models.HalfDayMorning)
	afternoon := string(models.HalfDayAfternoon)
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	seedAbsenceLeaveRequest(t, pool, tenantID, empID, typeID, start, end, absenceLeaveRequestOpts{
		isHalfDayStart: true, halfDayPeriodStart: &morning,
		isHalfDayEnd: true, halfDayPeriodEnd: &afternoon,
	})

	entries, err := repo.GetAbsenceCalendar(ctx, AbsenceFilter{
		TenantID: tenantID, StartDate: start, EndDate: end,
	})
	if err != nil {
		t.Fatalf("GetAbsenceCalendar: %v", err)
	}
	got := findAbsenceEntry(entries, empID)
	if got == nil {
		t.Fatalf("seeded employee %s not present in result", empID)
	}
	if !got.IsHalfDayStart || got.HalfDayPeriodStart == nil || *got.HalfDayPeriodStart != models.HalfDayMorning {
		t.Errorf("half-day start: got isHalfDay=%v period=%v, want true/morning", got.IsHalfDayStart, got.HalfDayPeriodStart)
	}
	if !got.IsHalfDayEnd || got.HalfDayPeriodEnd == nil || *got.HalfDayPeriodEnd != models.HalfDayAfternoon {
		t.Errorf("half-day end: got isHalfDay=%v period=%v, want true/afternoon", got.IsHalfDayEnd, got.HalfDayPeriodEnd)
	}
}

// ============================================================================
// Cross-tenant isolation
// ============================================================================

func TestRepository_GetAbsenceCalendar_CrossTenantExcluded(t *testing.T) {
	pool, tenantA, _ := newAbsenceFixture(t)
	tenantB := uuid.New()
	testutil.EnsureTenant(t, pool, tenantB, "Absence DB Test Tenant B")
	repo := NewPostgresAbsenceRepo(pool)

	typeA := seedAbsenceLeaveType(t, pool, tenantA)
	empA := seedAbsenceUser(t, pool, tenantA, "Secret", "Owner", "secret-"+uuid.NewString()+"@example.test")
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)
	seedAbsenceLeaveRequest(t, pool, tenantA, empA, typeA, start, end, absenceLeaveRequestOpts{})

	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	entriesB, err := repo.GetAbsenceCalendar(ctxB, AbsenceFilter{
		TenantID: tenantB, StartDate: start, EndDate: end,
	})
	if err != nil {
		t.Fatalf("GetAbsenceCalendar B: %v", err)
	}
	if findAbsenceEntry(entriesB, empA) != nil {
		t.Errorf("cross-tenant leak: tenant B sees tenant A's absence for employee %s", empA)
	}
}
