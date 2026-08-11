package schichten

// Covers the PostgresRepository methods left untested by tenant_write_test.go
// (plain INSERT paths) and own_scope_list_test.go (ListSwapRequests own-scope
// filter, which already exercises the repository, not just the service):
// Update/Delete/Get/List across shifts, assignments and templates,
// PublishShifts, the ArbZG rest-period lookups, ShiftExistsForTemplate,
// GetStats, the swap-request Get/UpdateStatus/atomic-swap trio, and
// IsMinorEmployee. Runs against the real schema, not a mock, so a forgotten
// WHERE clause, wrong ORDER BY, or mismatched column shows up as a genuine
// query failure or wrong row count/order instead of passing silently.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func setupSchichtenRepo(t *testing.T) (*PostgresRepository, *pgxpool.Pool, context.Context, uuid.UUID) {
	t.Helper()
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Schichten Repo Test Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	return repo, pool, ctx, tenantID
}

func newTestShift(t *testing.T, repo *PostgresRepository, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, title string, status ShiftStatus, start, end time.Time) *Shift {
	t.Helper()
	now := time.Now().UTC()
	shift := &Shift{
		ID: uuid.New(), TenantID: tenantID, Title: title, Status: status,
		StartTime: start, EndTime: end, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateShift(ctx, shift); err != nil {
		t.Fatalf("seed shift %s: %v", title, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "shifts", shift.ID) })
	return shift
}

func newTestAssignment(t *testing.T, repo *PostgresRepository, ctx context.Context, pool *pgxpool.Pool, tenantID, shiftID, employeeID uuid.UUID, assignedAt time.Time) *ShiftAssignment {
	t.Helper()
	a := &ShiftAssignment{ID: uuid.New(), TenantID: tenantID, ShiftID: shiftID, EmployeeID: employeeID, AssignedAt: assignedAt}
	if err := repo.CreateAssignment(ctx, a); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "shift_assignments", a.ID) })
	return a
}

func newTestTemplate(t *testing.T, repo *PostgresRepository, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, name string, dayOfWeek, startHour, startMinute, durationMinutes int) *ShiftTemplate {
	t.Helper()
	now := time.Now().UTC()
	tpl := &ShiftTemplate{
		ID: uuid.New(), TenantID: tenantID, Name: name, DayOfWeek: dayOfWeek,
		StartHour: startHour, StartMinute: startMinute, DurationMinutes: durationMinutes,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateTemplate(ctx, tpl); err != nil {
		t.Fatalf("seed template %s: %v", name, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "shift_templates", tpl.ID) })
	return tpl
}

func newTestSwapRequest(t *testing.T, repo *PostgresRepository, ctx context.Context, pool *pgxpool.Pool, tenantID, assignmentID, shiftID, requestedBy, swapWith uuid.UUID, idemKey string) *SwapRequest {
	t.Helper()
	now := time.Now().UTC()
	req := &SwapRequest{
		ID: uuid.New(), TenantID: tenantID, AssignmentID: assignmentID, ShiftID: shiftID,
		RequestedByEmployeeID: requestedBy, SwapWithEmployeeID: swapWith,
		Status: SwapRequestStatusPending, IdempotencyKey: idemKey,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSwapRequest(ctx, req); err != nil {
		t.Fatalf("seed swap request %s: %v", idemKey, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "shift_swap_requests", req.ID) })
	return req
}

// ============================================================================
// Shifts
// ============================================================================

func TestUpdateShift_UpdatesFieldsAndRejectsUnknownOrForeignID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	now := time.Now().UTC()

	shift := newTestShift(t, repo, ctx, pool, tenantID, "Fruehschicht", ShiftStatusDraft, now, now.Add(8*time.Hour))

	loc := "Halle 2"
	shift.Title = "Fruehschicht (revidiert)"
	shift.Description = "angepasst"
	shift.Status = ShiftStatusPublished
	shift.Location = &loc
	shift.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateShift(ctx, shift); err != nil {
		t.Fatalf("UpdateShift: %v", err)
	}

	reloaded, err := repo.GetShift(ctx, tenantID, shift.ID)
	if err != nil {
		t.Fatalf("GetShift after update: %v", err)
	}
	if reloaded.Title != "Fruehschicht (revidiert)" || reloaded.Status != ShiftStatusPublished {
		t.Fatalf("update did not persist: got title=%q status=%q", reloaded.Title, reloaded.Status)
	}
	if reloaded.Location == nil || *reloaded.Location != loc {
		t.Fatalf("location did not persist: got %v", reloaded.Location)
	}

	unknown := &Shift{ID: uuid.New(), TenantID: tenantID, Title: "x", UpdatedAt: time.Now().UTC()}
	if err := repo.UpdateShift(ctx, unknown); !errors.Is(err, ErrShiftNotFound) {
		t.Fatalf("expected ErrShiftNotFound for unknown shift, got %v", err)
	}

	foreignTenant := uuid.New()
	testutil.EnsureTenant(t, pool, foreignTenant, "Schichten Foreign Tenant")
	foreignAttempt := &Shift{ID: shift.ID, TenantID: foreignTenant, Title: "hijacked", UpdatedAt: time.Now().UTC()}
	if err := repo.UpdateShift(testutil.WithTenantCtx(context.Background(), foreignTenant), foreignAttempt); !errors.Is(err, ErrShiftNotFound) {
		t.Fatalf("expected ErrShiftNotFound for cross-tenant update, got %v", err)
	}
}

