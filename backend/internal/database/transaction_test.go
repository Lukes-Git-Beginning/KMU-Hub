package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

type rlsGUCs struct {
	tenant, user, role, roles string
	isSystem                  bool
}

func readGUCs(t *testing.T, ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) rlsGUCs {
	t.Helper()
	var g rlsGUCs
	err := q.QueryRow(ctx,
		`SELECT coalesce(current_setting('app.tenant_id', true), ''),
                coalesce(current_setting('app.user_id', true), ''),
                coalesce(current_setting('app.role', true), ''),
                coalesce(current_setting('app.user_roles', true), ''),
                is_system_context()`).
		Scan(&g.tenant, &g.user, &g.role, &g.roles, &g.isSystem)
	if err != nil {
		t.Fatalf("read GUCs: %v", err)
	}
	return g
}

func TestBeginRLSTx_TenantContextSetsAllGUCs(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	userID := uuid.New()
	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID.String())
	ctx = context.WithValue(ctx, middleware.UserRolesKey, []string{"admin", "hr_admin"})

	tx, err := database.BeginRLSTx(ctx, pool)
	if err != nil {
		t.Fatalf("BeginRLSTx: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	got := readGUCs(t, ctx, tx)
	want := rlsGUCs{tenant: testutil.TenantA.String(), user: userID.String(), role: "user", roles: "admin,hr_admin"}
	if got != want {
		t.Fatalf("GUCs = %+v; want %+v", got, want)
	}
}

func TestBeginRLSTx_SystemContextSetsSystemRole(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	ctx := database.WithSystemContext(context.Background())
	tx, err := database.BeginRLSTx(ctx, pool)
	if err != nil {
		t.Fatalf("BeginRLSTx: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	got := readGUCs(t, ctx, tx)
	if got.role != "system" || !got.isSystem {
		t.Fatalf("role = %q, is_system_context = %v; want system/true", got.role, got.isSystem)
	}
	if got.tenant != "" || got.user != "" {
		t.Fatalf("system tx must not carry tenant/user, got tenant=%q user=%q", got.tenant, got.user)
	}
}

func TestBeginRLSTx_MissingTenantFailsClosed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	for name, ctx := range map[string]context.Context{
		"empty":          context.Background(),
		"invalid tenant": context.WithValue(context.Background(), middleware.TenantIDKey, "not-a-uuid"),
	} {
		t.Run(name, func(t *testing.T) {
			tx, err := database.BeginRLSTx(ctx, pool)
			if !errors.Is(err, database.ErrMissingTenantContext) {
				if tx != nil {
					_ = tx.Rollback(ctx)
				}
				t.Fatalf("err = %v; want ErrMissingTenantContext", err)
			}
			if tx != nil {
				t.Fatalf("tx must be nil on error")
			}
		})
	}
}

func TestBeginRLSTx_InvalidUserIDIsDropped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctx = context.WithValue(ctx, middleware.UserIDKey, "service-account")

	tx, err := database.BeginRLSTx(ctx, pool)
	if err != nil {
		t.Fatalf("BeginRLSTx: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	got := readGUCs(t, ctx, tx)
	if got.user != "" {
		t.Fatalf("app.user_id = %q; a non-UUID user must not reach the GUC", got.user)
	}
	if got.tenant != testutil.TenantA.String() || got.role != "user" {
		t.Fatalf("tenant/role = %q/%q; want TenantA/user", got.tenant, got.role)
	}
}

// The GUCs are LOCAL: after commit a plain acquire without tenant context must
// not see the tenant of the finished transaction (no leak via pool reuse).
func TestBeginRLSTx_GUCsDoNotLeakAfterCommit(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	tx, err := database.BeginRLSTx(ctx, pool)
	if err != nil {
		t.Fatalf("BeginRLSTx: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	got := readGUCs(t, context.Background(), pool)
	if got.tenant != "" || got.isSystem {
		t.Fatalf("after commit tenant=%q system=%v; want clean connection", got.tenant, got.isSystem)
	}
}

func TestWithSystemContext_RoundTrip(t *testing.T) {
	if database.IsSystemContext(context.Background()) {
		t.Fatal("plain context must not be a system context")
	}
	if !database.IsSystemContext(database.WithSystemContext(context.Background())) {
		t.Fatal("WithSystemContext must mark the context")
	}
}
