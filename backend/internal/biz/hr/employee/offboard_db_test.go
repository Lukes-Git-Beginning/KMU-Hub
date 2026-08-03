package employee

// The offboard cascade is tested against a real database because what it
// promises cannot be proven against a fake: that the personnel record, the
// login and the role assignments move together or not at all, that a second
// offboard finds nothing left to do, and above all that handing a team to a
// successor who sits inside that team does not make anyone their own manager.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// Every test seeds its OWN tenant: "who reports to whom" and "who is the last
// role administrator" are properties of the tenant population, so a shared
// fixture would let a neighbour under -parallel decide the outcome.
func newOffboardFixture(t *testing.T) (*pgxpool.Pool, uuid.UUID, context.Context) {
	t.Helper()
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Offboard Tenant")

	return pool, tenantID, testutil.WithTenantCtx(context.Background(), tenantID)
}

func seedOffboardUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, email string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "email": email,
		"password_hash": "hash", "first_name": "Test", "last_name": "User",
		"is_active": true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
	return id
}

func seedOffboardProfile(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID, managerID *uuid.UUID) uuid.UUID {
	t.Helper()
	// The nullable text columns are seeded as empty strings, not left NULL:
	// scanEmployeeProfile reads them into plain Go strings, so a NULL from a
	// hand-written row breaks every read of that profile. The application
	// always writes "" (see Create), so this matches real rows.
	cols := map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "user_id": userID,
		"department": "Engineering", "position_title": "Engineer",
		"contract_type": "full_time", "work_days_per_week": 5,
		"annual_leave_days": 25, "start_date": "2020-01-01",
		"address_city": "Muenchen", "address_street": "Alte Strasse 1",
		"address_postal_code": "80331", "address_country": "DE",
		"emergency_contact_name": "", "emergency_contact_phone": "",
	}
	if managerID != nil {
		cols["manager_user_id"] = *managerID
	}
	id := testutil.SeedRow(t, pool, "hr_employee_profiles", cols)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_profiles", id) })
	return id
}

