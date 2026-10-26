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
	"github.com/kmuhub/kmuhub/internal/biz/hr/absence"
	"github.com/kmuhub/kmuhub/internal/biz/hr/leave"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// The absence calendar opens to everyone holding team:absence:read, but the leave type
// ('Krankheit' is a health datum) follows team:absence_data:view: a member sees who is away and
// the type of their own absences only, a manager also the type of their direct reports, admin and
// hr_admin the full entries. Real seed through ResolveTokenPermissions, real Auth middleware and
// route registration, a real HR gRPC server behind TenantInboundUnaryInterceptor, Postgres as
// kmuhub_app.
func TestHRAbsenceCalendar_ProjectsTheLeaveTypeByReach(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "HR Absence Calendar")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	absenceSvc := absence.NewService(absence.NewPostgresAbsenceRepo(pool), leave.NewPostgresHRSettingsRepo(pool))
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	hrv1.RegisterHRServiceServer(srv, server.NewHRGRPCServer(nil, nil, nil, absenceSvc, nil, nil))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("biz", lis.Addr().String())
	t.Cleanup(reg.Close)

	router := chi.NewRouter()
	NewHRRoutes(reg, nil).RegisterRoutes(router, middleware.Auth(authSvc))

	userFor := func(role string) (uuid.UUID, string) {
		t.Helper()
		userID := testutil.SeedRow(t, pool, "users", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "email": "hr-absence-cal-" + uuid.NewString() + "@test.local",
			"password_hash": "x", "first_name": "Kalender", "last_name": role,
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

	for _, p := range [][2]uuid.UUID{{reportID, managerID}, {strangerID, otherManagerID}} {
		id := testutil.SeedRow(t, pool, "hr_employee_profiles", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "user_id": p[0], "manager_user_id": p[1],
			"department": "Lager", "start_date": "2020-01-01",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_profiles", id) })
	}
	sickTypeID := testutil.SeedRow(t, pool, "hr_leave_types", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "name": "Krankheit", "key": "sick", "color": "#ef4444",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_leave_types", sickTypeID) })
	for _, employeeID := range []uuid.UUID{reportID, strangerID} {
		id := testutil.SeedRow(t, pool, "hr_leave_requests", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "employee_id": employeeID, "leave_type_id": sickTypeID,
			"start_date": "2026-03-02", "end_date": "2026-03-04", "total_days": 3, "status": "approved",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_leave_requests", id) })
	}

	calendar := func(t *testing.T, token string) map[string]map[string]any {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet,
			"/api/v1/hr/absences/calendar?start_date=2026-03-01&end_date=2026-03-31", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body struct {
			Entries []map[string]any `json:"entries"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		byEmployee := map[string]map[string]any{}
		for _, e := range body.Entries {
			byEmployee[e["employee_id"].(string)] = e
		}
		require.Len(t, byEmployee, 2, "everyone holding the guard sees who is away: %s", rec.Body.String())
		for _, e := range byEmployee {
			assert.Equal(t, "Lager", e["department"])
			assert.Equal(t, "2026-03-02", e["start_date"])
			assert.Equal(t, "2026-03-04", e["end_date"])
			assert.NotContains(t, e, "total_days")
		}
		return byEmployee
	}
	assertHidden := func(t *testing.T, e map[string]any) {
		t.Helper()
		assert.NotContains(t, e, "leave_type_name")
		assert.Equal(t, absenceNeutralColor, e["leave_type_color"])
		assert.Equal(t, "absent", e["absence_class"])
	}
	assertShown := func(t *testing.T, e map[string]any) {
		t.Helper()
		assert.Equal(t, "Krankheit", e["leave_type_name"])
		assert.Equal(t, "#ef4444", e["leave_type_color"])
	}

	t.Run("member sees the type of their own absence only", func(t *testing.T) {
		entries := calendar(t, reportToken)
		assertShown(t, entries[reportID.String()])
		assertHidden(t, entries[strangerID.String()])
	})
	t.Run("manager sees the type for their reporting line only", func(t *testing.T) {
		entries := calendar(t, managerToken)
		assertShown(t, entries[reportID.String()])
		assertHidden(t, entries[strangerID.String()])
	})
	t.Run("admin and hr_admin see the full entries", func(t *testing.T) {
		for _, token := range []string{adminToken, hrAdminToken} {
			entries := calendar(t, token)
			for _, id := range []uuid.UUID{reportID, strangerID} {
				assertShown(t, entries[id.String()])
				assert.NotContains(t, entries[id.String()], "absence_class")
			}
		}
	})
}
