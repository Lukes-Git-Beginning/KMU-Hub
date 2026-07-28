package caldav

// Exercises the real write path (PostgresAppPasswordRepository,
// PushSubscriptionService, SyncTokenService — not testutil.SeedRow) so a
// forgotten tenant_id column would show up as a genuine INSERT/constraint
// failure instead of being silently bypassed by a system-context fixture.
// Complements tenant_isolation_phase2_test.go, which only proves RLS SELECT
// filtering on hand-seeded rows.
//
// Also covers two real bugs found and fixed in this iteration:
//   - sync_token.go's INSERTs into caldav_sync_versions / caldav_change_log
//     never set tenant_id, which is NOT NULL (migration 000115) — every call
//     failed against real Postgres. Fixed by threading tenantID through
//     GetSyncToken/IncrementAndLog/GetChangesSince, resolved from the
//     Basic-Auth-authenticated userID via resolveTenantID (sysctx bypass,
//     the same sanctioned "identity not yet known" case as a login flow).
//   - FindActiveByUser/UpdateLastUsed (the Basic Auth credential check) ran
//     with no tenant context at all (CalDAV/CardDAV authenticate via an
//     app-specific password, never a JWT) — without a system-context bypass,
//     RLS's FORCE ROW LEVEL SECURITY blocked every row, so CalDAV Basic Auth
//     could never succeed against real Postgres. Fixed by wrapping both in
//     sysctx.With.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestCalDAVWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantWrite, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantWrite, "CalDAV Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CalDAV Write Other Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantWrite,
		"email":         fmt.Sprintf("caldav-write-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	ctxA := testutil.WithTenantCtx(context.Background(), tenantWrite)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantOther)

	appPwRepo := NewPostgresAppPasswordRepository(pool)
	pw := &models.AppSpecificPassword{
		ID:             uuid.New(),
		TenantID:       tenantWrite,
		UserID:         userID,
		Label:          "MacOS Calendar",
		PasswordHash:   "$2a$12$test",
		PasswordPrefix: "abcd",
		Scope:          "caldav_carddav",
	}
	if err := appPwRepo.Create(ctxA, pw); err != nil {
		t.Fatalf("AppPasswordRepository.Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "app_specific_passwords", pw.ID)

	pushSvc := NewPushSubscriptionService(pool)
	sub, err := pushSvc.Subscribe(ctxA, userID, tenantWrite, "calendar", uuid.New(), "https://push.example.com/"+uuid.New().String(), "", 0)
	if err != nil {
		t.Fatalf("PushSubscriptionService.Subscribe: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "caldav_push_subscriptions", sub.ID)

	rows := []struct {
		table string
		id    uuid.UUID
	}{
		{"app_specific_passwords", pw.ID},
		{"caldav_push_subscriptions", sub.ID},
	}

	for _, row := range rows {
		t.Run(row.table, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, row.table, row.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, row.table, row.id, 0)
		})
	}

	t.Run("SyncTokenService writes carry tenant_id", func(t *testing.T) {
		collID := uuid.New()
		syncSvc := NewSyncTokenService(pool)

		if err := syncSvc.IncrementAndLog(ctxA, tenantWrite, "calendar", collID, "/caldav/x.ics", "modified"); err != nil {
			t.Fatalf("IncrementAndLog: %v", err)
		}
		defer func() {
			sysCtx := testutil.WithSystemCtx(context.Background())
			_, _ = pool.Exec(sysCtx, `DELETE FROM caldav_change_log WHERE collection_id = $1`, collID)
			_, _ = pool.Exec(sysCtx, `DELETE FROM caldav_sync_versions WHERE collection_id = $1`, collID)
		}()

		var countA, countB int
		if err := pool.QueryRow(ctxA,
			`SELECT count(*) FROM caldav_sync_versions WHERE collection_id = $1`, collID,
		).Scan(&countA); err != nil {
			t.Fatalf("count caldav_sync_versions (ctxA): %v", err)
		}
		if err := pool.QueryRow(ctxB,
			`SELECT count(*) FROM caldav_sync_versions WHERE collection_id = $1`, collID,
		).Scan(&countB); err != nil {
			t.Fatalf("count caldav_sync_versions (ctxB): %v", err)
		}
		if countA != 1 {
			t.Errorf("ctxA: expected 1 row in caldav_sync_versions, got %d", countA)
		}
		if countB != 0 {
			t.Errorf("ctxB: expected 0 rows in caldav_sync_versions (tenant isolation), got %d", countB)
		}

		token, err := syncSvc.GetSyncToken(ctxA, tenantWrite, "calendar", collID)
		if err != nil {
			t.Fatalf("GetSyncToken: %v", err)
		}
		if token == "" {
			t.Fatal("expected non-empty sync token")
		}

		entries, _, err := syncSvc.GetChangesSince(ctxA, tenantWrite, "calendar", collID, 0)
		if err != nil {
			t.Fatalf("GetChangesSince: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("expected 1 change log entry, got %d", len(entries))
		}
	})

	t.Run("resolveTenantID finds the owning tenant via sysctx bypass", func(t *testing.T) {
		got, err := resolveTenantID(context.Background(), pool, userID)
		if err != nil {
			t.Fatalf("resolveTenantID: %v", err)
		}
		if got != tenantWrite {
			t.Fatalf("resolveTenantID: expected %s, got %s", tenantWrite, got)
		}
	})

	t.Run("FindActiveByUser and UpdateLastUsed work with no tenant in ctx", func(t *testing.T) {
		// This is the exact situation of an incoming CalDAV Basic Auth
		// request: no JWT, no tenant, only a claimed username/password.
		// Before the sysctx fix, RLS's FORCE ROW LEVEL SECURITY silently
		// returned zero rows here and Basic Auth could never succeed.
		bare := context.Background()

		active, err := appPwRepo.FindActiveByUser(bare, userID)
		if err != nil {
			t.Fatalf("FindActiveByUser (no tenant ctx): %v", err)
		}
		if len(active) != 1 {
			t.Fatalf("expected 1 active password with no tenant ctx, got %d", len(active))
		}

		if err := appPwRepo.UpdateLastUsed(bare, pw.ID); err != nil {
			t.Fatalf("UpdateLastUsed (no tenant ctx): %v", err)
		}

		sysCtx := testutil.WithSystemCtx(context.Background())
		var lastUsed *time.Time
		if err := pool.QueryRow(sysCtx,
			`SELECT last_used_at FROM app_specific_passwords WHERE id = $1`, pw.ID,
		).Scan(&lastUsed); err != nil {
			t.Fatalf("read back last_used_at: %v", err)
		}
		if lastUsed == nil {
			t.Fatal("expected last_used_at to be set after UpdateLastUsed")
		}
	})
}
