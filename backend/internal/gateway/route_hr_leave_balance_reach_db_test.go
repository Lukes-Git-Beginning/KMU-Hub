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
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/biz/hr/employee"
	"github.com/kmuhub/kmuhub/internal/biz/hr/leave"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// The team profile reads GET /hr/leave/balance/{userId} for a colleague. Since the token carries
// real scopes, hr:self at "own" gave every manager 403 for each report and hr_admin (no hr:* key)
// 403 for everyone. The balance now follows team:absence_data:view: manager -> self plus the
// reporting line, hr_admin/admin -> tenant, member -> self. The chain is the production one: the
// real seed through ResolveTokenPermissions, the real Auth middleware and route registration, a
// real HR gRPC server behind TenantInboundUnaryInterceptor and the Postgres leave and employee
// repositories.
func TestHRLeaveBalance_FollowsAbsenceDataReach(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "HR Leave Balance Reach")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	leaveSvc := leave.NewService(leave.NewPostgresLeaveRequestRepo(pool), leave.NewPostgresLeaveBalanceRepo(pool),
		leave.NewPostgresLeaveTypeRepo(pool), leave.NewPostgresHRSettingsRepo(pool), employee.NewPostgresEmployeeRepo(pool))
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	hrv1.RegisterHRServiceServer(srv, server.NewHRGRPCServer(leaveSvc, nil, nil, nil, nil, nil))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("biz", lis.Addr().String())
	t.Cleanup(reg.Close)

	router := chi.NewRouter()
	NewHRRoutes(reg, nil).RegisterRoutes(router, middleware.Auth(authSvc))

	userFor := func(role string, managerID *uuid.UUID) (uuid.UUID, string) {
		t.Helper()
		userID := testutil.SeedRow(t, pool, "users", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "email": "leave-reach-" + uuid.NewString() + "@test.local",
			"password_hash": "x", "first_name": "Balance", "last_name": role,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
		profile := map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "user_id": userID,
			"department": "Ops", "position_title": "Staff", "contract_type": "full_time",
			"work_days_per_week": 5, "annual_leave_days": 25, "start_date": "2020-01-01",
			"address_city": "", "address_street": "", "address_postal_code": "", "address_country": "DE",
			"emergency_contact_name": "", "emergency_contact_phone": "",
		}
		if managerID != nil {
			profile["manager_user_id"] = *managerID
		}
		profileID := testutil.SeedRow(t, pool, "hr_employee_profiles", profile)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_employee_profiles", profileID) })

		var roleID uuid.UUID
		if err := pool.QueryRow(sysCtx,
			`SELECT id FROM roles WHERE name = $1 AND tenant_id IS NULL`, role).Scan(&roleID); err != nil {
			t.Fatalf("preset role %q: %v", role, err)
		}
		if _, err := pool.Exec(sysCtx,
			`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`,
			userID, roleID, tenantID); err != nil {
			t.Fatalf("assign role: %v", err)
		}
		perms, err := authSvc.ResolveTokenPermissions(sysCtx, tenantID, userID)
		if err != nil {
			t.Fatalf("resolve permissions: %v", err)
		}
		token, err := tokenMaker.CreateAccessToken(userID, tenantID.String(), []string{role}, perms)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}
		return userID, token
	}

	managerID, managerToken := userFor("manager", nil)
	reportID, reportToken := userFor("member", &managerID)
	outsiderID, _ := userFor("member", nil)
	hrAdminID, hrAdminToken := userFor("hr_admin", nil)
	_, adminToken := userFor("admin", nil)
	// The service creates a balance row on first read; registered last, so it runs before the
	// profile and user cleanups.
	t.Cleanup(func() {
		if _, err := pool.Exec(sysCtx, `DELETE FROM hr_leave_balances WHERE tenant_id = $1`, tenantID); err != nil {
			t.Errorf("cleanup balances: %v", err)
		}
	})

	balance := func(token string, employeeID uuid.UUID) (int, string) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/leave/balance/"+employeeID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			return rec.Code, ""
		}
		var resp struct {
			EmployeeID string `json:"employee_id"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode balance: %v (body %s)", err, rec.Body.String())
		}
		return rec.Code, resp.EmployeeID
	}
	expect := func(t *testing.T, who, token string, employeeID uuid.UUID, wantCode int) {
		t.Helper()
		code, got := balance(token, employeeID)
		if code != wantCode {
			t.Fatalf("%s balance of %s = %d, want %d", who, employeeID, code, wantCode)
		}
		if code == http.StatusOK && got != employeeID.String() {
			t.Errorf("%s balance of %s carries employee_id %q", who, employeeID, got)
		}
	}

	t.Run("manager reads self and reporting line, not other teams", func(t *testing.T) {
		expect(t, "manager", managerToken, managerID, http.StatusOK)
		expect(t, "manager", managerToken, reportID, http.StatusOK)
		expect(t, "manager", managerToken, outsiderID, http.StatusForbidden)
		// An id without a profile answers like a stranger, not like a missing row.
		expect(t, "manager", managerToken, uuid.New(), http.StatusForbidden)
	})

	t.Run("member reads only self", func(t *testing.T) {
		expect(t, "member", reportToken, reportID, http.StatusOK)
		expect(t, "member", reportToken, managerID, http.StatusForbidden)
		expect(t, "member", reportToken, outsiderID, http.StatusForbidden)
	})

	t.Run("hr_admin reads the tenant", func(t *testing.T) {
		expect(t, "hr_admin", hrAdminToken, outsiderID, http.StatusOK)
		expect(t, "hr_admin", hrAdminToken, reportID, http.StatusOK)
		expect(t, "hr_admin", hrAdminToken, hrAdminID, http.StatusOK)
	})

	t.Run("admin reads the tenant", func(t *testing.T) {
		expect(t, "admin", adminToken, outsiderID, http.StatusOK)
		expect(t, "admin", adminToken, managerID, http.StatusOK)
	})
}
