package notification_test

// Repository coverage against the app-role database (kmuhub_app, RLS in force).
//
// Two kinds of context are used on purpose. The tenant context is what production
// runs under, but RLS then hides foreign rows on its own, so a dropped
// `tenant_id = $n` predicate would go unnoticed. The cross-tenant assertions
// therefore run under the system context (RLS wide open) with a foreign tenant
// as argument: only the explicit predicate keeps the row out of reach there.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/notification"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func newRepoWithPool(t *testing.T) (*notification.PostgresRepository, *pgxpool.Pool) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	// Registered first so LIFO closes the pool after the fixture cleanups.
	t.Cleanup(func() { pool.Close() })
	return notification.NewPostgresRepository(pool), pool
}

func strPtr(s string) *string { return &s }

func seedNotificationWith(t *testing.T, pool *pgxpool.Pool, fx fixture, cols map[string]any) uuid.UUID {
	t.Helper()
	row := map[string]any{
		"id": uuid.New(), "tenant_id": fx.tenant, "user_id": fx.user,
		"event_type_key": "test.event", "module_id": "test", "title": "Seeded",
	}
	for k, v := range cols {
		row[k] = v
	}
	id := testutil.SeedRow(t, pool, "notifications", row)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "notifications", id) })
	return id
}

func TestRepository_CreateRoundTripsAllFields(t *testing.T) {
	repo, pool := newRepoWithPool(t)
	fx := seedFixture(t, pool, "create")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	actor := uuid.New()
	n := &models.Notification{
		ID: uuid.New(), TenantID: fx.tenant, UserID: fx.user,
		EventTypeKey: "chat.mention", ModuleID: "chat", Priority: models.PriorityUrgent,
		ActorID: &actor, ActorName: strPtr("Ada"), ResourceID: strPtr("msg-1"),
		Title: "Hello", Body: strPtr("Body"), DeepLink: strPtr("/chat/1"),
		GroupKey: strPtr("g1"), GroupCount: 3, IsRead: false, IsPinned: true, IsDismissed: true,
		DeliveredDesktop: true, CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	if err := repo.Create(ctx, n); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "notifications", n.ID) })

	got, err := repo.GetByID(ctx, fx.tenant, n.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.UserID != fx.user || got.EventTypeKey != "chat.mention" || got.ModuleID != "chat" ||
		got.Priority != models.PriorityUrgent || got.Title != "Hello" || got.GroupCount != 3 {
		t.Errorf("scalar fields did not round-trip: %+v", got)
	}
	if got.ActorID == nil || *got.ActorID != actor || got.ActorName == nil || *got.ActorName != "Ada" {
		t.Errorf("actor fields did not round-trip: %v / %v", got.ActorID, got.ActorName)
	}
	if got.ResourceID == nil || *got.ResourceID != "msg-1" || got.Body == nil || *got.Body != "Body" ||
		got.DeepLink == nil || *got.DeepLink != "/chat/1" || got.GroupKey == nil || *got.GroupKey != "g1" {
		t.Errorf("optional text fields did not round-trip: %+v", got)
	}
	if !got.IsPinned || !got.IsDismissed || !got.DeliveredDesktop || got.IsRead {
		t.Errorf("flags did not round-trip: pinned=%v dismissed=%v desktop=%v read=%v",
			got.IsPinned, got.IsDismissed, got.DeliveredDesktop, got.IsRead)
	}
	if !got.CreatedAt.Equal(n.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, n.CreatedAt)
	}
}

func TestRepository_CreateWithNilTenantUsesSentinel(t *testing.T) {
	repo, pool := newRepoWithPool(t)
	fx := seedFixture(t, pool, "create-sentinel")
	// Legacy/system-generated notifications carry no tenant; the sentinel tenant
	// is only writable under the system context.
	ctx := testutil.WithSystemCtx(context.Background())

	n := &models.Notification{
		ID: uuid.New(), UserID: fx.user, EventTypeKey: "system.alert", ModuleID: "system",
		Priority: models.PriorityNormal, Title: "Legacy", GroupCount: 1, CreatedAt: time.Now(),
	}
	if err := repo.Create(ctx, n); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "notifications", n.ID) })

	sentinel := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	if _, err := repo.GetByID(ctx, sentinel, n.ID); err != nil {
		t.Fatalf("row not stored under the sentinel tenant: %v", err)
	}
	if _, err := repo.GetByID(ctx, fx.tenant, n.ID); err != notification.ErrNotificationNotFound {
		t.Fatalf("GetByID under the user's tenant = %v, want ErrNotificationNotFound", err)
	}
}

