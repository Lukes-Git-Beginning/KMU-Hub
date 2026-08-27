package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/middleware"
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

const wsTestSecret = "test-secret-minimum-32-characters!"

// fakeChatClient implements chatv1.ChatServiceClient by embedding the interface:
// only the RPCs the WebSocket hub actually calls are overridden, everything else
// panics with a nil-pointer dereference if a test ever reaches it (which is the
// intended signal — the hub must not call anything else from the message loop).
type fakeChatClient struct {
	chatv1.ChatServiceClient

	getChannel func(ctx context.Context, in *chatv1.GetChannelRequest) (*chatv1.GetChannelResponse, error)
}

func (f *fakeChatClient) GetChannel(ctx context.Context, in *chatv1.GetChannelRequest, _ ...grpc.CallOption) (*chatv1.GetChannelResponse, error) {
	return f.getChannel(ctx, in)
}

// fakeGuestValidator implements GuestSessionValidator with a scripted outcome.
type fakeGuestValidator struct {
	session *GuestSessionInfo
	err     error
}

func (f *fakeGuestValidator) ValidateToken(context.Context, string) (*GuestSessionInfo, error) {
	return f.session, f.err
}
func (f *fakeGuestValidator) CheckRateLimit(string) error                         { return nil }
func (f *fakeGuestValidator) UpdateLastActivity(context.Context, uuid.UUID) error { return nil }

// newAuthTestHub builds a hub with a real TokenMaker so HandleWebSocket can be
// driven end-to-end. accessTTL controls how long issued tokens stay valid.
func newAuthTestHub(t *testing.T, chatClient chatv1.ChatServiceClient, accessTTL time.Duration) (*WebSocketHub, *auth.TokenMaker) {
	t.Helper()

	tm := auth.NewTokenMaker(wsTestSecret, accessTTL, 7*24*time.Hour)
	h := newTestHub(nil)
	h.chatClient = chatClient
	h.tokenMaker = tm
	return h, tm
}

// issueToken mints an access token for userID in tenantID.
func issueToken(t *testing.T, tm *auth.TokenMaker, userID uuid.UUID, tenantID string) string {
	t.Helper()

	token, err := tm.CreateAccessToken(userID, tenantID, []string{"member"}, &auth.TokenPermissions{})
	require.NoError(t, err)
	return token
}

// serveHub exposes HandleWebSocket on a plain http.Server. httptest.Server is
// deliberately avoided: its Close waits for in-flight handlers, and a hub
// connection whose read-loop is still parked would hang the test. http.Server's
// Close knows nothing about hijacked connections and therefore never blocks.
func serveHub(t *testing.T, h *WebSocketHub) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := &http.Server{
		Handler:           http.HandlerFunc(h.HandleWebSocket),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	return "ws://" + ln.Addr().String()
}

// dialHub opens a client connection carrying token via Sec-WebSocket-Protocol.
func dialHub(t *testing.T, wsURL, token string) (*websocket.Conn, *http.Response) {
	t.Helper()

	conn, resp, err := websocket.Dial(context.Background(), wsURL, &websocket.DialOptions{
		Subprotocols: []string{wsAuthProtocol, token},
	})
	require.NoError(t, err)
	return conn, resp
}

// getStatus performs a plain (non-upgrading) GET and returns the status code
// HandleWebSocket rejected the handshake with.
func getStatus(t *testing.T, wsURL, query string) int {
	t.Helper()

	resp, err := http.Get(strings.Replace(wsURL, "ws://", "http://", 1) + query) //nolint:noctx // short-lived test request
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode
}

// waitFor polls cond every millisecond until it holds or the timeout expires.
func waitFor(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return cond()
}

func (h *WebSocketHub) connCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections[userID])
}

// newWSPairReadable is newConnectedWSConnPair without the passive client-side
// read loop: tests that assert on what the hub pushes must own the client
// reader themselves, otherwise the helper's loop consumes the frame first.
func newWSPairReadable(t *testing.T) (serverConn, clientConn *websocket.Conn) {
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
		for {
			var msg WSMessage
			if readErr := wsjson.Read(r.Context(), c, &msg); readErr != nil {
				return
			}
		}
	}))

	cc, resp, err := websocket.Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	t.Cleanup(func() {
		_ = cc.CloseNow()
		<-serverDone
		srv.Close()
	})

	return <-accepted, cc
}

