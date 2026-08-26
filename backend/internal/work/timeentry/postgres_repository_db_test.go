package timeentry

// Real-SQL DB tests for postgres_repository.go, covering the surfaces the
// existing tenant_isolation_phase2_test.go / tenant_write_test.go files
// leave untouched: the timer state machine (GetActiveTimer/StopActiveTimer,
// including the double-start-auto-stops-previous behaviour driven through
// the real service+repo, and stop-without-a-running-timer), pagination in
// ListByTask/ListByUser, and GetTaskTimeSummary's running-timer branch.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedTEUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("db-timeentry-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"first_name":    "DB",
		"last_name":     "Tester",
	})
}

func seedTEProject(t *testing.T, pool *pgxpool.Pool, tenantID, createdBy uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id":   tenantID,
		"name":        "DB TimeEntry Project",
		"project_key": "DT" + uuid.New().String()[:6],
		"created_by":  createdBy,
	})
}

func seedTETask(t *testing.T, pool *pgxpool.Pool, tenantID, projectID, createdBy uuid.UUID, taskNumber int) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantID,
		"project_id":  projectID,
		"title":       "DB TimeEntry Task",
		"task_number": taskNumber,
		"created_by":  createdBy,
	})
}

// TestStartTimer_DoubleStart_AutoStopsPrevious drives the real repo through
// the Service (not a mock) to prove the SQL underneath the "only one running
// timer per user" invariant actually holds: a second StartTimer call must
// leave exactly one active timer and must have closed out the first one with
// a real duration, not two rows both open.
func TestStartTimer_DoubleStart_AutoStopsPrevious(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "DoubleStart Tenant")

	owner := seedTEUser(t, pool, tenantID)
	defer testutil.CleanupRow(t, pool, "users", owner)
	projID := seedTEProject(t, pool, tenantID, owner)
	defer testutil.CleanupRow(t, pool, "projects", projID)
	taskA := seedTETask(t, pool, tenantID, projID, owner, 1)
	defer testutil.CleanupRow(t, pool, "tasks", taskA)
	taskB := seedTETask(t, pool, tenantID, projID, owner, 2)
	defer testutil.CleanupRow(t, pool, "tasks", taskB)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	defer func() {
		_, _ = pool.Exec(testutil.WithSystemCtx(context.Background()),
			`DELETE FROM time_entries WHERE task_id IN ($1, $2)`, taskA, taskB)
	}()

	first, stoppedFirst, err := svc.StartTimer(ctx, taskA, owner, tenantID)
	if err != nil {
		t.Fatalf("StartTimer(first): %v", err)
	}
	if stoppedFirst != nil {
		t.Fatalf("StartTimer(first): expected no previously-stopped timer, got %+v", stoppedFirst)
	}

	time.Sleep(10 * time.Millisecond) // ensure a measurable duration on the auto-stopped entry

	second, stoppedSecond, err := svc.StartTimer(ctx, taskB, owner, tenantID)
	if err != nil {
		t.Fatalf("StartTimer(second): %v", err)
	}
	if stoppedSecond == nil || stoppedSecond.ID != first.ID {
		t.Fatalf("StartTimer(second): expected the first entry to be auto-stopped, got %+v", stoppedSecond)
	}
	if stoppedSecond.EndedAt == nil || stoppedSecond.DurationSeconds == nil {
		t.Fatalf("auto-stopped entry must carry ended_at and duration_seconds: %+v", stoppedSecond)
	}
	if *stoppedSecond.DurationSeconds < 0 {
		t.Fatalf("auto-stopped entry has a negative duration: %d", *stoppedSecond.DurationSeconds)
	}

	active, err := repo.GetActiveTimer(ctx, owner, tenantID)
	if err != nil {
		t.Fatalf("GetActiveTimer: %v", err)
	}
	if active == nil || active.ID != second.ID {
		t.Fatalf("GetActiveTimer: expected the second entry to be the sole active timer, got %+v", active)
	}

	var openCount int
	if scanErr := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT count(*) FROM time_entries WHERE task_id IN ($1, $2) AND ended_at IS NULL`,
		taskA, taskB,
	).Scan(&openCount); scanErr != nil {
		t.Fatalf("count open entries: %v", scanErr)
	}
	if openCount != 1 {
		t.Fatalf("expected exactly 1 open time entry after double-start, got %d", openCount)
	}
}

// TestStopActiveTimer_NoRunningTimer_ReturnsNilNotError covers stop-without-
// start at the real-SQL level: the UPDATE...RETURNING matches zero rows,
// which pgx surfaces as pgx.ErrNoRows on Scan — the repo must translate that
// into (nil, nil), not bubble the raw error up.
func TestStopActiveTimer_NoRunningTimer_ReturnsNilNotError(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "StopNoTimer Tenant")

	owner := seedTEUser(t, pool, tenantID)
	defer testutil.CleanupRow(t, pool, "users", owner)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	stopped, err := repo.StopActiveTimer(ctx, owner, tenantID)
	if err != nil {
		t.Fatalf("StopActiveTimer(no timer): unexpected error %v", err)
	}
	if stopped != nil {
		t.Fatalf("StopActiveTimer(no timer): expected nil, got %+v", stopped)
	}

	// Service layer must turn the nil into ErrNoActiveTimer.
	svc := NewService(repo)
	if _, err := svc.StopTimer(ctx, owner, tenantID); err != ErrNoActiveTimer {
		t.Fatalf("Service.StopTimer(no timer): err=%v, want ErrNoActiveTimer", err)
	}
}

// TestGetActiveTimer_TenantScoped proves a running timer for tenant A is
// invisible under tenant B's context even for the same user id (were one to
// collide), and that a user with no timer at all gets (nil, nil).
func TestGetActiveTimer_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantMine := uuid.New()
	tenantOther := uuid.New()
	testutil.EnsureTenant(t, pool, tenantMine, "ActiveTimer Mine")
	testutil.EnsureTenant(t, pool, tenantOther, "ActiveTimer Other")

	owner := seedTEUser(t, pool, tenantMine)
	defer testutil.CleanupRow(t, pool, "users", owner)
	projID := seedTEProject(t, pool, tenantMine, owner)
	defer testutil.CleanupRow(t, pool, "projects", projID)
	taskID := seedTETask(t, pool, tenantMine, projID, owner, 1)
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	runningID := testutil.SeedRow(t, pool, "time_entries", map[string]any{
		"tenant_id":  tenantMine,
		"task_id":    taskID,
		"user_id":    owner,
		"started_at": time.Now().UTC(),
	})
	defer testutil.CleanupRow(t, pool, "time_entries", runningID)

	repo := NewPostgresRepository(pool)

	ctxMine := testutil.WithTenantCtx(context.Background(), tenantMine)
	mine, err := repo.GetActiveTimer(ctxMine, owner, tenantMine)
	if err != nil || mine == nil || mine.ID != runningID {
		t.Fatalf("GetActiveTimer(mine): %+v, %v, want the running entry", mine, err)
	}
	if mine.TaskTitle != "DB TimeEntry Task" {
		t.Fatalf("GetActiveTimer: task title not joined correctly: %q", mine.TaskTitle)
	}

	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	foreign, err := repo.GetActiveTimer(ctxOther, owner, tenantOther)
	if err != nil {
		t.Fatalf("GetActiveTimer(foreign): %v", err)
	}
	if foreign != nil {
		t.Fatalf("GetActiveTimer(foreign): expected nil for a foreign tenant, got %+v", foreign)
	}
}

// TestListByTask_And_ListByUser_PaginationAndTenantScoping covers pagination
// clamping (page<1, out-of-range pageSize), correct total counts, and
// tenant-scoped visibility for both list methods.
func TestListByTask_And_ListByUser_PaginationAndTenantScoping(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantMine := uuid.New()
	tenantOther := uuid.New()
	testutil.EnsureTenant(t, pool, tenantMine, "Pagination Mine")
	testutil.EnsureTenant(t, pool, tenantOther, "Pagination Other")

	owner := seedTEUser(t, pool, tenantMine)
	defer testutil.CleanupRow(t, pool, "users", owner)
	projID := seedTEProject(t, pool, tenantMine, owner)
	defer testutil.CleanupRow(t, pool, "projects", projID)
	taskID := seedTETask(t, pool, tenantMine, projID, owner, 1)
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	var entryIDs []uuid.UUID
	now := time.Now().UTC()
	for i := range 3 {
		id := testutil.SeedRow(t, pool, "time_entries", map[string]any{
			"tenant_id":        tenantMine,
			"task_id":          taskID,
			"user_id":          owner,
			"started_at":       now.Add(-time.Duration(i+1) * time.Hour),
			"ended_at":         now.Add(-time.Duration(i) * time.Hour),
			"duration_seconds": 3600,
		})
		entryIDs = append(entryIDs, id)
	}
	defer func() {
		for _, id := range entryIDs {
			testutil.CleanupRow(t, pool, "time_entries", id)
		}
	}()

	repo := NewPostgresRepository(pool)
	ctxMine := testutil.WithTenantCtx(context.Background(), tenantMine)

	// page=0 clamps to 1, pageSize=0 clamps to 20 -- all 3 entries on one page.
	byTask, total, err := repo.ListByTask(ctxMine, taskID, tenantMine, 0, 0)
	if err != nil {
		t.Fatalf("ListByTask(clamped): %v", err)
	}
	if total != 3 || len(byTask) != 3 {
		t.Fatalf("ListByTask(clamped): total=%d len=%d, want 3, 3", total, len(byTask))
	}

	// pageSize=2, page=2 -- second page has exactly 1 remaining entry, total still 3.
	page2, total2, err := repo.ListByTask(ctxMine, taskID, tenantMine, 2, 2)
	if err != nil {
		t.Fatalf("ListByTask(page2): %v", err)
	}
	if total2 != 3 || len(page2) != 1 {
		t.Fatalf("ListByTask(page2): total=%d len=%d, want 3, 1", total2, len(page2))
	}

	// pageSize=500 clamps to 20.
	clampedHigh, _, err := repo.ListByTask(ctxMine, taskID, tenantMine, 1, 500)
	if err != nil {
		t.Fatalf("ListByTask(pageSize=500): %v", err)
	}
	if len(clampedHigh) != 3 {
		t.Fatalf("ListByTask(pageSize=500): expected all 3 entries within the clamped page, got %d", len(clampedHigh))
	}

	byUser, totalUser, err := repo.ListByUser(ctxMine, owner, tenantMine, 1, 20)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if totalUser != 3 || len(byUser) != 3 {
		t.Fatalf("ListByUser: total=%d len=%d, want 3, 3", totalUser, len(byUser))
	}

	// Foreign tenant guessing the real task/user id sees nothing.
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	foreignByTask, foreignTotal, err := repo.ListByTask(ctxOther, taskID, tenantOther, 1, 20)
	if err != nil {
		t.Fatalf("ListByTask(foreign): %v", err)
	}
	if foreignTotal != 0 || len(foreignByTask) != 0 {
		t.Fatalf("ListByTask(foreign): expected 0 entries, got total=%d len=%d", foreignTotal, len(foreignByTask))
	}
	foreignByUser, foreignUserTotal, err := repo.ListByUser(ctxOther, owner, tenantOther, 1, 20)
	if err != nil {
		t.Fatalf("ListByUser(foreign): %v", err)
	}
	if foreignUserTotal != 0 || len(foreignByUser) != 0 {
		t.Fatalf("ListByUser(foreign): expected 0 entries, got total=%d len=%d", foreignUserTotal, len(foreignByUser))
	}
}

// TestGetTaskTimeSummary_IncludesRunningTimerAndIsTenantScoped covers the
// summary's CASE branch for a still-running entry (computed live via NOW()
// rather than the stored duration_seconds) alongside a completed entry, and
// that a foreign tenant gets a zeroed summary rather than an error.
func TestGetTaskTimeSummary_IncludesRunningTimerAndIsTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantMine := uuid.New()
	tenantOther := uuid.New()
	testutil.EnsureTenant(t, pool, tenantMine, "Summary Mine")
	testutil.EnsureTenant(t, pool, tenantOther, "Summary Other")

	owner := seedTEUser(t, pool, tenantMine)
	defer testutil.CleanupRow(t, pool, "users", owner)
	projID := seedTEProject(t, pool, tenantMine, owner)
	defer testutil.CleanupRow(t, pool, "projects", projID)
	taskID := seedTETask(t, pool, tenantMine, projID, owner, 1)
	defer testutil.CleanupRow(t, pool, "tasks", taskID)

	now := time.Now().UTC()
	completedID := testutil.SeedRow(t, pool, "time_entries", map[string]any{
		"tenant_id":        tenantMine,
		"task_id":          taskID,
		"user_id":          owner,
		"started_at":       now.Add(-2 * time.Hour),
		"ended_at":         now.Add(-1 * time.Hour),
		"duration_seconds": 3600,
	})
	defer testutil.CleanupRow(t, pool, "time_entries", completedID)

	runningStart := now.Add(-30 * time.Second)
	runningID := testutil.SeedRow(t, pool, "time_entries", map[string]any{
		"tenant_id":  tenantMine,
		"task_id":    taskID,
		"user_id":    owner,
		"started_at": runningStart,
	})
	defer testutil.CleanupRow(t, pool, "time_entries", runningID)

	repo := NewPostgresRepository(pool)
	ctxMine := testutil.WithTenantCtx(context.Background(), tenantMine)

	summary, err := repo.GetTaskTimeSummary(ctxMine, taskID, tenantMine)
	if err != nil {
		t.Fatalf("GetTaskTimeSummary: %v", err)
	}
	if summary.EntryCount != 2 {
		t.Fatalf("GetTaskTimeSummary: entry_count=%d, want 2", summary.EntryCount)
	}
	// Completed entry contributes exactly 3600s; running entry contributes
	// ~30s computed live. Assert a tolerant lower/upper bound rather than an
	// exact value to avoid flaking on scheduling jitter.
	if summary.TotalDurationSeconds < 3620 || summary.TotalDurationSeconds > 3700 {
		t.Fatalf("GetTaskTimeSummary: total_duration_seconds=%d, want ~3630 (3600 completed + ~30 running)", summary.TotalDurationSeconds)
	}

	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	foreignSummary, err := repo.GetTaskTimeSummary(ctxOther, taskID, tenantOther)
	if err != nil {
		t.Fatalf("GetTaskTimeSummary(foreign): %v", err)
	}
	if foreignSummary.EntryCount != 0 || foreignSummary.TotalDurationSeconds != 0 {
		t.Fatalf("GetTaskTimeSummary(foreign): expected a zeroed summary, got %+v", foreignSummary)
	}
}
