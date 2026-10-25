package gateway

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/rapporte"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	rapportev1 "github.com/kmuhub/kmuhub/proto/rapporte/v1"
)

// The member preset holds rapporte:measurement:read at scope "all" but
// rapporte:report:read only at "own" (000256). Reading measurements must follow
// the report scope: a member sees measurements of their own reports and those
// without a report, never the ones hanging off a colleague's report. The chain is
// the production one end to end — the real seed through ResolveTokenPermissions,
// the real Auth middleware and route registration, a real rapporte gRPC server
// behind TenantInboundUnaryInterceptor and the Postgres repository.

var measurementReadTenant = uuid.MustParse("e4fec700-0000-4000-8000-0000000000d2")

func TestRapporteMeasurementRead_FollowsReportScope(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	testutil.EnsureTenant(t, pool, measurementReadTenant, "MeasurementReadTenant")
	sysCtx := testutil.WithSystemCtx(context.Background())

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	repo := rapporte.NewPostgresRepository(pool)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	rapportev1.RegisterRapporteServiceServer(srv, server.NewRapporteGRPCServer(rapporte.NewService(repo), repo))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("rapporte", lis.Addr().String())
	t.Cleanup(reg.Close)

	router := chi.NewRouter()
	newRapporteRoutes(reg).RegisterRoutes(router, middleware.Auth(authSvc))

	tokenFor := func(role string) (uuid.UUID, string) {
		t.Helper()
		userID := testutil.SeedRow(t, pool, "users", map[string]any{
			"id":            uuid.New(),
			"tenant_id":     measurementReadTenant,
			"email":         "measurement-read-" + uuid.NewString() + "@test.local",
			"password_hash": "x",
			"first_name":    "Measurement",
			"last_name":     role,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })
		var roleID uuid.UUID
		if err := pool.QueryRow(sysCtx,
			`SELECT id FROM roles WHERE name = $1 AND tenant_id IS NULL`, role).Scan(&roleID); err != nil {
			t.Fatalf("preset role %q: %v", role, err)
		}
		if _, err := pool.Exec(sysCtx,
			`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`,
			userID, roleID, measurementReadTenant); err != nil {
			t.Fatalf("assign role: %v", err)
		}
		perms, err := authSvc.ResolveTokenPermissions(sysCtx, measurementReadTenant, userID)
		if err != nil {
			t.Fatalf("resolve permissions: %v", err)
		}
		token, err := tokenMaker.CreateAccessToken(userID, measurementReadTenant.String(), []string{role}, perms)
		if err != nil {
			t.Fatalf("create token: %v", err)
		}
		return userID, token
	}

	memberID, memberToken := tokenFor("member")
	colleagueID, _ := tokenFor("member")
	_, managerToken := tokenFor("manager")

	seedReport := func(author uuid.UUID) uuid.UUID {
		id := testutil.SeedRow(t, pool, "work_reports", map[string]any{
			"id":        uuid.New(),
			"tenant_id": measurementReadTenant,
			"title":     "Rapport",
			"author_id": author,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "work_reports", id) })
		return id
	}
	seedMeasurement := func(reportID *uuid.UUID) uuid.UUID {
		cols := map[string]any{"id": uuid.New(), "tenant_id": measurementReadTenant, "title": "Aufmaß"}
		if reportID != nil {
			cols["report_id"] = *reportID
		}
		id := testutil.SeedRow(t, pool, "measurements", cols)
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "measurements", id) })
		return id
	}
	ownReport := seedReport(memberID)
	foreignReport := seedReport(colleagueID)
	ownM := seedMeasurement(&ownReport)
	foreignM := seedMeasurement(&foreignReport)
	orphanM := seedMeasurement(nil)

	do := func(token, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	list := func(t *testing.T, token, query string) map[uuid.UUID]bool {
		t.Helper()
		rec := do(token, "/api/v1/rapporte/measurements"+query)
		if rec.Code != http.StatusOK {
			t.Fatalf("list%s = %d, body %s", query, rec.Code, rec.Body.String())
		}
		var resp rapportev1.ListMeasurementsResponse
		if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode list: %v", err)
		}
		seen := map[uuid.UUID]bool{}
		for _, m := range resp.GetMeasurements() {
			seen[uuid.MustParse(m.GetId())] = true
		}
		if int(resp.GetTotal()) != len(seen) {
			t.Errorf("list%s total = %d, items = %d", query, resp.GetTotal(), len(seen))
		}
		return seen
	}

	t.Run("member", func(t *testing.T) {
		seen := list(t, memberToken, "")
		if !seen[ownM] || !seen[orphanM] || seen[foreignM] {
			t.Errorf("member list own=%v orphan=%v foreign=%v, want true true false", seen[ownM], seen[orphanM], seen[foreignM])
		}
		if seen := list(t, memberToken, "?report_id="+foreignReport.String()); len(seen) != 0 {
			t.Errorf("member list for a colleague's report = %d items, want 0", len(seen))
		}
		for id, want := range map[uuid.UUID]int{ownM: http.StatusOK, orphanM: http.StatusOK, foreignM: http.StatusNotFound} {
			if rec := do(memberToken, "/api/v1/rapporte/measurements/"+id.String()+"/"); rec.Code != want {
				t.Errorf("member get %s = %d, want %d (body %s)", id, rec.Code, want, rec.Body.String())
			}
		}
	})

	t.Run("manager", func(t *testing.T) {
		seen := list(t, managerToken, "")
		if !seen[ownM] || !seen[orphanM] || !seen[foreignM] {
			t.Errorf("manager list own=%v orphan=%v foreign=%v, want all true", seen[ownM], seen[orphanM], seen[foreignM])
		}
		if rec := do(managerToken, "/api/v1/rapporte/measurements/"+foreignM.String()+"/"); rec.Code != http.StatusOK {
			t.Errorf("manager get foreign = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
	})
}