// ---------------------------------------------------------------------------
// Connection setup — rejection paths
// ---------------------------------------------------------------------------

func TestHandleWebSocket_NoToken_Returns401(t *testing.T) {
	h, _ := newAuthTestHub(t, nil, time.Hour)
	wsURL := serveHub(t, h)

	assert.Equal(t, http.StatusUnauthorized, getStatus(t, wsURL, "/api/v1/ws"))
}

func TestHandleWebSocket_InvalidToken_Returns401(t *testing.T) {
	h, _ := newAuthTestHub(t, nil, time.Hour)
	wsURL := serveHub(t, h)

	assert.Equal(t, http.StatusUnauthorized, getStatus(t, wsURL, "/api/v1/ws?token=not-a-jwt"))
}

func TestHandleWebSocket_ExpiredToken_RejectedAtHandshake(t *testing.T) {
	// Grenzfall: a structurally valid but expired JWT must never reach Accept.
	h, tm := newAuthTestHub(t, nil, time.Nanosecond)
	token := issueToken(t, tm, uuid.New(), uuid.New().String())
	time.Sleep(2 * time.Millisecond)
	wsURL := serveHub(t, h)

	assert.Equal(t, http.StatusUnauthorized, getStatus(t, wsURL, "/api/v1/ws?token="+token))
}

func TestHandleWebSocket_ConnectionLimitExceeded_Returns429(t *testing.T) {
	h, tm := newAuthTestHub(t, nil, time.Hour)
	userID := uuid.New()
	token := issueToken(t, tm, userID, uuid.New().String())

	// Pre-fill the connection map up to maxConnsPerUser without opening real
	// sockets: HandleWebSocket only reads len(h.connections[userID]).
	h.connections[userID.String()] = make(map[*websocket.Conn]struct{})
	for range maxConnsPerUser {
		h.connections[userID.String()][&websocket.Conn{}] = struct{}{}
	}

	wsURL := serveHub(t, h)
	assert.Equal(t, http.StatusTooManyRequests, getStatus(t, wsURL, "/api/v1/ws?token="+token))
}

func TestHandleWebSocket_GuestTokenWithoutGuestService_Returns503(t *testing.T) {
	h, _ := newAuthTestHub(t, nil, time.Hour)
	wsURL := serveHub(t, h)

	assert.Equal(t, http.StatusServiceUnavailable, getStatus(t, wsURL, "/api/v1/ws?guest_token=whatever"))
}

func TestHandleWebSocket_GuestTokenInvalid_Returns401(t *testing.T) {
	h, _ := newAuthTestHub(t, nil, time.Hour)
	h.SetGuestService(&fakeGuestValidator{err: assert.AnError})
	wsURL := serveHub(t, h)

	assert.Equal(t, http.StatusUnauthorized, getStatus(t, wsURL, "/api/v1/ws?guest_token=stale"))
}

// ---------------------------------------------------------------------------
// Connection setup — accepted path, then the message loop
// ---------------------------------------------------------------------------

func TestHandleWebSocket_ValidToken_RegistersConnectionAndTenant(t *testing.T) {
	h, tm := newAuthTestHub(t, nil, time.Hour)
	userID := uuid.New()
	tenantID := uuid.New().String()
	wsURL := serveHub(t, h)

	client, resp := dialHub(t, wsURL, issueToken(t, tm, userID, tenantID))
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	defer func() { _ = client.CloseNow() }()

	require.True(t, waitFor(time.Second, func() bool { return h.connCount(userID.String()) == 1 }),
		"the accepted connection must be registered under the caller's user ID")

	h.mu.RLock()
	gotTenant := h.userTenants[userID.String()]
	h.mu.RUnlock()
	assert.Equal(t, tenantID, gotTenant,
		"the tenant from the JWT tid claim must be recorded on connect")
}

func TestHandleWebSocket_MessageLoop_UnknownType_ReturnsErrorFrame(t *testing.T) {
	h, tm := newAuthTestHub(t, nil, time.Hour)
	wsURL := serveHub(t, h)

	client, resp := dialHub(t, wsURL, issueToken(t, tm, uuid.New(), uuid.New().String()))
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	defer func() { _ = client.CloseNow() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, wsjson.Write(ctx, client, WSMessage{Type: "does.not.exist"}))

	var got WSMessage
	require.NoError(t, wsjson.Read(ctx, client, &got))
	assert.Equal(t, WSError, got.Type)
	assert.Equal(t, "unknown message type: does.not.exist", got.Error)
}