// grantRoleAdmin gives the user a fresh custom role carrying roles:manage, so
// the last-admin guard sees a real active role administrator. The role is
// tenant-scoped; user_roles rows go with it via ON DELETE CASCADE.
func grantRoleAdmin(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID) {
	t.Helper()
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	roleID := testutil.SeedRow(t, pool, "roles", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"name": "offboard-test-" + uuid.NewString()[:8], "description": "test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "roles", roleID) })

	var permID uuid.UUID
	if err := pool.QueryRow(ctx,
		`SELECT id FROM permissions WHERE name = 'roles:manage'`).Scan(&permID); err != nil {
		t.Fatalf("look up roles:manage: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`, roleID, permID); err != nil {
		t.Fatalf("grant roles:manage: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)
		 ON CONFLICT DO NOTHING`, userID, roleID, tenantID); err != nil {
		t.Fatalf("assign role: %v", err)
	}
}

func newOffboardSvc(pool *pgxpool.Pool) *Service {
	return NewService(NewPostgresEmployeeRepo(pool), nil, nil)
}

func defaultOffboardInput() OffboardInput {
	lastDay := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	return OffboardInput{
		LastWorkDay: &lastDay,
		ExitDate:    time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		ExitType:    models.ExitTypeResignation,
		Reason:      "moving on",
	}
}

func managerOf(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID) *uuid.UUID {
	t.Helper()
	var mgr *uuid.UUID
	err := pool.QueryRow(testutil.WithTenantCtx(context.Background(), tenantID),
		`SELECT manager_user_id FROM hr_employee_profiles
		  WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID).Scan(&mgr)
	if err != nil {
		t.Fatalf("read manager of %s: %v", userID, err)
	}
	return mgr
}

// TestOffboard_CascadeLocksAccountAndDropsRoles is the whole promise of the
// route in one test: after it, the personnel file says the person left, the
// account cannot log in (auth refuses !IsActive on both login and refresh) and
// carries no roles. The seat comes back with is_active = false, since auth
// counts seats as active users.
func TestOffboard_CascadeLocksAccountAndDropsRoles(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	svc := newOffboardSvc(pool)

	adminID := seedOffboardUser(t, pool, tenantID, "admin-"+uuid.NewString()+"@example.test")
	grantRoleAdmin(t, pool, tenantID, adminID)

	leaverID := seedOffboardUser(t, pool, tenantID, "leaver-"+uuid.NewString()+"@example.test")
	grantRoleAdmin(t, pool, tenantID, leaverID)
	profileID := seedOffboardProfile(t, pool, tenantID, leaverID, nil)

	got, err := svc.OffboardEmployee(ctx, tenantID, profileID, adminID, defaultOffboardInput())
	if err != nil {
		t.Fatalf("OffboardEmployee: %v", err)
	}
	if got.Status != models.EmployeeStatusInactive {
		t.Errorf("status: got %q, want inactive", got.Status)
	}
	if got.ExitDate == nil || got.ExitType != models.ExitTypeResignation {
		t.Errorf("exit data not recorded: date=%v type=%q", got.ExitDate, got.ExitType)
	}

	var isActive bool
	if err := pool.QueryRow(ctx, `SELECT is_active FROM users WHERE id = $1`, leaverID).Scan(&isActive); err != nil {
		t.Fatalf("read users.is_active: %v", err)
	}
	if isActive {
		t.Error("the account is still active: the login was never locked")
	}

	var roles int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_roles WHERE user_id = $1`, leaverID).Scan(&roles); err != nil {
		t.Fatalf("count user_roles: %v", err)
	}
	if roles != 0 {
		t.Errorf("role assignments left: got %d, want 0", roles)
	}
}

// TestOffboard_SecondAttemptIsRefused: the conditional UPDATE is the
// concurrency guard, so a repeat cannot overwrite the recorded exit data.
func TestOffboard_SecondAttemptIsRefused(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	svc := newOffboardSvc(pool)

	adminID := seedOffboardUser(t, pool, tenantID, "admin2-"+uuid.NewString()+"@example.test")
	grantRoleAdmin(t, pool, tenantID, adminID)
	leaverID := seedOffboardUser(t, pool, tenantID, "leaver2-"+uuid.NewString()+"@example.test")
	profileID := seedOffboardProfile(t, pool, tenantID, leaverID, nil)

	if _, err := svc.OffboardEmployee(ctx, tenantID, profileID, adminID, defaultOffboardInput()); err != nil {
		t.Fatalf("first offboard: %v", err)
	}
	_, err := svc.OffboardEmployee(ctx, tenantID, profileID, adminID, defaultOffboardInput())
	if !errors.Is(err, ErrAlreadyOffboarded) {
		t.Fatalf("second offboard: got %v, want ErrAlreadyOffboarded", err)
	}
}

// TestOffboard_ReportsWithoutSuccessorIsRefused — the guard the unit exists
// for: approvals must not end up hanging off a locked account.
func TestOffboard_ReportsWithoutSuccessorIsRefused(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	svc := newOffboardSvc(pool)

	adminID := seedOffboardUser(t, pool, tenantID, "admin3-"+uuid.NewString()+"@example.test")
	grantRoleAdmin(t, pool, tenantID, adminID)
	leadID := seedOffboardUser(t, pool, tenantID, "lead3-"+uuid.NewString()+"@example.test")
	leadProfile := seedOffboardProfile(t, pool, tenantID, leadID, nil)
	reportID := seedOffboardUser(t, pool, tenantID, "report3-"+uuid.NewString()+"@example.test")
	seedOffboardProfile(t, pool, tenantID, reportID, &leadID)

	_, err := svc.OffboardEmployee(ctx, tenantID, leadProfile, adminID, defaultOffboardInput())
	if !errors.Is(err, ErrSuccessorRequired) {
		t.Fatalf("got %v, want ErrSuccessorRequired", err)
	}

	// And the refusal really rolled nothing forward.
	var status string
	if err := pool.QueryRow(ctx,
		`SELECT status FROM hr_employee_profiles WHERE id = $1`, leadProfile).Scan(&status); err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != string(models.EmployeeStatusActive) {
		t.Errorf("status after refusal: got %q, want active", status)
	}
}

// TestOffboard_SuccessorInsideTheTeamDoesNotBecomeTheirOwnManager covers the
// direct case: the successor is one of the leaver's own reports. Handing the
// team over unconditionally would set their manager to themselves.
func TestOffboard_SuccessorInsideTheTeamDoesNotBecomeTheirOwnManager(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	svc := newOffboardSvc(pool)

	adminID := seedOffboardUser(t, pool, tenantID, "admin4-"+uuid.NewString()+"@example.test")
	grantRoleAdmin(t, pool, tenantID, adminID)

	bossID := seedOffboardUser(t, pool, tenantID, "boss4-"+uuid.NewString()+"@example.test")
	seedOffboardProfile(t, pool, tenantID, bossID, nil)

	leadID := seedOffboardUser(t, pool, tenantID, "lead4-"+uuid.NewString()+"@example.test")
	leadProfile := seedOffboardProfile(t, pool, tenantID, leadID, &bossID)

	successorID := seedOffboardUser(t, pool, tenantID, "succ4-"+uuid.NewString()+"@example.test")
	seedOffboardProfile(t, pool, tenantID, successorID, &leadID)
	peerID := seedOffboardUser(t, pool, tenantID, "peer4-"+uuid.NewString()+"@example.test")
	seedOffboardProfile(t, pool, tenantID, peerID, &leadID)

	in := defaultOffboardInput()
	in.SuccessorUserID = &successorID
	if _, err := svc.OffboardEmployee(ctx, tenantID, leadProfile, adminID, in); err != nil {
		t.Fatalf("OffboardEmployee: %v", err)
	}

	// The successor is promoted into the leaver's slot, not under themselves.
	if mgr := managerOf(t, pool, tenantID, successorID); mgr == nil || *mgr != bossID {
		t.Errorf("successor's manager: got %v, want the leaver's manager %s", mgr, bossID)
	}
	// The rest of the team reports to the successor.
	if mgr := managerOf(t, pool, tenantID, peerID); mgr == nil || *mgr != successorID {
		t.Errorf("peer's manager: got %v, want the successor %s", mgr, successorID)
	}
}

// TestOffboard_SuccessorTwoLevelsDownDoesNotCloseALoop is the case a simple
// "skip the successor" fix would miss: the successor is a grandchild, so their
// own manager is among the rows being moved. Without walking the chain the two
// would end up managing each other.
func TestOffboard_SuccessorTwoLevelsDownDoesNotCloseALoop(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	svc := newOffboardSvc(pool)

	adminID := seedOffboardUser(t, pool, tenantID, "admin5-"+uuid.NewString()+"@example.test")
	grantRoleAdmin(t, pool, tenantID, adminID)

	bossID := seedOffboardUser(t, pool, tenantID, "boss5-"+uuid.NewString()+"@example.test")
	seedOffboardProfile(t, pool, tenantID, bossID, nil)

	leadID := seedOffboardUser(t, pool, tenantID, "lead5-"+uuid.NewString()+"@example.test")
	leadProfile := seedOffboardProfile(t, pool, tenantID, leadID, &bossID)

	// middle reports to the leaver, successor reports to middle.
	middleID := seedOffboardUser(t, pool, tenantID, "middle5-"+uuid.NewString()+"@example.test")
	seedOffboardProfile(t, pool, tenantID, middleID, &leadID)
	successorID := seedOffboardUser(t, pool, tenantID, "succ5-"+uuid.NewString()+"@example.test")
	seedOffboardProfile(t, pool, tenantID, successorID, &middleID)

	in := defaultOffboardInput()
	in.SuccessorUserID = &successorID
	if _, err := svc.OffboardEmployee(ctx, tenantID, leadProfile, adminID, in); err != nil {
		t.Fatalf("OffboardEmployee: %v", err)
	}

	// middle is on the chain from the successor upwards, so it inherits the
	// leaver's manager instead of reporting to its own former report.
	if mgr := managerOf(t, pool, tenantID, middleID); mgr == nil || *mgr != bossID {
		t.Errorf("middle's manager: got %v, want %s (a loop was closed)", mgr, bossID)
	}
	// And the successor still reports to middle — nothing moved them.
	if mgr := managerOf(t, pool, tenantID, successorID); mgr == nil || *mgr != middleID {
		t.Errorf("successor's manager: got %v, want %s", mgr, middleID)
	}
}

// TestOffboard_LastRoleAdminIsRefused: a tenant that loses its last role
// administrator cannot give the right back to itself.
func TestOffboard_LastRoleAdminIsRefused(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	svc := newOffboardSvc(pool)

	onlyAdminID := seedOffboardUser(t, pool, tenantID, "only6-"+uuid.NewString()+"@example.test")
	grantRoleAdmin(t, pool, tenantID, onlyAdminID)
	profileID := seedOffboardProfile(t, pool, tenantID, onlyAdminID, nil)

	actorID := seedOffboardUser(t, pool, tenantID, "actor6-"+uuid.NewString()+"@example.test")

	_, err := svc.OffboardEmployee(ctx, tenantID, profileID, actorID, defaultOffboardInput())
	if !errors.Is(err, ErrLastRoleAdmin) {
		t.Fatalf("got %v, want ErrLastRoleAdmin", err)
	}
}

// TestOffboard_SelfIsRefused: the session stays valid until it expires, so
// self-offboarding is a mistake that only surfaces at the next login.
func TestOffboard_SelfIsRefused(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	svc := newOffboardSvc(pool)

	otherAdminID := seedOffboardUser(t, pool, tenantID, "other7-"+uuid.NewString()+"@example.test")
	grantRoleAdmin(t, pool, tenantID, otherAdminID)

	selfID := seedOffboardUser(t, pool, tenantID, "self7-"+uuid.NewString()+"@example.test")
	profileID := seedOffboardProfile(t, pool, tenantID, selfID, nil)

	_, err := svc.OffboardEmployee(ctx, tenantID, profileID, selfID, defaultOffboardInput())
	if !errors.Is(err, ErrSelfOffboard) {
		t.Fatalf("got %v, want ErrSelfOffboard", err)
	}
}

// TestOffboard_SuccessorFromAnotherTenantIsRefused: RLS hides the foreign
// profile, so the lookup comes back empty rather than handing a team across the
// tenant boundary.
func TestOffboard_SuccessorFromAnotherTenantIsRefused(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	svc := newOffboardSvc(pool)

	foreignTenant := uuid.New()
	testutil.EnsureTenant(t, pool, foreignTenant, "Offboard Foreign Tenant")
	foreignUserID := seedOffboardUser(t, pool, foreignTenant, "foreign8-"+uuid.NewString()+"@example.test")
	seedOffboardProfile(t, pool, foreignTenant, foreignUserID, nil)

	adminID := seedOffboardUser(t, pool, tenantID, "admin8-"+uuid.NewString()+"@example.test")
	grantRoleAdmin(t, pool, tenantID, adminID)
	leadID := seedOffboardUser(t, pool, tenantID, "lead8-"+uuid.NewString()+"@example.test")
	leadProfile := seedOffboardProfile(t, pool, tenantID, leadID, nil)
	reportID := seedOffboardUser(t, pool, tenantID, "report8-"+uuid.NewString()+"@example.test")
	seedOffboardProfile(t, pool, tenantID, reportID, &leadID)

	in := defaultOffboardInput()
	in.SuccessorUserID = &foreignUserID
	_, err := svc.OffboardEmployee(ctx, tenantID, leadProfile, adminID, in)
	if !errors.Is(err, ErrInvalidSuccessor) {
		t.Fatalf("got %v, want ErrInvalidSuccessor", err)
	}
}

// TestOffboard_UnknownExitTypeIsRefused keeps a value the CHECK constraint
// would reject from reaching the database as a constraint error.
func TestOffboard_UnknownExitTypeIsRefused(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	svc := newOffboardSvc(pool)

	adminID := seedOffboardUser(t, pool, tenantID, "admin9-"+uuid.NewString()+"@example.test")
	leaverID := seedOffboardUser(t, pool, tenantID, "leaver9-"+uuid.NewString()+"@example.test")
	profileID := seedOffboardProfile(t, pool, tenantID, leaverID, nil)

	in := defaultOffboardInput()
	in.ExitType = "fired_by_email"
	_, err := svc.OffboardEmployee(ctx, tenantID, profileID, adminID, in)
	if !errors.Is(err, ErrInvalidExitType) {
		t.Fatalf("got %v, want ErrInvalidExitType", err)
	}
}
