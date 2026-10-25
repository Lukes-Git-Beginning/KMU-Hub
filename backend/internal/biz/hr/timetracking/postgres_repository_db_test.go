package timetracking_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// System context on purpose (see postgres_tenant_scope_test.go): RLS is off, so
// every foreign row that stays invisible was filtered by the query's own
// tenant_id predicate.

type repoFixture struct {
	pool      *pgxpool.Pool
	repo      *timetracking.PostgresWorkTimeRepo
	ctx       context.Context
	employeeA uuid.UUID
	employeeB uuid.UUID // other tenant
	day       time.Time // Monday 00:00 UTC, far enough back to stay clear of other tests
}

func newRepoFixture(t *testing.T) *repoFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	f := &repoFixture{
		pool: pool,
		repo: timetracking.NewPostgresWorkTimeRepo(pool),
		ctx:  testutil.WithSystemCtx(context.Background()),
		day:  time.Date(2023, 3, 6, 0, 0, 0, 0, time.UTC),
	}
	f.employeeA = f.seedUser(t, testutil.TenantA)
	f.employeeB = f.seedUser(t, testutil.TenantB)
	return f
}

func (f *repoFixture) seedUser(t *testing.T, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	testutil.SeedRow(t, f.pool, "users", map[string]any{
		"id":            id,
		"tenant_id":     tenantID,
		"email":         "ttrepo-" + id.String() + "@test.local",
		"password_hash": "x",
		"first_name":    "Repo",
		"last_name":     "Test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "users", id) })
	return id
}

// entry builds a finished entry on f.day+dayOffset starting at hour, lasting net
// minutes with no break.
func (f *repoFixture) entry(tenantID, employeeID uuid.UUID, dayOffset, hour, net int, status models.WorkTimeEntryStatus) *models.HRWorkTimeEntry {
	in := f.day.AddDate(0, 0, dayOffset).Add(time.Duration(hour) * time.Hour)
	out := in.Add(time.Duration(net) * time.Minute)
	n := net
	now := time.Now()
	return &models.HRWorkTimeEntry{
		ID: uuid.New(), TenantID: tenantID, EmployeeID: employeeID,
		ClockIn: in, ClockOut: &out, NetWorkMinutes: &n, Status: status,
		CreatedAt: now, UpdatedAt: now,
	}
}

func (f *repoFixture) create(t *testing.T, e *models.HRWorkTimeEntry) *models.HRWorkTimeEntry {
	t.Helper()
	if err := f.repo.Create(f.ctx, e); err != nil {
		t.Fatalf("create entry: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "hr_work_time_entries", e.ID) })
	return e
}

func (f *repoFixture) listFilter(emp *uuid.UUID) timetracking.WorkTimeFilter {
	return timetracking.WorkTimeFilter{TenantID: testutil.TenantA, EmployeeID: emp}
}

func TestCreateAndGetByID_RoundTripsAllColumns(t *testing.T) {
	f := newRepoFixture(t)
	lat, lng, addr := 49.45, 11.07, "Nürnberg"
	e := f.entry(testutil.TenantA, f.employeeA, 0, 9, 240, models.WorkTimeStatusCompleted)
	e.BreakMinutes, e.AutoBreakDeducted = 15, 15
	e.LocationLat, e.LocationLng, e.LocationAddress = &lat, &lng, &addr
	f.create(t, e)

	got, err := f.repo.GetByID(f.ctx, testutil.TenantA, e.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.BreakMinutes != 15 || got.AutoBreakDeducted != 15 || got.NetWorkMinutes == nil || *got.NetWorkMinutes != 240 {
		t.Fatalf("break/net not persisted: %+v", got)
	}
	if got.LocationLat == nil || *got.LocationLat != lat || got.LocationAddress == nil || *got.LocationAddress != addr {
		t.Fatalf("location not persisted: %+v", got)
	}
	if got.Status != models.WorkTimeStatusCompleted {
		t.Fatalf("status = %s", got.Status)
	}
}

func TestApproveCorrection_SupersedesOriginalAtomically(t *testing.T) {
	f := newRepoFixture(t)
	orig := f.create(t, f.entry(testutil.TenantA, f.employeeA, 0, 9, 120, models.WorkTimeStatusCompleted))

	corr := f.entry(testutil.TenantA, f.employeeA, 0, 9, 180, models.WorkTimeStatusCorrectionPending)
	corr.IsCorrection, corr.OriginalEntryID, corr.CorrectionReason = true, &orig.ID, "forgot to clock out"
	f.create(t, corr)

	approver := f.employeeA
	at := time.Now()
	corr.Status = models.WorkTimeStatusCorrectionApproved
	corr.CorrectionApprovedBy, corr.CorrectionApprovedAt, corr.UpdatedAt = &approver, &at, at

	if err := f.repo.ApproveCorrection(f.ctx, corr, &orig.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}

	gotCorr, err := f.repo.GetByID(f.ctx, testutil.TenantA, corr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotCorr.Status != models.WorkTimeStatusCorrectionApproved || gotCorr.CorrectionApprovedBy == nil || *gotCorr.CorrectionApprovedBy != approver {
		t.Fatalf("correction not approved: %+v", gotCorr)
	}
	gotOrig, err := f.repo.GetByID(f.ctx, testutil.TenantA, orig.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotOrig.Status != models.WorkTimeStatusSuperseded {
		t.Fatalf("original status = %s, want superseded", gotOrig.Status)
	}

	// The day now counts the correction once, not original + correction.
	sum, err := f.repo.GetDailySummary(f.ctx, testutil.TenantA, f.employeeA, f.day)
	if err != nil {
		t.Fatal(err)
	}
	if sum.NetWorkMinutes != 180 || sum.EntryCount != 1 {
		t.Fatalf("day = %d min / %d entries, want 180 / 1", sum.NetWorkMinutes, sum.EntryCount)
	}
}

func TestApproveCorrection_DoesNotTouchForeignTenantOriginal(t *testing.T) {
	f := newRepoFixture(t)
	foreign := f.create(t, f.entry(testutil.TenantB, f.employeeB, 0, 9, 120, models.WorkTimeStatusCompleted))

	corr := f.create(t, f.entry(testutil.TenantA, f.employeeA, 0, 9, 180, models.WorkTimeStatusCorrectionPending))
	corr.Status = models.WorkTimeStatusCorrectionApproved
	if err := f.repo.ApproveCorrection(f.ctx, corr, &foreign.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}

	got, err := f.repo.GetByID(f.ctx, testutil.TenantB, foreign.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.WorkTimeStatusCompleted {
		t.Fatalf("TenantA's approval superseded TenantB's entry: status = %s", got.Status)
	}
}

func TestApproveCorrection_AlreadyApprovedOriginalStaysPut(t *testing.T) {
	f := newRepoFixture(t)
	orig := f.create(t, f.entry(testutil.TenantA, f.employeeA, 0, 9, 120, models.WorkTimeStatusCorrectionApproved))

	corr := f.create(t, f.entry(testutil.TenantA, f.employeeA, 1, 9, 60, models.WorkTimeStatusCorrectionPending))
	corr.Status = models.WorkTimeStatusCorrectionApproved
	if err := f.repo.ApproveCorrection(f.ctx, corr, &orig.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	got, err := f.repo.GetByID(f.ctx, testutil.TenantA, orig.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.WorkTimeStatusCorrectionApproved {
		t.Fatalf("an approved correction was pulled to %s by a second approval", got.Status)
	}
}

func TestApproveCorrection_WithoutOriginal(t *testing.T) {
	f := newRepoFixture(t)
	corr := f.create(t, f.entry(testutil.TenantA, f.employeeA, 0, 9, 60, models.WorkTimeStatusCorrectionPending))
	corr.Status = models.WorkTimeStatusCorrectionApproved
	if err := f.repo.ApproveCorrection(f.ctx, corr, nil); err != nil {
		t.Fatalf("approve without original: %v", err)
	}
	got, err := f.repo.GetByID(f.ctx, testutil.TenantA, corr.ID)
	if err != nil || got.Status != models.WorkTimeStatusCorrectionApproved {
		t.Fatalf("got %+v, err %v", got, err)
	}
}

func TestList_FiltersPaginationAndTenant(t *testing.T) {
	f := newRepoFixture(t)
	for i := 0; i < 3; i++ {
		f.create(t, f.entry(testutil.TenantA, f.employeeA, i, 9, 60, models.WorkTimeStatusCompleted))
	}
	f.create(t, f.entry(testutil.TenantA, f.employeeA, 3, 9, 60, models.WorkTimeStatusCorrectionPending))
	f.create(t, f.entry(testutil.TenantB, f.employeeB, 0, 9, 60, models.WorkTimeStatusCompleted))

	emp := f.employeeA
	base := f.listFilter(&emp)

	all, total, err := f.repo.List(f.ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 || len(all) != 4 {
		t.Fatalf("employee filter: total %d, len %d, want 4/4", total, len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].ClockIn.Before(all[i].ClockIn) {
			t.Fatal("entries not ordered by clock_in DESC")
		}
	}

	pending := string(models.WorkTimeStatusCorrectionPending)
	st := base
	st.Status = &pending
	got, total, err := f.repo.List(f.ctx, st)
	if err != nil || total != 1 || len(got) != 1 {
		t.Fatalf("status filter: total %d len %d err %v", total, len(got), err)
	}

	// DateTo covers its whole day: days 0..1 -> 2 entries.
	from, to := f.day, f.day.AddDate(0, 0, 1)
	dr := base
	dr.DateFrom, dr.DateTo = &from, &to
	got, total, err = f.repo.List(f.ctx, dr)
	if err != nil || total != 2 || len(got) != 2 {
		t.Fatalf("date filter: total %d len %d err %v", total, len(got), err)
	}

	// Pagination: total stays 4, page 2 of size 3 holds the last one.
	pg := base
	pg.PerPage, pg.Page = 3, 2
	got, total, err = f.repo.List(f.ctx, pg)
	if err != nil || total != 4 || len(got) != 1 {
		t.Fatalf("page 2: total %d len %d err %v", total, len(got), err)
	}

	// Without an employee filter TenantA never sees TenantB's row on day 0.
	got, _, err = f.repo.List(f.ctx, timetracking.WorkTimeFilter{TenantID: testutil.TenantA, DateFrom: &from, DateTo: &from})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range got {
		if e.TenantID != testutil.TenantA {
			t.Fatalf("TenantA list returned a %s row", e.TenantID)
		}
	}

	none := uuid.New()
	empty, total, err := f.repo.List(f.ctx, f.listFilter(&none))
	if err != nil || total != 0 || empty == nil {
		t.Fatalf("empty list: total %d, nil=%v, err %v", total, empty == nil, err)
	}
}

func TestGetDailySummaryRange_BucketsAndBalanceStatuses(t *testing.T) {
	f := newRepoFixture(t)
	f.create(t, f.entry(testutil.TenantA, f.employeeA, 0, 8, 300, models.WorkTimeStatusCompleted))
	f.create(t, f.entry(testutil.TenantA, f.employeeA, 0, 14, 240, models.WorkTimeStatusCompleted)) // day 0: 540 -> 60 overtime
	f.create(t, f.entry(testutil.TenantA, f.employeeA, 2, 9, 100, models.WorkTimeStatusCorrectionApproved))
	f.create(t, f.entry(testutil.TenantA, f.employeeA, 1, 9, 500, models.WorkTimeStatusSuperseded))         // not in the balance
	f.create(t, f.entry(testutil.TenantA, f.employeeA, 1, 12, 500, models.WorkTimeStatusCorrectionPending)) // not in the balance
	f.create(t, f.entry(testutil.TenantB, f.employeeB, 0, 9, 999, models.WorkTimeStatusCompleted))

	days, err := f.repo.GetDailySummaryRange(f.ctx, testutil.TenantA, f.employeeA, f.day, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 3 {
		t.Fatalf("len = %d, want 3", len(days))
	}
	if days[0].NetWorkMinutes != 540 || days[0].EntryCount != 2 || days[0].OvertimeMinutes != 60 {
		t.Fatalf("day 0 = %+v", days[0])
	}
	if days[1].NetWorkMinutes != 0 || days[1].EntryCount != 0 {
		t.Fatalf("day 1 counts a superseded/pending entry: %+v", days[1])
	}
	if days[2].NetWorkMinutes != 100 || days[2].OvertimeMinutes != 0 {
		t.Fatalf("day 2 = %+v", days[2])
	}
	if !days[1].Date.Equal(f.day.AddDate(0, 0, 1)) {
		t.Fatalf("empty day lost its date label: %v", days[1].Date)
	}

	if none, err := f.repo.GetDailySummaryRange(f.ctx, testutil.TenantA, f.employeeA, f.day, 0); err != nil || len(none) != 0 {
		t.Fatalf("numDays 0: %v, %v", none, err)
	}
}

func TestGetProjectBreakdown_GroupsBillableByProject(t *testing.T) {
	f := newRepoFixture(t)
	projects := timetracking.NewPostgresTimeProjectRepo(f.pool)
	now := time.Now()
	p := &models.HRTimeProject{ID: uuid.New(), TenantID: testutil.TenantA, Name: "Relaunch", Color: "#112233", CreatedAt: now, UpdatedAt: now}
	if err := projects.Create(f.ctx, p); err != nil {
		t.Fatalf("create project: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "hr_time_projects", p.ID) })

	a := f.entry(testutil.TenantA, f.employeeA, 0, 9, 120, models.WorkTimeStatusCompleted)
	a.ProjectID = &p.ID
	f.create(t, a)
	b := f.entry(testutil.TenantA, f.employeeA, 1, 9, 60, models.WorkTimeStatusCorrectionApproved)
	b.ProjectID = &p.ID
	f.create(t, b)
	f.create(t, f.entry(testutil.TenantA, f.employeeA, 1, 13, 30, models.WorkTimeStatusCompleted))    // no project
	f.create(t, f.entry(testutil.TenantA, f.employeeA, 1, 15, 700, models.WorkTimeStatusSuperseded)) // not billable
	f.create(t, f.entry(testutil.TenantA, f.employeeA, 1, 17, 700, models.WorkTimeStatusActive))     // not billable

	got, err := f.repo.GetProjectBreakdown(f.ctx, testutil.TenantA, f.employeeA, f.day, f.day.AddDate(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d groups, want 2: %+v", len(got), got)
	}
	if got[0].ProjectID != p.ID || got[0].ProjectName != "Relaunch" || got[0].Minutes != 180 {
		t.Fatalf("first group = %+v, want Relaunch/180", got[0])
	}
	if got[1].ProjectID != uuid.Nil || got[1].ProjectName != "Kein Projekt" || got[1].Minutes != 30 {
		t.Fatalf("second group = %+v, want Kein Projekt/30", got[1])
	}

	other, err := f.repo.GetProjectBreakdown(f.ctx, testutil.TenantB, f.employeeA, f.day, f.day.AddDate(0, 0, 1))
	if err != nil || len(other) != 0 {
		t.Fatalf("foreign tenant breakdown = %+v, err %v", other, err)
	}
}

func TestGetActiveShiftEmployeeIDs_ScopedToTenant(t *testing.T) {
	f := newRepoFixture(t)
	active := f.entry(testutil.TenantA, f.employeeA, 0, 9, 0, models.WorkTimeStatusActive)
	active.ClockOut, active.NetWorkMinutes = nil, nil
	f.create(t, active)

	ids := []uuid.UUID{f.employeeA, f.employeeB}
	got, err := f.repo.GetActiveShiftEmployeeIDs(f.ctx, testutil.TenantA, ids)
	if err != nil {
		t.Fatal(err)
	}
	if !got[f.employeeA] || got[f.employeeB] {
		t.Fatalf("active set = %v", got)
	}
	if other, err := f.repo.GetActiveShiftEmployeeIDs(f.ctx, testutil.TenantB, ids); err != nil || other[f.employeeA] {
		t.Fatalf("TenantB saw TenantA's active shift: %v, %v", other, err)
	}
	if empty, err := f.repo.GetActiveShiftEmployeeIDs(f.ctx, testutil.TenantA, nil); err != nil || len(empty) != 0 {
		t.Fatalf("empty ids: %v, %v", empty, err)
	}
}

func TestTimeCategoryTemplateProjectRepos_CRUDAndTenantList(t *testing.T) {
	f := newRepoFixture(t)
	now := time.Now()

	cats := timetracking.NewPostgresTimeCategoryRepo(f.pool)
	c := &models.HRTimeCategory{ID: uuid.New(), TenantID: testutil.TenantA, Name: "Beratung", Color: "#abcdef", Icon: "Tag", CreatedAt: now, UpdatedAt: now}
	if err := cats.Create(f.ctx, c); err != nil {
		t.Fatalf("create category: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "hr_time_categories", c.ID) })
	foreign := &models.HRTimeCategory{ID: uuid.New(), TenantID: testutil.TenantB, Name: "Fremd", Color: "#abcdef", Icon: "Tag", CreatedAt: now, UpdatedAt: now}
	if err := cats.Create(f.ctx, foreign); err != nil {
		t.Fatalf("create foreign category: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "hr_time_categories", foreign.ID) })

	c.Name, c.IsDefault = "Beratung vor Ort", true
	if err := cats.Update(f.ctx, c); err != nil {
		t.Fatalf("update category: %v", err)
	}
	got, err := cats.GetByID(f.ctx, c.ID)
	if err != nil || got.Name != "Beratung vor Ort" || !got.IsDefault {
		t.Fatalf("get category: %+v, %v", got, err)
	}
	list, err := cats.List(f.ctx, testutil.TenantA)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range list {
		if x.ID == foreign.ID {
			t.Fatal("category list leaked a foreign tenant row")
		}
	}
	if _, err := cats.GetByID(f.ctx, uuid.New()); err != timetracking.ErrTimeCategoryNotFound {
		t.Fatalf("unknown category err = %v", err)
	}

	tpls := timetracking.NewPostgresTimeTemplateRepo(f.pool)
	tpl := &models.HRTimeTemplate{ID: uuid.New(), TenantID: testutil.TenantA, Name: "Jour fixe", CategoryID: &c.ID, EstimatedMinutes: 45, CreatedAt: now, UpdatedAt: now}
	if err := tpls.Create(f.ctx, tpl); err != nil {
		t.Fatalf("create template: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "hr_time_templates", tpl.ID) })
	tplList, err := tpls.List(f.ctx, testutil.TenantA)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, x := range tplList {
		if x.ID == tpl.ID {
			found = x.EstimatedMinutes == 45 && x.CategoryID != nil
		}
	}
	if !found {
		t.Fatalf("template not listed with its fields: %+v", tplList)
	}
	if otherTpls, err := tpls.List(f.ctx, testutil.TenantB); err != nil || len(otherTpls) != 0 {
		t.Fatalf("foreign template list: %+v, %v", otherTpls, err)
	}
	if err := tpls.Delete(f.ctx, tpl.ID); err != nil {
		t.Fatal(err)
	}
	if tplList, err = tpls.List(f.ctx, testutil.TenantA); err != nil {
		t.Fatal(err)
	}
	for _, x := range tplList {
		if x.ID == tpl.ID {
			t.Fatal("template survived delete")
		}
	}

	projects := timetracking.NewPostgresTimeProjectRepo(f.pool)
	p := &models.HRTimeProject{ID: uuid.New(), TenantID: testutil.TenantA, Name: "Audit", CustomerName: "ACME", Color: "#112233", BillableDefault: true, CreatedAt: now, UpdatedAt: now}
	if err := projects.Create(f.ctx, p); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "hr_time_projects", p.ID) })
	pl, err := projects.List(f.ctx, testutil.TenantA)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, x := range pl {
		if x.ID == p.ID {
			found = x.BillableDefault && x.CustomerName == "ACME"
		}
	}
	if !found {
		t.Fatalf("project not listed with its fields: %+v", pl)
	}

	if err := cats.Delete(f.ctx, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := cats.GetByID(f.ctx, c.ID); err != timetracking.ErrTimeCategoryNotFound {
		t.Fatalf("category survived delete: %v", err)
	}
}