func TestHandleWebSocket_MessageLoop_RateLimitExceeded_ReturnsErrorFrame(t *testing.T) {
	h, tm := newAuthTestHub(t, nil, time.Hour)
	userID := uuid.New()
	wsURL := serveHub(t, h)

	client, resp := dialHub(t, wsURL, issueToken(t, tm, userID, uuid.New().String()))
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	defer func() { _ = client.CloseNow() }()

	require.True(t, waitFor(time.Second, func() bool { return h.connCount(userID.String()) == 1 }))

	// Drain the user's bucket from the test side before sending: the limiter is
	// keyed by user ID, so the connection inherits an already-empty bucket.
	h.mu.Lock()
	h.rateLimiters[userID.String()] = &msgRateLimiter{tokens: 0, maxTokens: msgBurstSize, refillRate: 0, lastRefill: time.Now()}
	h.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, wsjson.Write(ctx, client, WSMessage{Type: WSTypingStart, ChannelID: "chan-1"}))

	var got WSMessage
	require.NoError(t, wsjson.Read(ctx, client, &got))
	assert.Equal(t, WSError, got.Type)
	assert.Equal(t, "rate limit exceeded", got.Error)
}

// ---------------------------------------------------------------------------
// Teardown: expired token, goroutine release, state cleanup
// ---------------------------------------------------------------------------

// TestHandleWebSocket_HardDisconnect_CleanupWaitsForTokenRevalidation pins down
// the actual teardown order after a hard abort (no close handshake, the socket
// just dies). It asserts two things in sequence, and the second only means
// something once the first has held:
//
//	(1) the read-loop error alone releases NOTHING. handleConnection blocks on
//	    <-done until the revalidation goroutine returns, and that goroutine only
//	    returns on ctx.Done() or on an invalid token. r.Context() belongs to a
//	    hijacked connection and is not cancelled when the socket dies, so in
//	    production the connection, its cached tenant, its rate-limiter bucket,
//	    its channel memberships and both goroutines survive the disconnect for
//	    up to wsTokenRevalidateInterval (5 minutes). With maxConnsPerUser = 5
//	    that also means five flaky reconnects can lock a user out with 429.
//	    Tracked as backlog unit fix-websocket-hard-disconnect-cleanup-delay.
//	(2) once the token is no longer valid, the revalidation tick closes the
//	    connection, which unblocks handleConnection, runs the deferred
//	    unregisterConnection and returns every goroutine of this connection.
//
// The test runs with a short revalidation interval and a short-lived token so
// both phases fit into a couple of seconds; production uses 5 minutes.
func TestHandleWebSocket_HardDisconnect_CleanupWaitsForTokenRevalidation(t *testing.T) {
	// 2s TTL, not milliseconds: JWT exp has second granularity, so anything
	// below a second can already be expired at handshake time.
	h, tm := newAuthTestHub(t, nil, 2*time.Second)
	h.revalidateInterval = 100 * time.Millisecond
	userID := uuid.New()
	wsURL := serveHub(t, h)

	before := runtime.NumGoroutine()

	client, resp := dialHub(t, wsURL, issueToken(t, tm, userID, uuid.New().String()))
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	require.True(t, waitFor(time.Second, func() bool { return h.connCount(userID.String()) == 1 }))
	h.mu.Lock()
	h.channelMembers["chan-1"] = map[string]struct{}{userID.String(): {}}
	h.rateLimiters[userID.String()] = newMsgRateLimiter()
	h.mu.Unlock()

	require.NoError(t, client.CloseNow())

	// (1) Several revalidation ticks pass while the token is still valid — and
	// nothing is released, because the read-loop error alone cannot get past
	// <-done.
	time.Sleep(400 * time.Millisecond)
	assert.Equal(t, 1, h.connCount(userID.String()),
		"a hard disconnect releases nothing on its own: handleConnection stays parked on <-done")
	h.mu.RLock()
	_, stillMember := h.channelMembers["chan-1"][userID.String()]
	h.mu.RUnlock()
	assert.True(t, stillMember, "channel membership also survives the dead socket until revalidation fires")

	// (2) Token expires -> revalidation closes the connection -> cleanup runs.
	require.True(t, waitFor(5*time.Second, func() bool { return h.connCount(userID.String()) == 0 }),
		"the expired token must terminate the dead connection and unregister it")

	h.mu.RLock()
	_, tenantLeft := h.userTenants[userID.String()]
	_, limiterLeft := h.rateLimiters[userID.String()]
	_, memberLeft := h.channelMembers["chan-1"][userID.String()]
	h.mu.RUnlock()
	assert.False(t, tenantLeft, "cached tenant must be dropped with the last connection")
	assert.False(t, limiterLeft, "rate limiter must be dropped with the last connection")
	assert.False(t, memberLeft, "channel membership must be dropped with the last connection")

	// Goroutine budget: read loop and revalidation loop must both be gone. The
	// small tolerance covers the http.Server's own per-connection bookkeeping,
	// which winds down asynchronously.
	require.True(t, waitFor(5*time.Second, func() bool { return runtime.NumGoroutine() <= before+2 }),
		"goroutines must return to the pre-connect baseline (before=%d, now=%d)", before, runtime.NumGoroutine())
}

