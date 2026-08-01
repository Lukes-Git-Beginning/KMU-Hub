package bookmark_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/chat/bookmark"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// fixture mirrors the chain internal/chat/message's tenant_write_test.go
// seeds: tenant -> user -> channel -> message. Tenants are minted per test
// (not testutil.TenantA/B) so fixtures don't collide under t.Parallel().
type fixture struct {
	tenant  uuid.UUID
	user    uuid.UUID
	message uuid.UUID
}

func seedFixture(t *testing.T, pool *pgxpool.Pool, name string) fixture {
	t.Helper()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "bookmark test "+name)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID,
		"email":         "bookmark-" + name + "-" + uuid.NewString() + "@test.local",
		"password_hash": "x", "first_name": "Bookmark", "last_name": "Test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "name": "bookmark-" + name, "created_by": userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "channels", channelID) })

	messageID := testutil.SeedRow(t, pool, "messages", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "channel_id": channelID,
		"content": "hallo", "lang": "german", "created_by": userID, "created_at": time.Now().UTC(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", messageID) })

	return fixture{tenant: tenantID, user: userID, message: messageID}
}

func TestRepository_AddExistsRemoveRoundtrip(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := bookmark.NewPostgresRepository(pool)
	mine := seedFixture(t, pool, "roundtrip")
	ctx := testutil.WithTenantCtx(context.Background(), mine.tenant)

	if exists, err := repo.Exists(ctx, mine.tenant, mine.user, mine.message); err != nil || exists {
		t.Fatalf("Exists before Add: %v, err=%v, want false", exists, err)
	}

	if err := repo.Add(ctx, mine.tenant, mine.user, mine.message); err != nil {
		t.Fatalf("Add: %v", err)
	}
	t.Cleanup(func() { _ = repo.Remove(context.Background(), mine.tenant, mine.user, mine.message) })

	if exists, err := repo.Exists(ctx, mine.tenant, mine.user, mine.message); err != nil || !exists {
		t.Fatalf("Exists after Add: %v, err=%v, want true", exists, err)
	}

	ids, err := repo.ListMessageIDs(ctx, mine.tenant, mine.user)
	if err != nil {
		t.Fatalf("ListMessageIDs: %v", err)
	}
	if len(ids) != 1 || ids[0] != mine.message {
		t.Fatalf("ListMessageIDs = %v, want [%s]", ids, mine.message)
	}

	if err := repo.Remove(ctx, mine.tenant, mine.user, mine.message); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if exists, err := repo.Exists(ctx, mine.tenant, mine.user, mine.message); err != nil || exists {
		t.Fatalf("Exists after Remove: %v, err=%v, want false", exists, err)
	}
}

func TestRepository_AddIsIdempotent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := bookmark.NewPostgresRepository(pool)
	mine := seedFixture(t, pool, "idempotent")
	ctx := testutil.WithTenantCtx(context.Background(), mine.tenant)

	if err := repo.Add(ctx, mine.tenant, mine.user, mine.message); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	t.Cleanup(func() { _ = repo.Remove(context.Background(), mine.tenant, mine.user, mine.message) })
	if err := repo.Add(ctx, mine.tenant, mine.user, mine.message); err != nil {
		t.Fatalf("second Add: %v, want no error (ON CONFLICT DO NOTHING)", err)
	}

	ids, err := repo.ListMessageIDs(ctx, mine.tenant, mine.user)
	if err != nil {
		t.Fatalf("ListMessageIDs: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("ListMessageIDs = %v, want exactly 1 despite double Add", ids)
	}
}

func TestRepository_ListMessageIDsIsScopedPerUserAndTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := bookmark.NewPostgresRepository(pool)
	mine := seedFixture(t, pool, "scope-own")
	other := seedFixture(t, pool, "scope-other")

	// A second user in mine's own tenant, sharing mine's channel/message.
	otherUserID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "tenant_id": mine.tenant,
		"email":         "bookmark-scope-peer-" + uuid.NewString() + "@test.local",
		"password_hash": "x", "first_name": "Peer", "last_name": "User",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", otherUserID) })

	ctxMine := testutil.WithTenantCtx(context.Background(), mine.tenant)
	if err := repo.Add(ctxMine, mine.tenant, mine.user, mine.message); err != nil {
		t.Fatalf("Add: %v", err)
	}
	t.Cleanup(func() { _ = repo.Remove(context.Background(), mine.tenant, mine.user, mine.message) })

	// A different user in the same tenant must not see mine's bookmark.
	ids, err := repo.ListMessageIDs(ctxMine, mine.tenant, otherUserID)
	if err != nil {
		t.Fatalf("ListMessageIDs peer user: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("ListMessageIDs peer user = %v, want empty (another user's bookmark leaked)", ids)
	}

	// A user in a different tenant, queried under their own tenant context,
	// must not see mine's bookmark either (RLS-backed, not just the WHERE
	// clause: this exercises the enable_tenant_rls('message_bookmarks')
	// policy under kmuhub_app, not the superuser role).
	ctxOther := testutil.WithTenantCtx(context.Background(), other.tenant)
	ids, err = repo.ListMessageIDs(ctxOther, other.tenant, other.user)
	if err != nil {
		t.Fatalf("ListMessageIDs foreign tenant: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("ListMessageIDs foreign tenant = %v, want empty", ids)
	}
}
