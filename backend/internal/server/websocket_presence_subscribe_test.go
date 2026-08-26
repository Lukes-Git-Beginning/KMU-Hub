package server

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

const (
	presenceTestTenantA = "11111111-1111-1111-1111-111111111111"
	presenceTestTenantB = "22222222-2222-2222-2222-222222222222"
)

// fakeUserInfoLookup mimics authClient.GetUser scoped by RLS: it returns an
// error unless the caller's tenant (read off the scoped context, exactly as
// the real TenantOutboundUnaryInterceptor -> TenantInboundUnaryInterceptor ->
// RLS policy chain would enforce) matches the target user's tenant.
func fakeUserInfoLookup(targetTenants map[string]string) UserInfoFunc {
	return func(ctx context.Context, userID string) (string, string, error) {
		callerTenant, _ := ctx.Value(middleware.TenantIDKey).(string)
		targetTenant, ok := targetTenants[userID]
		if !ok || callerTenant == "" || targetTenant != callerTenant {
			return "", "", errors.New("user not found")
		}
		return "First", "Last", nil
	}
}

func newPresenceTestHub(targetTenants map[string]string) *WebSocketHub {
	return NewWebSocketHub(nil, nil, fakeUserInfoLookup(targetTenants), nil)
}

func subscribeMsg(t *testing.T, userIDs ...string) *WSMessage {
	t.Helper()
	payload, err := json.Marshal(struct {
		UserIDs []string `json:"user_ids"`
	}{UserIDs: userIDs})
	require.NoError(t, err)
	return &WSMessage{Type: "presence.subscribe", Message: payload}
}

func isSubscribed(h *WebSocketHub, targetUserID, subscriberUserID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	subs, ok := h.presenceSubscribers[targetUserID]
	if !ok {
		return false
	}
	_, subscribed := subs[subscriberUserID]
	return subscribed
}

func TestHandlePresenceSubscribe_SameTenantTargetIsAllowed(t *testing.T) {
	h := newPresenceTestHub(map[string]string{
		"target-in-a": presenceTestTenantA,
	})
	h.registerUserTenant("subscriber-1", presenceTestTenantA)

	h.handlePresenceSubscribe(context.Background(), nil, "subscriber-1", subscribeMsg(t, "target-in-a"))

	assert.True(t, isSubscribed(h, "target-in-a", "subscriber-1"),
		"subscribing to a user in the caller's own tenant must succeed")
}

func TestHandlePresenceSubscribe_CrossTenantTargetIsRejected(t *testing.T) {
	h := newPresenceTestHub(map[string]string{
		"target-in-b": presenceTestTenantB,
	})
	h.registerUserTenant("subscriber-1", presenceTestTenantA)

	h.handlePresenceSubscribe(context.Background(), nil, "subscriber-1", subscribeMsg(t, "target-in-b"))

	assert.False(t, isSubscribed(h, "target-in-b", "subscriber-1"),
		"a user from tenant A must not be able to subscribe to a tenant B user's presence")
}

func TestHandlePresenceSubscribe_MixedBatchKeepsOnlySameTenantTargets(t *testing.T) {
	h := newPresenceTestHub(map[string]string{
		"target-in-a": presenceTestTenantA,
		"target-in-b": presenceTestTenantB,
	})
	h.registerUserTenant("subscriber-1", presenceTestTenantA)

	h.handlePresenceSubscribe(context.Background(), nil, "subscriber-1", subscribeMsg(t, "target-in-a", "target-in-b"))

	assert.True(t, isSubscribed(h, "target-in-a", "subscriber-1"), "same-tenant target in a mixed batch must still be admitted")
	assert.False(t, isSubscribed(h, "target-in-b", "subscriber-1"), "cross-tenant target in a mixed batch must still be rejected")
}

func TestHandlePresenceSubscribe_SelfSubscribeAlwaysAllowed(t *testing.T) {
	h := newPresenceTestHub(nil)
	// Deliberately never registered via registerUserTenant, mirroring a race
	// where a presence.subscribe message arrives before the tenant map is
	// populated. Self-subscribe must not depend on that lookup at all.

	h.handlePresenceSubscribe(context.Background(), nil, "subscriber-1", subscribeMsg(t, "subscriber-1"))

	assert.True(t, isSubscribed(h, "subscriber-1", "subscriber-1"), "a user must always be able to watch their own presence")
}

func TestHandlePresenceSubscribe_UnknownCallerTenantRejectsNonSelfTargets(t *testing.T) {
	h := newPresenceTestHub(map[string]string{
		"target-in-a": presenceTestTenantA,
	})
	// subscriber-1 never went through registerUserTenant (e.g. connection
	// registered but the tenant map write raced) — fail closed rather than
	// admitting an unverifiable target.

	h.handlePresenceSubscribe(context.Background(), nil, "subscriber-1", subscribeMsg(t, "target-in-a"))

	assert.False(t, isSubscribed(h, "target-in-a", "subscriber-1"),
		"an unknown caller tenant must reject non-self targets instead of admitting them")
}

func TestHandlePresenceSubscribe_NilUserInfoFuncRejectsNonSelfTargets(t *testing.T) {
	h := NewWebSocketHub(nil, nil, nil, nil)
	h.registerUserTenant("subscriber-1", presenceTestTenantA)

	h.handlePresenceSubscribe(context.Background(), nil, "subscriber-1", subscribeMsg(t, "target-in-a"))

	assert.False(t, isSubscribed(h, "target-in-a", "subscriber-1"),
		"without a lookup function to verify tenant membership, non-self targets must be rejected, not admitted")
}

func TestUnregisterConnection_ClearsUserTenantAfterLastConnection(t *testing.T) {
	h := newPresenceTestHub(map[string]string{"target-in-a": presenceTestTenantA})
	h.registerUserTenant("subscriber-1", presenceTestTenantA)

	serverConn, _, cleanup := newConnectedWSConnPair(t)
	defer cleanup()

	h.registerConnection("subscriber-1", serverConn)
	h.unregisterConnection("subscriber-1", serverConn)

	h.handlePresenceSubscribe(context.Background(), nil, "subscriber-1", subscribeMsg(t, "target-in-a"))

	assert.False(t, isSubscribed(h, "target-in-a", "subscriber-1"),
		"once the user's last connection closes, their cached tenant must be cleared so a stale subscribe cannot use it")
}
