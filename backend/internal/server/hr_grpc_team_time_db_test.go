package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/biz/hr/employee"
	"github.com/kmuhub/kmuhub/internal/biz/hr/leave"
	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/testutil"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// teamTimeFixture runs GetTeamTime over the real employee repository: the reporting-line
// filter is a WHERE clause on hr_employee_profiles.manager_user_id, so a mock repository
// would prove nothing about it. Tenant: a fresh one per test, so the "all" assertions are exact.
type teamTimeFixture struct {
	srv      *HRGRPCServer
	tenantID uuid.UUID
	manager  uuid.UUID
	reportA  uuid.UUID
	reportB  uuid.UUID
	outsider uuid.UUID // same tenant, reports to nobody
	otherMgr uuid.UUID // reports to nobody and has one report of their own
	otherRep uuid.UUID
}

func newTeamTimeFixture(t *testing.T, pool *pgxpool.Pool) *teamTimeFixture {
	t.Helper()
	f := &teamTimeFixture{tenantID: uuid.New()}
	testutil.EnsureTenant(t, pool, f.tenantID, "Team Time Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", f.tenantID) })

	seedUser := func(name string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"id": uuid.New(), "tenant_id": f.tenantID, "email": name + "-" + uuid.NewString() + "@test.local",
			"password_hash": "x", "first_name": name, "last_name": "Test", "is_active": true,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	// The nullable text columns are written as "" like the application does: the scan reads them
	// into plain strings.
	seedProfile := func(userID uuid.UUID, managerID *uuid.UUID) {
		cols := map[string]any{
			"id": uuid.New(), "tenant_id": f.tenantID, "user_id": userID,
			"department": "Ops", "position_title": "Staff", "contract_type": "full_time",
			"work_days_per_week": 5, "annual_leave_days": 25, "start_date": "2020-01-01",
			"address_city": "", "address_street": "", "address_postal_code": "", "address_country": "DE",
			"emergency_contact_name": "", "emergency_contact_phone": "",
		}
		if managerID != nil {
			cols["manager_user_id"] = *managerID
		}
		id := testutil.SeedRow(t, pool, "hr_employee_profiles", cols)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_profiles", id) })
	}

	f.manager, f.reportA, f.reportB = seedUser("manager"), seedUser("reporta"), seedUser("reportb")
	f.outsider, f.otherMgr, f.otherRep = seedUser("outsider"), seedUser("othermgr"), seedUser("otherrep")
	seedProfile(f.manager, nil)
	seedProfile(f.reportA, &f.manager)
	seedProfile(f.reportB, &f.manager)
	seedProfile(f.outsider, nil)
	seedProfile(f.otherMgr, nil)
	seedProfile(f.otherRep, &f.otherMgr)

	ttSvc := timetracking.NewService(timetracking.NewPostgresWorkTimeRepo(pool), nil,
		employee.NewPostgresEmployeeRepo(pool), leave.NewPostgresHRSettingsRepo(pool), pool)
	f.srv = NewHRGRPCServer(nil, ttSvc, nil, nil, nil, nil)
	return f
}

func (f *teamTimeFixture) ctxAs(caller uuid.UUID) context.Context {
	ctx := testutil.WithTenantCtx(context.Background(), f.tenantID)
	return context.WithValue(ctx, middleware.UserIDKey, caller.String())
}

func teamIDs(resp *hrv1.GetTeamTimeResp) []string {
	ids := make([]string, 0, len(resp.GetTeam()))
	for _, e := range resp.GetTeam() {
		ids = append(ids, e.GetEmployeeId())
	}
	return ids
}

func TestGetTeamTime_ScopeNarrowsToReportingLine(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	f := newTeamTimeFixture(t, pool)

	t.Run("scope team lists only the caller's reports", func(t *testing.T) {
		resp, err := f.srv.GetTeamTime(f.ctxAs(f.manager), &hrv1.GetTeamTimeReq{Scope: auth.ScopeTeam})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{f.reportA.String(), f.reportB.String()}, teamIDs(resp))
	})

	t.Run("scope all lists the whole tenant", func(t *testing.T) {
		resp, err := f.srv.GetTeamTime(f.ctxAs(f.manager), &hrv1.GetTeamTimeReq{Scope: auth.ScopeAll})
		require.NoError(t, err)
		assert.Len(t, resp.GetTeam(), 6)
	})

	t.Run("a missing or unknown scope falls back to the reporting line", func(t *testing.T) {
		for _, scope := range []string{"", "own", "bogus", "ALL"} {
			resp, err := f.srv.GetTeamTime(f.ctxAs(f.manager), &hrv1.GetTeamTimeReq{Scope: scope})
			require.NoError(t, err, "scope %q", scope)
			assert.ElementsMatch(t, []string{f.reportA.String(), f.reportB.String()}, teamIDs(resp), "scope %q", scope)
		}
	})

	t.Run("a manager without reports gets an empty list, not an error", func(t *testing.T) {
		resp, err := f.srv.GetTeamTime(f.ctxAs(f.outsider), &hrv1.GetTeamTimeReq{Scope: auth.ScopeTeam})
		require.NoError(t, err)
		assert.NotNil(t, resp.GetTeam(), "the wire list must be [] not null")
		assert.Empty(t, resp.GetTeam())
	})

	t.Run("the other manager sees only their own report", func(t *testing.T) {
		resp, err := f.srv.GetTeamTime(f.ctxAs(f.otherMgr), &hrv1.GetTeamTimeReq{Scope: auth.ScopeTeam})
		require.NoError(t, err)
		assert.Equal(t, []string{f.otherRep.String()}, teamIDs(resp))
	})
}

// Scope team needs a caller: without one there is no reporting line to resolve, and falling
// through to the whole tenant would be the leak this scope exists to close.
func TestGetTeamTime_NonAllScopeNeedsACaller(t *testing.T) {
	ctx := ctxWithTenant(uuid.New())
	for _, scope := range []string{"", auth.ScopeTeam, auth.ScopeOwn, "bogus"} {
		_, err := newTestHRServer().GetTeamTime(ctx, &hrv1.GetTeamTimeReq{Scope: scope})
		requireGRPCCode(t, err, codes.Unauthenticated)
	}
}
