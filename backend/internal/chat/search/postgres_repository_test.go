package search

// The postgres repository's full-text search had no direct integration
// test — service_test.go only exercises the service against an in-memory
// MockRepository, so the real tsvector columns and triggers from migration
// 000019_add_chat_search.up.sql were never proven to actually run. This
// file covers SearchMessages/SearchFiles/GetUserChannelIDs against a real
// DB and proves channel scoping and the is_deleted filter really narrow
// the result set instead of just looking like they do against the mock.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

type searchFixture struct {
	tenant                uuid.UUID
	userA                 uuid.UUID
	channelIn, channelOut uuid.UUID
}

func newSearchFixture(t *testing.T, pool *pgxpool.Pool) searchFixture {
	t.Helper()
	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Search Read Tenant")

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("search-read-a-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Sina",
		"last_name":     "Searcher",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userA) })

	channelIn := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id": tenant, "name": "search-in-scope", "created_by": userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "channels", channelIn) })

	channelOut := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id": tenant, "name": "search-out-of-scope", "created_by": userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "channels", channelOut) })

	return searchFixture{tenant: tenant, userA: userA, channelIn: channelIn, channelOut: channelOut}
}

// seedSearchMembership inserts a channel_memberships row directly — the
// table has a composite primary key (channel_id, user_id), so
// testutil.SeedRow (which always does RETURNING id) does not apply.
func seedSearchMembership(t *testing.T, pool *pgxpool.Pool, channelID, userID, tenantID uuid.UUID) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(ctx,
		`INSERT INTO channel_memberships (channel_id, user_id, tenant_id, role, joined_at) VALUES ($1, $2, $3, 'member', $4)`,
		channelID, userID, tenantID, time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("seed membership: %v", err)
	}
}

func TestPostgresRepository_SearchMessages(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newSearchFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	inScope := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id": fx.tenant, "channel_id": fx.channelIn, "content": "quirkzebra migration notes",
		"lang": "simple", "created_by": fx.userA, "created_at": time.Now().UTC(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", inScope) })

	outOfScope := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id": fx.tenant, "channel_id": fx.channelOut, "content": "quirkzebra in the wrong channel",
		"lang": "simple", "created_by": fx.userA, "created_at": time.Now().UTC(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", outOfScope) })

	deleted := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id": fx.tenant, "channel_id": fx.channelIn, "content": "quirkzebra but deleted",
		"lang": "simple", "created_by": fx.userA, "created_at": time.Now().UTC(), "is_deleted": true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", deleted) })

	results, total, err := repo.SearchMessages(ctxOwn, "quirkzebra", "simple", []uuid.UUID{fx.channelIn}, 10, 0)
	if err != nil {
		t.Fatalf("SearchMessages: %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Fatalf("SearchMessages: expected 1 result (channel-scoped, deleted excluded), got total=%d len=%d", total, len(results))
	}
	if results[0].MessageID == nil || *results[0].MessageID != inScope {
		t.Fatalf("SearchMessages: expected the in-scope message, got %+v", results[0])
	}
	if results[0].FirstName != "Sina" {
		t.Fatalf("SearchMessages: expected sender Sina, got %q", results[0].FirstName)
	}

	// Error/edge path: no lexeme match.
	noHit, totalNoHit, err := repo.SearchMessages(ctxOwn, "nonexistentterm", "simple", []uuid.UUID{fx.channelIn}, 10, 0)
	if err != nil {
		t.Fatalf("SearchMessages no hit: %v", err)
	}
	if totalNoHit != 0 || len(noHit) != 0 {
		t.Fatalf("SearchMessages no hit: expected 0 results, got total=%d len=%d", totalNoHit, len(noHit))
	}

	// Error/edge path: caller has no channels at all.
	empty, totalEmpty, err := repo.SearchMessages(ctxOwn, "quirkzebra", "simple", nil, 10, 0)
	if err != nil {
		t.Fatalf("SearchMessages no channels: %v", err)
	}
	if totalEmpty != 0 || len(empty) != 0 {
		t.Fatalf("SearchMessages no channels: expected 0 results, got total=%d len=%d", totalEmpty, len(empty))
	}
}

func TestPostgresRepository_SearchFiles(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newSearchFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	msgID := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id": fx.tenant, "channel_id": fx.channelIn, "content": "see attachment",
		"lang": "simple", "created_by": fx.userA, "created_at": time.Now().UTC(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", msgID) })

	inScope := testutil.SeedRow(t, pool, "chat_files", map[string]any{
		"tenant_id": fx.tenant, "message_id": msgID, "channel_id": fx.channelIn,
		"filename": "quirkzebra report.pdf", "mime_type": "application/pdf", "file_size": 1024,
		"storage_key": "chat/quirkzebra-report.pdf", "uploaded_by": fx.userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "chat_files", inScope) })

	outMsgID := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id": fx.tenant, "channel_id": fx.channelOut, "content": "see attachment",
		"lang": "simple", "created_by": fx.userA, "created_at": time.Now().UTC(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "messages", outMsgID) })

	outOfScope := testutil.SeedRow(t, pool, "chat_files", map[string]any{
		"tenant_id": fx.tenant, "message_id": outMsgID, "channel_id": fx.channelOut,
		"filename": "quirkzebra out-of-scope.pdf", "mime_type": "application/pdf", "file_size": 512,
		"storage_key": "chat/quirkzebra-out-of-scope.pdf", "uploaded_by": fx.userA,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "chat_files", outOfScope) })

	results, total, err := repo.SearchFiles(ctxOwn, "quirkzebra", []uuid.UUID{fx.channelIn}, 10, 0)
	if err != nil {
		t.Fatalf("SearchFiles: %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Fatalf("SearchFiles: expected 1 channel-scoped result, got total=%d len=%d", total, len(results))
	}
	if results[0].FileID == nil || *results[0].FileID != inScope {
		t.Fatalf("SearchFiles: expected the in-scope file, got %+v", results[0])
	}

	// Error/edge path: no lexeme match.
	noHit, totalNoHit, err := repo.SearchFiles(ctxOwn, "nonexistentterm", []uuid.UUID{fx.channelIn}, 10, 0)
	if err != nil {
		t.Fatalf("SearchFiles no hit: %v", err)
	}
	if totalNoHit != 0 || len(noHit) != 0 {
		t.Fatalf("SearchFiles no hit: expected 0 results, got total=%d len=%d", totalNoHit, len(noHit))
	}
}

func TestPostgresRepository_GetUserChannelIDs(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	fx := newSearchFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)
	repo := NewPostgresRepository(pool)

	seedSearchMembership(t, pool, fx.channelIn, fx.userA, fx.tenant)
	seedSearchMembership(t, pool, fx.channelOut, fx.userA, fx.tenant)

	ids, err := repo.GetUserChannelIDs(ctxOwn, fx.userA)
	if err != nil {
		t.Fatalf("GetUserChannelIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("GetUserChannelIDs: expected 2 channels, got %d", len(ids))
	}
	seen := map[uuid.UUID]bool{ids[0]: true, ids[1]: true}
	if !seen[fx.channelIn] || !seen[fx.channelOut] {
		t.Fatalf("GetUserChannelIDs: expected channelIn and channelOut, got %v", ids)
	}

	// Error/edge path: a user with no memberships gets an empty slice.
	empty, err := repo.GetUserChannelIDs(ctxOwn, uuid.New())
	if err != nil {
		t.Fatalf("GetUserChannelIDs unknown user: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("GetUserChannelIDs unknown user: expected 0 channels, got %d", len(empty))
	}
}
