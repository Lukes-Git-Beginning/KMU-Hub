package gateway

// The desktop sends, edits and deletes chat messages over REST and learns about
// everyone else's writes only through the WebSocket hub. These tests drive the
// real hub (server.WebSocketHub over a real socket) next to the REST handlers:
// B subscribes to the channel, A writes over REST, B has to see the event.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/server"
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

const (
	rtChannelID = "44444444-4444-4444-8444-444444444444"
	rtMessageID = "55555555-5555-4555-8555-555555555555"
	rtParentID  = "66666666-6666-4666-8666-666666666666"
)

var (
	rtWriterID    = uuid.MustParse("77777777-7777-4777-8777-777777777777")
	rtReaderID    = uuid.MustParse("88888888-8888-4888-8888-888888888888")
	rtStrangerID  = uuid.MustParse("99999999-9999-4999-8999-999999999999")
	rtOtherTenant = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
)

// realtimeChatStub plays the chat service: the channel belongs to testTenantID
// and only writer and reader are members. GetChannel enforces both the tenant
// the gateway forwards (x-tenant-id) and the membership, as RLS plus the
// membership check do in production.
type realtimeChatStub struct {
	chatv1.UnimplementedChatServiceServer
}

func (realtimeChatStub) GetChannel(ctx context.Context, r *chatv1.GetChannelRequest) (*chatv1.GetChannelResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	tenant := md.Get("x-tenant-id")
	member := r.GetUserId() == rtWriterID.String() || r.GetUserId() == rtReaderID.String()
	if r.GetId() != rtChannelID || len(tenant) != 1 || tenant[0] != testTenantID.String() || !member {
		return nil, status.Error(codes.NotFound, "channel not found")
	}
	return &chatv1.GetChannelResponse{Channel: &chatv1.ChannelInfo{Id: rtChannelID}}, nil
}

func (realtimeChatStub) SendMessage(_ context.Context, r *chatv1.SendMessageRequest) (*chatv1.SendMessageResponse, error) {
	return &chatv1.SendMessageResponse{Message: &chatv1.MessageInfo{
		Id: rtMessageID, ChannelId: r.GetChannelId(), Content: r.GetContent(), ParentMessageId: r.ParentMessageId,
	}}, nil
}

func (realtimeChatStub) UpdateMessage(_ context.Context, r *chatv1.UpdateMessageRequest) (*chatv1.UpdateMessageResponse, error) {
	return &chatv1.UpdateMessageResponse{Message: &chatv1.MessageInfo{
		Id: r.GetId(), ChannelId: rtChannelID, Content: r.GetContent(),
	}}, nil
}

func (realtimeChatStub) DeleteMessage(context.Context, *chatv1.DeleteMessageRequest) (*chatv1.DeleteMessageResponse, error) {
	return &chatv1.DeleteMessageResponse{ChannelId: rtChannelID}, nil
}

func (realtimeChatStub) ToggleReaction(_ context.Context, r *chatv1.ToggleReactionRequest) (*chatv1.ToggleReactionResponse, error) {
	return &chatv1.ToggleReactionResponse{Added: true, ChannelId: rtChannelID}, nil
}

type realtimeEnv struct {
	routes *ChatRoutes
	hub    *server.WebSocketHub
	wsURL  string
	tokens *auth.TokenMaker
}

