package database_test

import (
	"context"
	"testing"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestPrepareConn_SystemContext_SetsAppRoleSystem verifies that the
// PrepareConn pool hook stamps app.role='system' when a context produced by
// testutil.WithSystemCtx (i.e. sysctx.With) is used to acquire a connection.
// This is the end-to-end wiring test: Pool → PrepareConn hook → set_config →
// current_setting round-trip.
func TestPrepareConn_SystemContext_SetsAppRoleSystem(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	ctxSys := testutil.WithSystemCtx(context.Background())

	var role string
	if err := pool.QueryRow(ctxSys, "SELECT current_setting('app.role', true)").Scan(&role); err != nil {
		t.Fatalf("query app.role: %v", err)
	}
	if role != "system" {
		t.Fatalf("expected app.role='system', got %q", role)
	}
}

// TestPrepareConn_TenantContext_SetsAppTenantId verifies that the hook stamps
// app.tenant_id with the UUID carried by testutil.WithTenantCtx.
func TestPrepareConn_TenantContext_SetsAppTenantId(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)

	var tid string
	if err := pool.QueryRow(ctxA, "SELECT current_setting('app.tenant_id', true)").Scan(&tid); err != nil {
		t.Fatalf("query app.tenant_id: %v", err)
	}
	if tid != testutil.TenantA.String() {
		t.Fatalf("expected app.tenant_id=%s, got %s", testutil.TenantA, tid)
	}
}

// TestPrepareConn_AfterRelease_ClearsGUCs verifies that GUCs from a
// system-context acquire do not leak into a subsequent tenant-context acquire
// on the same underlying connection.
func TestPrepareConn_AfterRelease_ClearsGUCs(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	// First acquire as system — sets app.role='system'.
	ctxSys := testutil.WithSystemCtx(context.Background())
	var sysRole string
	if err := pool.QueryRow(ctxSys, "SELECT current_setting('app.role', true)").Scan(&sysRole); err != nil {
		t.Fatalf("system acquire: %v", err)
	}
	if sysRole != "system" {
		t.Fatalf("expected app.role='system', got %q", sysRole)
	}

	// Second acquire as TenantA — app.role must be 'user', not 'system'.
	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	var tenantRole, tenantID string
	row := pool.QueryRow(ctxA,
		"SELECT current_setting('app.role', true), current_setting('app.tenant_id', true)")
	if err := row.Scan(&tenantRole, &tenantID); err != nil {
		t.Fatalf("tenant acquire: %v", err)
	}
	if tenantRole != "user" {
		t.Errorf("app.role after AfterRelease: got %q, want 'user' (system GUC leaked)", tenantRole)
	}
	if tenantID != testutil.TenantA.String() {
		t.Errorf("app.tenant_id: got %q, want %s", tenantID, testutil.TenantA)
	}
}