// TestHandleWebSocket_TokenExpiresMidSession_ClosesWithPolicyViolation proves
// end-to-end — through HandleWebSocket, not just the isolated loop — that a
// token which stops being valid while the connection is open terminates that
// connection. A WebSocket authenticates once and then lives for hours, so this
// is the only thing standing between a revoked token and an open session.
func TestHandleWebSocket_TokenExpiresMidSession_ClosesWithPolicyViolation(t *testing.T) {
	h, tm := newAuthTestHub(t, nil, 2*time.Second)
	h.revalidateInterval = 100 * time.Millisecond
	userID := uuid.New()
	wsURL := serveHub(t, h)

	client, resp := dialHub(t, wsURL, issueToken(t, tm, userID, uuid.New().String()))
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	defer func() { _ = client.CloseNow() }()

	require.True(t, waitFor(time.Second, func() bool { return h.connCount(userID.String()) == 1 }),
		"the connection must be accepted while the token is still valid")

	readCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var msg WSMessage
	err := wsjson.Read(readCtx, client, &msg)
	require.Error(t, err, "the server must close the connection once the token expires")
	assert.Equal(t, websocket.StatusPolicyViolation, websocket.CloseStatus(err),
		"an expired token must close with PolicyViolation, distinct from a normal closure")

	assert.True(t, waitFor(3*time.Second, func() bool { return h.connCount(userID.String()) == 0 }),
		"the closed connection must be unregistered")
}

// ---------------------------------------------------------------------------
// Rate limiter scope: per user — not global, not per connection
// ---------------------------------------------------------------------------

func TestCheckMsgRate_IsPerUser_NotGlobal(t *testing.T) {
	h := newTestHub(nil)

	for i := range msgBurstSize {
		require.True(t, h.checkMsgRate("user-a"), "call %d of user-a must fit into the burst budget", i+1)
	}
	require.False(t, h.checkMsgRate("user-a"), "user-a's own budget must now be exhausted")

	assert.True(t, h.checkMsgRate("user-b"),
		"user-b must be unaffected: an exhausted budget of one user may never throttle another")
}

func TestCheckMsgRate_IsPerUser_NotPerConnection(t *testing.T) {
	// Two connections of the same user share one bucket: checkMsgRate is keyed
	// by user ID only, so a user cannot multiply their allowance by opening
	// more connections (up to maxConnsPerUser).
	h := newTestHub(nil)
	userID := uuid.New().String()

	for range msgBurstSize {
		require.True(t, h.checkMsgRate(userID))
	}

	h.mu.RLock()
	limiterCount := len(h.rateLimiters)
	h.mu.RUnlock()
	assert.Equal(t, 1, limiterCount, "one bucket per user, regardless of connection count")
	assert.False(t, h.checkMsgRate(userID),
		"a second connection of the same user draws from the same exhausted bucket")
}

func TestCheckMsgRate_ReconnectResetsTheBucket(t *testing.T) {
	// Documents the current behaviour: unregisterConnection deletes the bucket
	// once the user's last connection closes, so reconnecting hands out a fresh
	// burst allowance. See backlog unit fix-websocket-rate-limit-reset-on-reconnect.
	h := newTestHub(nil)
	serverConn, _, cleanup := newConnectedWSConnPair(t)
	defer cleanup()

	userID := uuid.New().String()
	h.registerConnection(userID, serverConn)
	for range msgBurstSize {
		require.True(t, h.checkMsgRate(userID))
	}
	require.False(t, h.checkMsgRate(userID))

	h.unregisterConnection(userID, serverConn)

	assert.True(t, h.checkMsgRate(userID),
		"after the last connection closed the bucket is gone, so the next connect starts from a full burst")
}