func newRealtimeEnv(t *testing.T) *realtimeEnv {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	grpcSrv := grpc.NewServer()
	chatv1.RegisterChatServiceServer(grpcSrv, realtimeChatStub{})
	go func() { _ = grpcSrv.Serve(lis) }()
	t.Cleanup(grpcSrv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("chat", lis.Addr().String())
	t.Cleanup(reg.Close)
	conn, err := reg.GetConnection("chat")
	require.NoError(t, err)

	tokens := auth.NewTokenMaker("realtime-test-secret-at-least-32-bytes!", time.Hour, time.Hour)
	hub := server.NewWebSocketHub(chatv1.NewChatServiceClient(conn), tokens,
		func(context.Context, string) (string, string, error) { return "", "", nil }, nil)

	// http.Server instead of httptest.Server: its Close does not wait for the
	// hijacked WebSocket connections, so a parked hub read loop cannot hang the test.
	wsLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	wsSrv := &http.Server{Handler: http.HandlerFunc(hub.HandleWebSocket), ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = wsSrv.Serve(wsLis) }()
	t.Cleanup(func() { _ = wsSrv.Close() })

	return &realtimeEnv{routes: NewChatRoutes(reg), hub: hub, wsURL: "ws://" + wsLis.Addr().String(), tokens: tokens}
}

// subscribe connects userID of tenantID and sends channel.subscribe. It returns
// once the hub has processed the subscription, and whether the hub refused it.
// The hub sends no ack, so a second, invalid subscribe follows: the hub handles
// one connection's frames in order, so its error reply proves the first is done.
func (e *realtimeEnv) subscribe(t *testing.T, userID, tenantID uuid.UUID) (*websocket.Conn, bool) {
	t.Helper()

	token, err := e.tokens.CreateAccessToken(userID, tenantID.String(), []string{"member"}, &auth.TokenPermissions{})
	require.NoError(t, err)
	conn, resp, err := websocket.Dial(context.Background(), e.wsURL, &websocket.DialOptions{
		Subprotocols: []string{"access_token", token},
	})
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	t.Cleanup(func() { _ = conn.CloseNow() })

	ctx := context.Background()
	require.NoError(t, wsjson.Write(ctx, conn, map[string]string{"type": "channel.subscribe", "channel_id": rtChannelID}))
	require.NoError(t, wsjson.Write(ctx, conn, map[string]string{"type": "channel.subscribe"}))

	refused := false
	for {
		msg := readFrame(t, conn, 2*time.Second)
		require.NotNil(t, msg, "hub never answered the subscribe")
		if msg.Type != server.WSError {
			continue
		}
		if msg.Error == "channel_id is required" {
			return conn, refused
		}
		refused = true
	}
}

// readFrame returns the next frame, or nil once timeout passes without one.
func readFrame(t *testing.T, conn *websocket.Conn, timeout time.Duration) *server.WSMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var msg server.WSMessage
	if err := wsjson.Read(ctx, conn, &msg); err != nil {
		return nil
	}
	return &msg
}

// awaitType skips frames of other types (presence and the like) and returns
// the first frame of type want, or nil if none arrives within timeout.
func awaitType(t *testing.T, conn *websocket.Conn, want string, timeout time.Duration) *server.WSMessage {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		msg := readFrame(t, conn, time.Until(deadline))
		if msg == nil {
			return nil
		}
		if msg.Type == want {
			return msg
		}
	}
	return nil
}

// rawField decodes one top-level field of a WebSocket event's message payload.
func rawField(t *testing.T, raw json.RawMessage, key string) any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	return m[key]
}

func (e *realtimeEnv) send(t *testing.T, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	req := withChiURLParam(withAuth(httptest.NewRequest(http.MethodPost, "/x", jsonBody(t, body)), rtWriterID.String(), testTenantID), "id", rtChannelID)
	rec := httptest.NewRecorder()
	e.routes.HandleSendMessage(rec, req)
	return rec
}

func TestChatRealtime_RESTSendReachesSubscriber(t *testing.T) {
	env := newRealtimeEnv(t)
	env.routes.SetWSHub(env.hub)
	reader, refused := env.subscribe(t, rtReaderID, testTenantID)
	require.False(t, refused)

	rec := env.send(t, map[string]any{"content": "hallo"})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	msg := awaitType(t, reader, server.WSMessageNew, 2*time.Second)
	require.NotNil(t, msg, "B never received message.new for A's REST send")
	assert.Equal(t, rtChannelID, msg.ChannelID)
	assert.Equal(t, rtMessageID, rawField(t, msg.Message, "id"))
	assert.Equal(t, "hallo", rawField(t, msg.Message, "content"))
	assert.Equal(t, rtChannelID, rawField(t, msg.Message, "channel_id"), "useMessageWebSocket filters on message.channel_id")
}

func TestChatRealtime_RESTThreadReplyIsThreadEvent(t *testing.T) {
	env := newRealtimeEnv(t)
	env.routes.SetWSHub(env.hub)
	reader, _ := env.subscribe(t, rtReaderID, testTenantID)

	rec := env.send(t, map[string]any{"content": "antwort", "parent_message_id": rtParentID})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	msg := readFrame(t, reader, 2*time.Second)
	require.NotNil(t, msg)
	assert.Equal(t, server.WSThreadReplyNew, msg.Type, "a reply must not land in the channel list as message.new")
	assert.Equal(t, rtParentID, msg.ParentMessageID)
	assert.Equal(t, rtParentID, rawField(t, msg.Message, "parent_message_id"), "useThreadWebSocket filters on message.parent_message_id")
}

