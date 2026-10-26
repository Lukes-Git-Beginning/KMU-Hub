package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/schichten"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	schichtenv1 "github.com/kmuhub/kmuhub/proto/schichten/v1"
)

// The member preset holds schichten:assignment:read at scope "own" and
// schichten:template:read (000355): a member sees their own roster — the rows
// under their user id and under their HR profile id, which the desktop roster
// assigns by — but not the colleagues' on the same shift; a manager sees all.
// The chain is the production one: the real seed through
// ResolveTokenPermissions, the real Auth middleware and route registration, a
// real schichten gRPC server behind TenantInboundUnaryInterceptor and the
// Postgres repository.

var ownAssignmentsTenant = uuid.MustParse("e4fec700-0000-4000-8000-0000000000e5")

// schichtenSeedHarness wires the production chain for one tenant: users with
// a preset role and a token from the real seed, HR profiles, shifts with
// assignments, and the registered schichten routes behind middleware.Auth.
type schichtenSeedHarness struct {
	t          *testing.T
	pool       *pgxpool.Pool
	tenant     uuid.UUID
	repo       *schichten.PostgresRepository
	authSvc    *auth.Service
	tokenMaker *auth.TokenMaker
	router     chi.Router
}

func newSchichtenSeedHarness(t *testing.T, tenant uuid.UUID) *schichtenSeedHarness {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	testutil.EnsureTenant(t, pool, tenant, "SchichtenSeedTenant")

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	repo := schichten.NewPostgresRepository(pool)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	schichtenv1.RegisterSchichtenServiceServer(srv, server.NewSchichtenGRPCServer(schichten.NewService(repo)))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("schichten", lis.Addr().String())
	t.Cleanup(reg.Close)

	router := chi.NewRouter()
	newSchichtenRoutes(reg).RegisterRoutes(router, middleware.Auth(authSvc))

	return &schichtenSeedHarness{t: t, pool: pool, tenant: tenant, repo: repo, authSvc: authSvc, tokenMaker: tokenMaker, router: router}
}

// token seeds a user with the preset role and returns the user id and an
// access token whose permissions come from the seed.
func (h *schichtenSeedHarness) token(role string) (uuid.UUID, string) {
	t := h.t
	t.Helper()
	sysCtx := testutil.WithSystemCtx(context.Background())
	userID := testutil.SeedRow(t, h.pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     h.tenant,
		"email":         "schichten-seed-" + uuid.NewString() + "@test.local",
		"password_hash": "x",
		"first_name":    "Roster",
		"last_name":     role,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, h.pool, "users", userID) })
	var roleID uuid.UUID
	if err := h.pool.QueryRow(sysCtx,
		`SELECT id FROM roles WHERE name = $1 AND tenant_id IS NULL`, role).Scan(&roleID); err != nil {
		t.Fatalf("preset role %q: %v", role, err)
	}
	if _, err := h.pool.Exec(sysCtx,
		`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`,
		userID, roleID, h.tenant); err != nil {
		t.Fatalf("assign role: %v", err)
	}
	perms, err := h.authSvc.ResolveTokenPermissions(sysCtx, h.tenant, userID)
	if err != nil {
		t.Fatalf("resolve permissions: %v", err)
	}
	token, err := h.tokenMaker.CreateAccessToken(userID, h.tenant.String(), []string{role}, perms)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	return userID, token
}

// profile seeds the user's HR profile and returns its id — the identity the
// desktop roster assigns with.
func (h *schichtenSeedHarness) profile(userID uuid.UUID) uuid.UUID {
	t := h.t
	t.Helper()
	id := testutil.SeedRow(t, h.pool, "hr_employee_profiles", map[string]any{
		"id": uuid.New(), "tenant_id": h.tenant, "user_id": userID, "contract_type": "full_time",
		"work_days_per_week": 5, "annual_leave_days": 25, "start_date": time.Now().UTC(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, h.pool, "hr_employee_profiles", id) })
	return id
}

