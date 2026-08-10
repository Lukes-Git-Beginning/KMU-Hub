package wopi

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

type wopiLockFixture struct {
	tenant uuid.UUID
	user   uuid.UUID
	file   uuid.UUID
}

// tenantCtx stamps ctx with the fixture's tenant, exactly like withTenantCtx
// does from a real WOPI request — required for the RLS-aware pool to admit
// wopi_locks rows for this tenant.
func (fx wopiLockFixture) tenantCtx() context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, fx.tenant.String())
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

// seedRealWopiLock inserts directly into wopi_locks, bypassing LockService —
// used to set up pre-existing lock state for conflict/expiry scenarios.
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

func TestLockService_Lock_AcquiresAndRefreshesAndConflicts(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	svc := NewLockService(pool)
	fx := seedWopiLockFixture(t, pool, "lock-acquire")
	ctx := fx.tenantCtx()

	if err := svc.Lock(ctx, fx.file.String(), "lock-a", fx.user.String(), fx.tenant); err != nil {
		t.Fatalf("Lock (acquire): %v", err)
	}

	// Same lockID again -- refresh, not a conflict.
	if err := svc.Lock(ctx, fx.file.String(), "lock-a", fx.user.String(), fx.tenant); err != nil {
		t.Fatalf("Lock (refresh, same lockID): %v", err)
	}

	// Different lockID while "lock-a" is active and unexpired -- conflict.
	err := svc.Lock(ctx, fx.file.String(), "lock-b", fx.user.String(), fx.tenant)
	var conflictErr *LockConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("Lock (conflict) err = %v, want *LockConflictError", err)
	}
	if conflictErr.CurrentLockID != "lock-a" {
		t.Errorf("conflictErr.CurrentLockID = %q, want lock-a", conflictErr.CurrentLockID)
	}
}

func TestLockService_Lock_TakesOverExpiredLock(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	svc := NewLockService(pool)
	fx := seedWopiLockFixture(t, pool, "lock-expired-takeover")
	seedRealWopiLock(t, pool, fx, "stale-lock", time.Now().Add(-time.Hour))

	ctx := fx.tenantCtx()
	if err := svc.Lock(ctx, fx.file.String(), "fresh-lock", fx.user.String(), fx.tenant); err != nil {
		t.Fatalf("Lock over expired lock: %v", err)
	}

	got, err := svc.GetLock(ctx, fx.file.String(), fx.tenant)
	if err != nil {
		t.Fatalf("GetLock after takeover: %v", err)
	}
	if got != "fresh-lock" {
		t.Errorf("GetLock = %q, want fresh-lock", got)
	}
}

func TestLockService_Unlock(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	svc := NewLockService(pool)
	fx := seedWopiLockFixture(t, pool, "unlock")
	ctx := fx.tenantCtx()

	if err := svc.Lock(ctx, fx.file.String(), "lock-a", fx.user.String(), fx.tenant); err != nil {
		t.Fatalf("Lock: %v", err)
	}

	t.Run("wrong lockID does not unlock", func(t *testing.T) {
		err := svc.Unlock(ctx, fx.file.String(), "wrong-lock", fx.tenant)
		if !errors.Is(err, ErrLockNotFound) {
			t.Fatalf("Unlock (wrong lockID) err = %v, want ErrLockNotFound", err)
		}
		got, getErr := svc.GetLock(ctx, fx.file.String(), fx.tenant)
		if getErr != nil || got != "lock-a" {
			t.Fatalf("lock should still be held: got=%q err=%v", got, getErr)
		}
	})

	t.Run("correct lockID unlocks", func(t *testing.T) {
		if err := svc.Unlock(ctx, fx.file.String(), "lock-a", fx.tenant); err != nil {
			t.Fatalf("Unlock: %v", err)
		}
		_, err := svc.GetLock(ctx, fx.file.String(), fx.tenant)
		if !errors.Is(err, ErrLockNotFound) {
			t.Fatalf("GetLock after unlock err = %v, want ErrLockNotFound", err)
		}
	})
}

