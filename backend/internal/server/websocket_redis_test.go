package server

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/coder/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestHub returns a minimal WebSocketHub suitable for unit tests.
// It does not wire chatClient or tokenMaker; tests that need those should
// build their own hub directly (cf. websocket_revalidate_test.go).
func newTestHub(rc *redis.Client) *WebSocketHub {
	return &WebSocketHub{
		connections:         make(map[string]map[*websocket.Conn]struct{}),
		channelMembers:      make(map[string]map[string]struct{}),
		presenceSubscribers: make(map[string]map[string]struct{}),
		userNames:           make(map[string]struct{ firstName, lastName string }),
		rateLimiters:        make(map[string]*msgRateLimiter),
		guestConnections:    make(map[string]map[*websocket.Conn]struct{}),
		guestChannels:       make(map[string]string),
		redisClient:         rc,
	}
}

// mustMembers is a test helper that retrieves set members from miniredis.
// It returns an empty slice for non-existent keys (miniredis returns an error
// "ERR no such key" when a key doesn't exist, which is semantically "empty set").
func mustMembers(t *testing.T, mr *miniredis.Miniredis, key string) []string {
	t.Helper()
	members, err := mr.Members(key)
	if err != nil {
		// "ERR no such key" means the key was deleted (empty set) — return empty slice.
		return []string{}
	}
	return members
}

// ----------------------------------------------------------------------------
// Tests for redisSubscribe / redisUnsubscribe with a real miniredis instance
// ----------------------------------------------------------------------------

func TestRedisSubscribe_AddsMemberAndSetsTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })

	h := newTestHub(rc)
	channelID := "chan-abc"
	userID := "user-123"

	h.redisSubscribe(channelID, userID)

	// Verify the member was added to the set.
	members := mustMembers(t, mr, wsSubscriptionKeyPrefix+channelID)
	require.Contains(t, members, userID, "userID should be in the Redis set after redisSubscribe")

	// Verify a TTL was set (miniredis returns 0 if no TTL is configured).
	ttl := mr.TTL(wsSubscriptionKeyPrefix + channelID)
	assert.Greater(t, ttl, time.Duration(0), "key should have a positive TTL after redisSubscribe")
}

func TestRedisSubscribe_RefreshesTTLOnRepeatCall(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })

	h := newTestHub(rc)
	channelID := "chan-ttl"
	userID := "user-456"

	// First subscribe sets the TTL.
	h.redisSubscribe(channelID, userID)
	first := mr.TTL(wsSubscriptionKeyPrefix + channelID)

	// Fast-forward time in miniredis so the remaining TTL decreases.
	mr.FastForward(wsSubscriptionTTL / 2)

	// Second subscribe should reset the TTL back to the full 24h value.
	h.redisSubscribe(channelID, userID)
	second := mr.TTL(wsSubscriptionKeyPrefix + channelID)

	assert.GreaterOrEqual(t, second, first,
		"TTL after second subscribe should be >= TTL right after first subscribe (refreshed to full value)")
}

func TestRedisUnsubscribe_RemovesMember(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })

	h := newTestHub(rc)
	channelID := "chan-xyz"
	userA := "user-A"
	userB := "user-B"

	h.redisSubscribe(channelID, userA)
	h.redisSubscribe(channelID, userB)

	// Unsubscribe one user; the other must remain.
	h.redisUnsubscribe(channelID, userA)

	members := mustMembers(t, mr, wsSubscriptionKeyPrefix+channelID)
	assert.NotContains(t, members, userA, "userA should have been removed by SREM")
	assert.Contains(t, members, userB, "userB should still be in the set")
}

func TestRedisSubscribe_MultipleChannels(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })

	h := newTestHub(rc)
	userID := "user-multi"

	h.redisSubscribe("chan-1", userID)
	h.redisSubscribe("chan-2", userID)

	assert.Contains(t, mustMembers(t, mr, wsSubscriptionKeyPrefix+"chan-1"), userID)
	assert.Contains(t, mustMembers(t, mr, wsSubscriptionKeyPrefix+"chan-2"), userID)

	h.redisUnsubscribe("chan-1", userID)
	assert.NotContains(t, mustMembers(t, mr, wsSubscriptionKeyPrefix+"chan-1"), userID)
	assert.Contains(t, mustMembers(t, mr, wsSubscriptionKeyPrefix+"chan-2"), userID)
}

// ----------------------------------------------------------------------------
// Graceful degradation: nil redisClient must not panic or cause errors
// ----------------------------------------------------------------------------

func TestGracefulDegradation_NilRedisClient(t *testing.T) {
	h := newTestHub(nil) // no Redis client

	// All Redis-touching methods must be silent no-ops when redisClient is nil.
	assert.NotPanics(t, func() { h.redisSubscribe("chan-1", "user-1") })
	assert.NotPanics(t, func() { h.redisUnsubscribe("chan-1", "user-1") })
}

func TestGracefulDegradation_SetRedisClientNil(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	h := newTestHub(rc)

	// Nullify the client after construction.
	h.SetRedisClient(nil)

	assert.NotPanics(t, func() { h.redisSubscribe("chan-x", "u1") })
	assert.NotPanics(t, func() { h.redisUnsubscribe("chan-x", "u1") })

	// No key should have been created in miniredis.
	members, _ := mr.Members(wsSubscriptionKeyPrefix + "chan-x")
	assert.Empty(t, members, "no Redis key should be written when client is nil")
}
