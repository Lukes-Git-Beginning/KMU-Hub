package reaction

// TestPostgresAddReaction_ConcurrentToggle_OnlyOneRow proves the concurrency
// question required by cov-gateway-chat-reactions-bookmarks-search: what
// happens when HandleToggleReaction's underlying AddReaction runs twice at
// once for the same (message, user, emoji)? Service.ToggleReaction itself is
// check-then-act (ReactionExists, then Add/Remove) and racy on its own, but
// postgres_repository.go's AddReaction relies on message_reactions' composite
// PRIMARY KEY (message_id, user_id, emoji) plus "ON CONFLICT DO NOTHING" as
// the real backstop -- the same pattern proven for message_bookmarks'
// PRIMARY KEY (user_id, message_id) in bookmark.Service.Toggle. This test
// exercises the real INSERT path, not a mock, so it actually proves the
// constraint fires under concurrency instead of just reading the SQL text.

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPostgresAddReaction_ConcurrentToggle_OnlyOneRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Reaction Concurrent Toggle")
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         "reaction-concurrent-" + uuid.New().String()[:8] + "@tenanta.local",
		"password_hash": "$argon2id$v=19$test",
		"first_name":    "Rita",
		"last_name":     "Reactor",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id": tenantID, "name": "reaction-concurrent", "created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	messageID := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id": tenantID, "channel_id": channelID, "content": "react to me",
		"lang": "simple", "created_by": userID, "created_at": time.Now().UTC(),
	})
	defer testutil.CleanupRow(t, pool, "messages", messageID)

	repo := NewPostgresRepository(pool)

	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, 5)
	for i := range 5 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = repo.AddReaction(ctx, messageID, userID, "👍")
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: AddReaction returned an error instead of silently no-opping on conflict: %v", i, err)
		}
	}

	var count int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM message_reactions WHERE message_id=$1 AND user_id=$2 AND emoji=$3",
		messageID, userID, "👍",
	).Scan(&count); err != nil {
		t.Fatalf("count message_reactions: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 reaction row after 5 concurrent AddReaction calls, got %d", count)
	}
}
