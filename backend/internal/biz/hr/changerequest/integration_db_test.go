package changerequest

// These tests run against a real database because what they prove cannot be
// proven against a fake: that the partial unique index really refuses a second
// open proposal for the same field, that an approval and the profile write land
// together or not at all, that the team scope resolves against a real reporting
// line, and that RLS keeps one tenant's proposals out of another's reads under
// kmuhub_app (NOSUPERUSER NOBYPASSRLS).

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// Every test seeds its OWN tenant. "The tenant's pending requests" and "who
// reports to whom" are properties of the tenant population, so a neighbour's
// fixture under -parallel would decide another test's result.
func newFixture(t *testing.T) (*pgxpool.Pool, uuid.UUID, context.Context) {
	t.Helper()
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Change Request Tenant")

	return pool, tenantID, testutil.WithTenantCtx(context.Background(), tenantID)
}

func seedUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, email string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "email": email,
		"password_hash": "hash", "first_name": "Test", "last_name": "User",
		"is_active": true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
	return id
}

func seedProfile(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID, managerID *uuid.UUID) uuid.UUID {
	t.Helper()
	cols := map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "user_id": userID,
		"department": "Engineering", "position_title": "Engineer",
		"contract_type": "full_time", "work_days_per_week": 5,
		"annual_leave_days": 25, "start_date": "2020-01-01",
		"address_city": "Muenchen", "address_street": "Alte Strasse 1",
	}
	if managerID != nil {
		cols["manager_user_id"] = *managerID
	}
	id := testutil.SeedRow(t, pool, "hr_employee_profiles", cols)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_profiles", id) })
	return id
}

func newSvc(pool *pgxpool.Pool) *Service {
	return NewService(NewPostgresRepository(pool))
}

func submit(t *testing.T, ctx context.Context, svc *Service, tenantID, userID uuid.UUID, field, newValue string) *models.HRProfileChangeRequest {
	t.Helper()
	req, err := svc.Create(ctx, CreateInput{
		TenantID: tenantID, ActorID: userID,
		Drawer: "personal", Field: field, FieldLabel: field,
		OldValue: "old", NewValue: newValue,
	})
	if err != nil {
		t.Fatalf("Create(%s): %v", field, err)
	}
	return req
}

func TestCreate_SecondPendingRequestForSameFieldIsRefused(t *testing.T) {
	pool, tenantID, ctx := newFixture(t)
	svc := newSvc(pool)

	employeeID := seedUser(t, pool, tenantID, "dup-"+uuid.NewString()+"@example.test")
	seedProfile(t, pool, tenantID, employeeID, nil)

	first := submit(t, ctx, svc, tenantID, employeeID, "addressCity", "Berlin")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_profile_change_requests", first.ID) })

	_, err := svc.Create(ctx, CreateInput{
		TenantID: tenantID, ActorID: employeeID,
		Drawer: "personal", Field: "addressCity", FieldLabel: "Wohnort",
		NewValue: "Hamburg",
	})
	if !errors.Is(err, ErrPendingRequestExists) {
		t.Fatalf("second pending request for the same field: got %v, want ErrPendingRequestExists", err)
	}

	// A different field is a different proposal and must still be accepted.
	second := submit(t, ctx, svc, tenantID, employeeID, "addressStreet", "Neue Strasse 2")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_profile_change_requests", second.ID) })
}

