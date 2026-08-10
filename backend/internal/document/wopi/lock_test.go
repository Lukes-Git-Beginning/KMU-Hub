package wopi

// LockService currently queries a table named "document_wopi_locks", but no
// migration ever created that table — migration 000044 created "wopi_locks",
// and every migration since (000114 tenant_id retrofit, 000122 RLS) refers to
// that same name. Every method below therefore fails against the real
// schema. These tests document that CURRENT (broken) behavior precisely so a
// fix (rename the six query targets in lock.go to "wopi_locks") has a
// red-then-green signal to work against — see
// fix-document-wopi-lock-table-name-mismatch in BACKLOG.yml. Do not delete
// these on a whim; replace them with real round-trip tests once the fix
// lands.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

const undefinedTable = "42P01"

type wopiLockFixture struct {
	tenant uuid.UUID
	user   uuid.UUID
	file   uuid.UUID
}

func seedWopiLockFixture(t *testing.T, pool *pgxpool.Pool, name string) wopiLockFixture {
	t.Helper()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "wopi lock test "+name)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"email":         "wopi-" + name + "-" + uuid.NewString() + "@test.local",
		"password_hash": "x", "first_name": "Wopi", "last_name": "Tester",
	})

	folderID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "name": "WOPI Lock Test Folder",
		"space_type": "personal", "space_id": uuid.New(), "created_by": userID,
	})

	fileID := testutil.SeedRow(t, pool, "document_files", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "folder_id": folderID,
		"filename": "wopi-test.docx", "mime_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"file_size": 2048, "storage_key": "documents/wopi-test/" + uuid.NewString(), "owner_id": userID,
	})

	return wopiLockFixture{tenant: tenantID, user: userID, file: fileID}
}

// seedRealWopiLock inserts directly into the ACTUAL table (wopi_locks) that
// migration 000044 created, bypassing LockService entirely — proving what a
// genuine lock row looks like so the "always reports unlocked" gap below is
// demonstrated against real data, not just an empty table.
func seedRealWopiLock(t *testing.T, pool *pgxpool.Pool, fx wopiLockFixture, lockID string, expiresAt time.Time) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(ctx,
		`INSERT INTO wopi_locks (file_id, lock_id, locked_by, tenant_id, expires_at) VALUES ($1, $2, $3, $4, $5)`,
		fx.file, lockID, fx.user, fx.tenant, expiresAt,
	)
	if err != nil {
		t.Fatalf("seed wopi_locks: %v", err)
	}
}

func assertUndefinedTableErr(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error because LockService queries a nonexistent table " +
			"\"document_wopi_locks\" (real table is \"wopi_locks\", migration 000044) — " +
			"if this now succeeds, the table-name bug has been fixed; replace this test " +
			"with a real round-trip (see fix-document-wopi-lock-table-name-mismatch in BACKLOG.yml)")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != undefinedTable {
		t.Fatalf("expected undefined_table (42P01) error, got: %v", err)
	}
}

func TestLockService_Lock_QueriesNonexistentTable_DocumentsCurrentGap(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	svc := NewLockService(pool)
	err := svc.Lock(context.Background(), uuid.NewString(), "lock-1", uuid.NewString())
	assertUndefinedTableErr(t, err)
}

func TestLockService_Unlock_QueriesNonexistentTable_DocumentsCurrentGap(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	svc := NewLockService(pool)
	err := svc.Unlock(context.Background(), uuid.NewString(), "lock-1")
	assertUndefinedTableErr(t, err)
}

func TestLockService_RefreshLock_QueriesNonexistentTable_DocumentsCurrentGap(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	svc := NewLockService(pool)
	err := svc.RefreshLock(context.Background(), uuid.NewString(), "lock-1")
	assertUndefinedTableErr(t, err)
}

func TestLockService_CleanExpired_QueriesNonexistentTable_DocumentsCurrentGap(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	svc := NewLockService(pool)
	fx := seedWopiLockFixture(t, pool, "clean-expired")
	seedRealWopiLock(t, pool, fx, "expired-lock", time.Now().Add(-time.Hour))

	n, err := svc.CleanExpired(context.Background())
	assertUndefinedTableErr(t, err)
	if n != 0 {
		t.Errorf("CleanExpired returned n=%d on error, want 0", n)
	}

	// The real, genuinely-expired row in wopi_locks was never touched — the
	// 5-minute cleanup goroutine in cmd/document/main.go has been failing on
	// every tick since this table was introduced, and expired locks
	// accumulate in wopi_locks forever.
	var count int
	ctx := testutil.WithSystemCtx(context.Background())
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM wopi_locks WHERE file_id = $1`, fx.file).Scan(&count); err != nil {
		t.Fatalf("count wopi_locks: %v", err)
	}
	if count != 1 {
		t.Errorf("wopi_locks row count = %d, want 1 (CleanExpired must not have deleted it)", count)
	}
}

// TestLockService_GetLock_AlwaysReportsUnlocked_DocumentsCurrentGap is the
// sharpest demonstration of the impact: GetLock swallows every query error
// (including "relation does not exist") into ErrLockNotFound, so it reports
// "no lock" even for a file carrying a real, active, non-expired lock row
// in wopi_locks — silently disabling every caller's lock-conflict check
// (see TestHandler_PutFile and TestHandler_HandleLockOperation in
// handler_test.go for the caller-level consequence).
func TestLockService_GetLock_AlwaysReportsUnlocked_DocumentsCurrentGap(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	svc := NewLockService(pool)
	fx := seedWopiLockFixture(t, pool, "get-lock")
	seedRealWopiLock(t, pool, fx, "active-lock", time.Now().Add(30*time.Minute))

	_, err := svc.GetLock(context.Background(), fx.file.String())
	if !errors.Is(err, ErrLockNotFound) {
		t.Fatalf("GetLock err = %v, want ErrLockNotFound (even though an active lock row exists in "+
			"wopi_locks) — if this now finds the lock, the table-name bug has been fixed; replace "+
			"this test with a real round-trip", err)
	}
}