func TestDeleteShift_RemovesRowAndRejectsDoubleDelete(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	now := time.Now().UTC()

	shift := newTestShift(t, repo, ctx, pool, tenantID, "Loeschbar", ShiftStatusDraft, now, now.Add(4*time.Hour))

	if err := repo.DeleteShift(ctx, tenantID, shift.ID); err != nil {
		t.Fatalf("DeleteShift: %v", err)
	}
	if _, err := repo.GetShift(ctx, tenantID, shift.ID); !errors.Is(err, ErrShiftNotFound) {
		t.Fatalf("expected ErrShiftNotFound after delete, got %v", err)
	}
	if err := repo.DeleteShift(ctx, tenantID, shift.ID); !errors.Is(err, ErrShiftNotFound) {
		t.Fatalf("expected ErrShiftNotFound on double delete, got %v", err)
	}
}

func TestGetShift_ReturnsRowOrNotFound(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	now := time.Now().UTC()

	shift := newTestShift(t, repo, ctx, pool, tenantID, "Sichtbar", ShiftStatusDraft, now, now.Add(2*time.Hour))

	got, err := repo.GetShift(ctx, tenantID, shift.ID)
	if err != nil {
		t.Fatalf("GetShift: %v", err)
	}
	if got.ID != shift.ID || got.Title != shift.Title {
		t.Fatalf("GetShift returned wrong row: %+v", got)
	}

	if _, err := repo.GetShift(ctx, tenantID, uuid.New()); !errors.Is(err, ErrShiftNotFound) {
		t.Fatalf("expected ErrShiftNotFound for unknown id, got %v", err)
	}
}

func TestListShifts_FiltersOrdersAndPaginates(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	base := time.Now().UTC().Truncate(time.Second).Add(48 * time.Hour)

	s1 := newTestShift(t, repo, ctx, pool, tenantID, "S1", ShiftStatusDraft, base.Add(1*time.Hour), base.Add(2*time.Hour))
	s2 := newTestShift(t, repo, ctx, pool, tenantID, "S2", ShiftStatusPublished, base.Add(2*time.Hour), base.Add(3*time.Hour))
	s3 := newTestShift(t, repo, ctx, pool, tenantID, "S3", ShiftStatusDraft, base.Add(3*time.Hour), base.Add(4*time.Hour))

	draft := ShiftStatusDraft
	shifts, total, err := repo.ListShifts(ctx, tenantID, ListShiftsFilter{Status: &draft}, 0, 50)
	if err != nil {
		t.Fatalf("ListShifts (status filter): %v", err)
	}
	if total != 2 || len(shifts) != 2 {
		t.Fatalf("status filter: got %d rows / total %d, want 2 / 2", len(shifts), total)
	}
	if shifts[0].ID != s1.ID || shifts[1].ID != s3.ID {
		t.Fatalf("status filter: expected [s1, s3] ordered by start_time, got [%s, %s]", shifts[0].Title, shifts[1].Title)
	}

	from := base.Add(2 * time.Hour)
	shifts, total, err = repo.ListShifts(ctx, tenantID, ListShiftsFilter{From: &from}, 0, 50)
	if err != nil {
		t.Fatalf("ListShifts (from filter): %v", err)
	}
	if total != 2 || shifts[0].ID != s2.ID || shifts[1].ID != s3.ID {
		t.Fatalf("from filter: expected [s2, s3], got total=%d", total)
	}

	all, total, err := repo.ListShifts(ctx, tenantID, ListShiftsFilter{}, 1, 2)
	if err != nil {
		t.Fatalf("ListShifts (pagination): %v", err)
	}
	if total != 3 || len(all) != 2 || all[0].ID != s2.ID || all[1].ID != s3.ID {
		t.Fatalf("pagination: expected [s2, s3] with total 3, got %d rows total=%d", len(all), total)
	}
}

