package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

// newBroadcastConnPair is a variant of newConnectedWSConnPair (see
// websocket_helpers_test.go) that leaves the client side free for the test
// to read explicitly instead of draining it in a background goroutine — the
// broadcast/notification tests need to inspect exactly what the hub wrote.
//
// The server side still runs a passive read loop purely to keep the HTTP
// handler (and therefore serverConn) alive until the client closes; the
// hub under test never reads from serverConn in these tests, it only writes.
func newBroadcastConnPair(t *testing.T) (serverConn, clientConn *websocket.Conn, cleanup func()) {
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

	cleanup = func() {
		_ = cc.CloseNow()
		<-serverDone
		srv.Close()
	}
	return sc, cc, cleanup
}

// readWSMessage reads exactly one WSMessage from conn, failing the test if
// none arrives within the timeout.
func readWSMessage(t *testing.T, conn *websocket.Conn) WSMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var msg WSMessage
	require.NoError(t, wsjson.Read(ctx, conn, &msg))
	return msg
}

// assertNoWSMessage fails the test if a message arrives on conn within a
// short window — used for negative delivery assertions (excluded/unrelated
// recipients must not receive anything).
func assertNoWSMessage(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var msg WSMessage
	err := wsjson.Read(ctx, conn, &msg)
	assert.Error(t, err, "expected no message to arrive, got %+v", msg)
}

func rawMessageField(t *testing.T, raw json.RawMessage, key string) interface{} {
	t.Helper()
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload[key]
}

// ----------------------------------------------------------------------------
// sendToUser
// ----------------------------------------------------------------------------

func TestSendToUser_DeliversToConnectedUser(t *testing.T) {
	h := newTestHub(nil)
	serverConn, clientConn, cleanup := newBroadcastConnPair(t)
	defer cleanup()
	h.connections["user-1"] = map[*websocket.Conn]struct{}{serverConn: {}}

	h.sendToUser(context.Background(), "user-1", WSMessage{Type: "custom.event", UserID: "user-1"})

	msg := readWSMessage(t, clientConn)
	assert.Equal(t, "custom.event", msg.Type)
	assert.Equal(t, "user-1", msg.UserID)
}

func TestSendToUser_NoConnections_NoPanic(t *testing.T) {
	// Grenzfall: the recipient has no open connections at all.
	h := newTestHub(nil)

	assert.NotPanics(t, func() {
		h.sendToUser(context.Background(), "nobody-connected", WSMessage{Type: "custom.event"})
	})
}

// ----------------------------------------------------------------------------
// broadcastToChannel / broadcastToChannelExcept
// ----------------------------------------------------------------------------

func TestBroadcastToChannel_DeliversToAllMembers(t *testing.T) {
	h := newTestHub(nil)
	sc1, cc1, cleanup1 := newBroadcastConnPair(t)
	defer cleanup1()
	sc2, cc2, cleanup2 := newBroadcastConnPair(t)
	defer cleanup2()

	h.connections["user-1"] = map[*websocket.Conn]struct{}{sc1: {}}
	h.connections["user-2"] = map[*websocket.Conn]struct{}{sc2: {}}
	h.channelMembers["chan-1"] = map[string]struct{}{"user-1": {}, "user-2": {}}

	h.broadcastToChannel(context.Background(), "chan-1", WSMessage{Type: "custom.event", ChannelID: "chan-1"})

	msg1 := readWSMessage(t, cc1)
	msg2 := readWSMessage(t, cc2)
	assert.Equal(t, "chan-1", msg1.ChannelID)
	assert.Equal(t, "chan-1", msg2.ChannelID)
}

