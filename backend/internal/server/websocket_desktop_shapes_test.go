package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin the wire shapes the desktop reads and sends (useMessages.ts typing hooks,
// PresenceProvider.tsx, useIncomingCallListener.ts). The legacy nested/content forms stay
// covered next to them so neither side breaks the other.

type stubPresenceService struct{ got string }

func (s *stubPresenceService) Heartbeat(context.Context, string) (string, error) { return "", nil }
func (s *stubPresenceService) SetManualStatus(_ context.Context, _, status string) error {
	s.got = status
	return nil
}

func TestHandleTypingIndicator_SendsInboundTypeAsEventType(t *testing.T) {
	for _, typ := range []string{WSTypingStart, WSTypingStop} {
		t.Run(typ, func(t *testing.T) {
			h := newTestHub(nil)
			senderSc, senderCc, senderCleanup := newBroadcastConnPair(t)
			defer senderCleanup()
			peerSc, peerCc, peerCleanup := newBroadcastConnPair(t)
			defer peerCleanup()
			h.connections["sender"] = map[*websocket.Conn]struct{}{senderSc: {}}
			h.connections["peer"] = map[*websocket.Conn]struct{}{peerSc: {}}
			h.channelMembers["chan-1"] = map[string]struct{}{"sender": {}, "peer": {}}
			h.userNames["sender"] = struct{ firstName, lastName string }{"Ada", "Lovelace"}

			h.handleTypingIndicator(context.Background(), "sender", &WSMessage{Type: typ, ChannelID: "chan-1"})

			msg := readWSMessage(t, peerCc)
			assert.Equal(t, typ, msg.Type, "desktop subscribes to typing.start/typing.stop as event types")
			assert.Equal(t, "chan-1", msg.ChannelID)
			assert.Equal(t, "sender", msg.UserID)
			assert.Equal(t, "Ada", msg.FirstName)
			assert.Equal(t, typ, msg.Content, "legacy readers keep the content marker")
			assertNoWSMessage(t, senderCc)
		})
	}
}

func TestBroadcastPresenceUpdate_CarriesStatusOnTopLevelAndNested(t *testing.T) {
	h := newTestHub(nil)
	subSc, subCc, cleanup := newBroadcastConnPair(t)
	defer cleanup()
	h.connections["sub"] = map[*websocket.Conn]struct{}{subSc: {}}
	h.presenceSubscribers["target"] = map[string]struct{}{"sub": {}}

	h.BroadcastPresenceUpdate(context.Background(), "target", "away")

	msg := readWSMessage(t, subCc)
	assert.Equal(t, "away", msg.Status, "PresenceProvider reads data.status")
	assert.Equal(t, "target", msg.UserID)
	assert.Equal(t, "away", rawMessageField(t, msg.Message, "status"), "legacy nested reader")
}

func TestHandlePresenceSetStatus_AcceptsStatusAndLegacyContent(t *testing.T) {
	for name, in := range map[string]*WSMessage{
		"status (desktop)": {Type: WSPresenceSetStatus, Status: "away"},
		"content (legacy)": {Type: WSPresenceSetStatus, Content: "away"},
	} {
		t.Run(name, func(t *testing.T) {
			h := newTestHub(nil)
			svc := &stubPresenceService{}
			h.SetPresenceService(svc)
			subSc, subCc, cleanup := newBroadcastConnPair(t)
			defer cleanup()
			h.connections["sub"] = map[*websocket.Conn]struct{}{subSc: {}}
			h.presenceSubscribers["me"] = map[string]struct{}{"sub": {}}

			h.handlePresenceSetStatus(context.Background(), nil, "me", in)

			assert.Equal(t, "away", svc.got)
			assert.Equal(t, "away", readWSMessage(t, subCc).Status)
		})
	}
}

func TestHandlePresenceSetStatus_MissingStatusIsRejected(t *testing.T) {
	h := newTestHub(nil)
	svc := &stubPresenceService{}
	h.SetPresenceService(svc)
	sc, cc, cleanup := newBroadcastConnPair(t)
	defer cleanup()

	h.handlePresenceSetStatus(context.Background(), sc, "me", &WSMessage{Type: WSPresenceSetStatus})

	assert.Equal(t, WSError, readWSMessage(t, cc).Type)
	assert.Empty(t, svc.got)
}

func TestHandlePresenceSubscribe_TopLevelUserIDsRespectTenant(t *testing.T) {
	h := newPresenceTestHub(map[string]string{
		"in-a":  presenceTestTenantA,
		"in-b":  presenceTestTenantB,
		"other": presenceTestTenantA,
	})
	h.registerUserTenant("sub", presenceTestTenantA)

	h.handlePresenceSubscribe(context.Background(), nil, "sub",
		&WSMessage{Type: WSPresenceSubscribe, UserIDs: []string{"in-a", "in-b"}})

	assert.True(t, isSubscribed(h, "in-a", "sub"), "desktop sends user_ids on the top level")
	assert.False(t, isSubscribed(h, "in-b", "sub"), "foreign tenant stays rejected for the flat form")
}

func TestHandlePresenceSubscribe_LegacyNestedUserIDsStillWork(t *testing.T) {
	h := newPresenceTestHub(map[string]string{"in-a": presenceTestTenantA})
	h.registerUserTenant("sub", presenceTestTenantA)

	h.handlePresenceSubscribe(context.Background(), nil, "sub", subscribeMsg(t, "in-a"))

	assert.True(t, isSubscribed(h, "in-a", "sub"))
}

func TestBroadcastCall_CarriesPayloadUnderCallAndMessage(t *testing.T) {
	data := map[string]interface{}{"call_id": "call-1", "caller_id": "u-9", "reason": "declined"}
	for name, send := range map[string]func(*WebSocketHub){
		WSCallIncoming: func(h *WebSocketHub) { h.BroadcastCallIncoming(context.Background(), "user-1", data) },
		WSCallEnded:    func(h *WebSocketHub) { h.BroadcastCallEnded(context.Background(), "user-1", data) },
	} {
		t.Run(name, func(t *testing.T) {
			h := newTestHub(nil)
			sc, cc, cleanup := newBroadcastConnPair(t)
			defer cleanup()
			h.connections["user-1"] = map[*websocket.Conn]struct{}{sc: {}}

			send(h)

			msg := readWSMessage(t, cc)
			require.Equal(t, name, msg.Type)
			assert.Equal(t, "call-1", rawMessageField(t, msg.Call, "call_id"), "useIncomingCallListener unwraps `call`")
			assert.Equal(t, "call-1", rawMessageField(t, msg.Message, "call_id"), "legacy nested reader")
			var raw map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(msg.Call, &raw))
			assert.Contains(t, raw, "caller_id")
		})
	}
}