func TestPublishShifts_OnlyDraftShiftsInRangeChange(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	base := time.Now().UTC().Truncate(time.Second).Add(72 * time.Hour)

	inRangeDraft := newTestShift(t, repo, ctx, pool, tenantID, "InRangeDraft", ShiftStatusDraft, base.Add(1*time.Hour), base.Add(2*time.Hour))
	outOfRangeDraft := newTestShift(t, repo, ctx, pool, tenantID, "OutOfRangeDraft", ShiftStatusDraft, base.Add(10*time.Hour), base.Add(11*time.Hour))
	alreadyPublished := newTestShift(t, repo, ctx, pool, tenantID, "AlreadyPublished", ShiftStatusPublished, base.Add(1*time.Hour), base.Add(2*time.Hour))

	affected, err := repo.PublishShifts(ctx, tenantID, base, base.Add(5*time.Hour))
	if err != nil {
		t.Fatalf("PublishShifts: %v", err)
	}
	if affected != 1 {
		t.Fatalf("expected exactly 1 shift to change status, got %d", affected)
	}

	reloadedIn, err := repo.GetShift(ctx, tenantID, inRangeDraft.ID)
	if err != nil || reloadedIn.Status != ShiftStatusPublished {
		t.Fatalf("in-range draft was not published: status=%v err=%v", reloadedIn, err)
	}
	reloadedOut, err := repo.GetShift(ctx, tenantID, outOfRangeDraft.ID)
	if err != nil || reloadedOut.Status != ShiftStatusDraft {
		t.Fatalf("out-of-range draft should stay draft: status=%v err=%v", reloadedOut, err)
	}
	reloadedPublished, err := repo.GetShift(ctx, tenantID, alreadyPublished.ID)
	if err != nil || reloadedPublished.Status != ShiftStatusPublished {
		t.Fatalf("already-published shift should stay published: status=%v err=%v", reloadedPublished, err)
	}
}

// ============================================================================
// Assignments
// ============================================================================

func TestDeleteAssignment_RemovesRowAndRejectsDoubleDelete(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	now := time.Now().UTC()

	shift := newTestShift(t, repo, ctx, pool, tenantID, "Assign Shift", ShiftStatusDraft, now, now.Add(4*time.Hour))
	employeeID := uuid.New()
	newTestAssignment(t, repo, ctx, pool, tenantID, shift.ID, employeeID, now)

	if err := repo.DeleteAssignment(ctx, tenantID, shift.ID, employeeID); err != nil {
		t.Fatalf("DeleteAssignment: %v", err)
	}
	if _, err := repo.GetAssignment(ctx, tenantID, shift.ID, employeeID); !errors.Is(err, ErrAssignmentNotFound) {
		t.Fatalf("expected ErrAssignmentNotFound after delete, got %v", err)
	}
	if err := repo.DeleteAssignment(ctx, tenantID, shift.ID, employeeID); !errors.Is(err, ErrAssignmentNotFound) {
		t.Fatalf("expected ErrAssignmentNotFound on double delete, got %v", err)
	}
}

func TestGetAssignment_ReturnsRowOrNotFound(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	now := time.Now().UTC()

	shift := newTestShift(t, repo, ctx, pool, tenantID, "Assign Shift", ShiftStatusDraft, now, now.Add(4*time.Hour))
	employeeID := uuid.New()
	a := newTestAssignment(t, repo, ctx, pool, tenantID, shift.ID, employeeID, now)

	got, err := repo.GetAssignment(ctx, tenantID, shift.ID, employeeID)
	if err != nil {
		t.Fatalf("GetAssignment: %v", err)
	}
	if got.ID != a.ID {
		t.Fatalf("GetAssignment returned wrong row: got %s want %s", got.ID, a.ID)
	}

	if _, err := repo.GetAssignment(ctx, tenantID, shift.ID, uuid.New()); !errors.Is(err, ErrAssignmentNotFound) {
		t.Fatalf("expected ErrAssignmentNotFound for unknown employee, got %v", err)
	}
}

