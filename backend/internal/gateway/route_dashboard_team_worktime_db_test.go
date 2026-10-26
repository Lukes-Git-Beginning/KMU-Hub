package gateway

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/biz/hr/employee"
	"github.com/kmuhub/kmuhub/internal/biz/hr/leave"
	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// The team-worktime widget lists this week's minutes with the reach of GET /hr/time/team: a
// manager sees the reporting line, admin and hr_admin the whole tenant, a member (no team grant)
// an empty list with 200. Real seed through ResolveTokenPermissions, real Auth middleware and
// route registration, a real HR gRPC server behind TenantInboundUnaryInterceptor, Postgres as
// kmuhub_app.
func TestDashboardTeamWorktime_FollowsTheTeamTimeReach(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Dashboard Team Worktime")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	ttSvc := timetracking.NewService(timetracking.NewPostgresWorkTimeRepo(pool), nil,
		employee.NewPostgresEmployeeRepo(pool), leave.NewPostgresHRSettingsRepo(pool), pool)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	hrv1.RegisterHRServiceServer(srv, server.NewHRGRPCServer(nil, ttSvc, nil, nil, nil, nil))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("biz", lis.Addr().String())
	t.Cleanup(reg.Close)

	router := chi.NewRouter()
	NewDashboardRoutes(NewDashboardService(newMockDashboardRepo())).WithHR(reg).
		RegisterRoutes(router, middleware.Auth(authSvc))

	userFor := func(role string) (uuid.UUID, string) {
		t.Helper()
		userID := testutil.SeedRow(t, pool, "users", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "email": "dash-worktime-" + uuid.NewString() + "@test.local",
			"password_hash": "x", "first_name": "Woche", "last_name": role,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
		var roleID uuid.UUID
		require.NoError(t, pool.QueryRow(sysCtx,
			`SELECT id FROM roles WHERE name = $1 AND tenant_id IS NULL`, role).Scan(&roleID))
		_, err := pool.Exec(sysCtx,
			`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`, userID, roleID, tenantID)
		require.NoError(t, err)
		perms, err := authSvc.ResolveTokenPermissions(sysCtx, tenantID, userID)
		require.NoError(t, err)
		token, err := tokenMaker.CreateAccessToken(userID, tenantID.String(), []string{role}, perms)
		require.NoError(t, err)
		return userID, token
	}

	managerID, managerToken := userFor("manager")
	otherManagerID, _ := userFor("manager")
	reportID, reportToken := userFor("member")
	strangerID, _ := userFor("member")
	_, adminToken := userFor("admin")
	_, hrAdminToken := userFor("hr_admin")

	// The nullable text columns are written as "" like the application does: the employee list
	// scans them into plain strings.
	for _, p := range [][2]uuid.UUID{{reportID, managerID}, {strangerID, otherManagerID}} {
		id := testutil.SeedRow(t, pool, "hr_employee_profiles", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "user_id": p[0], "manager_user_id": p[1],
			"department": "Lager", "position_title": "Lager", "contract_type": "full_time",
			"work_days_per_week": 5, "annual_leave_days": 25, "start_date": "2020-01-01",
			"address_city": "", "address_street": "", "address_postal_code": "", "address_country": "DE",
			"emergency_contact_name": "", "emergency_contact_phone": "",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_profiles", id) })
	}
	// clock_in = now lies in the current week whatever day the test runs.
	clockIn := time.Now()
	entryID := testutil.SeedRow(t, pool, "hr_work_time_entries", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "employee_id": reportID,
		"clock_in": clockIn, "clock_out": clockIn.Add(7 * time.Hour), "net_work_minutes": 420, "status": "completed",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_work_time_entries", entryID) })

	worktime := func(t *testing.T, token string) map[string]float64 {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/team-worktime", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body map[string][]map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		members, ok := body["members"]
		require.True(t, ok, "the wrapper key must be present: %s", rec.Body.String())
		require.NotNil(t, members, "the wire list must be [] not null: %s", rec.Body.String())
		byID := map[string]float64{}
		for _, m := range members {
			assert.NotEmpty(t, m["name"])
			byID[m["id"].(string)] = m["minutes"].(float64)
		}
		return byID
	}

	t.Run("manager sees the reporting line with this week's minutes", func(t *testing.T) {
		assert.Equal(t, map[string]float64{reportID.String(): 420}, worktime(t, managerToken))
	})
	t.Run("member gets an empty list, not 403", func(t *testing.T) {
		assert.Empty(t, worktime(t, reportToken))
	})
	t.Run("admin and hr_admin see the whole tenant", func(t *testing.T) {
		for _, token := range []string{adminToken, hrAdminToken} {
			got := worktime(t, token)
			assert.Len(t, got, 2)
			assert.Equal(t, float64(420), got[reportID.String()])
			assert.Contains(t, got, strangerID.String())
		}
	})
}