// shift seeds a shift with one assignment per employee id and returns the
// shift id and the assignment id per employee id.
func (h *schichtenSeedHarness) shift(employees ...uuid.UUID) (uuid.UUID, map[uuid.UUID]uuid.UUID) {
	t := h.t
	t.Helper()
	ctx := testutil.WithTenantCtx(context.Background(), h.tenant)
	now := time.Now().UTC()
	shift := &schichten.Shift{
		ID: uuid.New(), TenantID: h.tenant, Title: "Roster",
		StartTime: now, EndTime: now.Add(8 * time.Hour), Status: schichten.ShiftStatusDraft,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := h.repo.CreateShift(ctx, shift); err != nil {
		t.Fatalf("CreateShift: %v", err)
	}
	// Assignments and swap requests go with the shift (ON DELETE CASCADE).
	t.Cleanup(func() { testutil.CleanupRow(t, h.pool, "shifts", shift.ID) })
	assigned := map[uuid.UUID]uuid.UUID{}
	for _, emp := range employees {
		a := &schichten.ShiftAssignment{ID: uuid.New(), TenantID: h.tenant, ShiftID: shift.ID, EmployeeID: emp, AssignedAt: now}
		if err := h.repo.CreateAssignment(ctx, a); err != nil {
			t.Fatalf("CreateAssignment: %v", err)
		}
		assigned[emp] = a.ID
	}
	return shift.ID, assigned
}

func (h *schichtenSeedHarness) do(method, token, path, body string) *httptest.ResponseRecorder {
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.NewString())
	}
	rec := httptest.NewRecorder()
	h.router.ServeHTTP(rec, req)
	return rec
}

func TestSchichtenAssignments_MemberSeesOwnOnly(t *testing.T) {
	h := newSchichtenSeedHarness(t, ownAssignmentsTenant)

	memberID, memberToken := h.token("member")
	colleagueID, colleagueToken := h.token("member")
	_, managerToken := h.token("manager")
	memberProfile := h.profile(memberID)

	// One shift with the member by user id and the colleague, one with the
	// member by HR profile id (the desktop roster's identity).
	apiShift, _ := h.shift(memberID, colleagueID)
	rosterShift, _ := h.shift(memberProfile, colleagueID)

	assignments := func(t *testing.T, token string, shiftID uuid.UUID) map[string]bool {
		t.Helper()
		rec := h.do(http.MethodGet, token, "/api/v1/schichten/shifts/"+shiftID.String()+"/assignments", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("list assignments = %d, body %s", rec.Code, rec.Body.String())
		}
		var resp schichtenv1.ListAssignmentsResponse
		if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		seen := map[string]bool{}
		for _, a := range resp.GetAssignments() {
			seen[a.GetEmployeeId()] = true
		}
		return seen
	}

	t.Run("member sees their own assignment, not the colleague's", func(t *testing.T) {
		if seen := assignments(t, memberToken, apiShift); len(seen) != 1 || !seen[memberID.String()] {
			t.Errorf("member on the API shift = %v, want only %s", seen, memberID)
		}
		if seen := assignments(t, memberToken, rosterShift); len(seen) != 1 || !seen[memberProfile.String()] {
			t.Errorf("member on the roster shift = %v, want only their profile %s", seen, memberProfile)
		}
		if seen := assignments(t, colleagueToken, apiShift); len(seen) != 1 || !seen[colleagueID.String()] {
			t.Errorf("colleague on the API shift = %v, want only %s", seen, colleagueID)
		}
	})

	t.Run("a shift without the member's assignment comes back empty, not refused", func(t *testing.T) {
		other, _ := h.shift(colleagueID)
		rec := h.do(http.MethodGet, memberToken, "/api/v1/schichten/shifts/"+other.String()+"/assignments", "")
		var body map[string]any
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &body) != nil {
			t.Fatalf("member on a foreign shift = %d %s, want 200", rec.Code, rec.Body.String())
		}
		if list, ok := body["assignments"].([]any); !ok || len(list) != 0 {
			t.Errorf("member on a foreign shift = %s, want {\"assignments\":[]}", rec.Body.String())
		}
	})

	t.Run("manager sees the whole roster", func(t *testing.T) {
		if seen := assignments(t, managerToken, apiShift); len(seen) != 2 {
			t.Errorf("manager on the API shift = %v, want both", seen)
		}
	})

	t.Run("member reads the shift templates", func(t *testing.T) {
		if rec := h.do(http.MethodGet, memberToken, "/api/v1/schichten/templates", ""); rec.Code != http.StatusOK {
			t.Errorf("member templates = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
	})
}
