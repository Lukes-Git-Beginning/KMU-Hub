package database

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

// The Welle-2 PrepareConn hook is verified against a real Postgres connection
// because pgx.Conn is not mockable in a way that exercises set_config /
// current_setting. Tests skip when DATABASE_URL is unset.

// TestPrepareConn_TenantContext stamps tenant_id/user_id/role on a connection
// acquired with a tenant-bearing context and verifies current_setting reads
// them back.
func TestPrepareConn_TenantContext(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("TEST_DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := NewPostgresPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tctx := context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())
	tctx = context.WithValue(tctx, middleware.UserIDKey, userID.String())

	var gotTenant, gotUser, gotRole string
	row := pool.QueryRow(tctx,
		`SELECT current_setting('app.tenant_id', true),
                current_setting('app.user_id',   true),
                current_setting('app.role',      true)`)
	if err := row.Scan(&gotTenant, &gotUser, &gotRole); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if gotTenant != tenantID.String() {
		t.Errorf("app.tenant_id = %q; want %q", gotTenant, tenantID)
	}
	if gotUser != userID.String() {
		t.Errorf("app.user_id = %q; want %q", gotUser, userID)
	}
	if gotRole != "user" {
		t.Errorf("app.role = %q; want user", gotRole)
	}
}

// TestPrepareConn_SystemContext verifies that sysctx.With sets app.role='system'
// regardless of any tenant in the context, so RLS policies bypass via
// is_system_context().
func TestPrepareConn_SystemContext(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("TEST_DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := NewPostgresPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	sctx := sysctx.With(ctx)

	var gotTenant, gotRole string
	row := pool.QueryRow(sctx,
		`SELECT current_setting('app.tenant_id', true),
                current_setting('app.role',      true)`)
	if err := row.Scan(&gotTenant, &gotRole); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if gotTenant != "" {
		t.Errorf("app.tenant_id under system context = %q; want empty", gotTenant)
	}
	if gotRole != "system" {
		t.Errorf("app.role = %q; want system", gotRole)
	}
}

// TestPrepareConn_NoTenantNoSystem leaves the GUCs at the AfterRelease default
// (empty) when neither sysctx nor a tenant is set. RLS policies then admit
// nothing, which is the safe default for accidentally-unwrapped paths.
func TestPrepareConn_NoTenantNoSystem(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("TEST_DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := NewPostgresPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	var gotTenant, gotUser, gotRole string
	row := pool.QueryRow(ctx,
		`SELECT current_setting('app.tenant_id', true),
                current_setting('app.user_id',   true),
                current_setting('app.role',      true)`)
	if err := row.Scan(&gotTenant, &gotUser, &gotRole); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if gotTenant != "" {
		t.Errorf("app.tenant_id = %q; want empty (no tenant, no system)", gotTenant)
	}
	if gotUser != "" {
		t.Errorf("app.user_id = %q; want empty", gotUser)
	}
	if gotRole != "" {
		t.Errorf("app.role = %q; want empty", gotRole)
	}
}

// TestAfterRelease_ResetsGUCs verifies that GUCs from one acquire do not leak
// to the next acquire of the same connection. Tenant-A acquires, sets GUCs,
// releases; Tenant-B acquires the same conn (in practice; pool reuse) and
// must see Tenant-B's stamp, not Tenant-A's.
func TestAfterRelease_ResetsGUCs(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("TEST_DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := NewPostgresPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	tenantA := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	tenantB := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	// First acquire as Tenant A.
	ctxA := context.WithValue(ctx, middleware.TenantIDKey, tenantA.String())
	var aTenant string
	row := pool.QueryRow(ctxA, `SELECT current_setting('app.tenant_id', true)`)
	if err := row.Scan(&aTenant); err != nil {
		t.Fatalf("scan A: %v", err)
	}
	if aTenant != tenantA.String() {
		t.Fatalf("first acquire: app.tenant_id = %q; want %q", aTenant, tenantA)
	}

	// Second acquire as Tenant B: PrepareConn must overwrite with B; AfterRelease
	// must have cleared between acquires.
	ctxB := context.WithValue(ctx, middleware.TenantIDKey, tenantB.String())
	var bTenant string
	row = pool.QueryRow(ctxB, `SELECT current_setting('app.tenant_id', true)`)
	if err := row.Scan(&bTenant); err != nil {
		t.Fatalf("scan B: %v", err)
	}
	if bTenant != tenantB.String() {
		t.Errorf("second acquire: app.tenant_id = %q; want %q (no leak between acquires)", bTenant, tenantB)
	}
}