func TestBroadcastToChannelExcept_ExcludesGivenUser(t *testing.T) {
	h := newTestHub(nil)
	sc1, cc1, cleanup1 := newBroadcastConnPair(t)
	defer cleanup1()
	sc2, cc2, cleanup2 := newBroadcastConnPair(t)
	defer cleanup2()

	h.connections["user-1"] = map[*websocket.Conn]struct{}{sc1: {}}
	h.connections["user-2"] = map[*websocket.Conn]struct{}{sc2: {}}
	h.channelMembers["chan-1"] = map[string]struct{}{"user-1": {}, "user-2": {}}

	h.broadcastToChannelExcept(context.Background(), "chan-1", "user-1", WSMessage{Type: "typing", ChannelID: "chan-1"})

	msg2 := readWSMessage(t, cc2)
	assert.Equal(t, "typing", msg2.Type)
	assertNoWSMessage(t, cc1)
}

// ----------------------------------------------------------------------------
// BroadcastMessageCreated / Updated / Deleted
// ----------------------------------------------------------------------------

func TestBroadcastMessageCreated_WireShape(t *testing.T) {
	h := newTestHub(nil)
	serverConn, clientConn, cleanup := newBroadcastConnPair(t)
	defer cleanup()
	h.connections["user-1"] = map[*websocket.Conn]struct{}{serverConn: {}}
	h.channelMembers["chan-1"] = map[string]struct{}{"user-1": {}}

	h.BroadcastMessageCreated(context.Background(), "chan-1", &chatv1.MessageInfo{
		Id:        "msg-1",
		ChannelId: "chan-1",
		Content:   "hello",
	})

	msg := readWSMessage(t, clientConn)
	assert.Equal(t, WSMessageNew, msg.Type)
	assert.Equal(t, "chan-1", msg.ChannelID)
	require.NotNil(t, msg.Message)
	assert.Equal(t, "msg-1", rawMessageField(t, msg.Message, "id"))
	assert.Equal(t, "hello", rawMessageField(t, msg.Message, "content"))
}

func TestBroadcastMessageUpdated_WireShape(t *testing.T) {
	h := newTestHub(nil)
	serverConn, clientConn, cleanup := newBroadcastConnPair(t)
	defer cleanup()
	h.connections["user-1"] = map[*websocket.Conn]struct{}{serverConn: {}}
	h.channelMembers["chan-1"] = map[string]struct{}{"user-1": {}}

	h.BroadcastMessageUpdated(context.Background(), "chan-1", &chatv1.MessageInfo{
		Id:        "msg-1",
		ChannelId: "chan-1",
		Content:   "edited",
	})

	msg := readWSMessage(t, clientConn)
	assert.Equal(t, WSMessageUpdated, msg.Type)
	assert.Equal(t, "edited", rawMessageField(t, msg.Message, "content"))
}

func TestBroadcastMessageDeleted_WireShape(t *testing.T) {
	h := newTestHub(nil)
	serverConn, clientConn, cleanup := newBroadcastConnPair(t)
	defer cleanup()
	h.connections["user-1"] = map[*websocket.Conn]struct{}{serverConn: {}}
	h.channelMembers["chan-1"] = map[string]struct{}{"user-1": {}}

	h.BroadcastMessageDeleted(context.Background(), "chan-1", "msg-1")

	msg := readWSMessage(t, clientConn)
	assert.Equal(t, WSMessageDeleted, msg.Type)
	assert.Equal(t, "chan-1", msg.ChannelID)
	assert.Equal(t, "msg-1", msg.MessageID)
	assert.Nil(t, msg.Message, "deleted event carries no message payload")
}

// ----------------------------------------------------------------------------
// SendNotificationToUser / SendNotificationRead / SendNotificationReadAll /
// SendNotificationUnreadCount
// ----------------------------------------------------------------------------

func TestSendNotificationToUser_WireShape(t *testing.T) {
	h := newTestHub(nil)
	serverConn, clientConn, cleanup := newBroadcastConnPair(t)
	defer cleanup()
	h.connections["user-1"] = map[*websocket.Conn]struct{}{serverConn: {}}

	h.SendNotificationToUser(context.Background(), "user-1", map[string]string{"title": "New lead"}, true, "chime")

	msg := readWSMessage(t, clientConn)
	assert.Equal(t, WSNotificationNew, msg.Type)
	assert.Equal(t, true, rawMessageField(t, msg.Message, "desktop_push"))
	assert.Equal(t, "chime", rawMessageField(t, msg.Message, "sound"))
	notification, ok := rawMessageField(t, msg.Message, "notification").(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "New lead", notification["title"])
}