func TestRepository_CreateRejectsForeignTenantUnderTenantContext(t *testing.T) {
	repo, pool := newRepoWithPool(t)
	fx := seedFixture(t, pool, "create-rls")
	other := seedFixture(t, pool, "create-rls-other")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	n := &models.Notification{
		ID: uuid.New(), TenantID: other.tenant, UserID: other.user, EventTypeKey: "x.y", ModuleID: "x",
		Priority: models.PriorityNormal, Title: "Smuggled", GroupCount: 1, CreatedAt: time.Now(),
	}
	err := repo.Create(ctx, n)
	assertRLSRejected(t, err, "Create stamped with a foreign tenant")
}

func TestRepository_GetByIDIsTenantScoped(t *testing.T) {
	repo, pool := newRepoWithPool(t)
	fx := seedFixture(t, pool, "get")
	other := seedFixture(t, pool, "get-other")
	id := seedNotification(t, pool, fx.tenant, fx.user, false)
	sys := testutil.WithSystemCtx(context.Background())

	if _, err := repo.GetByID(sys, other.tenant, id); err != notification.ErrNotificationNotFound {
		t.Fatalf("GetByID with foreign tenant = %v, want ErrNotificationNotFound", err)
	}
	if _, err := repo.GetByID(sys, fx.tenant, uuid.New()); err != notification.ErrNotificationNotFound {
		t.Fatalf("GetByID unknown id = %v, want ErrNotificationNotFound", err)
	}
	if got, err := repo.GetByID(sys, fx.tenant, id); err != nil || got.ID != id {
		t.Fatalf("GetByID own tenant = %v, %v", got, err)
	}
}

func TestRepository_ListFiltersOrderingAndPaging(t *testing.T) {
	repo, pool := newRepoWithPool(t)
	fx := seedFixture(t, pool, "list")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)
	now := time.Now().UTC()

	oldest := seedNotificationWith(t, pool, fx, map[string]any{"module_id": "chat", "created_at": now.Add(-3 * time.Hour)})
	middle := seedNotificationWith(t, pool, fx, map[string]any{"module_id": "crm", "is_read": true, "created_at": now.Add(-2 * time.Hour)})
	newest := seedNotificationWith(t, pool, fx, map[string]any{"module_id": "chat", "created_at": now.Add(-1 * time.Hour)})
	pinned := seedNotificationWith(t, pool, fx, map[string]any{"module_id": "crm", "is_pinned": true, "created_at": now.Add(-5 * time.Hour)})
	dismissed := seedNotificationWith(t, pool, fx, map[string]any{"module_id": "chat", "is_dismissed": true, "created_at": now.Add(-4 * time.Hour)})

	base := notification.ListFilter{TenantID: fx.tenant, UserID: fx.user}
	ids := func(ns []*models.Notification) []uuid.UUID {
		out := make([]uuid.UUID, len(ns))
		for i, n := range ns {
			out[i] = n.ID
		}
		return out
	}
	assertOrder := func(name string, got []*models.Notification, want ...uuid.UUID) {
		t.Helper()
		g := ids(got)
		if len(g) != len(want) {
			t.Fatalf("%s: got %d rows, want %d (%v)", name, len(g), len(want), g)
		}
		for i := range want {
			if g[i] != want[i] {
				t.Fatalf("%s: position %d = %s, want %s", name, i, g[i], want[i])
			}
		}
	}

	all, total, err := repo.List(ctx, base, 0, 50)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	// Pinned first, then newest first.
	assertOrder("unfiltered", all, pinned, newest, middle, oldest, dismissed)

	f := base
	f.ModuleID = strPtr("chat")
	got, total, err := repo.List(ctx, f, 0, 50)
	if err != nil {
		t.Fatalf("List module: %v", err)
	}
	if total != 3 {
		t.Errorf("module total = %d, want 3", total)
	}
	assertOrder("module=chat", got, newest, oldest, dismissed)

	f = base
	read := true
	f.IsRead = &read
	got, total, err = repo.List(ctx, f, 0, 50)
	if err != nil {
		t.Fatalf("List is_read: %v", err)
	}
	if total != 1 {
		t.Errorf("is_read total = %d, want 1", total)
	}
	assertOrder("is_read=true", got, middle)

	f = base
	yes := true
	f.IsPinned = &yes
	got, _, err = repo.List(ctx, f, 0, 50)
	if err != nil {
		t.Fatalf("List is_pinned: %v", err)
	}
	assertOrder("is_pinned=true", got, pinned)

	f = base
	f.IsDismissed = &yes
	got, _, err = repo.List(ctx, f, 0, 50)
	if err != nil {
		t.Fatalf("List is_dismissed: %v", err)
	}
	assertOrder("is_dismissed=true", got, dismissed)

	f = base
	no := false
	f.IsDismissed = &no
	got, total, err = repo.List(ctx, f, 0, 50)
	if err != nil {
		t.Fatalf("List is_dismissed=false: %v", err)
	}
	if total != 4 {
		t.Errorf("not-dismissed total = %d, want 4", total)
	}
	assertOrder("is_dismissed=false", got, pinned, newest, middle, oldest)

	// Paging: total ignores LIMIT/OFFSET, the page does not.
	got, total, err = repo.List(ctx, base, 1, 2)
	if err != nil {
		t.Fatalf("List page: %v", err)
	}
	if total != 5 {
		t.Errorf("paged total = %d, want 5", total)
	}
	assertOrder("offset=1 limit=2", got, newest, middle)
}

