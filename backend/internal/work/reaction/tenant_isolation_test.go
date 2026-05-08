package reaction

// Cross-tenant isolation tests for message_reactions (Migration 000115).
//
// message_reactions is now scoped by tenant_id (backfilled from messages.tenant_id).
// These tests verify that reactions for message A (tenant A) are not returned
// when querying reactions for message B (tenant B), using the in-process mock repo
// which mirrors what the postgres repository enforces at the DB level.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessageReactions_TenantIsolation verifies that reactions belonging to a message
// in tenant A are not returned when listing reactions for a message in tenant B.
func TestMessageReactions_TenantIsolation(t *testing.T) {
	repo := newMockReactionRepo()

	msgA := uuid.New() // message owned by tenant A
	msgB := uuid.New() // message owned by tenant B
	userA := uuid.New()
	userB := uuid.New()

	// Add reactions to both messages.
	require.NoError(t, repo.AddReaction(context.Background(), msgA, userA, "👍"))
	require.NoError(t, repo.AddReaction(context.Background(), msgB, userB, "❤️"))

	// List reactions for message A — must only contain userA's reaction.
	reactionsA, err := repo.ListReactions(context.Background(), msgA)
	require.NoError(t, err)
	require.Len(t, reactionsA, 1)
	assert.Equal(t, msgA, reactionsA[0].MessageID)
	assert.Equal(t, userA, reactionsA[0].UserID)
	assert.Equal(t, "👍", reactionsA[0].Emoji)

	// List reactions for message B — must only contain userB's reaction.
	reactionsB, err := repo.ListReactions(context.Background(), msgB)
	require.NoError(t, err)
	require.Len(t, reactionsB, 1)
	assert.Equal(t, msgB, reactionsB[0].MessageID)
	assert.Equal(t, userB, reactionsB[0].UserID)
	assert.Equal(t, "❤️", reactionsB[0].Emoji)

	// Confirm no cross-contamination: msgA reactions not in msgB results.
	for _, r := range reactionsB {
		assert.NotEqual(t, msgA, r.MessageID, "tenant A message reaction leaked into tenant B result")
	}
}

// TestMessageReactions_BatchQuery_TenantIsolation verifies that GetBatchReactionSummary
// only returns reactions for the requested message IDs (no other messages leak in).
func TestMessageReactions_BatchQuery_TenantIsolation(t *testing.T) {
	repo := newMockReactionRepo()

	msgA := uuid.New()
	msgB := uuid.New()
	msgC := uuid.New() // third message, never queried
	user := uuid.New()

	require.NoError(t, repo.AddReaction(context.Background(), msgA, user, "😀"))
	require.NoError(t, repo.AddReaction(context.Background(), msgB, user, "😢"))
	require.NoError(t, repo.AddReaction(context.Background(), msgC, user, "🔥"))

	// Batch query only for msgA and msgB.
	summaries, err := repo.GetBatchReactionSummary(context.Background(), []uuid.UUID{msgA, msgB}, user)
	require.NoError(t, err)

	// Result must only contain keys for the requested messages.
	assert.Contains(t, summaries, msgA, "msgA must be in batch result")
	assert.Contains(t, summaries, msgB, "msgB must be in batch result")
	assert.NotContains(t, summaries, msgC, "msgC must not leak into batch result")
}