func TestListAssignments_OrdersByAssignedAt(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	now := time.Now().UTC()

	shift := newTestShift(t, repo, ctx, pool, tenantID, "Assign Shift", ShiftStatusDraft, now, now.Add(8*time.Hour))
	later := newTestAssignment(t, repo, ctx, pool, tenantID, shift.ID, uuid.New(), now.Add(2*time.Hour))
	earlier := newTestAssignment(t, repo, ctx, pool, tenantID, shift.ID, uuid.New(), now.Add(1*time.Hour))

	assignments, err := repo.ListAssignments(ctx, tenantID, shift.ID)
	if err != nil {
		t.Fatalf("ListAssignments: %v", err)
	}
	if len(assignments) != 2 || assignments[0].ID != earlier.ID || assignments[1].ID != later.ID {
		t.Fatalf("expected [earlier, later] ordered by assigned_at, got %+v", assignments)
	}
}

func TestCountAssignments_CountsOnlyMatchingShift(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	now := time.Now().UTC()

	shiftA := newTestShift(t, repo, ctx, pool, tenantID, "Shift A", ShiftStatusDraft, now, now.Add(4*time.Hour))
	shiftB := newTestShift(t, repo, ctx, pool, tenantID, "Shift B", ShiftStatusDraft, now, now.Add(4*time.Hour))
	newTestAssignment(t, repo, ctx, pool, tenantID, shiftA.ID, uuid.New(), now)
	newTestAssignment(t, repo, ctx, pool, tenantID, shiftA.ID, uuid.New(), now)

	countA, err := repo.CountAssignments(ctx, tenantID, shiftA.ID)
	if err != nil {
		t.Fatalf("CountAssignments (A): %v", err)
	}
	if countA != 2 {
		t.Fatalf("expected 2 assignments for shift A, got %d", countA)
	}

	countB, err := repo.CountAssignments(ctx, tenantID, shiftB.ID)
	if err != nil {
		t.Fatalf("CountAssignments (B): %v", err)
	}
	if countB != 0 {
		t.Fatalf("expected 0 assignments for shift B, got %d", countB)
	}
}

// ============================================================================
// ArbZG rest-period lookups
// ============================================================================

func TestLatestShiftEndBeforeForEmployee_FindsMostRecentPriorShiftStrictlyBefore(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	base := time.Now().UTC().Truncate(time.Second).Add(96 * time.Hour)
	employeeID := uuid.New()

	older := newTestShift(t, repo, ctx, pool, tenantID, "Older", ShiftStatusPublished, base.Add(-12*time.Hour), base.Add(-10*time.Hour))
	newTestAssignment(t, repo, ctx, pool, tenantID, older.ID, employeeID, base.Add(-12*time.Hour))
	mostRecent := newTestShift(t, repo, ctx, pool, tenantID, "MostRecent", ShiftStatusPublished, base.Add(-4*time.Hour), base.Add(-2*time.Hour))
	newTestAssignment(t, repo, ctx, pool, tenantID, mostRecent.ID, employeeID, base.Add(-4*time.Hour))
	// Ends exactly at base: must NOT count as "before" (strict <), the doc
	// comment on LatestShiftEndBeforeForEmployee is explicit about this.
	exactBoundary := newTestShift(t, repo, ctx, pool, tenantID, "ExactBoundary", ShiftStatusPublished, base.Add(-1*time.Hour), base)
	newTestAssignment(t, repo, ctx, pool, tenantID, exactBoundary.ID, employeeID, base.Add(-1*time.Hour))

	got, err := repo.LatestShiftEndBeforeForEmployee(ctx, tenantID, employeeID, base)
	if err != nil {
		t.Fatalf("LatestShiftEndBeforeForEmployee: %v", err)
	}
	if got == nil || !got.Equal(mostRecent.EndTime) {
		t.Fatalf("expected most recent prior shift end %v, got %v", mostRecent.EndTime, got)
	}

	fresh, err := repo.LatestShiftEndBeforeForEmployee(ctx, tenantID, uuid.New(), base)
	if err != nil {
		t.Fatalf("LatestShiftEndBeforeForEmployee (no shifts): %v", err)
	}
	if fresh != nil {
		t.Fatalf("expected nil for employee with no prior shifts, got %v", fresh)
	}
}