func TestRepository_ListAndCountAreTenantAndUserScoped(t *testing.T) {
	repo, pool := newRepoWithPool(t)
	fx := seedFixture(t, pool, "list-scope")
	other := seedFixture(t, pool, "list-scope-other")
	mine := seedNotification(t, pool, fx.tenant, fx.user, false)
	// A second user in the same tenant, and the same user id asked for as a foreign tenant.
	coworker := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": fx.tenant, "email": "notif-co-" + uuid.NewString() + "@test.local",
		"password_hash": "x", "first_name": "Co", "last_name": "Worker",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", coworker) })
	seedNotification(t, pool, fx.tenant, coworker, false)
	sys := testutil.WithSystemCtx(context.Background())

	got, total, err := repo.List(sys, notification.ListFilter{TenantID: fx.tenant, UserID: fx.user}, 0, 50)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 || len(got) != 1 || got[0].ID != mine {
		t.Fatalf("List own = %d rows total=%d, want only %s", len(got), total, mine)
	}
	got, total, err = repo.List(sys, notification.ListFilter{TenantID: other.tenant, UserID: fx.user}, 0, 50)
	if err != nil {
		t.Fatalf("List foreign tenant: %v", err)
	}
	if total != 0 || len(got) != 0 {
		t.Fatalf("List with foreign tenant leaked %d rows (total %d)", len(got), total)
	}

	if n, err := repo.GetUnreadCount(sys, fx.tenant, fx.user); err != nil || n != 1 {
		t.Fatalf("GetUnreadCount own = %d, %v; want 1", n, err)
	}
	if n, err := repo.GetUnreadCount(sys, other.tenant, fx.user); err != nil || n != 0 {
		t.Fatalf("GetUnreadCount foreign tenant = %d, %v; want 0", n, err)
	}
}

func TestRepository_MarkReadSetsReadAtOnceAndIsTenantScoped(t *testing.T) {
	repo, pool := newRepoWithPool(t)
	fx := seedFixture(t, pool, "markread")
	other := seedFixture(t, pool, "markread-other")
	id := seedNotification(t, pool, fx.tenant, fx.user, false)
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)
	sys := testutil.WithSystemCtx(context.Background())

	// Foreign tenant: neither updated nor reported as already-read.
	if err := repo.MarkRead(sys, other.tenant, id, time.Now()); err != notification.ErrNotificationNotFound {
		t.Fatalf("MarkRead foreign tenant = %v, want ErrNotificationNotFound", err)
	}
	if err := repo.MarkRead(ctx, fx.tenant, uuid.New(), time.Now()); err != notification.ErrNotificationNotFound {
		t.Fatalf("MarkRead unknown = %v, want ErrNotificationNotFound", err)
	}
	if got, _ := repo.GetByID(ctx, fx.tenant, id); got.IsRead {
		t.Fatal("foreign-tenant MarkRead flipped is_read")
	}

	first := time.Now().Add(-time.Hour).UTC().Truncate(time.Microsecond)
	if err := repo.MarkRead(ctx, fx.tenant, id, first); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	// Second call: already read is not an error and must not move read_at.
	if err := repo.MarkRead(ctx, fx.tenant, id, time.Now()); err != nil {
		t.Fatalf("MarkRead (already read): %v", err)
	}
	got, err := repo.GetByID(ctx, fx.tenant, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if !got.IsRead || got.ReadAt == nil || !got.ReadAt.Equal(first) {
		t.Fatalf("after MarkRead twice: read=%v read_at=%v, want true / %v", got.IsRead, got.ReadAt, first)
	}
}