func TestLockService_RefreshLock(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	svc := NewLockService(pool)
	fx := seedWopiLockFixture(t, pool, "refresh")
	seedRealWopiLock(t, pool, fx, "lock-a", time.Now().Add(time.Minute))
	ctx := fx.tenantCtx()

	t.Run("wrong lockID fails", func(t *testing.T) {
		err := svc.RefreshLock(ctx, fx.file.String(), "wrong-lock", fx.tenant)
		if !errors.Is(err, ErrLockNotFound) {
			t.Fatalf("RefreshLock (wrong lockID) err = %v, want ErrLockNotFound", err)
		}
	})

	t.Run("correct lockID extends expiry past the original near-term value", func(t *testing.T) {
		if err := svc.RefreshLock(ctx, fx.file.String(), "lock-a", fx.tenant); err != nil {
			t.Fatalf("RefreshLock: %v", err)
		}
		var expiresAt time.Time
		sysCtx := testutil.WithSystemCtx(context.Background())
		if err := pool.QueryRow(sysCtx, `SELECT expires_at FROM wopi_locks WHERE file_id = $1`, fx.file).Scan(&expiresAt); err != nil {
			t.Fatalf("query expires_at: %v", err)
		}
		if !expiresAt.After(time.Now().Add(20 * time.Minute)) {
			t.Errorf("expires_at = %v, want > 20 minutes from now (RefreshLock should extend by 30m)", expiresAt)
		}
	})
}

func TestLockService_GetLock(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	svc := NewLockService(pool)
	fx := seedWopiLockFixture(t, pool, "get-lock")
	ctx := fx.tenantCtx()

	t.Run("no lock", func(t *testing.T) {
		_, err := svc.GetLock(ctx, fx.file.String(), fx.tenant)
		if !errors.Is(err, ErrLockNotFound) {
			t.Fatalf("GetLock err = %v, want ErrLockNotFound", err)
		}
	})

	t.Run("active lock", func(t *testing.T) {
		seedRealWopiLock(t, pool, fx, "active-lock", time.Now().Add(30*time.Minute))
		got, err := svc.GetLock(ctx, fx.file.String(), fx.tenant)
		if err != nil {
			t.Fatalf("GetLock: %v", err)
		}
		if got != "active-lock" {
			t.Errorf("GetLock = %q, want active-lock", got)
		}
	})
}

func TestLockService_CleanExpired(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	svc := NewLockService(pool)
	expiredFx := seedWopiLockFixture(t, pool, "clean-expired")
	activeFx := seedWopiLockFixture(t, pool, "clean-active")
	seedRealWopiLock(t, pool, expiredFx, "expired-lock", time.Now().Add(-time.Hour))
	seedRealWopiLock(t, pool, activeFx, "active-lock", time.Now().Add(30*time.Minute))

	// CleanExpired is a cross-tenant maintenance job: it must run under a
	// system context (see cmd/document/main.go's cleanup goroutine), not a
	// single tenant's context, or RLS admits zero rows and nothing is cleaned.
	sysCtx := testutil.WithSystemCtx(context.Background())
	n, err := svc.CleanExpired(sysCtx)
	if err != nil {
		t.Fatalf("CleanExpired: %v", err)
	}
	if n < 1 {
		t.Fatalf("CleanExpired returned n=%d, want >= 1 (the expired row)", n)
	}

	if _, err := svc.GetLock(expiredFx.tenantCtx(), expiredFx.file.String(), expiredFx.tenant); !errors.Is(err, ErrLockNotFound) {
		t.Errorf("expired lock should be gone, GetLock err = %v", err)
	}
	got, err := svc.GetLock(activeFx.tenantCtx(), activeFx.file.String(), activeFx.tenant)
	if err != nil || got != "active-lock" {
		t.Errorf("active lock must survive CleanExpired: got=%q err=%v", got, err)
	}
}
