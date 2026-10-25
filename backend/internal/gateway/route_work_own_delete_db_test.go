package gateway

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/testutil"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

// Deleting an own task comment or own time entry must work for the member
// preset. The token here is minted the way login does it: the real seed
// (migration 000256 and its predecessors) through ResolveTokenPermissions, then
// the real Auth middleware and the real route registration. A hand-built token
// would have kept the old guard green: member holds tasks:write, not
// tasks:delete.
//
// Who may delete WHICH comment or entry is the service's call (comment.Service
// Delete, timeentry.Service DeleteEntry, both covered in their packages); this
// test pins the gateway half — who gets through the guard and what the RPC
// carries.

var ownDeleteTenant = uuid.MustParse("e4fec700-0000-4000-8000-0000000000d1")

type fakeWorkDeleteServer struct {
	workv1.UnimplementedWorkServiceServer

	mu       sync.Mutex
	comments []*workv1.DeleteTaskCommentRequest
	entries  []*workv1.DeleteTimeEntryRequest
}

func (f *fakeWorkDeleteServer) DeleteTaskComment(_ context.Context, req *workv1.DeleteTaskCommentRequest) (*workv1.DeleteTaskCommentResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.comments = append(f.comments, req)
	return &workv1.DeleteTaskCommentResponse{}, nil
}

func (f *fakeWorkDeleteServer) DeleteTimeEntry(_ context.Context, req *workv1.DeleteTimeEntryRequest) (*workv1.DeleteTimeEntryResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, req)
	return &workv1.DeleteTimeEntryResponse{}, nil
}

func (f *fakeWorkDeleteServer) calls() (comments, entries int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.comments), len(f.entries)
}

func TestWorkOwnDelete_GuardAgainstRealPresetToken(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	testutil.EnsureTenant(t, pool, ownDeleteTenant, "OwnDeleteTenant")

	tokenMaker := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	authSvc := auth.NewService(auth.NewPostgresRepository(pool), tokenMaker)

	fake := &fakeWorkDeleteServer{}
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	workv1.RegisterWorkServiceServer(srv, fake)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("work", lis.Addr().String())
	t.Cleanup(reg.Close)

	router := chi.NewRouter()
	NewWorkRoutes(reg).RegisterRoutes(router, middleware.Auth(authSvc))

	sysCtx := testutil.WithSystemCtx(context.Background())

	tests := []struct {
		role string
		want int
	}{
		{"member", http.StatusOK},
		{"manager", http.StatusOK},
		{"admin", http.StatusOK},
		{"readonly", http.StatusForbidden},
		{"extern", http.StatusForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.role, func(t *testing.T) {
			userID := testutil.SeedRow(t, pool, "users", map[string]any{
				"id":            uuid.New(),
				"tenant_id":     ownDeleteTenant,
				"email":         "own-delete-" + tc.role + "@test.local",
				"password_hash": "x",
				"first_name":    "Own",
				"last_name":     "Delete",
			})
			t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

			var roleID uuid.UUID
			if err := pool.QueryRow(sysCtx,
				`SELECT id FROM roles WHERE name = $1 AND tenant_id IS NULL`, tc.role).Scan(&roleID); err != nil {
				t.Fatalf("preset role %q: %v", tc.role, err)
			}
			if _, err := pool.Exec(sysCtx,
				`INSERT INTO user_roles (user_id, role_id, tenant_id) VALUES ($1, $2, $3)`,
				userID, roleID, ownDeleteTenant); err != nil {
				t.Fatalf("assign role: %v", err)
			}

			perms, err := authSvc.ResolveTokenPermissions(sysCtx, ownDeleteTenant, userID)
			if err != nil {
				t.Fatalf("resolve permissions: %v", err)
			}
			token, err := tokenMaker.CreateAccessToken(userID, ownDeleteTenant.String(), []string{tc.role}, perms)
			if err != nil {
				t.Fatalf("create token: %v", err)
			}

			commentsBefore, entriesBefore := fake.calls()
			for _, path := range []string{
				"/api/v1/task-comments/" + uuid.NewString(),
				"/api/v1/time-entries/" + uuid.NewString(),
			} {
				req := httptest.NewRequest(http.MethodDelete, path, nil)
				req.Header.Set("Authorization", "Bearer "+token)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if rec.Code != tc.want {
					t.Errorf("%s as %s = %d, want %d (body %s)", path, tc.role, rec.Code, tc.want, rec.Body.String())
				}
			}

			commentsAfter, entriesAfter := fake.calls()
			reached := tc.want == http.StatusOK
			if got := commentsAfter-commentsBefore == 1 && entriesAfter-entriesBefore == 1; got != reached {
				t.Errorf("RPC reached = %v for %s, want %v", got, tc.role, reached)
			}
		})
	}

	// The comment RPC must carry the caller's admin flag from the token roles,
	// never a client value: member and manager stay false, so the service's
	// author check applies to them.
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.comments) != 3 {
		t.Fatalf("comment RPCs = %d, want 3 (member, manager, admin)", len(fake.comments))
	}
	adminFlags := []bool{fake.comments[0].IsAdmin, fake.comments[1].IsAdmin, fake.comments[2].IsAdmin}
	if adminFlags[0] || adminFlags[1] || !adminFlags[2] {
		t.Errorf("IsAdmin per call (member, manager, admin) = %v, want [false false true]", adminFlags)
	}
}
