package gateway

// POST /api/v1/channels/dm as the desktop sends it: useStartConversation
// (desktop/src/renderer/src/api/hooks/useChannels.ts) posts
// {participant_ids:[…]} for two or more picked people and {other_user_id} for
// one. The group body has to reach GetOrCreateGroupDM, the 1:1 body
// GetOrCreateDM, and the desktop reads result.channel.id from either answer.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

const (
	groupDMChannelID = "12121212-1212-4212-8212-121212121212"
	groupDMPeerA     = "13131313-1313-4313-8313-131313131313"
	groupDMPeerB     = "14141414-1414-4414-8414-141414141414"
	groupDMCaller    = "15151515-1515-4515-8515-151515151515"
)

type groupDMChatStub struct {
	chatv1.UnimplementedChatServiceServer
	mu      sync.Mutex
	group   *chatv1.GetOrCreateGroupDMRequest
	direct  *chatv1.GetOrCreateDMRequest
	created bool
}

func (s *groupDMChatStub) GetOrCreateGroupDM(_ context.Context, r *chatv1.GetOrCreateGroupDMRequest) (*chatv1.GetOrCreateDMResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.group = r
	return &chatv1.GetOrCreateDMResponse{Channel: &chatv1.ChannelInfo{Id: groupDMChannelID, IsDm: true, MemberCount: 3}, Created: s.created}, nil
}

func (s *groupDMChatStub) GetOrCreateDM(_ context.Context, r *chatv1.GetOrCreateDMRequest) (*chatv1.GetOrCreateDMResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.direct = r
	return &chatv1.GetOrCreateDMResponse{Channel: &chatv1.ChannelInfo{Id: groupDMChannelID, IsDm: true, MemberCount: 2}, Created: s.created}, nil
}

func newGroupDMRoutes(t *testing.T) (*ChatRoutes, *groupDMChatStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &groupDMChatStub{}
	srv := grpc.NewServer()
	chatv1.RegisterChatServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("chat", lis.Addr().String())
	t.Cleanup(reg.Close)
	return NewChatRoutes(reg), stub
}

func postDM(t *testing.T, routes *ChatRoutes, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/channels/dm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenantID(withUserID(req, groupDMCaller), testTenantID)
	rec := httptest.NewRecorder()
	routes.HandleGetOrCreateDM(rec, req)
	return rec
}

func TestChatGroupDM_DesktopBodyReachesGroupRPC(t *testing.T) {
	routes, stub := newGroupDMRoutes(t)

	stub.created = true
	rec := postDM(t, routes, `{"participant_ids":["`+groupDMPeerA+`","`+groupDMPeerB+`"]}`)
	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.group, "the group body must reach GetOrCreateGroupDM")
	assert.Nil(t, stub.direct)
	assert.Equal(t, groupDMCaller, stub.group.GetUserId(), "the caller comes from the token, not the body")
	assert.Equal(t, []string{groupDMPeerA, groupDMPeerB}, stub.group.GetParticipantIds())
	assert.Nil(t, stub.group.Name, "no name key sends no name")

	var resp struct {
		Channel struct {
			ID string `json:"id"`
		} `json:"channel"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, groupDMChannelID, resp.Channel.ID, "the desktop opens result.channel.id")

	stub.created = false
	rec = postDM(t, routes, `{"participant_ids":["`+groupDMPeerB+`","`+groupDMPeerA+`"],"name":"Projekt"}`)
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, "Projekt", stub.group.GetName())
}

func TestChatGroupDM_OneToOneBodyStaysOnTheDirectRPC(t *testing.T) {
	routes, stub := newGroupDMRoutes(t)

	stub.created = true
	rec := postDM(t, routes, `{"other_user_id":"`+groupDMPeerA+`"}`)
	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.direct)
	assert.Nil(t, stub.group, "a 1:1 body must not take the group path")
	assert.Equal(t, groupDMPeerA, stub.direct.GetOtherUserId())
}

func TestChatGroupDM_RejectsAmbiguousOrMalformedBodies(t *testing.T) {
	routes, stub := newGroupDMRoutes(t)

	for name, body := range map[string]string{
		"neither":            `{}`,
		"empty list":         `{"participant_ids":[]}`,
		"both":               `{"other_user_id":"` + groupDMPeerA + `","participant_ids":["` + groupDMPeerB + `"]}`,
		"non-uuid member":    `{"participant_ids":["usr-1","` + groupDMPeerB + `"]}`,
		"name over 100":      `{"participant_ids":["` + groupDMPeerA + `","` + groupDMPeerB + `"],"name":"` + strings.Repeat("x", 101) + `"}`,
		"over fifty members": `{"participant_ids":[` + strings.TrimSuffix(strings.Repeat(`"`+groupDMPeerA+`",`, 51), ",") + `]}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := postDM(t, routes, body)
			assertStatus(t, rec, http.StatusBadRequest)
			assert.Nil(t, stub.group, "a rejected body must not reach the service")
			assert.Nil(t, stub.direct, "a rejected body must not reach the service")
		})
	}

	rec := postDM(t, routes, `{}`)
	assertValidationError(t, rec, "other_user_id")
}
