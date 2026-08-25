package gateway

// route_hr_manual_entry_idempotency_db_test.go belongs to
// fix-hr-manual-entry-idempotency-key-not-enforced (BACKLOG.yml). The unit's
// premise ("duplicate manual entries slip through") was wrong: POST
// /api/v1/hr/time/entries runs through cmd/gateway/main.go's
// authWithIdempotency chain (authMiddleware(idempotencyMW(next))) like every
// other route registered via reg.RegisterRoutes, so middleware.Idempotency
// already deduplicates it. This test builds that exact chain — a real
// middleware.Idempotency backed by Postgres, wrapping HandleCreateManualEntry,
// which itself reaches a REAL HR gRPC server (via a loopback grpc.ClientConn,
// same registry.GetConnection() path production uses) backed by a REAL
// timetracking.Service and PostgresWorkTimeRepo — and proves the existing
// wiring holds: a repeated Idempotency-Key replays instead of writing a
// second row, and a different key with the identical body creates an
// independent one.

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

// hrManualEntryIdempotencyFixture wires a real HR gRPC server (loopback TCP,
// same TenantInboundUnaryInterceptor production uses) plus the real gateway
// idempotency chain in front of HandleCreateManualEntry.
type hrManualEntryIdempotencyFixture struct {
	pool       *pgxpool.Pool
	tenantID   uuid.UUID
	employeeID uuid.UUID
	idempRepo  idempotency.Repository
	chain      http.Handler
	grpcServer *grpc.Server
}

func newHRManualEntryIdempotencyFixture(t *testing.T) *hrManualEntryIdempotencyFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "HR Manual Entry Idempotency")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	employeeID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     tenantID,
		"email":         "hr-idem-" + uuid.New().String() + "@test.local",
		"password_hash": "x",
		"first_name":    "Idem",
		"last_name":     "Test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", employeeID) })
	ctxSystem := testutil.WithSystemCtx(context.Background())
	t.Cleanup(func() {
		if _, err := pool.Exec(ctxSystem, "DELETE FROM hr_work_time_entries WHERE employee_id = $1", employeeID); err != nil {
			t.Logf("cleanup hr_work_time_entries employee=%s: %v", employeeID, err)
		}
	})

	// Real HR gRPC server, only the timetracking dependency wired: every
	// other *Service param stays nil because CreateManualEntry only touches
	// workTimeRepo (assertWeekEditable no-ops when weekApprovalRepo is nil —
	// see service_week_lock.go:49-53).
	workTimeRepo := timetracking.NewPostgresWorkTimeRepo(pool)
	timetrackingSvc := timetracking.NewService(workTimeRepo, nil, nil, nil, pool)
	hrServer := server.NewHRGRPCServer(nil, timetrackingSvc, nil, nil, nil, nil)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()),
	)
	hrv1.RegisterHRServiceServer(grpcServer, hrServer)
	go func() { _ = grpcServer.Serve(lis) }()
	t.Cleanup(grpcServer.GracefulStop)

	registry := NewServiceRegistry(nil)
	registry.Register("biz", lis.Addr().String())
	routes := NewHRRoutes(registry, nil)

	idempRepo := idempotency.NewPostgresRepository(pool)
	idempMW := middleware.Idempotency(idempRepo, middleware.WarnMode)

	// Mirrors cmd/gateway/main.go:205-206's authWithIdempotency :=
	// authMiddleware(idempotencyMW(next)) — auth runs first and stamps the
	// tenant/user context idempotencyMW and the HR handler both read.
	fakeAuth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, withAuth(r, employeeID.String(), tenantID))
		})
	}
	chain := fakeAuth(idempMW(http.HandlerFunc(routes.HandleCreateManualEntry)))

	return &hrManualEntryIdempotencyFixture{
		pool:       pool,
		tenantID:   tenantID,
		employeeID: employeeID,
		idempRepo:  idempRepo,
		chain:      chain,
		grpcServer: grpcServer,
	}
}

