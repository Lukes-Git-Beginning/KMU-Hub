//go:build integration

package pgtc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/middleware"
)

func init() {
	// Disable the Ryuk reaper — it can fail in Windows/Docker Desktop environments.
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
}

// StartPostgres starts a fresh pgvector/pgvector:pg16 container, runs all
// *.up.sql migrations as superuser, grants kmuhub_app a test password, and
// returns:
//   - appPool: *pgxpool.Pool connected as kmuhub_app (NOSUPERUSER NOBYPASSRLS).
//     The PrepareConn hook stamps the tenant GUC, so RLS is enforced exactly
//     as in production.
//   - superURL: DSN for a raw superuser connection (for direct-query helpers
//     that need to bypass RLS in assertions).
//
// t.Cleanup closes pools and terminates the container.
func StartPostgres(t *testing.T) (appPool *pgxpool.Pool, superURL string) {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx,
		"pgvector/pgvector:pg16",
		tcpostgres.WithDatabase("kmuhub"),
		tcpostgres.WithUsername("kmuhub"),
		tcpostgres.WithPassword("verify"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	rawURL, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get container connection string: %v", err)
	}
	superURL = rawURL

	// Open a superuser pool to run migrations and setup.
	superPool, err := pgxpool.New(ctx, superURL)
	if err != nil {
		t.Fatalf("open superuser pool: %v", err)
	}

	// Run all *.up.sql migrations in lexical (= numerical) order.
	migrationsDir := resolveMigrationsDir(t)
	runMigrations(t, ctx, superPool, migrationsDir)

	// Grant kmuhub_app a deterministic test password so the app pool can connect.
	if _, err := superPool.Exec(ctx, "ALTER ROLE kmuhub_app WITH PASSWORD 'apptest'"); err != nil {
		t.Fatalf("set kmuhub_app password: %v", err)
	}

	// Build the app pool URL (kmuhub_app, NOSUPERUSER NOBYPASSRLS → RLS enforced).
	host, err := ctr.Host(ctx)
	if err != nil {
		t.Fatalf("get container host: %v", err)
	}
	port, err := ctr.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("get container port: %v", err)
	}
	appURL := fmt.Sprintf(
		"postgres://kmuhub_app:apptest@%s:%s/kmuhub?sslmode=disable",
		host, port.Port(),
	)

	// database.NewPostgresPool installs the PrepareConn hook that stamps
	// app.tenant_id from the context — mandatory for RLS to work.
	appPool, err = database.NewPostgresPool(ctx, appURL)
	if err != nil {
		t.Fatalf("open app pool (kmuhub_app): %v", err)
	}

	t.Cleanup(func() {
		appPool.Close()
		superPool.Close()
		if tErr := ctr.Terminate(context.Background()); tErr != nil {
			t.Logf("warn: terminate container: %v", tErr)
		}
	})

	return appPool, superURL
}

// TenantCtx returns a child context with tenantID stamped as middleware.TenantIDKey.
// The PrepareConn hook in NewPostgresPool reads this value to set app.tenant_id.
func TenantCtx(parent context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(parent, middleware.TenantIDKey, tenantID.String())
}

// SuperPool opens a raw pgxpool connection as the superuser (kmuhub / verify).
// Used for direct-query assertions that need to bypass RLS (e.g. counting rows
// in finance_invoice_lines without a tenant filter).
// Caller must close the returned pool — typically via t.Cleanup.
func SuperPool(t *testing.T, superURL string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), superURL)
	if err != nil {
		t.Fatalf("open super pool: %v", err)
	}
	return pool
}

// resolveMigrationsDir resolves the path to the repo-root migrations/ directory.
// It walks up from this file's location until it finds the go.mod.
func resolveMigrationsDir(t *testing.T) string {
	t.Helper()

	// ENV override — useful in CI or unusual directory layouts.
	if dir := os.Getenv("MIGRATIONS_DIR"); dir != "" {
		return dir
	}

	// Walk up from this file to the repo root (directory containing go.mod).
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed — cannot resolve migrations dir")
	}

	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "migrations")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find go.mod walking up from %s", filepath.Dir(thisFile))
		}
		dir = parent
	}
}

// runMigrations reads all *.up.sql files from dir, sorts them lexically
// (which equals numerical order given zero-padded names), and executes each
// one as a single Exec call against superPool.
// pgx's simple-query protocol handles multi-statement scripts (including
// BEGIN/COMMIT and $$ dollar-quoted blocks) correctly.
func runMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations dir %q: %v", dir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files) // lexical = numerical for zero-padded names

	for _, name := range files {
		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			t.Fatalf("execute migration %s: %v", name, err)
		}
	}
	t.Logf("applied %d migrations from %s", len(files), dir)
}
