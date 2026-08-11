package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newConnectedWSConnPair spins up a throwaway httptest server, accepts one
// WebSocket connection and dials it from a client, returning both live
// *websocket.Conn handles. Used by register/unregister tests that need a
// real conn as a map key (register/unregister call conn.Close on cleanup).
//
// Both sides run a passive read loop (mirrors handleConnection) so that a
// close frame from either party — whether from cleanup() or from production
// code like unregisterConnection closing serverConn directly — is acked
// immediately instead of the closer blocking on coder/websocket's default
// close-handshake timeout.
func newConnectedWSConnPair(t *testing.T) (serverConn, clientConn *websocket.Conn, cleanup func()) {
	t.Helper()

	accepted := make(chan *websocket.Conn, 1)
	serverDone := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(serverDone)
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		accepted <- c
		ctx := r.Context()
		for {
			var msg WSMessage
			if readErr := wsjson.Read(ctx, c, &msg); readErr != nil {
				return
			}
		}
	}))

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	cc, resp, err := websocket.Dial(context.Background(), wsURL, nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	sc := <-accepted

	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		for {
			var msg WSMessage
			if readErr := wsjson.Read(context.Background(), cc, &msg); readErr != nil {
				return
			}
		}
	}()

	cleanup = func() {
		// CloseNow (not Close): the other side may have already closed its end
		// (e.g. unregisterConnection closing serverConn), in which case a
		// graceful Close would block waiting for a close-handshake ack.
		_ = cc.CloseNow()
		<-serverDone
		<-clientDone
		srv.Close()
	}
	return sc, cc, cleanup
}

// ----------------------------------------------------------------------------
// msgRateLimiter.allow
// ----------------------------------------------------------------------------

func TestMsgRateLimiter_Allow_WithinLimit(t *testing.T) {
	rl := &msgRateLimiter{
		tokens:     5,
		maxTokens:  5,
		refillRate: 0, // no refill during the test window
		lastRefill: time.Now(),
	}

	for i := range 5 {
		assert.True(t, rl.allow(), "call %d should be allowed within the 5-token budget", i+1)
	}
}

func TestMsgRateLimiter_Allow_ExceedsLimit(t *testing.T) {
	rl := &msgRateLimiter{
		tokens:     1,
		maxTokens:  1,
		refillRate: 0, // no refill: the single token is not replenished
		lastRefill: time.Now(),
	}

	require.True(t, rl.allow(), "first call should consume the only token")
	assert.False(t, rl.allow(), "second call should be rejected: no tokens left and no refill")
}

func TestMsgRateLimiter_Allow_RefillsOverElapsedTime(t *testing.T) {
	rl := &msgRateLimiter{
		tokens:     0,
		maxTokens:  10,
		refillRate: 100, // 100 tokens/sec
		lastRefill: time.Now().Add(-1 * time.Second),
	}

	// One second elapsed at 100 tokens/sec would refill far past maxTokens,
	// so allow() must clamp to maxTokens and still permit the call.
	assert.True(t, rl.allow(), "token bucket should have refilled (and clamped) after 1s at 100 tok/s")
	assert.LessOrEqual(t, rl.tokens, rl.maxTokens, "tokens must never exceed maxTokens")
}

// ----------------------------------------------------------------------------
// extractToken
// ----------------------------------------------------------------------------

func TestExtractToken_FromSecWebSocketProtocolHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/ws", nil)
	r.Header.Set("Sec-WebSocket-Protocol", "access_token, my-jwt-value")

	assert.Equal(t, "my-jwt-value", extractToken(r))
}

func TestExtractToken_FallsBackToQueryParam(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/ws?token=query-jwt", nil)

	assert.Equal(t, "query-jwt", extractToken(r))
}

func TestExtractToken_HeaderWithOnlyMarker_FallsBackToQuery(t *testing.T) {
	// Grenzfall: header present but carries only the marker value, no JWT.
	r := httptest.NewRequest(http.MethodGet, "/ws?token=fallback-jwt", nil)
	r.Header.Set("Sec-WebSocket-Protocol", "access_token")

	assert.Equal(t, "fallback-jwt", extractToken(r))
}

func TestExtractToken_NoneProvided(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/ws", nil)

	assert.Equal(t, "", extractToken(r))
}

// ----------------------------------------------------------------------------
// ValidateChannelID
// ----------------------------------------------------------------------------

func TestValidateChannelID_Valid(t *testing.T) {
	assert.True(t, ValidateChannelID(uuid.New().String()))
}

func TestValidateChannelID_Invalid(t *testing.T) {
	assert.False(t, ValidateChannelID("not-a-uuid"))
}

// ----------------------------------------------------------------------------
// wsSubscriptionKey
// ----------------------------------------------------------------------------

func TestWsSubscriptionKey_BuildsPrefixedKey(t *testing.T) {
	assert.Equal(t, "ws:subscriptions:chan-1", wsSubscriptionKey("chan-1"))
}