func TestRepository_MarkAllReadScopesByUserModuleAndTenant(t *testing.T) {
	repo, pool := newRepoWithPool(t)
	fx := seedFixture(t, pool, "markall")
	other := seedFixture(t, pool, "markall-other")
	chatA := seedNotificationWith(t, pool, fx, map[string]any{"module_id": "chat"})
	chatB := seedNotificationWith(t, pool, fx, map[string]any{"module_id": "chat"})
	crm := seedNotificationWith(t, pool, fx, map[string]any{"module_id": "crm"})
	foreign := seedNotificationWith(t, pool, other, map[string]any{"module_id": "chat"})
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)
	sys := testutil.WithSystemCtx(context.Background())
	readAt := time.Now().UTC().Truncate(time.Microsecond)

	isRead := func(id uuid.UUID, tenant uuid.UUID) bool {
		t.Helper()
		n, err := repo.GetByID(sys, tenant, id)
		if err != nil {
			t.Fatalf("GetByID %s: %v", id, err)
		}
		return n.IsRead
	}

	// Foreign tenant asked: nothing of ours may change.
	if n, err := repo.MarkAllRead(sys, other.tenant, fx.user, nil, readAt); err != nil || n != 0 {
		t.Fatalf("MarkAllRead foreign tenant = %d, %v; want 0", n, err)
	}
	if n, err := repo.MarkAllRead(sys, other.tenant, fx.user, strPtr("chat"), readAt); err != nil || n != 0 {
		t.Fatalf("MarkAllRead foreign tenant (module) = %d, %v; want 0", n, err)
	}

	n, err := repo.MarkAllRead(ctx, fx.tenant, fx.user, strPtr("chat"), readAt)
	if err != nil || n != 2 {
		t.Fatalf("MarkAllRead module = %d, %v; want 2", n, err)
	}
	if !isRead(chatA, fx.tenant) || !isRead(chatB, fx.tenant) || isRead(crm, fx.tenant) {
		t.Fatal("module-scoped MarkAllRead touched the wrong rows")
	}

	n, err = repo.MarkAllRead(ctx, fx.tenant, fx.user, nil, readAt)
	if err != nil || n != 1 {
		t.Fatalf("MarkAllRead all = %d, %v; want 1 (only the crm row was left)", n, err)
	}
	if !isRead(crm, fx.tenant) {
		t.Fatal("unscoped MarkAllRead left the crm row unread")
	}
	if isRead(foreign, other.tenant) {
		t.Fatal("MarkAllRead touched a foreign tenant's notification")
	}
}

func TestRepository_MarkDeliveredDesktopIsTenantScoped(t *testing.T) {
	repo, pool := newRepoWithPool(t)
	fx := seedFixture(t, pool, "desktop")
	other := seedFixture(t, pool, "desktop-other")
	id := seedNotification(t, pool, fx.tenant, fx.user, false)
	sys := testutil.WithSystemCtx(context.Background())

	if err := repo.MarkDeliveredDesktop(sys, other.tenant, id); err != nil {
		t.Fatalf("MarkDeliveredDesktop foreign tenant: %v", err)
	}
	if got, _ := repo.GetByID(sys, fx.tenant, id); got.DeliveredDesktop {
		t.Fatal("foreign-tenant MarkDeliveredDesktop flipped the flag")
	}
	if err := repo.MarkDeliveredDesktop(sys, fx.tenant, id); err != nil {
		t.Fatalf("MarkDeliveredDesktop: %v", err)
	}
	if got, _ := repo.GetByID(sys, fx.tenant, id); !got.DeliveredDesktop {
		t.Fatal("MarkDeliveredDesktop did not set the flag")
	}
}