func TestSendNotificationRead_WireShape(t *testing.T) {
	h := newTestHub(nil)
	serverConn, clientConn, cleanup := newBroadcastConnPair(t)
	defer cleanup()
	h.connections["user-1"] = map[*websocket.Conn]struct{}{serverConn: {}}

	h.SendNotificationRead(context.Background(), "user-1", "notif-1")

	msg := readWSMessage(t, clientConn)
	assert.Equal(t, WSNotificationRead, msg.Type)
	assert.Equal(t, "notif-1", rawMessageField(t, msg.Message, "notification_id"))
}

func TestSendNotificationReadAll_WireShape(t *testing.T) {
	h := newTestHub(nil)
	serverConn, clientConn, cleanup := newBroadcastConnPair(t)
	defer cleanup()
	h.connections["user-1"] = map[*websocket.Conn]struct{}{serverConn: {}}

	h.SendNotificationReadAll(context.Background(), "user-1", 7)

	msg := readWSMessage(t, clientConn)
	assert.Equal(t, WSNotificationReadAll, msg.Type)
	assert.Equal(t, float64(7), rawMessageField(t, msg.Message, "marked_count"))
}

func TestSendNotificationUnreadCount_WireShape(t *testing.T) {
	h := newTestHub(nil)
	serverConn, clientConn, cleanup := newBroadcastConnPair(t)
	defer cleanup()
	h.connections["user-1"] = map[*websocket.Conn]struct{}{serverConn: {}}

	h.SendNotificationUnreadCount(context.Background(), "user-1", 3)

	msg := readWSMessage(t, clientConn)
	assert.Equal(t, WSNotificationUnreadCount, msg.Type)
	assert.Equal(t, float64(3), rawMessageField(t, msg.Message, "unread_count"))
}

// ----------------------------------------------------------------------------
// BroadcastPresenceUpdate
// ----------------------------------------------------------------------------

func TestBroadcastPresenceUpdate_DeliversToSubscribersOnly(t *testing.T) {
	h := newTestHub(nil)
	subSc, subCc, subCleanup := newBroadcastConnPair(t)
	defer subCleanup()
	otherSc, otherCc, otherCleanup := newBroadcastConnPair(t)
	defer otherCleanup()

	h.connections["subscriber-1"] = map[*websocket.Conn]struct{}{subSc: {}}
	h.connections["not-a-subscriber"] = map[*websocket.Conn]struct{}{otherSc: {}}
	h.presenceSubscribers["target-user"] = map[string]struct{}{"subscriber-1": {}}

	h.BroadcastPresenceUpdate(context.Background(), "target-user", "online")

	msg := readWSMessage(t, subCc)
	assert.Equal(t, WSPresenceUpdate, msg.Type)
	assert.Equal(t, "target-user", msg.UserID)
	assert.Equal(t, "online", rawMessageField(t, msg.Message, "status"))
	assertNoWSMessage(t, otherCc)
}

func TestBroadcastPresenceUpdate_NoSubscribers_NoPanic(t *testing.T) {
	// Grenzfall: nobody is subscribed to this user's presence.
	h := newTestHub(nil)

	assert.NotPanics(t, func() {
		h.BroadcastPresenceUpdate(context.Background(), "lonely-user", "online")
	})
}

// ----------------------------------------------------------------------------
// BroadcastCallIncoming / BroadcastCallEnded
// ----------------------------------------------------------------------------

func TestBroadcastCallIncoming_WireShape(t *testing.T) {
	h := newTestHub(nil)
	serverConn, clientConn, cleanup := newBroadcastConnPair(t)
	defer cleanup()
	h.connections["user-1"] = map[*websocket.Conn]struct{}{serverConn: {}}

	h.BroadcastCallIncoming(context.Background(), "user-1", map[string]interface{}{"call_id": "call-1"})

	msg := readWSMessage(t, clientConn)
	assert.Equal(t, WSCallIncoming, msg.Type)
	assert.Equal(t, "call-1", rawMessageField(t, msg.Message, "call_id"))
}

