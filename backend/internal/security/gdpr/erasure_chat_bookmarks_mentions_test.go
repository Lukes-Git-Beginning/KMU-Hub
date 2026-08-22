package gdpr

// Integration coverage for the message_bookmarks and message_mentions gap in
// ChatErasureHandler (fix-chat-erasure-missing-bookmarks-mentions). Both
// tables are CASCADE on users(id), but AuthErasureHandler anonymizes the user
// row instead of deleting it, so the CASCADE never fires — the same reasoning
// already applied to channel_memberships and project_members.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedMessageBookmark inserts into message_bookmarks, which has a composite
// primary key (user_id, message_id) and no id column.
func seedMessageBookmark(t *testing.T, pool *pgxpool.Pool, tenantID, messageID, userID uuid.UUID) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(ctx,
		`INSERT INTO message_bookmarks (tenant_id, message_id, user_id) VALUES ($1, $2, $3)`,
		tenantID, messageID, userID,
	)
	if err != nil {
		t.Fatalf("seed message_bookmarks: %v", err)
	}
}

// seedMessageMention inserts into message_mentions, which has a composite
// primary key (message_id, user_id) and no id column.
func seedMessageMention(t *testing.T, pool *pgxpool.Pool, tenantID, messageID, mentionedUserID uuid.UUID) {
	t.Helper()
	ctx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(ctx,
		`INSERT INTO message_mentions (tenant_id, message_id, user_id) VALUES ($1, $2, $3)`,
		tenantID, messageID, mentionedUserID,
	)
	if err != nil {
		t.Fatalf("seed message_mentions: %v", err)
	}
}

func TestChatErasureHandler_ExecuteErasure_BookmarksAndMentions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Chat Erasure Bookmarks Mentions Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userID := seedExportUser(t, pool, tenantOwn, "chat-erasure-bookmarks")
	defer testutil.CleanupRow(t, pool, "users", userID)
	colleagueID := seedExportUser(t, pool, tenantOwn, "chat-erasure-colleague-2")
	defer testutil.CleanupRow(t, pool, "users", colleagueID)

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "projekt-delta",
		"created_by": colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	// A colleague's message that the subject bookmarked and was mentioned in.
	foreignMessageID := testutil.SeedRow(t, pool, "messages", map[string]any{
		"tenant_id":  tenantOwn,
		"channel_id": channelID,
		"content":    "@subject bitte pruefen",
		"created_by": colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "messages", foreignMessageID)

	seedMessageBookmark(t, pool, tenantOwn, foreignMessageID, userID)
	seedMessageMention(t, pool, tenantOwn, foreignMessageID, userID)

	// The colleague is also mentioned in the same message — their mention row
	// must survive the subject's erasure untouched.
	seedMessageMention(t, pool, tenantOwn, foreignMessageID, colleagueID)

	h := NewChatErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	affected, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureAnonymize)
	require.NoError(t, err)
	assert.Equal(t, 2, affected, "one bookmark deleted + one mention of the subject deleted")

	// The mentioning message survives untouched — it belongs to the colleague,
	// not the subject.
	var content string
	var isDeleted bool
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT content, is_deleted FROM messages WHERE id = $1`, foreignMessageID,
	).Scan(&content, &isDeleted))
	assert.Equal(t, "@subject bitte pruefen", content, "a foreign message must survive the mentioned user's erasure untouched")
	assert.False(t, isDeleted)

	// The subject's bookmark is gone.
	var bookmarkRows int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM message_bookmarks WHERE user_id = $1`, userID).Scan(&bookmarkRows))
	assert.Equal(t, 0, bookmarkRows, "the subject's bookmark must be deleted")

	// Only the subject's mention is gone; the colleague's mention in the same
	// message survives.
	var subjectMentionRows, colleagueMentionRows int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM message_mentions WHERE user_id = $1`, userID).Scan(&subjectMentionRows))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM message_mentions WHERE user_id = $1`, colleagueID).Scan(&colleagueMentionRows))
	assert.Equal(t, 0, subjectMentionRows, "the subject's mention must be deleted")
	assert.Equal(t, 1, colleagueMentionRows, "a colleague's mention in the same message must survive")
}
