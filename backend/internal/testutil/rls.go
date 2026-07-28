// Package testutil provides shared helpers for cross-tenant RLS verification
// against the local Compose Postgres. Tests skip cleanly when DATABASE_URL is
// unset (CI without DB) and use sysctx-marked contexts for fixture setup so
// the seed inserts bypass the very policies the tests are exercising.
package testutil

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

// SkipIfNoDB skips the test if DATABASE_URL is unset. Used to keep RLS
// integration tests off CI runners without Postgres.
func SkipIfNoDB(t *testing.T) {
	t.Helper()
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set — skipping RLS integration test")
	}
}

// PoolFromEnv connects to DATABASE_URL via the production pool constructor
// so the PrepareConn hook is active. Tests must Close the pool.
func PoolFromEnv(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL not set — call SkipIfNoDB before PoolFromEnv")
	}
	pool, err := database.NewPostgresPool(context.Background(), url)
	if err != nil {
		t.Fatalf("connect to %s: %v", url, err)
	}
	return pool
}

// WithTenantCtx returns a context whose middleware.TenantIDKey value is the
// given tenantID — exactly what JWTAuth or TenantInboundUnaryInterceptor would
// inject in production. Combined with the PrepareConn pool hook, the next
// pool.Acquire stamps app.tenant_id on the connection.
func WithTenantCtx(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())
}

// WithSystemCtx flags the context as a system operation. The pool hook then
// sets app.role='system' and is_system_context() admits every row.
func WithSystemCtx(ctx context.Context) context.Context {
	return sysctx.With(ctx)
}

// SeedRow inserts a row as system-context and returns its id.
// `cols` must include either an "id" key (caller-provided UUID) or rely on
// the table's DEFAULT to generate one — we call RETURNING id either way.
// Used to pre-populate fixtures for cross-tenant tests; using SystemCtx for
// the seed lets us insert rows for arbitrary tenants without RLS rejection.
func SeedRow(t *testing.T, pool *pgxpool.Pool, table string, cols map[string]any) uuid.UUID {
	t.Helper()
	if len(cols) == 0 {
		t.Fatalf("SeedRow: cols is empty for table %s", table)
	}

	colNames := make([]string, 0, len(cols))
	placeholders := make([]string, 0, len(cols))
	values := make([]any, 0, len(cols))
	idx := 1
	for k, v := range cols {
		colNames = append(colNames, k)
		placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
		values = append(values, v)
		idx++
	}

	query := fmt.Sprintf(
		`INSERT INTO %s (%s) VALUES (%s) RETURNING id`,
		table,
		strings.Join(colNames, ", "),
		strings.Join(placeholders, ", "),
	)

	ctx := WithSystemCtx(context.Background())
	var id uuid.UUID
	if err := pool.QueryRow(ctx, query, values...).Scan(&id); err != nil {
		t.Fatalf("seed %s: %v", table, err)
	}
	return id
}

// CleanupRow deletes a row by id under system-context.
// Safe to call with uuid.Nil — no-op.
func CleanupRow(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID) {
	t.Helper()
	if id == uuid.Nil {
		return
	}
	ctx := WithSystemCtx(context.Background())
	if _, err := pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE id = $1", table), id); err != nil {
		t.Logf("cleanup %s id=%s: %v", table, id, err)
	}
}

// AssertRowCount runs `SELECT count(*) FROM <table> WHERE id = $1` under the
// given ctx (which determines RLS visibility) and fails if the count differs.
func AssertRowCount(t *testing.T, pool *pgxpool.Pool, ctx context.Context, table string, id uuid.UUID, expected int) {
	t.Helper()
	var count int
	query := fmt.Sprintf("SELECT count(*) FROM %s WHERE id = $1", table)
	if err := pool.QueryRow(ctx, query, id).Scan(&count); err != nil {
		t.Fatalf("count %s id=%s: %v", table, id, err)
	}
	if count != expected {
		t.Fatalf("table %s id=%s: expected %d row(s), got %d", table, id, expected, count)
	}
}

// AssertUpdateRowsAffected runs an UPDATE under ctx and asserts the affected
// row count. Used to verify a cross-tenant UPDATE matches zero rows.
func AssertUpdateRowsAffected(t *testing.T, pool *pgxpool.Pool, ctx context.Context, query string, expected int64, args ...any) {
	t.Helper()
	tag, err := pool.Exec(ctx, query, args...)
	if err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
	if got := tag.RowsAffected(); got != expected {
		t.Fatalf("query %q: expected %d affected rows, got %d", query, expected, got)
	}
}

// TenantA / TenantB are stable UUIDs reserved for cross-tenant test fixtures.
// They never collide with the real DefaultTenantID
// (00000000-0000-0000-0000-000000000001).
var (
	TenantA = uuid.MustParse("aaaa0000-0000-0000-0000-000000000001")
	TenantB = uuid.MustParse("bbbb0000-0000-0000-0000-000000000002")
)

// AssertWriteCarriesTenant bundles the three-step check that wp-* module
// tests across this backlog have hand-rolled since iteration 13: perform a
// real repo write under a freshly minted tenant's context, then confirm the
// resulting row is visible reading back as that tenant and invisible reading
// back as a different one. It exercises the actual repo method rather than
// SeedRow, which is the only way to catch a write that silently omits or
// hardcodes tenant_id — the class of bug closed across caldav, formulare,
// automation and others in this backlog (SeedRow-based RLS tests never touch
// the real INSERT/UPDATE path, so they cannot see it).
//
// write receives a context scoped to the freshly minted tenant and that
// tenant's ID — set it on the object being persisted before calling the real
// repo method, the same way existing tenant_write_test.go files do by hand.
// table/id identify the row to check afterwards.
func AssertWriteCarriesTenant(t *testing.T, pool *pgxpool.Pool, write func(ctx context.Context, tenantID uuid.UUID) error, table string, id uuid.UUID) {
	t.Helper()

	tenantWrite, tenantOther := uuid.New(), uuid.New()
	EnsureTenant(t, pool, tenantWrite, "AssertWriteCarriesTenant Write Tenant")
	EnsureTenant(t, pool, tenantOther, "AssertWriteCarriesTenant Other Tenant")

	ctxWrite := WithTenantCtx(context.Background(), tenantWrite)
	if err := write(ctxWrite, tenantWrite); err != nil {
		t.Fatalf("write: %v", err)
	}

	AssertRowCount(t, pool, ctxWrite, table, id, 1)
	AssertRowCount(t, pool, WithTenantCtx(context.Background(), tenantOther), table, id, 0)
}

// EnsureTenant inserts a tenants row for the given id (idempotent — uses
// ON CONFLICT). Required because RLS on `users` and other Pilot-0 tables
// enforces FK to tenants(id), so seed rows for TenantA/TenantB need the
// parent row to exist.
func EnsureTenant(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, name string) {
	t.Helper()
	ctx := WithSystemCtx(context.Background())
	_, err := pool.Exec(ctx,
		`INSERT INTO tenants (id, name) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING`,
		id, name,
	)
	if err != nil {
		t.Fatalf("ensure tenant %s: %v", id, err)
	}
}
