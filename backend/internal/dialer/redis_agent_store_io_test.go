package dialer

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAgentStatusStore(t *testing.T) (*AgentStatusStore, *miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rc.Close() })
	return NewAgentStatusStore(rc), mr, rc
}

// campaignMembers reads a campaign agent set through the redis protocol
// (SMEMBERS), which — unlike miniredis's direct Go API — returns an empty
// slice rather than an error once the set has been emptied and deleted.
func campaignMembers(t *testing.T, rc *redis.Client, key string) []string {
	t.Helper()
	members, err := rc.SMembers(context.Background(), key).Result()
	require.NoError(t, err)
	return members
}

func TestSetStatus_InvalidTransition_RejectsWithoutWriting(t *testing.T) {
	store, _, _ := setupAgentStatusStore(t)
	ctx := context.Background()
	userID := uuid.New()
	tenantID := uuid.New()

	// A brand-new agent has no status key, so GetStatus treats it as offline.
	// offline -> on_call is not in validTransitions, so this must be rejected.
	err := store.SetStatus(ctx, userID, AgentStatusOnCall, nil, tenantID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidTransition)

	entry, err := store.GetStatus(ctx, userID)
	require.NoError(t, err)
	assert.Nil(t, entry, "rejected transition must not write a status key")
}

func TestSetStatus_ValidTransition_WritesWithTTL(t *testing.T) {
	store, mr, _ := setupAgentStatusStore(t)
	ctx := context.Background()
	userID := uuid.New()
	tenantID := uuid.New()

	err := store.SetStatus(ctx, userID, AgentStatusAvailable, nil, tenantID)
	require.NoError(t, err)

	entry, err := store.GetStatus(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, userID, entry.UserID)
	assert.Equal(t, AgentStatusAvailable, entry.Status)

	ttl := mr.TTL(agentStatusKey(userID))
	assert.Equal(t, agentStatusTTL, ttl)
}

func TestSetStatus_CampaignSwitch_MovesAgentBetweenSets(t *testing.T) {
	store, _, rc := setupAgentStatusStore(t)
	ctx := context.Background()
	userID := uuid.New()
	tenantID := uuid.New()
	campaignA := uuid.New()
	campaignB := uuid.New()

	require.NoError(t, store.SetStatus(ctx, userID, AgentStatusAvailable, &campaignA, tenantID))
	assert.Contains(t, campaignMembers(t, rc, campaignAgentsKey(campaignA)), userID.String())

	// available -> on_call is valid; switching to campaign B must move the
	// agent out of A's set and into B's set.
	require.NoError(t, store.SetStatus(ctx, userID, AgentStatusOnCall, &campaignB, tenantID))

	assert.NotContains(t, campaignMembers(t, rc, campaignAgentsKey(campaignA)), userID.String(),
		"agent must be removed from the old campaign set")
	assert.Contains(t, campaignMembers(t, rc, campaignAgentsKey(campaignB)), userID.String())
}

func TestGetAvailableAgents_IgnoresExpiredTTL(t *testing.T) {
	store, mr, _ := setupAgentStatusStore(t)
	ctx := context.Background()
	tenantID := uuid.New()
	campaignID := uuid.New()
	staleAgent := uuid.New()
	freshAgent := uuid.New()

	require.NoError(t, store.SetStatus(ctx, staleAgent, AgentStatusAvailable, &campaignID, tenantID))
	require.NoError(t, store.SetStatus(ctx, freshAgent, AgentStatusAvailable, &campaignID, tenantID))

	// Push time past the 8h status TTL. Both status keys expire, but the
	// campaign membership set (SADD, no TTL) still lists both agents.
	mr.FastForward(agentStatusTTL + time.Minute)

	// Refresh only freshAgent — from Redis's point of view its prior status
	// key is gone, so this is an offline -> available transition again.
	require.NoError(t, store.SetStatus(ctx, freshAgent, AgentStatusAvailable, &campaignID, tenantID))

	entries, err := store.GetCampaignAgents(ctx, campaignID)
	require.NoError(t, err)
	require.Len(t, entries, 1, "expired agent status must be skipped, not returned")
	assert.Equal(t, freshAgent, entries[0].UserID)

	available, err := store.GetAvailableAgents(ctx, campaignID)
	require.NoError(t, err)
	require.Len(t, available, 1)
	assert.Equal(t, freshAgent, available[0])
}

func TestGetAvailableAgents_ExcludesNonAvailableStatus(t *testing.T) {
	store, _, _ := setupAgentStatusStore(t)
	ctx := context.Background()
	tenantID := uuid.New()
	campaignID := uuid.New()
	onCallAgent := uuid.New()

	require.NoError(t, store.SetStatus(ctx, onCallAgent, AgentStatusAvailable, &campaignID, tenantID))
	require.NoError(t, store.SetStatus(ctx, onCallAgent, AgentStatusOnCall, &campaignID, tenantID))

	available, err := store.GetAvailableAgents(ctx, campaignID)
	require.NoError(t, err)
	assert.Empty(t, available, "an on_call agent must not be reported as available")
}

func TestRemoveFromCampaign_RemovesAgentFromSet(t *testing.T) {
	store, _, rc := setupAgentStatusStore(t)
	ctx := context.Background()
	tenantID := uuid.New()
	campaignID := uuid.New()
	userID := uuid.New()

	require.NoError(t, store.SetStatus(ctx, userID, AgentStatusAvailable, &campaignID, tenantID))
	require.Contains(t, campaignMembers(t, rc, campaignAgentsKey(campaignID)), userID.String())

	require.NoError(t, store.RemoveFromCampaign(ctx, userID, campaignID))

	assert.NotContains(t, campaignMembers(t, rc, campaignAgentsKey(campaignID)), userID.String())

	// Removing an agent that was never a member must not error.
	require.NoError(t, store.RemoveFromCampaign(ctx, userID, campaignID))
}

func TestGetCampaignAgents_EmptyCampaign(t *testing.T) {
	store, _, _ := setupAgentStatusStore(t)
	ctx := context.Background()

	entries, err := store.GetCampaignAgents(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestGetStatus_UnknownAgent_ReturnsNilWithoutError(t *testing.T) {
	store, _, _ := setupAgentStatusStore(t)
	ctx := context.Background()

	entry, err := store.GetStatus(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, entry)
}