func TestChatRealtime_RESTEditAndDeleteReachSubscriber(t *testing.T) {
	env := newRealtimeEnv(t)
	env.routes.SetWSHub(env.hub)
	reader, _ := env.subscribe(t, rtReaderID, testTenantID)

	req := withChiURLParam(withAuth(httptest.NewRequest(http.MethodPut, "/x", jsonBody(t, map[string]any{"content": "neu"})), rtWriterID.String(), testTenantID), "id", rtMessageID)
	rec := httptest.NewRecorder()
	env.routes.HandleUpdateMessage(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	msg := awaitType(t, reader, server.WSMessageUpdated, 2*time.Second)
	require.NotNil(t, msg, "B never received message.updated")
	assert.Equal(t, rtChannelID, msg.ChannelID)
	assert.Equal(t, "neu", rawField(t, msg.Message, "content"))

	req = withChiURLParam(withAuth(httptest.NewRequest(http.MethodDelete, "/x", nil), rtWriterID.String(), testTenantID), "id", rtMessageID)
	rec = httptest.NewRecorder()
	env.routes.HandleDeleteMessage(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	msg = awaitType(t, reader, server.WSMessageDeleted, 2*time.Second)
	require.NotNil(t, msg, "B never received message.deleted")
	assert.Equal(t, rtMessageID, msg.MessageID, "useMessageWebSocket reads data.message_id")
	assert.Equal(t, rtChannelID, msg.ChannelID)
}

func TestChatRealtime_OtherTenantSubscriberGetsNothing(t *testing.T) {
	env := newRealtimeEnv(t)
	env.routes.SetWSHub(env.hub)
	reader, _ := env.subscribe(t, rtReaderID, testTenantID)
	// Same channel ID, different tenant: the hub asks the chat service, which
	// does not find the channel under that tenant, so the subscribe is refused.
	stranger, refused := env.subscribe(t, rtStrangerID, rtOtherTenant)
	assert.True(t, refused, "a foreign tenant must not subscribe to the channel")

	rec := env.send(t, map[string]any{"content": "intern"})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	require.NotNil(t, awaitType(t, reader, server.WSMessageNew, 2*time.Second), "control: the member does receive it")
	for msg := readFrame(t, stranger, 300*time.Millisecond); msg != nil; msg = readFrame(t, stranger, 300*time.Millisecond) {
		assert.NotContains(t, []string{server.WSMessageNew, server.WSThreadReplyNew, server.WSMessageUpdated, server.WSMessageDeleted}, msg.Type,
			"foreign tenant received a chat event")
	}
}

func TestChatRealtime_NoHubRepliesUnchanged(t *testing.T) {
	env := newRealtimeEnv(t)

	rec := env.send(t, map[string]any{"content": "ohne hub"})
	assert.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	req := withChiURLParam(withAuth(httptest.NewRequest(http.MethodPut, "/x", jsonBody(t, map[string]any{"content": "neu"})), rtWriterID.String(), testTenantID), "id", rtMessageID)
	rec = httptest.NewRecorder()
	env.routes.HandleUpdateMessage(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	req = withChiURLParam(withAuth(httptest.NewRequest(http.MethodDelete, "/x", nil), rtWriterID.String(), testTenantID), "id", rtMessageID)
	rec = httptest.NewRecorder()
	env.routes.HandleDeleteMessage(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "message deleted", decodeObject(t, rec)["status"], "delete reply shape is unchanged")
}

func (e *realtimeEnv) react(t *testing.T, emoji string) *httptest.ResponseRecorder {
	t.Helper()
	req := withChiURLParam(withAuth(httptest.NewRequest(http.MethodPost, "/x", jsonBody(t, map[string]any{"emoji": emoji})), rtWriterID.String(), testTenantID), "id", rtMessageID)
	rec := httptest.NewRecorder()
	e.routes.HandleToggleReaction(rec, req)
	return rec
}

func TestChatRealtime_RESTReactionReachesSubscriber(t *testing.T) {
	env := newRealtimeEnv(t)
	env.routes.SetWSHub(env.hub)
	reader, refused := env.subscribe(t, rtReaderID, testTenantID)
	require.False(t, refused)

	rec := env.react(t, "👍")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, rtChannelID, decodeObject(t, rec)["channel_id"])

	msg := awaitType(t, reader, server.WSReactionToggled, 2*time.Second)
	require.NotNil(t, msg, "B never received reaction.toggled for A's REST reaction")
	assert.Equal(t, rtChannelID, msg.ChannelID)
	assert.Equal(t, rtMessageID, rawField(t, msg.Message, "message_id"))
	assert.Equal(t, rtWriterID.String(), rawField(t, msg.Message, "user_id"))
	assert.Equal(t, "👍", rawField(t, msg.Message, "emoji"))
	assert.Equal(t, true, rawField(t, msg.Message, "added"))
}

func TestChatRealtime_ReactionWithoutHubRepliesUnchanged(t *testing.T) {
	env := newRealtimeEnv(t)

	rec := env.react(t, "👍")
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}
