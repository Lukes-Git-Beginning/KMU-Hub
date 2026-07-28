package timetracking_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// The whole file runs in SYSTEM context on purpose. System context satisfies
// is_system_context() and therefore switches the RLS policy on
// hr_work_time_entries off — so anything that still fails to find a foreign row
// was filtered by the query's own tenant_id predicate, which is exactly what
// these tests exist to pin. Run under a tenant context instead and RLS would
// pass every assertion even if the predicates were removed again.

type scopeFixture struct {
	pool      *pgxpool.Pool
	repo      *timetracking.PostgresWorkTimeRepo
	ctx       context.Context
	employeeA uuid.UUID
	employeeB uuid.UUID
	entryA    uuid.UUID
	entryB    uuid.UUID
	clockIn   time.Time
}

// newScopeFixture seeds one employee and one work time entry per tenant.
// employee IDs differ per tenant because employee_id is an FK on users and a
// user belongs to exactly one tenant — the attack this guards against is a
// caller in TenantA naming TenantB's employee or entry id, not a shared id.
func newScopeFixture(t *testing.T) *scopeFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	f := &scopeFixture{
		pool:    pool,
		repo:    timetracking.NewPostgresWorkTimeRepo(pool),
		ctx:     testutil.WithSystemCtx(context.Background()),
		clockIn: time.Now().Add(-4 * time.Hour).Truncate(time.Second),
	}

	f.employeeA = testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantA,
		"email":         "timescope-a@test.local",
		"password_hash": "x",
		"first_name":    "Scope",
		"last_name":     "A",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", f.employeeA) })

	f.employeeB = testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     testutil.TenantB,
		"email":         "timescope-b@test.local",
		"password_hash": "x",
		"first_name":    "Scope",
		"last_name":     "B",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", f.employeeB) })

	f.entryA = f.seedEntry(t, testutil.TenantA, f.employeeA)
	f.entryB = f.seedEntry(t, testutil.TenantB, f.employeeB)

	return f
}

func (f *scopeFixture) seedEntry(t *testing.T, tenantID, employeeID uuid.UUID) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, f.pool, "hr_work_time_entries", map[string]any{
		"id":               uuid.New(),
		"tenant_id":        tenantID,
		"employee_id":      employeeID,
		"clock_in":         f.clockIn,
		"clock_out":        f.clockIn.Add(3 * time.Hour),
		"break_minutes":    0,
		"net_work_minutes": 180,
		"status":           "completed",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "hr_work_time_entries", id) })
	return id
}

// TestGetByID_ForeignTenantNotFound is the read half of the finding: without the
// predicate a caller who learned another tenant's entry id got the row back.
func TestGetByID_ForeignTenantNotFound(t *testing.T) {
	f := newScopeFixture(t)

	own, err := f.repo.GetByID(f.ctx, testutil.TenantA, f.entryA)
	if err != nil {
		t.Fatalf("own entry must be readable: %v", err)
	}
	if own.ID != f.entryA {
		t.Fatalf("got entry %s, want %s", own.ID, f.entryA)
	}

	if _, err = f.repo.GetByID(f.ctx, testutil.TenantA, f.entryB); err == nil {
		t.Fatal("TenantA read TenantB's work time entry")
	}
}

func TestGetActiveShift_ForeignTenantNotFound(t *testing.T) {
	f := newScopeFixture(t)

	// One active shift for TenantB only. The unique index on employee_id WHERE
	// status='active' allows exactly one per employee, so this stays valid.
	activeID := testutil.SeedRow(t, f.pool, "hr_work_time_entries", map[string]any{
		"id":          uuid.New(),
		"tenant_id":   testutil.TenantB,
		"employee_id": f.employeeB,
		"clock_in":    time.Now().Add(-time.Hour),
		"status":      "active",
	})
	defer testutil.CleanupRow(t, f.pool, "hr_work_time_entries", activeID)

	shift, err := f.repo.GetActiveShift(f.ctx, testutil.TenantB, f.employeeB)
	if err != nil {
		t.Fatalf("own active shift: %v", err)
	}
	if shift == nil {
		t.Fatal("TenantB must see its own active shift")
	}

	shift, err = f.repo.GetActiveShift(f.ctx, testutil.TenantA, f.employeeB)
	if err != nil {
		t.Fatalf("cross-tenant lookup must not error: %v", err)
	}
	if shift != nil {
		t.Fatal("TenantA saw TenantB's active shift")
	}
}

