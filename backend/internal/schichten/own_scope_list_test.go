package schichten

// The "own" data scope for swap requests reaches the database as a
// requested_by_employee_id OR swap_with_employee_id predicate, matched
// against BOTH employee fields since either party is a legitimate viewer of
// their own swap. Driven through the service (not the repository) because
// that is where page/pageSize become offset/limit — a filter applied after
// that split would still hand back the tenant-wide count. Mirrors
// internal/rapporte/own_scope_list_test.go.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestListSwapRequests_OwnScopeMatchesBothEmployeeFields(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Schichten Own Scope Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	now := time.Now().UTC()

	shift := &Shift{
		ID: uuid.New(), TenantID: tenant, Title: "Own Scope Shift",
		StartTime: now, EndTime: now.Add(8 * time.Hour), Status: ShiftStatusDraft,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateShift(ctx, shift); err != nil {
		t.Fatalf("CreateShift: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "shifts", shift.ID)

	me, colleague1, colleague2 := uuid.New(), uuid.New(), uuid.New()

	newAssignment := func(employeeID uuid.UUID) *ShiftAssignment {
		a := &ShiftAssignment{ID: uuid.New(), TenantID: tenant, ShiftID: shift.ID, EmployeeID: employeeID, AssignedAt: now}
		if err := repo.CreateAssignment(ctx, a); err != nil {
			t.Fatalf("CreateAssignment(%s): %v", employeeID, err)
		}
		return a
	}
	assignMine := newAssignment(me)
	defer testutil.CleanupRow(t, pool, "shift_assignments", assignMine.ID)
	assignForeign := newAssignment(colleague1)
	defer testutil.CleanupRow(t, pool, "shift_assignments", assignForeign.ID)

	newSwap := func(assignmentID uuid.UUID, requestedBy, swapWith uuid.UUID, idemKey string) *SwapRequest {
		req := &SwapRequest{
			ID: uuid.New(), TenantID: tenant, ShiftID: shift.ID, AssignmentID: assignmentID,
			RequestedByEmployeeID: requestedBy, SwapWithEmployeeID: swapWith,
			Status: SwapRequestStatusPending, IdempotencyKey: idemKey,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateSwapRequest(ctx, req); err != nil {
			t.Fatalf("CreateSwapRequest(%s): %v", idemKey, err)
		}
		return req
	}

	// I am the requester.
	requested := newSwap(assignMine.ID, me, colleague1, "own-scope-requested")
	defer testutil.CleanupRow(t, pool, "shift_swap_requests", requested.ID)
	// I am only the swap partner.
	partnered := newSwap(assignForeign.ID, colleague1, me, "own-scope-partnered")
	defer testutil.CleanupRow(t, pool, "shift_swap_requests", partnered.ID)
	// Neither field is me.
	foreign := newSwap(assignForeign.ID, colleague1, colleague2, "own-scope-foreign")
	defer testutil.CleanupRow(t, pool, "shift_swap_requests", foreign.ID)

	reqs, total, err := svc.ListSwapRequests(ctx, ListSwapRequestsInput{
		TenantID: tenant, OwnEmployeeID: &me, Page: 1, PageSize: 50,
	})
	if err != nil {
		t.Fatalf("ListSwapRequests (own scope): %v", err)
	}
	if total != 2 || len(reqs) != 2 {
		t.Fatalf("own scope: got %d rows / total %d, want 2 / 2", len(reqs), total)
	}
	for _, req := range reqs {
		if req.RequestedByEmployeeID != me && req.SwapWithEmployeeID != me {
			t.Fatalf("a swap request involving neither employee field leaked into the own-scoped list: %q", req.ID)
		}
	}

	// Without the filter the same tenant still shows all three — the
	// narrowing must come from the scope, not from a change in default
	// behaviour for everyone else.
	all, allTotal, err := svc.ListSwapRequests(ctx, ListSwapRequestsInput{TenantID: tenant, Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("ListSwapRequests (unscoped): %v", err)
	}
	if allTotal != 3 || len(all) != 3 {
		t.Fatalf("unscoped list: got %d rows / total %d, want 3 / 3", len(all), allTotal)
	}
}