// ---------------------------------------------------------------------------
// Broadcast and tenant isolation
// ---------------------------------------------------------------------------

// TestHandleSubscribeChannel_SendsContextWithoutTenantToChatService reproduces a
// production defect: /api/v1/ws is registered without authMiddleware
// (cmd/gateway/main.go), so r.Context() carries no middleware.TenantIDKey. The
// hub hands that bare context to the chat client, and ChatGRPCServer.GetChannel
// (internal/server/chat_grpc.go) rejects a missing tenant with Unauthenticated.
// The fake below asserts exactly what the real server checks.
func TestHandleSubscribeChannel_SendsContextWithoutTenantToChatService(t *testing.T) {
	var sawTenant bool
	chatClient := &fakeChatClient{
		getChannel: func(ctx context.Context, _ *chatv1.GetChannelRequest) (*chatv1.GetChannelResponse, error) {
			if _, err := middleware.GetTenantID(ctx); err == nil {
				sawTenant = true
				return &chatv1.GetChannelResponse{}, nil
			}
			// Mirrors chat_grpc.go GetChannel's own guard.
			return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
		},
	}

	h, tm := newAuthTestHub(t, chatClient, time.Hour)
	userID := uuid.New()
	channelID := uuid.New().String()
	wsURL := serveHub(t, h)

	client, resp := dialHub(t, wsURL, issueToken(t, tm, userID, uuid.New().String()))
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	defer func() { _ = client.CloseNow() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, wsjson.Write(ctx, client, WSMessage{Type: WSSubscribeChannel, ChannelID: channelID}))

	var got WSMessage
	require.NoError(t, wsjson.Read(ctx, client, &got))
	assert.Equal(t, WSError, got.Type)
	assert.Equal(t, "cannot subscribe to channel", got.Error)
	assert.False(t, sawTenant,
		"the WebSocket path carries no tenant into the gRPC context — every chat RPC from the message loop fails closed")

	h.mu.RLock()
	_, subscribed := h.channelMembers[channelID]
	h.mu.RUnlock()
	assert.False(t, subscribed, "a rejected GetChannel must not create a subscription")
}

func TestHandleSubscribeChannel_EmptyChannelID_ReturnsErrorFrame(t *testing.T) {
	h, tm := newAuthTestHub(t, &fakeChatClient{}, time.Hour)
	wsURL := serveHub(t, h)

	client, resp := dialHub(t, wsURL, issueToken(t, tm, uuid.New(), uuid.New().String()))
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	defer func() { _ = client.CloseNow() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, wsjson.Write(ctx, client, WSMessage{Type: WSSubscribeChannel}))

	var got WSMessage
	require.NoError(t, wsjson.Read(ctx, client, &got))
	assert.Equal(t, WSError, got.Type)
	assert.Equal(t, "channel_id is required", got.Error)
}