func TestEarliestShiftStartAfterForEmployee_FindsEarliestFollowingShiftStrictlyAfter(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	base := time.Now().UTC().Truncate(time.Second).Add(120 * time.Hour)
	employeeID := uuid.New()

	soonest := newTestShift(t, repo, ctx, pool, tenantID, "Soonest", ShiftStatusPublished, base.Add(2*time.Hour), base.Add(4*time.Hour))
	newTestAssignment(t, repo, ctx, pool, tenantID, soonest.ID, employeeID, base.Add(2*time.Hour))
	later := newTestShift(t, repo, ctx, pool, tenantID, "Later", ShiftStatusPublished, base.Add(10*time.Hour), base.Add(12*time.Hour))
	newTestAssignment(t, repo, ctx, pool, tenantID, later.ID, employeeID, base.Add(10*time.Hour))
	// Starts exactly at base: must NOT count as "after" (strict >).
	exactBoundary := newTestShift(t, repo, ctx, pool, tenantID, "ExactBoundary", ShiftStatusPublished, base, base.Add(1*time.Hour))
	newTestAssignment(t, repo, ctx, pool, tenantID, exactBoundary.ID, employeeID, base)

	got, err := repo.EarliestShiftStartAfterForEmployee(ctx, tenantID, employeeID, base)
	if err != nil {
		t.Fatalf("EarliestShiftStartAfterForEmployee: %v", err)
	}
	if got == nil || !got.Equal(soonest.StartTime) {
		t.Fatalf("expected earliest following shift start %v, got %v", soonest.StartTime, got)
	}

	fresh, err := repo.EarliestShiftStartAfterForEmployee(ctx, tenantID, uuid.New(), base)
	if err != nil {
		t.Fatalf("EarliestShiftStartAfterForEmployee (no shifts): %v", err)
	}
	if fresh != nil {
		t.Fatalf("expected nil for employee with no following shifts, got %v", fresh)
	}
}

func TestShiftExistsForTemplate_MatchesExactFields(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	base := time.Now().UTC().Truncate(time.Second).Add(144 * time.Hour)

	newTestShift(t, repo, ctx, pool, tenantID, "ExistsCheck", ShiftStatusDraft, base, base.Add(8*time.Hour))

	exists, err := repo.ShiftExistsForTemplate(ctx, tenantID, base, base.Add(8*time.Hour), "ExistsCheck")
	if err != nil {
		t.Fatalf("ShiftExistsForTemplate (match): %v", err)
	}
	if !exists {
		t.Fatal("expected shift to exist for matching tenant/start/end/title")
	}

	notExists, err := repo.ShiftExistsForTemplate(ctx, tenantID, base, base.Add(8*time.Hour), "DifferentTitle")
	if err != nil {
		t.Fatalf("ShiftExistsForTemplate (mismatch): %v", err)
	}
	if notExists {
		t.Fatal("expected no match for a different title")
	}
}

// ============================================================================
// Templates
// ============================================================================

func TestUpdateTemplate_UpdatesFieldsAndRejectsUnknownID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)

	tpl := newTestTemplate(t, repo, ctx, pool, tenantID, "Wochentemplate", 1, 8, 0, 480)

	tpl.Name = "Wochentemplate (neu)"
	tpl.DurationMinutes = 240
	tpl.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateTemplate(ctx, tpl); err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}

	reloaded, err := repo.GetTemplate(ctx, tenantID, tpl.ID)
	if err != nil {
		t.Fatalf("GetTemplate after update: %v", err)
	}
	if reloaded.Name != "Wochentemplate (neu)" || reloaded.DurationMinutes != 240 {
		t.Fatalf("update did not persist: got %+v", reloaded)
	}

	unknown := &ShiftTemplate{ID: uuid.New(), TenantID: tenantID, Name: "x", DayOfWeek: 1, DurationMinutes: 60, UpdatedAt: time.Now().UTC()}
	if err := repo.UpdateTemplate(ctx, unknown); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("expected ErrTemplateNotFound for unknown template, got %v", err)
	}
}

func TestDeleteTemplate_RemovesRowAndRejectsDoubleDelete(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)

	tpl := newTestTemplate(t, repo, ctx, pool, tenantID, "Loeschbares Template", 2, 9, 0, 240)

	if err := repo.DeleteTemplate(ctx, tenantID, tpl.ID); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}
	if _, err := repo.GetTemplate(ctx, tenantID, tpl.ID); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("expected ErrTemplateNotFound after delete, got %v", err)
	}
	if err := repo.DeleteTemplate(ctx, tenantID, tpl.ID); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("expected ErrTemplateNotFound on double delete, got %v", err)
	}
}

func TestGetTemplate_ReturnsRowOrNotFound(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)

	tpl := newTestTemplate(t, repo, ctx, pool, tenantID, "Sichtbares Template", 3, 10, 30, 120)

	got, err := repo.GetTemplate(ctx, tenantID, tpl.ID)
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if got.ID != tpl.ID || got.Name != tpl.Name {
		t.Fatalf("GetTemplate returned wrong row: %+v", got)
	}

	if _, err := repo.GetTemplate(ctx, tenantID, uuid.New()); !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("expected ErrTemplateNotFound for unknown id, got %v", err)
	}
}