func TestWsSubscriptionKey_EmptyChannelID(t *testing.T) {
	// Grenzfall: an empty channel ID still produces a (degenerate) key rather
	// than panicking — callers are expected to validate the ID beforehand.
	assert.Equal(t, wsSubscriptionKeyPrefix, wsSubscriptionKey(""))
}

// ----------------------------------------------------------------------------
// mustMarshal
// ----------------------------------------------------------------------------

func TestMustMarshal_Success(t *testing.T) {
	payload := map[string]string{"foo": "bar"}

	data := mustMarshal(payload)

	assert.JSONEq(t, `{"foo":"bar"}`, string(data))
}

func TestMustMarshal_UnmarshalableValue_ReturnsNilWithoutPanic(t *testing.T) {
	// Grenzfall: channels cannot be JSON-marshalled. mustMarshal must swallow
	// the error and return nil rather than panicking.
	unmarshalable := make(chan int)

	assert.NotPanics(t, func() {
		data := mustMarshal(unmarshalable)
		assert.Nil(t, data)
	})
}

// ----------------------------------------------------------------------------
// cacheUserName / getUserName
// ----------------------------------------------------------------------------

func TestCacheUserName_NilUserInfoFunc_NoOp(t *testing.T) {
	h := newTestHub(nil)
	h.userInfoFunc = nil

	h.cacheUserName(context.Background(), "user-1")

	first, last := h.getUserName("user-1")
	assert.Empty(t, first)
	assert.Empty(t, last)
}

func TestCacheUserName_AlreadyCached_SkipsCall(t *testing.T) {
	h := newTestHub(nil)
	calls := 0
	h.userInfoFunc = func(ctx context.Context, userID string) (string, string, error) {
		calls++
		return "Should", "NotBeCalled", nil
	}
	h.userNames["user-1"] = struct{ firstName, lastName string }{"Cached", "Name"}

	h.cacheUserName(context.Background(), "user-1")

	assert.Equal(t, 0, calls, "userInfoFunc must not be invoked for an already-cached user")
	first, last := h.getUserName("user-1")
	assert.Equal(t, "Cached", first)
	assert.Equal(t, "Name", last)
}

func TestCacheUserName_ErrorFromUserInfoFunc_NotCached(t *testing.T) {
	// Grenzfall: lookup fails, nothing should be cached.
	h := newTestHub(nil)
	h.userInfoFunc = func(ctx context.Context, userID string) (string, string, error) {
		return "", "", assert.AnError
	}

	h.cacheUserName(context.Background(), "user-1")

	first, last := h.getUserName("user-1")
	assert.Empty(t, first)
	assert.Empty(t, last)
}

func TestCacheUserName_Success_CachesAndGetUserNameReturnsIt(t *testing.T) {
	h := newTestHub(nil)
	h.userInfoFunc = func(ctx context.Context, userID string) (string, string, error) {
		return "Ada", "Lovelace", nil
	}

	h.cacheUserName(context.Background(), "user-1")

	first, last := h.getUserName("user-1")
	assert.Equal(t, "Ada", first)
	assert.Equal(t, "Lovelace", last)
}

func TestGetUserName_UnknownUser_ReturnsEmptyStrings(t *testing.T) {
	h := newTestHub(nil)

	first, last := h.getUserName("never-seen")

	assert.Empty(t, first)
	assert.Empty(t, last)
}

// ----------------------------------------------------------------------------
// cleanupPresenceSubscriptions
// ----------------------------------------------------------------------------

func TestCleanupPresenceSubscriptions_RemovesFromTargetsAndOwnList(t *testing.T) {
	h := newTestHub(nil)

	// user-1 subscribes to presence updates for target-a and target-b.
	h.presenceSubscribers["target-a"] = map[string]struct{}{"user-1": {}, "user-2": {}}
	h.presenceSubscribers["target-b"] = map[string]struct{}{"user-1": {}}
	// user-1 itself has subscribers watching it.
	h.presenceSubscribers["user-1"] = map[string]struct{}{"user-3": {}}

	h.cleanupPresenceSubscriptions("user-1")

	assert.NotContains(t, h.presenceSubscribers["target-a"], "user-1")
	assert.Contains(t, h.presenceSubscribers["target-a"], "user-2", "unrelated subscriber must remain")
	_, targetBStillExists := h.presenceSubscribers["target-b"]
	assert.False(t, targetBStillExists, "target-b's subscriber set becomes empty and must be pruned")
	_, ownListStillExists := h.presenceSubscribers["user-1"]
	assert.False(t, ownListStillExists, "user-1's own subscriber list must be removed on cleanup")
}

func TestCleanupPresenceSubscriptions_UnknownUser_NoPanic(t *testing.T) {
	// Grenzfall: cleaning up a user with no subscriptions at all must be a no-op.
	h := newTestHub(nil)

	assert.NotPanics(t, func() {
		h.cleanupPresenceSubscriptions("ghost-user")
	})
}