func TestGetPreviousShiftEnd_ForeignTenantNotFound(t *testing.T) {
	f := newScopeFixture(t)
	before := f.clockIn.Add(24 * time.Hour)

	end, err := f.repo.GetPreviousShiftEnd(f.ctx, testutil.TenantB, f.employeeB, before)
	if err != nil {
		t.Fatalf("own previous shift end: %v", err)
	}
	if end == nil {
		t.Fatal("TenantB must see its own previous shift end")
	}

	end, err = f.repo.GetPreviousShiftEnd(f.ctx, testutil.TenantA, f.employeeB, before)
	if err != nil {
		t.Fatalf("cross-tenant lookup must not error: %v", err)
	}
	if end != nil {
		t.Fatal("TenantA saw TenantB's previous shift end — an 11h rest check would read foreign time")
	}
}

// TestDailyAndWeeklyAggregates_ForeignTenantContributesNothing covers the
// balance path. A leak here is not a wrong display: the minutes of another
// tenant's employee would land in this tenant's overtime balance.
func TestDailyAndWeeklyAggregates_ForeignTenantContributesNothing(t *testing.T) {
	f := newScopeFixture(t)
	day := f.clockIn

	daily, err := f.repo.GetDailySummary(f.ctx, testutil.TenantB, f.employeeB, day)
	if err != nil {
		t.Fatalf("own daily summary: %v", err)
	}
	if daily.EntryCount != 1 {
		t.Fatalf("TenantB daily entry_count = %d, want 1", daily.EntryCount)
	}

	daily, err = f.repo.GetDailySummary(f.ctx, testutil.TenantA, f.employeeB, day)
	if err != nil {
		t.Fatalf("cross-tenant daily summary must not error: %v", err)
	}
	if daily.EntryCount != 0 || daily.NetWorkMinutes != 0 {
		t.Fatalf("TenantA counted TenantB's day: entries=%d net=%d",
			daily.EntryCount, daily.NetWorkMinutes)
	}

	// Same for the batched path, which feeds the team view and the week balance.
	weekStart := day.AddDate(0, 0, -int((day.Weekday()+6)%7))
	batch, err := f.repo.GetWeeklySummaryBatch(f.ctx, testutil.TenantA,
		[]uuid.UUID{f.employeeA, f.employeeB}, weekStart)
	if err != nil {
		t.Fatalf("weekly batch: %v", err)
	}
	if got := batch[f.employeeB]; got == nil || got.NetWorkMinutes != 0 {
		t.Fatalf("TenantA's weekly batch contains TenantB minutes: %+v", got)
	}
	if got := batch[f.employeeA]; got == nil || got.NetWorkMinutes == 0 {
		t.Fatalf("TenantA's own minutes went missing: %+v", got)
	}

	active, err := f.repo.GetActiveShiftEmployeeIDs(f.ctx, testutil.TenantA,
		[]uuid.UUID{f.employeeA, f.employeeB})
	if err != nil {
		t.Fatalf("active shift ids: %v", err)
	}
	if active[f.employeeB] {
		t.Fatal("TenantA saw TenantB's employee as clocked in")
	}
}

// TestUpdate_ForeignTenantWriteMissesRow is the write half, and the reason this
// unit is more than tidiness: `WHERE id = $1` alone reaches another tenant's row
// in any context where app.tenant_id is not set.
func TestUpdate_ForeignTenantWriteMissesRow(t *testing.T) {
	f := newScopeFixture(t)

	entryB, err := f.repo.GetByID(f.ctx, testutil.TenantB, f.entryB)
	if err != nil {
		t.Fatalf("load TenantB entry: %v", err)
	}

	// Same row, but stamped with the attacking tenant.
	forged := *entryB
	forged.TenantID = testutil.TenantA
	forged.NetWorkMinutes = ptrInt(999)
	forged.UpdatedAt = time.Now()

	if err = f.repo.Update(f.ctx, &forged); err != nil {
		t.Fatalf("update must not error, it must simply match no row: %v", err)
	}

	after, err := f.repo.GetByID(f.ctx, testutil.TenantB, f.entryB)
	if err != nil {
		t.Fatalf("reload TenantB entry: %v", err)
	}
	if after.NetWorkMinutes == nil {
		t.Fatal("TenantA's write blanked TenantB's net_work_minutes")
	}
	if *after.NetWorkMinutes != 180 {
		t.Fatalf("TenantA's write reached TenantB's row: net_work_minutes = %d",
			*after.NetWorkMinutes)
	}
}

func ptrInt(v int) *int { return &v }