func TestListTemplates_OrdersByDayHourMinuteAndPaginates(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)

	late := newTestTemplate(t, repo, ctx, pool, tenantID, "Late", 2, 14, 0, 240)
	early := newTestTemplate(t, repo, ctx, pool, tenantID, "Early", 1, 8, 0, 240)
	sameDayLater := newTestTemplate(t, repo, ctx, pool, tenantID, "SameDayLater", 1, 8, 30, 240)

	templates, total, err := repo.ListTemplates(ctx, tenantID, 0, 50)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if total != 3 || len(templates) != 3 {
		t.Fatalf("expected 3 templates, got %d (total %d)", len(templates), total)
	}
	if templates[0].ID != early.ID || templates[1].ID != sameDayLater.ID || templates[2].ID != late.ID {
		t.Fatalf("expected [early, sameDayLater, late] ordered by day/hour/minute, got %+v", templates)
	}

	page, total, err := repo.ListTemplates(ctx, tenantID, 1, 1)
	if err != nil {
		t.Fatalf("ListTemplates (pagination): %v", err)
	}
	if total != 3 || len(page) != 1 || page[0].ID != sameDayLater.ID {
		t.Fatalf("pagination: expected [sameDayLater] with total 3, got %+v total=%d", page, total)
	}
}

// ============================================================================
// Stats
// ============================================================================

func TestGetStats_AggregatesShiftsAndAssignmentsWithOptionalRange(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	base := time.Now().UTC().Truncate(time.Second).Add(168 * time.Hour)

	inRangePublishedA := newTestShift(t, repo, ctx, pool, tenantID, "InRangePublishedA", ShiftStatusPublished, base.Add(1*time.Hour), base.Add(2*time.Hour))
	inRangePublishedB := newTestShift(t, repo, ctx, pool, tenantID, "InRangePublishedB", ShiftStatusPublished, base.Add(2*time.Hour), base.Add(3*time.Hour))
	inRangeDraft := newTestShift(t, repo, ctx, pool, tenantID, "InRangeDraft", ShiftStatusDraft, base.Add(1*time.Hour), base.Add(2*time.Hour))
	outOfRangePublished := newTestShift(t, repo, ctx, pool, tenantID, "OutOfRangePublished", ShiftStatusPublished, base.Add(10*time.Hour), base.Add(11*time.Hour))

	employee1, employee2, employee3 := uuid.New(), uuid.New(), uuid.New()
	newTestAssignment(t, repo, ctx, pool, tenantID, inRangePublishedA.ID, employee1, base.Add(1*time.Hour))
	newTestAssignment(t, repo, ctx, pool, tenantID, inRangePublishedB.ID, employee2, base.Add(2*time.Hour))
	newTestAssignment(t, repo, ctx, pool, tenantID, inRangeDraft.ID, employee1, base.Add(1*time.Hour))
	newTestAssignment(t, repo, ctx, pool, tenantID, outOfRangePublished.ID, employee3, base.Add(10*time.Hour))

	from, to := base, base.Add(3*time.Hour)
	scoped, err := repo.GetStats(ctx, tenantID, &from, &to)
	if err != nil {
		t.Fatalf("GetStats (range): %v", err)
	}
	if scoped.TotalShifts != 3 || scoped.PublishedShifts != 2 || scoped.DraftShifts != 1 {
		t.Fatalf("range stats: expected total=3 published=2 draft=1, got %+v", scoped)
	}
	if scoped.TotalAssignments != 3 || scoped.UniqueEmployees != 2 {
		t.Fatalf("range stats: expected assignments=3 unique=2 (employee1+2, employee3 excluded by range), got %+v", scoped)
	}

	all, err := repo.GetStats(ctx, tenantID, nil, nil)
	if err != nil {
		t.Fatalf("GetStats (unscoped): %v", err)
	}
	if all.TotalShifts != 4 || all.PublishedShifts != 3 || all.DraftShifts != 1 {
		t.Fatalf("unscoped stats: expected total=4 published=3 draft=1, got %+v", all)
	}
	if all.TotalAssignments != 4 || all.UniqueEmployees != 3 {
		t.Fatalf("unscoped stats: expected assignments=4 unique=3, got %+v", all)
	}
}

// ============================================================================
// SwapRequests
// ============================================================================

