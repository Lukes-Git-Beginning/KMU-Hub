package task

// Coverage for BACKLOG unit "g-work-start-date": the start_date column, its
// start_date <= due_date invariant, and the due_from/due_to range filter that
// rides the same List query. The filter itself (DueDateFrom/DueDateTo) already
// existed in TaskFilters/postgres_repository.go before this unit -- only the
// gateway query-param wiring was missing (route_work_tasks.go HandleListTasks).
// This test exercises the repository layer directly, the same layer the
// gateway now calls into.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestService_Create_StartDateAfterDueDateRejected covers the service-level
// validation (mock repo, no DB): a create with start_date > due_date must
// never reach the repository.
func TestService_Create_StartDateAfterDueDateRejected(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)

	userID := uuid.New()
	start := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	due := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	_, err := svc.Create(ctx, CreateInput{
		Title:     "Backwards schedule",
		CreatedBy: userID,
		StartDate: &start,
		DueDate:   &due,
	})
	if err != ErrInvalidDateRange {
		t.Fatalf("expected ErrInvalidDateRange, got %v", err)
	}
	if len(repo.tasks) != 0 {
		t.Fatalf("expected no task to be persisted, got %d", len(repo.tasks))
	}
}

// TestService_Update_StartDateAfterDueDateRejected covers the same invariant
// on the update path, including the case where only one of the two dates is
// part of the update (the other comes from the existing row).
func TestService_Update_StartDateAfterDueDateRejected(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	projRepo := newMockProjectRepo()
	svc := NewService(repo, projRepo)

	userID := uuid.New()
	due := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	task1, err := svc.Create(ctx, CreateInput{
		Title:     "Task with due date only",
		CreatedBy: userID,
		DueDate:   &due,
	})
	if err != nil {
		t.Fatalf("setup Create: %v", err)
	}

	// start_date later than the task's existing due_date, due_date untouched.
	start := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	_, err = svc.Update(ctx, uuid.Nil, task1.ID, UpdateInput{StartDate: &start}, userID)
	if err != ErrInvalidDateRange {
		t.Fatalf("expected ErrInvalidDateRange, got %v", err)
	}

	// Valid order is accepted and persisted.
	start = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	updated, err := svc.Update(ctx, uuid.Nil, task1.ID, UpdateInput{StartDate: &start}, userID)
	if err != nil {
		t.Fatalf("expected valid start_date to be accepted, got %v", err)
	}
	if updated.StartDate == nil || !updated.StartDate.Equal(start) {
		t.Fatalf("expected start_date to be persisted, got %v", updated.StartDate)
	}
}

// TestListTasks_DueDateRangeFilter is a DB-backed test: it seeds three tasks
// (two with due_date in different ranges, one fully unscheduled with both
// dates NULL) and confirms PostgresRepository.List's due_date_from/to filter
// selects the right rows -- and that a NULL start_date scans cleanly instead
// of erroring, which a plain "build is green" check would not catch.
func TestListTasks_DueDateRangeFilter(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "StartDate Filter Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userOwn := seedWorkUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)
	projectOwn := seedWorkProject(t, pool, tenantOwn, userOwn)
	defer testutil.CleanupRow(t, pool, "projects", projectOwn)

	early := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	mid := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)
	midStart := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	// Task A: due_date only, no start_date (must scan a NULL start_date fine).
	taskA := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"project_id":  projectOwn,
		"title":       "Task A - early, unscheduled start",
		"task_number": taskNumber(),
		"created_by":  userOwn,
		"due_date":    early,
	})
	defer testutil.CleanupRow(t, pool, "tasks", taskA)

	// Task B: both dates set, in a later range than Task A.
	taskB := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"project_id":  projectOwn,
		"title":       "Task B - mid, scheduled",
		"task_number": taskNumber(),
		"created_by":  userOwn,
		"start_date":  midStart,
		"due_date":    mid,
	})
	defer testutil.CleanupRow(t, pool, "tasks", taskB)

	// Task C: fully unscheduled, must never satisfy a due_date range filter.
	taskC := testutil.SeedRow(t, pool, "tasks", map[string]any{
		"tenant_id":   tenantOwn,
		"project_id":  projectOwn,
		"title":       "Task C - unscheduled",
		"task_number": taskNumber(),
		"created_by":  userOwn,
	})
	defer testutil.CleanupRow(t, pool, "tasks", taskC)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)

	from := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	results, total, err := repo.List(ctxOwn, tenantOwn, TaskFilters{
		ProjectID:   &projectOwn,
		DueDateFrom: &from,
		DueDateTo:   &to,
		Page:        1,
		PageSize:    20,
	})
	if err != nil {
		t.Fatalf("List (early range): %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Fatalf("expected exactly Task A in range %s..%s, got %d results", from, to, total)
	}
	if results[0].ID.String() != taskA.String() {
		t.Fatalf("expected Task A, got %q", results[0].Title)
	}
	if results[0].StartDate != nil {
		t.Fatalf("expected Task A's start_date to scan as nil, got %v", results[0].StartDate)
	}

	from = time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	to = time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC)
	results, total, err = repo.List(ctxOwn, tenantOwn, TaskFilters{
		ProjectID:   &projectOwn,
		DueDateFrom: &from,
		DueDateTo:   &to,
		Page:        1,
		PageSize:    20,
	})
	if err != nil {
		t.Fatalf("List (mid range): %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Fatalf("expected exactly Task B in range %s..%s, got %d results", from, to, total)
	}
	if results[0].ID.String() != taskB.String() {
		t.Fatalf("expected Task B, got %q", results[0].Title)
	}
	if results[0].StartDate == nil || !results[0].StartDate.Equal(midStart) {
		t.Fatalf("expected Task B's start_date %v, got %v", midStart, results[0].StartDate)
	}

	// Sanity: Task C (both dates NULL) never satisfies a due_date range filter
	// -- it must not show up in either of the two result sets above.
	for _, tw := range results {
		if tw.ID == taskC {
			t.Fatalf("Task C (unscheduled) unexpectedly matched a due_date range filter")
		}
	}
}
