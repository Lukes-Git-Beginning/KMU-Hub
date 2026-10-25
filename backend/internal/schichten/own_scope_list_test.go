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

// The "own" scope of schichten:assignment:read keeps a member's roster to
// their own assignments. employee_id holds the user id when an API caller
// assigned and the HR profile id when the desktop roster did, so both
// identities of the caller must count — and nobody else's.
func TestListAssignments_OwnScopeMatchesUserAndProfileID(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Schichten Own Assignments Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	now := time.Now().UTC()

	seedUser := func(name string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"id": uuid.New(), "tenant_id": tenant, "email": name + "-" + uuid.NewString()[:8] + "@test.local",
			"password_hash": "x", "first_name": name, "last_name": "Test", "is_active": true,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	seedProfile := func(userID uuid.UUID) uuid.UUID {
		id := testutil.SeedRow(t, pool, "hr_employee_profiles", map[string]any{
			"id": uuid.New(), "tenant_id": tenant, "user_id": userID, "contract_type": "full_time",
			"work_days_per_week": 5, "annual_leave_days": 25, "start_date": now,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_profiles", id) })
		return id
	}
	me, colleague, noProfile := seedUser("Me"), seedUser("Colleague"), seedUser("NoProfile")
	myProfile, colleagueProfile := seedProfile(me), seedProfile(colleague)

	shift := &Shift{
		ID: uuid.New(), TenantID: tenant, Title: "Own Assignments Shift",
		StartTime: now, EndTime: now.Add(8 * time.Hour), Status: ShiftStatusDraft,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateShift(ctx, shift); err != nil {
		t.Fatalf("CreateShift: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "shifts", shift.ID) })

	for _, emp := range []uuid.UUID{me, myProfile, colleague, colleagueProfile, noProfile} {
		a := &ShiftAssignment{ID: uuid.New(), TenantID: tenant, ShiftID: shift.ID, EmployeeID: emp, AssignedAt: now}
		if err := repo.CreateAssignment(ctx, a); err != nil {
			t.Fatalf("CreateAssignment(%s): %v", emp, err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "shift_assignments", a.ID) })
	}

	employees := func(t *testing.T, own *uuid.UUID) map[uuid.UUID]bool {
		t.Helper()
		got, err := svc.ListAssignments(ctx, tenant, shift.ID, own)
		if err != nil {
			t.Fatalf("ListAssignments: %v", err)
		}
		out := map[uuid.UUID]bool{}
		for _, a := range got {
			out[a.EmployeeID] = true
		}
		return out
	}

	t.Run("own scope returns the rows under the user id and the HR profile id", func(t *testing.T) {
		got := employees(t, &me)
		if len(got) != 2 || !got[me] || !got[myProfile] {
			t.Fatalf("own scope: got %v, want exactly %s and %s", got, me, myProfile)
		}
	})
	t.Run("a user without HR profile sees the row under the user id only", func(t *testing.T) {
		got := employees(t, &noProfile)
		if len(got) != 1 || !got[noProfile] {
			t.Fatalf("own scope without profile: got %v, want exactly %s", got, noProfile)
		}
	})
	t.Run("a user without any assignment gets an empty roster", func(t *testing.T) {
		stranger := uuid.New()
		if got := employees(t, &stranger); len(got) != 0 {
			t.Fatalf("own scope of a stranger: got %v, want none", got)
		}
	})
	t.Run("without the filter the whole roster stays visible", func(t *testing.T) {
		if got := employees(t, nil); len(got) != 5 {
			t.Fatalf("unscoped: got %d employees, want 5", len(got))
		}
	})
}