func TestGetSwapRequest_ReturnsRowOrNotFound(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	now := time.Now().UTC()

	shift := newTestShift(t, repo, ctx, pool, tenantID, "Swap Shift", ShiftStatusPublished, now, now.Add(8*time.Hour))
	requester := uuid.New()
	assignment := newTestAssignment(t, repo, ctx, pool, tenantID, shift.ID, requester, now)
	req := newTestSwapRequest(t, repo, ctx, pool, tenantID, assignment.ID, shift.ID, requester, uuid.New(), "schichten-repo-test-get-swap")

	got, err := repo.GetSwapRequest(ctx, tenantID, req.ID)
	if err != nil {
		t.Fatalf("GetSwapRequest: %v", err)
	}
	if got.ID != req.ID || got.Status != SwapRequestStatusPending {
		t.Fatalf("GetSwapRequest returned wrong row: %+v", got)
	}

	if _, err := repo.GetSwapRequest(ctx, tenantID, uuid.New()); !errors.Is(err, ErrSwapRequestNotFound) {
		t.Fatalf("expected ErrSwapRequestNotFound for unknown id, got %v", err)
	}
}

func TestUpdateSwapRequestStatus_UpdatesAndRejectsUnknownID(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	now := time.Now().UTC()

	shift := newTestShift(t, repo, ctx, pool, tenantID, "Swap Shift", ShiftStatusPublished, now, now.Add(8*time.Hour))
	requester := uuid.New()
	assignment := newTestAssignment(t, repo, ctx, pool, tenantID, shift.ID, requester, now)
	req := newTestSwapRequest(t, repo, ctx, pool, tenantID, assignment.ID, shift.ID, requester, uuid.New(), "schichten-repo-test-status")

	if err := repo.UpdateSwapRequestStatus(ctx, tenantID, req.ID, SwapRequestStatusRejected); err != nil {
		t.Fatalf("UpdateSwapRequestStatus: %v", err)
	}
	reloaded, err := repo.GetSwapRequest(ctx, tenantID, req.ID)
	if err != nil {
		t.Fatalf("GetSwapRequest after status update: %v", err)
	}
	if reloaded.Status != SwapRequestStatusRejected {
		t.Fatalf("expected status rejected, got %q", reloaded.Status)
	}

	if err := repo.UpdateSwapRequestStatus(ctx, tenantID, uuid.New(), SwapRequestStatusApproved); !errors.Is(err, ErrSwapRequestNotFound) {
		t.Fatalf("expected ErrSwapRequestNotFound for unknown id, got %v", err)
	}
}

// TestSwapAssignmentsForRequest_SilentlyNoOpsWhenTargetNotYetOnShift
// documents a verified production bug (case 1 of 2, see the sibling test
// below for case 2): SwapAssignmentsForRequest's second UPDATE is scoped by
// (shift_id, employee_id = swap_with), not by an assignment id disjoint from
// the first UPDATE's target. When the swap partner has NO prior assignment
// on the shift, step one moves the requester's row onto the partner's
// employee_id -- and step two, running in the same transaction, immediately
// re-matches that SAME now-updated row (it now has employee_id = swap_with)
// and flips it straight back to the requester. Net effect: zero change to
// the database, no error returned, yet the caller (ApproveSwapRequest) marks
// the swap request "approved". Verified directly against the local DB
// (two sequential UPDATEs on a single seeded row) before writing this test.
// Not fixed here (coverage units change no behavior) -- filed as
// fix-schichten-swap-assignments-unique-violation for the next run, together
// with the unique-violation case below (same root cause: step two isn't
// scoped to a specific row).
func TestSwapAssignmentsForRequest_SilentlyNoOpsWhenTargetNotYetOnShift(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	now := time.Now().UTC()

	shift := newTestShift(t, repo, ctx, pool, tenantID, "Swap Shift", ShiftStatusPublished, now, now.Add(8*time.Hour))
	requester, partner := uuid.New(), uuid.New()
	assignment := newTestAssignment(t, repo, ctx, pool, tenantID, shift.ID, requester, now)
	req := newTestSwapRequest(t, repo, ctx, pool, tenantID, assignment.ID, shift.ID, requester, partner, "schichten-repo-test-swap-move")

	if err := repo.SwapAssignmentsForRequest(ctx, req); err != nil {
		t.Fatalf("SwapAssignmentsForRequest: %v", err)
	}

	// If this now finds the row under the partner, the bug is fixed --
	// rewrite this test to assert the actual move instead of the no-op.
	if _, err := repo.GetAssignment(ctx, tenantID, shift.ID, partner); !errors.Is(err, ErrAssignmentNotFound) {
		t.Fatalf("expected the swap to still be a no-op (bug fixed?): partner lookup returned %v", err)
	}
	stillRequester, err := repo.GetAssignment(ctx, tenantID, shift.ID, requester)
	if err != nil || stillRequester.ID != assignment.ID {
		t.Fatalf("expected the assignment to have silently reverted to the requester, got %+v / %v", stillRequester, err)
	}
}

