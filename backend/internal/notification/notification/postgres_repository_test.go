package notification_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/notification/notification"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// fixture mints a fresh tenant + user per test so seeds never collide under
// t.Parallel() (same reasoning as internal/chat/bookmark's tests).
type fixture struct {
	tenant uuid.UUID
	user   uuid.UUID
}

func seedFixture(t *testing.T, pool *pgxpool.Pool, name string) fixture {
	t.Helper()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "notification test "+name)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"email":         "notif-" + name + "-" + uuid.NewString() + "@test.local",
		"password_hash": "x", "first_name": "Notif", "last_name": "Test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	return fixture{tenant: tenantID, user: userID}
}

func seedNotification(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID, isRead bool) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "notifications", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "user_id": userID,
		"event_type_key": "test.event", "module_id": "test", "title": "Test notification",
		"is_read": isRead,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "notifications", id) })
	return id
}

func TestRepository_SnoozeSetsUntilAndMarksRead(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	// Registered before any fixture cleanup below, so LIFO runs this last —
	// a `defer pool.Close()` here would close the pool before t.Cleanup rows
	// are deleted (real, pre-existing footgun documented in JOURNAL.md
	// iteration 36).
	t.Cleanup(func() { pool.Close() })

	repo := notification.NewPostgresRepository(pool)
	fx := seedFixture(t, pool, "snooze")
	notifID := seedNotification(t, pool, fx.tenant, fx.user, false)
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	until := time.Now().Add(time.Hour).UTC()
	if err := repo.Snooze(ctx, fx.tenant, notifID, until); err != nil {
		t.Fatalf("Snooze: %v", err)
	}

	got, err := repo.GetByID(ctx, fx.tenant, notifID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	// Postgres timestamptz stores microsecond precision, so compare within a
	// tolerance rather than exact equality (Go's time.Time carries nanoseconds).
	if got.SnoozedUntil == nil || got.SnoozedUntil.Sub(until).Abs() > time.Millisecond {
		t.Fatalf("SnoozedUntil = %v, want %v", got.SnoozedUntil, until)
	}
	if !got.IsRead {
		t.Fatal("IsRead = false after Snooze, want true")
	}
}

func TestRepository_SnoozeUnknownIDReturnsNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	// Registered before any fixture cleanup below, so LIFO runs this last —
	// a `defer pool.Close()` here would close the pool before t.Cleanup rows
	// are deleted (real, pre-existing footgun documented in JOURNAL.md
	// iteration 36).
	t.Cleanup(func() { pool.Close() })

	repo := notification.NewPostgresRepository(pool)
	fx := seedFixture(t, pool, "snooze-404")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	err := repo.Snooze(ctx, fx.tenant, uuid.New(), time.Now().Add(time.Hour))
	if err != notification.ErrNotificationNotFound {
		t.Fatalf("Snooze unknown id error = %v, want ErrNotificationNotFound", err)
	}
}

// TestRepository_ListAndUnreadCountExcludeCurrentlySnoozed proves the List and
// GetUnreadCount filter added in this unit: a notification whose snooze
// window is still open must be invisible to both queries, while one whose
// snooze already elapsed must reappear in List (it stays "read" though —
// Snooze marks is_read=true and there is no un-snooze worker, matching the
// notifications MSW mock's semantics).
func TestRepository_ListAndUnreadCountExcludeCurrentlySnoozed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	// Registered before any fixture cleanup below, so LIFO runs this last —
	// a `defer pool.Close()` here would close the pool before t.Cleanup rows
	// are deleted (real, pre-existing footgun documented in JOURNAL.md
	// iteration 36).
	t.Cleanup(func() { pool.Close() })

	repo := notification.NewPostgresRepository(pool)
	fx := seedFixture(t, pool, "list-filter")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	visible := seedNotification(t, pool, fx.tenant, fx.user, false)
	snoozed := seedNotification(t, pool, fx.tenant, fx.user, false)
	expired := seedNotification(t, pool, fx.tenant, fx.user, false)

	if err := repo.Snooze(ctx, fx.tenant, snoozed, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("Snooze (future): %v", err)
	}
	if err := repo.Snooze(ctx, fx.tenant, expired, time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("Snooze (already elapsed): %v", err)
	}

	notifs, total, err := repo.List(ctx, notification.ListFilter{TenantID: fx.tenant, UserID: fx.user}, 0, 50)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	seen := make(map[uuid.UUID]bool, len(notifs))
	for _, n := range notifs {
		seen[n.ID] = true
	}
	if !seen[visible] {
		t.Errorf("List missing never-snoozed notification %s", visible)
	}
	if seen[snoozed] {
		t.Errorf("List leaked currently-snoozed notification %s", snoozed)
	}
	if !seen[expired] {
		t.Errorf("List missing notification %s whose snooze already elapsed", expired)
	}
	if total != 2 {
		t.Errorf("List total = %d, want 2 (visible + expired, currently-snoozed excluded)", total)
	}

	count, err := repo.GetUnreadCount(ctx, fx.tenant, fx.user)
	if err != nil {
		t.Fatalf("GetUnreadCount: %v", err)
	}
	if count != 1 {
		t.Errorf("GetUnreadCount = %d, want 1 (only the never-snoozed, still-unread item)", count)
	}
}