func TestBroadcastCallEnded_WireShape(t *testing.T) {
	h := newTestHub(nil)
	serverConn, clientConn, cleanup := newBroadcastConnPair(t)
	defer cleanup()
	h.connections["user-1"] = map[*websocket.Conn]struct{}{serverConn: {}}

	h.BroadcastCallEnded(context.Background(), "user-1", map[string]interface{}{"call_id": "call-1", "reason": "declined"})

	msg := readWSMessage(t, clientConn)
	assert.Equal(t, WSCallEnded, msg.Type)
	assert.Equal(t, "declined", rawMessageField(t, msg.Message, "reason"))
}

// ----------------------------------------------------------------------------
// BroadcastRecordingStarted
// ----------------------------------------------------------------------------

func TestBroadcastRecordingStarted_DeliversToAllConnectedUsers(t *testing.T) {
	h := newTestHub(nil)
	sc1, cc1, cleanup1 := newBroadcastConnPair(t)
	defer cleanup1()
	sc2, cc2, cleanup2 := newBroadcastConnPair(t)
	defer cleanup2()

	h.connections["user-1"] = map[*websocket.Conn]struct{}{sc1: {}}
	h.connections["user-2"] = map[*websocket.Conn]struct{}{sc2: {}}

	h.BroadcastRecordingStarted(context.Background(), "meeting-1", "recording-1", "user-1")

	for _, cc := range []*websocket.Conn{cc1, cc2} {
		msg := readWSMessage(t, cc)
		assert.Equal(t, WSRecordingStarted, msg.Type)
		assert.Equal(t, "meeting-1", rawMessageField(t, msg.Message, "meeting_id"))
		assert.Equal(t, "recording-1", rawMessageField(t, msg.Message, "recording_id"))
		assert.Equal(t, "user-1", rawMessageField(t, msg.Message, "initiated_by"))
	}
}

func TestBroadcastRecordingStarted_NoConnections_NoPanic(t *testing.T) {
	// Grenzfall: no users connected at all — tenant-wide broadcast over an
	// empty connection set must not panic.
	h := newTestHub(nil)

	assert.NotPanics(t, func() {
		h.BroadcastRecordingStarted(context.Background(), "meeting-1", "recording-1", "user-1")
	})
}

// ----------------------------------------------------------------------------
// BroadcastReactionToggled
// ----------------------------------------------------------------------------

func TestBroadcastReactionToggled_WireShape(t *testing.T) {
	h := newTestHub(nil)
	serverConn, clientConn, cleanup := newBroadcastConnPair(t)
	defer cleanup()
	h.connections["user-1"] = map[*websocket.Conn]struct{}{serverConn: {}}
	h.channelMembers["chan-1"] = map[string]struct{}{"user-1": {}}

	h.BroadcastReactionToggled(context.Background(), "chan-1", map[string]interface{}{
		"message_id": "msg-1",
		"emoji":      "👍",
	})

	msg := readWSMessage(t, clientConn)
	assert.Equal(t, WSReactionToggled, msg.Type)
	assert.Equal(t, "chan-1", msg.ChannelID)
	assert.Equal(t, "msg-1", rawMessageField(t, msg.Message, "message_id"))
}

// ----------------------------------------------------------------------------
// sendError
// ----------------------------------------------------------------------------

func TestSendError_WritesErrorMessage(t *testing.T) {
	h := newTestHub(nil)
	serverConn, clientConn, cleanup := newBroadcastConnPair(t)
	defer cleanup()

	h.sendError(context.Background(), serverConn, "channel_id is required")

	msg := readWSMessage(t, clientConn)
	assert.Equal(t, WSError, msg.Type)
	assert.Equal(t, "channel_id is required", msg.Error)
}