func (f *hrManualEntryIdempotencyFixture) postEntry(t *testing.T, key string, clockIn, clockOut time.Time) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"clock_in":      clockIn.Format(time.RFC3339),
		"clock_out":     clockOut.Format(time.RFC3339),
		"break_minutes": 0,
		"activity":      "idempotency-probe",
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/hr/time/entries", bytes.NewReader(body))
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	f.chain.ServeHTTP(rec, req)
	return rec
}

// waitForCompletion polls the idempotency record until the async
// repo.Complete goroutine (middleware/idempotency.go:165) has written a
// response — without this, a second request racing the first would see an
// in-flight reservation (409) instead of the replay this test is proving.
func (f *hrManualEntryIdempotencyFixture) waitForCompletion(t *testing.T, key string) {
	t.Helper()
	ctx := testutil.WithTenantCtx(context.Background(), f.tenantID)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rec, err := f.idempRepo.Get(ctx, f.tenantID, key)
		if err == nil && rec.CompletedAt != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("idempotency key %q did not complete within deadline", key)
}

func (f *hrManualEntryIdempotencyFixture) countEntries(t *testing.T) int {
	t.Helper()
	// Tenant-scoped ctx, not system ctx: this goes through the same RLS
	// path a production read does (PoolFromEnv's PrepareConn hook stamps
	// app.tenant_id from ctx on Acquire).
	ctx := testutil.WithTenantCtx(context.Background(), f.tenantID)
	var count int
	if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM hr_work_time_entries WHERE employee_id = $1", f.employeeID).Scan(&count); err != nil {
		t.Fatalf("count hr_work_time_entries: %v", err)
	}
	return count
}

func TestHandleCreateManualEntry_SameIdempotencyKey_ReplaysInsteadOfDuplicating(t *testing.T) {
	f := newHRManualEntryIdempotencyFixture(t)

	clockIn := time.Now().Add(-3 * time.Hour).Truncate(time.Second)
	clockOut := clockIn.Add(90 * time.Minute)
	key := "hr-manual-entry-" + uuid.New().String()

	first := f.postEntry(t, key, clockIn, clockOut)
	assertStatus(t, first, http.StatusCreated)
	if first.Header().Get("Idempotency-Replayed") != "" {
		t.Fatalf("first request must not be marked replayed, got header %q", first.Header().Get("Idempotency-Replayed"))
	}

	f.waitForCompletion(t, key)

	second := f.postEntry(t, key, clockIn, clockOut)
	assertStatus(t, second, http.StatusCreated)
	if got := second.Header().Get("Idempotency-Replayed"); got != "true" {
		t.Fatalf("second request Idempotency-Replayed header = %q, want %q", got, "true")
	}

	if got := f.countEntries(t); got != 1 {
		t.Fatalf("hr_work_time_entries count for employee = %d, want 1 (replay must not insert a second row)", got)
	}
}

func TestHandleCreateManualEntry_DifferentIdempotencyKey_CreatesIndependentEntry(t *testing.T) {
	f := newHRManualEntryIdempotencyFixture(t)

	clockIn := time.Now().Add(-5 * time.Hour).Truncate(time.Second)
	clockOut := clockIn.Add(45 * time.Minute)

	first := f.postEntry(t, "hr-manual-entry-a-"+uuid.New().String(), clockIn, clockOut)
	assertStatus(t, first, http.StatusCreated)

	second := f.postEntry(t, "hr-manual-entry-b-"+uuid.New().String(), clockIn, clockOut)
	assertStatus(t, second, http.StatusCreated)
	if second.Header().Get("Idempotency-Replayed") != "" {
		t.Fatalf("a different key must not replay, got header %q", second.Header().Get("Idempotency-Replayed"))
	}

	if got := f.countEntries(t); got != 2 {
		t.Fatalf("hr_work_time_entries count for employee = %d, want 2 (different keys must not be deduplicated)", got)
	}
}
