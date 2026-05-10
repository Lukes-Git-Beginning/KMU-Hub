package message

// Cross-tenant isolation tests for message_mentions (Migration 000115).
//
// message_mentions.tenant_id is NOT NULL after Migration 000115.
// These tests verify that Mention structs created by processMentions carry the
// correct TenantID (propagated from the parent message's tenantID parameter),
// ensuring no cross-tenant contamination in the batch INSERT.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// TestMentions_TenantIDStoredOnCreate verifies that when a message with mentions
// is created, all Mention records stored via CreateMentions carry the message's TenantID.
func TestMentions_TenantIDStoredOnCreate(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	channelID := uuid.New()
	createdBy := uuid.New()
	mentionedUser := uuid.New()

	// Register channel, creator, and mentioned user as members.
	repo.channels[channelID] = struct{ archived bool }{false}
	repo.memberships[membershipKey(channelID, createdBy)] = models.ChannelRoleMember
	repo.memberships[membershipKey(channelID, mentionedUser)] = models.ChannelRoleMember
	repo.users[createdBy] = struct{ firstName, lastName string }{"Alice", "Sender"}
	repo.users[mentionedUser] = struct{ firstName, lastName string }{"Bob", "Mentioned"}

	result, err := svc.Create(context.Background(), CreateInput{
		TenantID:         tenantID,
		ChannelID:        channelID,
		Content:          "Hello @Bob",
		CreatedBy:        createdBy,
		MentionedUserIDs: []uuid.UUID{mentionedUser},
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Inspect stored mentions keyed by the created message ID.
	msgID := result.ID
	stored := repo.mentions[msgID]
	require.NotEmpty(t, stored, "mentions must have been stored via CreateMentions")
	for _, m := range stored {
		assert.Equal(t, tenantID, m.TenantID,
			"mention must carry TenantID from parent message's tenant")
		assert.Equal(t, mentionedUser, m.UserID)
	}
}

// TestMentions_CrossTenantIsolation verifies that mentions for message A (tenant A)
// are not returned when querying mentions for message B (tenant B).
func TestMentions_CrossTenantIsolation(t *testing.T) {
	repo := NewMockRepository()

	msgA := uuid.New()
	msgB := uuid.New()
	tenantA := uuid.New()
	tenantB := uuid.New()
	userA := uuid.New()
	userB := uuid.New()

	now := time.Now()

	// Seed mentions for msgA (tenant A) and msgB (tenant B) directly via mock.
	repo.mentions[msgA] = []models.Mention{
		{MessageID: msgA, TenantID: tenantA, UserID: userA, MentionType: models.MentionTypeUser, CreatedAt: now},
	}
	repo.mentions[msgB] = []models.Mention{
		{MessageID: msgB, TenantID: tenantB, UserID: userB, MentionType: models.MentionTypeUser, CreatedAt: now},
	}

	// Fetch mentions for both messages.
	result, err := repo.GetMentionsByMessages(context.Background(), []uuid.UUID{msgA, msgB})
	require.NoError(t, err)

	// msgA mentions must only contain userA (tenant A).
	mentionsA := result[msgA]
	require.Len(t, mentionsA, 1)
	assert.Equal(t, tenantA, mentionsA[0].TenantID)
	assert.Equal(t, userA, mentionsA[0].UserID)

	// msgB mentions must only contain userB (tenant B).
	mentionsB := result[msgB]
	require.Len(t, mentionsB, 1)
	assert.Equal(t, tenantB, mentionsB[0].TenantID)
	assert.Equal(t, userB, mentionsB[0].UserID)

	// No cross-contamination.
	for _, m := range mentionsB {
		assert.NotEqual(t, tenantA, m.TenantID, "tenant A mention must not appear in tenant B result")
	}
}