// TestSwapAssignmentsForRequest_FailsWhenBothEmployeesAlreadyAssignedToShift
// documents case 2 of 2, the other failure mode of the same root cause: the
// two-step UPDATE in SwapAssignmentsForRequest is not order-safe against the
// uq_shift_assignments_tenant UNIQUE(tenant_id, shift_id, employee_id)
// constraint (migration 000102). When both the requester and the swap
// partner already hold an assignment on the SAME shift -- the primary,
// intended meaning of a "shift swap" between two colleagues -- step one
// rewrites the requester's row to the partner's employee_id while the
// partner's own row still carries that same employee_id, so the UPDATE
// hits a duplicate-key violation and the whole transaction (and thus
// ApproveSwapRequest) fails every time. Verified directly against the local
// DB before writing this test. Not fixed here (coverage units change no
// behavior) -- filed as fix-schichten-swap-assignments-unique-violation for
// the next run.
func TestSwapAssignmentsForRequest_FailsWhenBothEmployeesAlreadyAssignedToShift(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)
	now := time.Now().UTC()

	shift := newTestShift(t, repo, ctx, pool, tenantID, "Swap Shift", ShiftStatusPublished, now, now.Add(8*time.Hour))
	requester, partner := uuid.New(), uuid.New()
	requesterAssignment := newTestAssignment(t, repo, ctx, pool, tenantID, shift.ID, requester, now)
	newTestAssignment(t, repo, ctx, pool, tenantID, shift.ID, partner, now)
	req := newTestSwapRequest(t, repo, ctx, pool, tenantID, requesterAssignment.ID, shift.ID, requester, partner, "schichten-repo-test-swap-conflict")

	err := repo.SwapAssignmentsForRequest(ctx, req)
	if err == nil {
		t.Fatal("expected SwapAssignmentsForRequest to fail with a unique constraint violation when both employees already hold an assignment on the shift -- if this now succeeds, the bug is fixed and this test should be rewritten to assert the swap, not the failure")
	}
	if !strings.Contains(err.Error(), "duplicate key") && !strings.Contains(err.Error(), "unique") {
		t.Fatalf("expected a unique-constraint error, got: %v", err)
	}

	// Both original assignments must be untouched -- the transaction rolled back.
	stillRequester, err := repo.GetAssignment(ctx, tenantID, shift.ID, requester)
	if err != nil || stillRequester.ID != requesterAssignment.ID {
		t.Fatalf("requester's assignment should be unchanged after rollback: %+v / %v", stillRequester, err)
	}
}

// ============================================================================
// IsMinorEmployee
// ============================================================================

func TestIsMinorEmployee_MatchesProfileIDOrUserIDAndDefaultsFalseWithoutProfile(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupSchichtenRepo(t)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         "minor-" + uuid.New().String() + "@test.local",
		"password_hash": "x",
		"first_name":    "Minor",
		"last_name":     "Employee",
		"is_active":     true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	profileID := testutil.SeedRow(t, pool, "hr_employee_profiles", map[string]any{
		"tenant_id":  tenantID,
		"user_id":    userID,
		"start_date": time.Now().UTC().Format("2006-01-02"),
		"is_minor":   true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_profiles", profileID) })

	byUserID, err := repo.IsMinorEmployee(ctx, tenantID, userID)
	if err != nil {
		t.Fatalf("IsMinorEmployee (by user_id): %v", err)
	}
	if !byUserID {
		t.Fatal("expected is_minor=true when matched via user_id")
	}

	byProfileID, err := repo.IsMinorEmployee(ctx, tenantID, profileID)
	if err != nil {
		t.Fatalf("IsMinorEmployee (by profile id): %v", err)
	}
	if !byProfileID {
		t.Fatal("expected is_minor=true when matched via the profile's own id")
	}

	noProfile, err := repo.IsMinorEmployee(ctx, tenantID, uuid.New())
	if err != nil {
		t.Fatalf("IsMinorEmployee (no profile): %v", err)
	}
	if noProfile {
		t.Fatal("expected false for an employee id with no HR profile at all")
	}
}