func TestCreate_UnknownFieldIsRefusedBeforeItIsStored(t *testing.T) {
	pool, tenantID, ctx := newFixture(t)
	svc := newSvc(pool)

	employeeID := seedUser(t, pool, tenantID, "unknown-"+uuid.NewString()+"@example.test")
	seedProfile(t, pool, tenantID, employeeID, nil)

	// "mobile" is offered by the desktop UI but has no column behind it. Storing
	// the proposal would produce a request approval could never carry out.
	_, err := svc.Create(ctx, CreateInput{
		TenantID: tenantID, ActorID: employeeID,
		Drawer: "personal", Field: "mobile", NewValue: "+49 170 1234567",
	})
	if !errors.Is(err, ErrFieldNotProposable) {
		t.Fatalf("proposal for a field without a column: got %v, want ErrFieldNotProposable", err)
	}

	requests, err := svc.List(ctx, ListInput{TenantID: tenantID, ActorID: employeeID, OwnOnly: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(requests) != 0 {
		t.Fatalf("refused proposal was stored anyway: %d row(s)", len(requests))
	}
}

func TestApprove_WritesTheValueToTheProfile(t *testing.T) {
	pool, tenantID, ctx := newFixture(t)
	svc := newSvc(pool)

	employeeID := seedUser(t, pool, tenantID, "approve-"+uuid.NewString()+"@example.test")
	seedProfile(t, pool, tenantID, employeeID, nil)
	approverID := seedUser(t, pool, tenantID, "hr-"+uuid.NewString()+"@example.test")

	req := submit(t, ctx, svc, tenantID, employeeID, "addressCity", "Hamburg")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_profile_change_requests", req.ID) })

	updated, err := svc.Approve(ctx, DecideInput{
		TenantID: tenantID, ActorID: approverID, ActorScope: auth.ScopeAll, RequestID: req.ID,
	})
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if updated.Status != models.HRChangeRequestApproved {
		t.Fatalf("status after approval = %q, want approved", updated.Status)
	}
	if updated.DecidedAt == nil || updated.DecidedBy == nil || *updated.DecidedBy != approverID {
		t.Fatalf("decision not recorded: decided_at=%v decided_by=%v", updated.DecidedAt, updated.DecidedBy)
	}

	var city string
	if err := pool.QueryRow(ctx,
		`SELECT address_city FROM hr_employee_profiles WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, employeeID,
	).Scan(&city); err != nil {
		t.Fatalf("read back profile: %v", err)
	}
	if city != "Hamburg" {
		t.Fatalf("profile address_city = %q, want Hamburg — the approval did not reach the record", city)
	}

	// Approving twice must not apply the value a second time under a new
	// decision; the request is no longer pending.
	if _, err := svc.Approve(ctx, DecideInput{
		TenantID: tenantID, ActorID: approverID, ActorScope: auth.ScopeAll, RequestID: req.ID,
	}); !errors.Is(err, ErrNotPending) {
		t.Fatalf("second approval: got %v, want ErrNotPending", err)
	}
}

func TestApprove_TeamScopeReachesOnlyDirectReports(t *testing.T) {
	pool, tenantID, ctx := newFixture(t)
	svc := newSvc(pool)

	managerID := seedUser(t, pool, tenantID, "mgr-"+uuid.NewString()+"@example.test")
	reportID := seedUser(t, pool, tenantID, "report-"+uuid.NewString()+"@example.test")
	strangerID := seedUser(t, pool, tenantID, "stranger-"+uuid.NewString()+"@example.test")
	seedProfile(t, pool, tenantID, managerID, nil)
	seedProfile(t, pool, tenantID, reportID, &managerID)
	seedProfile(t, pool, tenantID, strangerID, nil)

	fromReport := submit(t, ctx, svc, tenantID, reportID, "addressCity", "Koeln")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_profile_change_requests", fromReport.ID) })
	fromStranger := submit(t, ctx, svc, tenantID, strangerID, "addressCity", "Bonn")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_profile_change_requests", fromStranger.ID) })

	// The whole point of the flow: a manager may not decide about somebody who
	// does not report to them, even though they hold the permission.
	_, err := svc.Approve(ctx, DecideInput{
		TenantID: tenantID, ActorID: managerID, ActorScope: auth.ScopeTeam, RequestID: fromStranger.ID,
	})
	if !errors.Is(err, ErrOutOfScope) {
		t.Fatalf("approving outside the reporting line: got %v, want ErrOutOfScope", err)
	}

	if _, err := svc.Approve(ctx, DecideInput{
		TenantID: tenantID, ActorID: managerID, ActorScope: auth.ScopeTeam, RequestID: fromReport.ID,
	}); err != nil {
		t.Fatalf("approving own report: %v", err)
	}

	// The list has to agree with what approve accepts, or the inbox offers rows
	// that answer 403 when clicked.
	visible, err := svc.List(ctx, ListInput{TenantID: tenantID, ActorID: managerID, ActorScope: auth.ScopeTeam})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, r := range visible {
		if r.UserID == strangerID {
			t.Fatal("team-scoped list leaks a proposal the manager may not decide")
		}
	}
}

func TestReject_NeedsAReasonAndLeavesTheProfileAlone(t *testing.T) {
	pool, tenantID, ctx := newFixture(t)
	svc := newSvc(pool)

	employeeID := seedUser(t, pool, tenantID, "reject-"+uuid.NewString()+"@example.test")
	seedProfile(t, pool, tenantID, employeeID, nil)
	approverID := seedUser(t, pool, tenantID, "hr2-"+uuid.NewString()+"@example.test")

	req := submit(t, ctx, svc, tenantID, employeeID, "addressCity", "Leipzig")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_profile_change_requests", req.ID) })

	if _, err := svc.Reject(ctx, DecideInput{
		TenantID: tenantID, ActorID: approverID, ActorScope: auth.ScopeAll, RequestID: req.ID,
	}); !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("rejection without a reason: got %v, want ErrReasonRequired", err)
	}

	updated, err := svc.Reject(ctx, DecideInput{
		TenantID: tenantID, ActorID: approverID, ActorScope: auth.ScopeAll,
		RequestID: req.ID, Reason: "Bitte Meldebescheinigung nachreichen",
	})
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if updated.Status != models.HRChangeRequestRejected || updated.Reason == "" {
		t.Fatalf("rejection not recorded: status=%q reason=%q", updated.Status, updated.Reason)
	}

	var city string
	if err := pool.QueryRow(ctx,
		`SELECT address_city FROM hr_employee_profiles WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, employeeID,
	).Scan(&city); err != nil {
		t.Fatalf("read back profile: %v", err)
	}
	if city != "Muenchen" {
		t.Fatalf("profile address_city = %q after a rejection, want the seeded Muenchen", city)
	}
}

func TestCancel_OnlyTheProposerAndOnlyWhilePending(t *testing.T) {
	pool, tenantID, ctx := newFixture(t)
	svc := newSvc(pool)

	employeeID := seedUser(t, pool, tenantID, "cancel-"+uuid.NewString()+"@example.test")
	seedProfile(t, pool, tenantID, employeeID, nil)
	otherID := seedUser(t, pool, tenantID, "other-"+uuid.NewString()+"@example.test")

	req := submit(t, ctx, svc, tenantID, employeeID, "addressCity", "Dresden")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_profile_change_requests", req.ID) })

	if _, err := svc.Cancel(ctx, tenantID, otherID, req.ID); !errors.Is(err, ErrNotProposer) {
		t.Fatalf("cancelling somebody else's proposal: got %v, want ErrNotProposer", err)
	}

	cancelled, err := svc.Cancel(ctx, tenantID, employeeID, req.ID)
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if cancelled.Status != models.HRChangeRequestCancelled {
		t.Fatalf("status after cancel = %q, want cancelled", cancelled.Status)
	}
	// A withdrawal is not somebody else's decision, so it records no decider.
	if cancelled.DecidedBy != nil || cancelled.DecidedAt != nil {
		t.Fatalf("cancel recorded a decider: decided_by=%v decided_at=%v", cancelled.DecidedBy, cancelled.DecidedAt)
	}

	if _, err := svc.Cancel(ctx, tenantID, employeeID, req.ID); !errors.Is(err, ErrNotPending) {
		t.Fatalf("cancelling twice: got %v, want ErrNotPending", err)
	}

	// The cancelled row must no longer block a fresh proposal for that field —
	// the unique index is partial for exactly this reason.
	again := submit(t, ctx, svc, tenantID, employeeID, "addressCity", "Erfurt")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_profile_change_requests", again.ID) })
}

func TestChangeRequests_AreInvisibleToAnotherTenant(t *testing.T) {
	pool, tenantID, ctx := newFixture(t)
	svc := newSvc(pool)

	employeeID := seedUser(t, pool, tenantID, "rls-"+uuid.NewString()+"@example.test")
	seedProfile(t, pool, tenantID, employeeID, nil)
	req := submit(t, ctx, svc, tenantID, employeeID, "addressCity", "Kiel")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_profile_change_requests", req.ID) })

	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "Change Request Foreign Tenant")
	foreignCtx := testutil.WithTenantCtx(context.Background(), otherTenant)

	// The read side is the half that gets forgotten and then produces phantom
	// 404s, so it is checked directly rather than through the write path.
	if _, err := svc.List(foreignCtx, ListInput{TenantID: otherTenant, ActorID: employeeID}); err != nil {
		t.Fatalf("List under the foreign tenant: %v", err)
	}
	testutil.AssertRowCount(t, pool, foreignCtx, "hr_profile_change_requests", req.ID, 0)
	testutil.AssertRowCount(t, pool, ctx, "hr_profile_change_requests", req.ID, 1)

	// A foreign tenant asking for the row by id must not reach it either — the
	// service must answer not-found, never the row.
	if _, err := svc.repo.GetByID(foreignCtx, otherTenant, req.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID across tenants: got %v, want ErrNotFound", err)
	}
}
