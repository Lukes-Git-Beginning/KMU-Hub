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
	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// The team page's correction tab lists GET /hr/time/entries?status=correction_pending without
// an employee. Since the token carries real scopes, hr:self at "own" pinned every manager to
// themselves and hr_admin (no hr:* key at all) got 403. The queue now follows the approver's
// reach: manager -> own entries plus the reporting line, hr_admin/admin -> tenant, member ->
// own. The chain is the production one: the real seed through ResolveTokenPermissions, the
// real Auth middleware and route registration, a real HR gRPC server behind
// TenantInboundUnaryInterceptor and the Postgres work-time repository.
func TestHRCorrectionQueue_FollowsApproverReach(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "HR Correction Queue")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	timetrackingSvc := timetracking.NewService(timetracking.NewPostgresWorkTimeRepo(pool), nil, nil, nil, pool)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	hrv1.RegisterHRServiceServer(srv, server.NewHRGRPCServer(nil, timetrackingSvc, nil, nil, nil, nil))
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
			"id": uuid.New(), "tenant_id": tenantID, "email": "corr-queue-" + uuid.NewString() + "@test.local",
			"password_hash": "x", "first_name": "Queue", "last_name": role,
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
	_, hrAdminToken := userFor("hr_admin", nil)
	_, adminToken := userFor("admin", nil)

	seedEntry := func(employeeID uuid.UUID, status string) uuid.UUID {
		clockIn := time.Now().Add(-48 * time.Hour)
		id := testutil.SeedRow(t, pool, "hr_work_time_entries", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "employee_id": employeeID,
			"clock_in": clockIn, "clock_out": clockIn.Add(8 * time.Hour), "net_work_minutes": 480,
			"status": status, "is_correction": status == "correction_pending",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_work_time_entries", id) })
		return id
	}
	managerPending := seedEntry(managerID, "correction_pending")
	reportPending := seedEntry(reportID, "correction_pending")
	outsiderPending := seedEntry(outsiderID, "correction_pending")
	reportCompleted := seedEntry(reportID, "completed")

	get := func(token, query string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/hr/time/entries"+query, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	list := func(t *testing.T, token, query string) map[uuid.UUID]bool {
		t.Helper()
		rec := get(token, query)
		if rec.Code != http.StatusOK {
			t.Fatalf("list%s = %d, body %s", query, rec.Code, rec.Body.String())
		}
		var resp struct {
			Entries []struct {
				ID uuid.UUID `json:"id"`
			} `json:"entries"`
			Total int `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode list: %v (body %s)", err, rec.Body.String())
		}
		seen := map[uuid.UUID]bool{}
		for _, e := range resp.Entries {
			seen[e.ID] = true
		}
		if resp.Total != len(seen) {
			t.Errorf("list%s total = %d, items = %d", query, resp.Total, len(seen))
		}
		return seen
	}
	const queue = "?status=correction_pending"
	want := func(t *testing.T, who string, seen map[uuid.UUID]bool, expected ...uuid.UUID) {
		t.Helper()
		if len(seen) != len(expected) {
			t.Errorf("%s sees %d entries, want %d", who, len(seen), len(expected))
		}
		for _, id := range expected {
			if !seen[id] {
				t.Errorf("%s misses entry %s", who, id)
			}
		}
	}

	t.Run("manager reviews own and reporting line, not other teams", func(t *testing.T) {
		seen := list(t, managerToken, queue)
		want(t, "manager queue", seen, managerPending, reportPending)
		if seen[outsiderPending] {
			t.Error("manager queue contains an employee outside the reporting line")
		}
	})

	t.Run("manager stays pinned outside the queue", func(t *testing.T) {
		// Header widget and export send the unfiltered list for the caller's own entries.
		want(t, "manager unfiltered", list(t, managerToken, ""), managerPending)
		if rec := get(managerToken, "?employee_id="+reportID.String()); rec.Code != http.StatusForbidden {
			t.Errorf("manager foreign employee_id = %d, want 403", rec.Code)
		}
	})

	t.Run("member sees only own corrections", func(t *testing.T) {
		want(t, "member queue", list(t, reportToken, queue), reportPending)
		want(t, "member unfiltered", list(t, reportToken, ""), reportPending, reportCompleted)
	})

	t.Run("hr_admin reviews the tenant, nothing beyond the queue", func(t *testing.T) {
		want(t, "hr_admin queue", list(t, hrAdminToken, queue), managerPending, reportPending, outsiderPending)
		if rec := get(hrAdminToken, ""); rec.Code != http.StatusForbidden {
			t.Errorf("hr_admin unfiltered = %d, want 403 (review keys open only the queue)", rec.Code)
		}
	})

	t.Run("admin reviews the tenant", func(t *testing.T) {
		want(t, "admin queue", list(t, adminToken, queue), managerPending, reportPending, outsiderPending)
	})
}