// ----------------------------------------------------------------------------
// registerConnection / unregisterConnection
// ----------------------------------------------------------------------------

func TestRegisterConnection_AddsToConnectionsMap(t *testing.T) {
	h := newTestHub(nil)
	serverConn, _, cleanup := newConnectedWSConnPair(t)
	defer cleanup()

	h.registerConnection("user-1", serverConn)

	assert.Len(t, h.connections["user-1"], 1)
	_, ok := h.connections["user-1"][serverConn]
	assert.True(t, ok)
}

func TestUnregisterConnection_RemovesConnectionChannelMembershipAndRateLimiter(t *testing.T) {
	h := newTestHub(nil)
	serverConn, _, cleanup := newConnectedWSConnPair(t)
	defer cleanup()

	h.registerConnection("user-1", serverConn)
	h.channelMembers["chan-1"] = map[string]struct{}{"user-1": {}, "user-2": {}}
	h.rateLimiters["user-1"] = newMsgRateLimiter()
	h.presenceSubscribers["user-1"] = map[string]struct{}{"user-3": {}}

	h.unregisterConnection("user-1", serverConn)

	_, hasConn := h.connections["user-1"]
	assert.False(t, hasConn, "last connection for the user must remove the user entry entirely")

	assert.NotContains(t, h.channelMembers["chan-1"], "user-1", "user must be removed from channel membership")
	assert.Contains(t, h.channelMembers["chan-1"], "user-2", "unrelated channel member must remain")

	_, hasRateLimiter := h.rateLimiters["user-1"]
	assert.False(t, hasRateLimiter, "rate limiter must be cleared once the user has no more connections")

	_, hasOwnPresenceList := h.presenceSubscribers["user-1"]
	assert.False(t, hasOwnPresenceList, "presence cleanup must run as part of unregister")
}

func TestUnregisterConnection_OneOfMultipleConnections_KeepsUserRegistered(t *testing.T) {
	// Grenzfall: a user with two open connections should only lose the one
	// being unregistered; the rate limiter must survive since a connection remains.
	h := newTestHub(nil)
	serverConn1, _, cleanup1 := newConnectedWSConnPair(t)
	defer cleanup1()
	serverConn2, _, cleanup2 := newConnectedWSConnPair(t)
	defer cleanup2()

	h.registerConnection("user-1", serverConn1)
	h.registerConnection("user-1", serverConn2)
	h.rateLimiters["user-1"] = newMsgRateLimiter()

	h.unregisterConnection("user-1", serverConn1)

	conns, hasConn := h.connections["user-1"]
	require.True(t, hasConn, "user must remain registered while a second connection is still open")
	assert.Len(t, conns, 1)
	_, stillHasSecond := conns[serverConn2]
	assert.True(t, stillHasSecond)

	_, hasRateLimiter := h.rateLimiters["user-1"]
	assert.True(t, hasRateLimiter, "rate limiter must not be cleared while connections remain")
}

// ----------------------------------------------------------------------------
// registerGuestConnection / unregisterGuestConnection
// ----------------------------------------------------------------------------

func TestRegisterGuestConnection_AddsToGuestMaps(t *testing.T) {
	h := newTestHub(nil)
	serverConn, _, cleanup := newConnectedWSConnPair(t)
	defer cleanup()

	h.registerGuestConnection("session-1", "chan-1", serverConn)

	assert.Len(t, h.guestConnections["session-1"], 1)
	assert.Equal(t, "chan-1", h.guestChannels["session-1"])
}

func TestUnregisterGuestConnection_LastConnection_RemovesSessionEntirely(t *testing.T) {
	h := newTestHub(nil)
	serverConn, _, cleanup := newConnectedWSConnPair(t)
	defer cleanup()

	h.registerGuestConnection("session-1", "chan-1", serverConn)

	h.unregisterGuestConnection("session-1", serverConn)

	_, hasConns := h.guestConnections["session-1"]
	assert.False(t, hasConns)
	_, hasChannel := h.guestChannels["session-1"]
	assert.False(t, hasChannel, "guest channel mapping must be cleared once the session has no more connections")
}

func TestUnregisterGuestConnection_OneOfMultiple_KeepsSessionRegistered(t *testing.T) {
	// Grenzfall: a guest session with two connections keeps its channel mapping
	// until the last connection is unregistered.
	h := newTestHub(nil)
	serverConn1, _, cleanup1 := newConnectedWSConnPair(t)
	defer cleanup1()
	serverConn2, _, cleanup2 := newConnectedWSConnPair(t)
	defer cleanup2()

	h.registerGuestConnection("session-1", "chan-1", serverConn1)
	h.registerGuestConnection("session-1", "chan-1", serverConn2)

	h.unregisterGuestConnection("session-1", serverConn1)

	conns, hasConns := h.guestConnections["session-1"]
	require.True(t, hasConns)
	assert.Len(t, conns, 1)
	assert.Equal(t, "chan-1", h.guestChannels["session-1"], "channel mapping must survive while a connection remains")
}