func TestRepository_FindRecentByGroupKey(t *testing.T) {
	repo, pool := newRepoWithPool(t)
	fx := seedFixture(t, pool, "group")
	other := seedFixture(t, pool, "group-other")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)
	sys := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	older := seedNotificationWith(t, pool, fx, map[string]any{"group_key": "k", "created_at": now.Add(-20 * time.Second)})
	newer := seedNotificationWith(t, pool, fx, map[string]any{"group_key": "k", "created_at": now.Add(-5 * time.Second)})
	seedNotificationWith(t, pool, fx, map[string]any{"group_key": "k", "is_read": true, "created_at": now.Add(-1 * time.Second)})
	seedNotificationWith(t, pool, fx, map[string]any{"group_key": "stale", "created_at": now.Add(-10 * time.Minute)})
	_ = older

	since := now.Add(-time.Minute)
	got, err := repo.FindRecentByGroupKey(ctx, fx.tenant, fx.user, "k", since)
	if err != nil {
		t.Fatalf("FindRecentByGroupKey: %v", err)
	}
	if got == nil || got.ID != newer {
		t.Fatalf("got %v, want the newest unread row %s (read rows are skipped)", got, newer)
	}

	if got, err := repo.FindRecentByGroupKey(ctx, fx.tenant, fx.user, "stale", since); err != nil || got != nil {
		t.Fatalf("stale group = %v, %v; want nil, nil (outside the window)", got, err)
	}
	if got, err := repo.FindRecentByGroupKey(ctx, fx.tenant, fx.user, "nope", since); err != nil || got != nil {
		t.Fatalf("unknown group = %v, %v; want nil, nil", got, err)
	}
	if got, err := repo.FindRecentByGroupKey(sys, other.tenant, fx.user, "k", since); err != nil || got != nil {
		t.Fatalf("foreign tenant = %v, %v; want nil, nil", got, err)
	}
	if got, err := repo.FindRecentByGroupKey(sys, fx.tenant, uuid.New(), "k", since); err != nil || got != nil {
		t.Fatalf("other user = %v, %v; want nil, nil", got, err)
	}
}

func TestRepository_IncrementGroupCount(t *testing.T) {
	repo, pool := newRepoWithPool(t)
	fx := seedFixture(t, pool, "increment")
	id := seedNotificationWith(t, pool, fx, map[string]any{"group_key": "k", "group_count": 1, "title": "one"})
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	if err := repo.IncrementGroupCount(ctx, id, "two"); err != nil {
		t.Fatalf("IncrementGroupCount: %v", err)
	}
	if err := repo.IncrementGroupCount(ctx, id, "three"); err != nil {
		t.Fatalf("IncrementGroupCount: %v", err)
	}
	got, err := repo.GetByID(ctx, fx.tenant, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.GroupCount != 3 || got.Title != "three" {
		t.Fatalf("group_count=%d title=%q, want 3 / three", got.GroupCount, got.Title)
	}
}

func TestRepository_FlagSettersToggleAndAreTenantScoped(t *testing.T) {
	repo, pool := newRepoWithPool(t)
	fx := seedFixture(t, pool, "flags")
	other := seedFixture(t, pool, "flags-other")
	id := seedNotification(t, pool, fx.tenant, fx.user, false)
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)
	sys := testutil.WithSystemCtx(context.Background())

	// SetPinned
	if err := repo.SetPinned(sys, other.tenant, id, true); err != notification.ErrNotificationNotFound {
		t.Fatalf("SetPinned foreign tenant = %v, want ErrNotificationNotFound", err)
	}
	if err := repo.SetPinned(ctx, fx.tenant, uuid.New(), true); err != notification.ErrNotificationNotFound {
		t.Fatalf("SetPinned unknown = %v, want ErrNotificationNotFound", err)
	}
	if err := repo.SetPinned(ctx, fx.tenant, id, true); err != nil {
		t.Fatalf("SetPinned: %v", err)
	}
	if got, _ := repo.GetByID(ctx, fx.tenant, id); !got.IsPinned {
		t.Fatal("SetPinned(true) did not stick")
	}
	if err := repo.SetPinned(ctx, fx.tenant, id, false); err != nil {
		t.Fatalf("SetPinned(false): %v", err)
	}
	if got, _ := repo.GetByID(ctx, fx.tenant, id); got.IsPinned {
		t.Fatal("SetPinned(false) did not clear the flag")
	}

	// SetDismissed
	if err := repo.SetDismissed(sys, other.tenant, id, true); err != notification.ErrNotificationNotFound {
		t.Fatalf("SetDismissed foreign tenant = %v, want ErrNotificationNotFound", err)
	}
	if err := repo.SetDismissed(ctx, fx.tenant, uuid.New(), true); err != notification.ErrNotificationNotFound {
		t.Fatalf("SetDismissed unknown = %v, want ErrNotificationNotFound", err)
	}
	if err := repo.SetDismissed(ctx, fx.tenant, id, true); err != nil {
		t.Fatalf("SetDismissed: %v", err)
	}
	if got, _ := repo.GetByID(ctx, fx.tenant, id); !got.IsDismissed {
		t.Fatal("SetDismissed(true) did not stick")
	}
	if err := repo.SetDismissed(ctx, fx.tenant, id, false); err != nil {
		t.Fatalf("SetDismissed(false): %v", err)
	}
	if got, _ := repo.GetByID(ctx, fx.tenant, id); got.IsDismissed {
		t.Fatal("SetDismissed(false) did not clear the flag")
	}

	// Snooze (the happy path lives in postgres_repository_test.go)
	if err := repo.Snooze(sys, other.tenant, id, time.Now().Add(time.Hour)); err != notification.ErrNotificationNotFound {
		t.Fatalf("Snooze foreign tenant = %v, want ErrNotificationNotFound", err)
	}
	if got, _ := repo.GetByID(ctx, fx.tenant, id); got.SnoozedUntil != nil || got.IsRead {
		t.Fatal("foreign-tenant Snooze modified the row")
	}
}