// TestHandleSubscribeChannel_ForeignTenantRejected_ReceivesNoBroadcast is the
// positive half: the chat service is the only thing that keeps a foreign tenant
// out of a channel's subscriber set, and when it says no, no broadcast reaches
// that user.
func TestHandleSubscribeChannel_ForeignTenantRejected_ReceivesNoBroadcast(t *testing.T) {
	ownTenant := uuid.New().String()
	channelID := uuid.New().String()

	chatClient := &fakeChatClient{
		getChannel: func(ctx context.Context, in *chatv1.GetChannelRequest) (*chatv1.GetChannelResponse, error) {
			// Stands in for the tenant-scoped repository lookup
			// (channel/postgres_repository.go: WHERE id = $1 AND tenant_id = $2).
			if tenantID, _ := middleware.GetTenantID(ctx); tenantID.String() == ownTenant {
				return &chatv1.GetChannelResponse{}, nil
			}
			return nil, status.Error(codes.NotFound, "channel not found")
		},
	}

	h := newTestHub(nil)
	h.chatClient = chatClient

	insider, insiderClient := newWSPairReadable(t)
	outsider, _, cleanupOutsider := newConnectedWSConnPair(t)
	defer cleanupOutsider()

	insiderID, outsiderID := uuid.New().String(), uuid.New().String()
	h.registerConnection(insiderID, insider)
	h.registerConnection(outsiderID, outsider)

	ownCtx := context.WithValue(context.Background(), middleware.TenantIDKey, ownTenant)
	foreignCtx := context.WithValue(context.Background(), middleware.TenantIDKey, uuid.New().String())

	h.handleSubscribeChannel(ownCtx, insider, insiderID, &WSMessage{ChannelID: channelID})
	h.handleSubscribeChannel(foreignCtx, outsider, outsiderID, &WSMessage{ChannelID: channelID})

	h.mu.RLock()
	members := len(h.channelMembers[channelID])
	_, insiderIn := h.channelMembers[channelID][insiderID]
	_, outsiderIn := h.channelMembers[channelID][outsiderID]
	h.mu.RUnlock()

	require.Equal(t, 1, members)
	assert.True(t, insiderIn, "the caller from the channel's own tenant must be subscribed")
	assert.False(t, outsiderIn, "a caller from a foreign tenant must not enter the subscriber set")

	// The broadcast reaches the insider only. Reading from the insider's client
	// side proves delivery; the outsider is not a member and gets nothing.
	h.broadcastToChannel(context.Background(), channelID, WSMessage{Type: WSMessageNew, ChannelID: channelID})

	readCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var got WSMessage
	require.NoError(t, wsjson.Read(readCtx, insiderClient, &got))
	assert.Equal(t, WSMessageNew, got.Type)
}

// TestBroadcastToChannel_HubDoesNotFilterByTenant pins down WHERE the isolation
// lives: nowhere in the broadcast path. broadcastToChannel fans out to every
// user ID in channelMembers without ever consulting userTenants, so the
// GetChannel gate in handleSubscribeChannel is the single point of enforcement.
// If a future change ever writes into channelMembers without that gate, foreign
// tenants receive the traffic — this test is here so that change breaks loudly.
func TestBroadcastToChannel_HubDoesNotFilterByTenant(t *testing.T) {
	h := newTestHub(nil)

	foreign, foreignClient := newWSPairReadable(t)

	foreignID := uuid.New().String()
	channelID := uuid.New().String()

	h.registerConnection(foreignID, foreign)
	h.registerUserTenant(foreignID, uuid.New().String())
	h.mu.Lock()
	h.channelMembers[channelID] = map[string]struct{}{foreignID: {}}
	h.mu.Unlock()

	h.broadcastToChannel(context.Background(), channelID, WSMessage{Type: WSMessageNew, ChannelID: channelID})

	readCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var got WSMessage
	require.NoError(t, wsjson.Read(readCtx, foreignClient, &got),
		"documented behaviour: membership in channelMembers is sufficient, the tenant is never re-checked here")
	assert.Equal(t, WSMessageNew, got.Type)
}

// ---------------------------------------------------------------------------
// Guest connection loop
// ---------------------------------------------------------------------------

func TestHandleGuestWebSocket_ValidSession_RegistersAndUnregisters(t *testing.T) {
	sessionID, channelID := uuid.New(), uuid.New()
	h := newTestHub(nil)
	h.SetGuestService(&fakeGuestValidator{session: &GuestSessionInfo{
		ID:          sessionID,
		ChannelID:   channelID,
		DisplayName: "Gast",
		IsActive:    true,
	}})

	srv := httptest.NewServer(http.HandlerFunc(h.HandleWebSocket))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/ws?guest_token=ok"
	client, resp, err := websocket.Dial(context.Background(), wsURL, nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	require.True(t, waitFor(time.Second, func() bool {
		h.mu.RLock()
		defer h.mu.RUnlock()
		return len(h.guestConnections[sessionID.String()]) == 1
	}), "the guest session must be registered under its session ID")

	h.mu.RLock()
	gotChannel := h.guestChannels[sessionID.String()]
	h.mu.RUnlock()
	assert.Equal(t, channelID.String(), gotChannel)

	// A graceful close wakes the guest read-loop, which has no revalidation
	// goroutine to wait for, so teardown is immediate.
	require.NoError(t, client.Close(websocket.StatusNormalClosure, "done"))
	require.True(t, waitFor(2*time.Second, func() bool {
		h.mu.RLock()
		defer h.mu.RUnlock()
		_, ok := h.guestConnections[sessionID.String()]
		return !ok
	}), "closing the guest connection must unregister the session")
}
