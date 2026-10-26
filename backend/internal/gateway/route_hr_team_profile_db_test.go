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
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// A manager reads the full profile of their direct reports (team:data_personal:view and
// team:data_job:view at scope team, 000256) and only the directory fields of anyone else; a
// member stays on the directory everywhere. The chain is the production one: the real seed
// through ResolveTokenPermissions, the real Auth middleware and route registration, a real HR
// gRPC server behind TenantInboundUnaryInterceptor and the Postgres employee repository as
// kmuhub_app.
func TestHREmployee_ManagerSeesFullProfileOfReportsOnly(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "HR Team Profile")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	employeeSvc := employee.NewService(employee.NewPostgresEmployeeRepo(pool),
		employee.NewPostgresDocCategoryRepo(pool), employee.NewPostgresEmployeeDocRepo(pool))
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	hrv1.RegisterHRServiceServer(srv, server.NewHRGRPCServer(nil, nil, employeeSvc, nil, nil, nil))
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
			"id": uuid.New(), "tenant_id": tenantID, "email": "hr-team-profile-" + uuid.NewString() + "@test.local",
			"password_hash": "x", "first_name": "Profil", "last_name": role,
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
	_, hrAdminToken := userFor("hr_admin")

	profileFor := func(userID, managerUserID uuid.UUID) {
		t.Helper()
		id := testutil.SeedRow(t, pool, "hr_employee_profiles", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "user_id": userID, "manager_user_id": managerUserID,
			"department": "Vertrieb", "position_title": "Account Manager", "contract_type": "part_time",
			"work_days_per_week": 3, "annual_leave_days": 30, "start_date": "2020-01-01",
			"emergency_contact_name": "Hans Muster", "emergency_contact_phone": "+49 170 0000000",
			"address_street": "Hauptstr. 1", "address_city": "Berlin", "address_postal_code": "10115",
			"is_minor": true,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_profiles", id) })
	}
	profileFor(reportID, managerID)
	profileFor(strangerID, otherManagerID)

	get := func(t *testing.T, token string, userID uuid.UUID) map[string]any {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/employees/"+userID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body struct {
			Employee map[string]any `json:"employee"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.NotNil(t, body.Employee, "the profile window reads `{employee}`")
		assert.Equal(t, userID.String(), body.Employee["user_id"])
		return body.Employee
	}
	fullFields := []string{"contract_type", "work_days_per_week", "annual_leave_days", "start_date",
		"emergency_contact_name", "address_city", "is_minor"}
	assertFull := func(t *testing.T, emp map[string]any) {
		t.Helper()
		for _, f := range fullFields {
			assert.Contains(t, emp, f, "full profile carries %s", f)
		}
	}
	assertDirectory := func(t *testing.T, emp map[string]any) {
		t.Helper()
		for _, f := range directoryHiddenFields {
			assert.NotContains(t, emp, f, "directory projection must not carry %s", f)
		}
		assert.Equal(t, "Vertrieb", emp["department"])
	}

	t.Run("manager sees the full profile of a direct report", func(t *testing.T) {
		emp := get(t, managerToken, reportID)
		assertFull(t, emp)
		assert.Equal(t, "Berlin", emp["address_city"])
		assert.Equal(t, managerID.String(), emp["manager_user_id"])
	})
	t.Run("manager sees only the directory fields of someone else's report", func(t *testing.T) {
		assertDirectory(t, get(t, managerToken, strangerID))
	})
	t.Run("member sees only the directory fields of a colleague", func(t *testing.T) {
		assertDirectory(t, get(t, reportToken, strangerID))
	})
	t.Run("hr_admin holds the keys at scope all and sees every full profile", func(t *testing.T) {
		assertFull(t, get(t, hrAdminToken, strangerID))
	})
}