func TestRepository_SnoozeKeepsExistingReadAt(t *testing.T) {
	repo, pool := newRepoWithPool(t)
	fx := seedFixture(t, pool, "snooze-readat")
	readAt := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Microsecond)
	id := seedNotificationWith(t, pool, fx, map[string]any{"is_read": true, "read_at": readAt})
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	if err := repo.Snooze(ctx, fx.tenant, id, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("Snooze: %v", err)
	}
	got, err := repo.GetByID(ctx, fx.tenant, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ReadAt == nil || !got.ReadAt.Equal(readAt) {
		t.Fatalf("read_at = %v, want the original %v (COALESCE must keep it)", got.ReadAt, readAt)
	}
}

func TestRepository_NotifyDelivery(t *testing.T) {
	repo, _ := newRepoWithPool(t)
	if err := repo.NotifyDelivery(context.Background(), `{"notification_id":"x"}`); err != nil {
		t.Fatalf("NotifyDelivery: %v", err)
	}
}

func TestRepository_ClosedPoolSurfacesErrors(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	repo := notification.NewPostgresRepository(pool)
	pool.Close()
	ctx := context.Background()
	id, tenant := uuid.New(), uuid.New()

	checks := map[string]error{
		"Create":               repo.Create(ctx, &models.Notification{ID: id, TenantID: tenant}),
		"MarkRead":             repo.MarkRead(ctx, tenant, id, time.Now()),
		"MarkDeliveredDesktop": repo.MarkDeliveredDesktop(ctx, tenant, id),
		"IncrementGroupCount":  repo.IncrementGroupCount(ctx, id, "t"),
		"SetPinned":            repo.SetPinned(ctx, tenant, id, true),
		"SetDismissed":         repo.SetDismissed(ctx, tenant, id, true),
		"Snooze":               repo.Snooze(ctx, tenant, id, time.Now()),
		"NotifyDelivery":       repo.NotifyDelivery(ctx, "{}"),
	}
	_, _, listErr := repo.List(ctx, notification.ListFilter{TenantID: tenant, UserID: id}, 0, 10)
	checks["List"] = listErr
	_, countErr := repo.GetUnreadCount(ctx, tenant, id)
	checks["GetUnreadCount"] = countErr
	_, markAllErr := repo.MarkAllRead(ctx, tenant, id, nil, time.Now())
	checks["MarkAllRead"] = markAllErr
	_, markAllModErr := repo.MarkAllRead(ctx, tenant, id, strPtr("m"), time.Now())
	checks["MarkAllRead(module)"] = markAllModErr
	_, findErr := repo.FindRecentByGroupKey(ctx, tenant, id, "k", time.Now())
	checks["FindRecentByGroupKey"] = findErr
	_, getErr := repo.GetByID(ctx, tenant, id)
	checks["GetByID"] = getErr

	for name, err := range checks {
		if err == nil {
			t.Errorf("%s on a closed pool returned nil, want an error", name)
		}
	}
}
