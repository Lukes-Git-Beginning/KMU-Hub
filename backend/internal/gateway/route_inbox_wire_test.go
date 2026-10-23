package gateway

// Wire shape of the inbox routes against the desktop types (api/inbox-types.ts,
// api/inbox-client.ts): tags always an array, team enums as strings, bare
// team/member bodies and lists, `counts` on unread-count, the /bulk/* paths and
// the `message` sample of /rules/test. A real gRPC server on loopback stands in
// for the notification service.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	inboxv1 "github.com/kmuhub/kmuhub/proto/inbox/v1"
)

const wireTeamID = "22222222-2222-2222-2222-222222222222"

type inboxWireStub struct {
	inboxv1.UnimplementedInboxServiceServer
	bulkRead    *inboxv1.BulkMarkReadRequest
	bulkArchive *inboxv1.BulkArchiveRequest
	addMember   *inboxv1.AddTeamMemberRequest
	testRule    *inboxv1.TestRoutingRuleRequest
	matches     bool
}

func wireTeam() *inboxv1.TeamInboxInfo {
	return &inboxv1.TeamInboxInfo{
		Id: wireTeamID, Name: "Support",
		AssignmentMode: inboxv1.AssignmentMode_ASSIGNMENT_MODE_ROUND_ROBIN,
		Visibility:     inboxv1.TeamInboxVisibility_VISIBILITY_PRIVATE,
	}
}

func (s *inboxWireStub) ListMessages(context.Context, *inboxv1.ListMessagesRequest) (*inboxv1.ListMessagesResponse, error) {
	return &inboxv1.ListMessagesResponse{
		Messages:   []*inboxv1.InboxMessageInfo{{Id: "m-1", Channel: inboxv1.Channel_CHANNEL_EMAIL}},
		TotalCount: 1,
	}, nil
}

func (s *inboxWireStub) GetMessage(context.Context, *inboxv1.GetMessageRequest) (*inboxv1.GetMessageResponse, error) {
	return &inboxv1.GetMessageResponse{Message: &inboxv1.InboxMessageInfo{Id: "m-1"}}, nil
}

func (s *inboxWireStub) GetUnreadCount(context.Context, *inboxv1.GetUnreadCountRequest) (*inboxv1.GetUnreadCountResponse, error) {
	return &inboxv1.GetUnreadCountResponse{
		Total:     3,
		ByChannel: []*inboxv1.UnreadCountByChannel{{Channel: inboxv1.Channel_CHANNEL_EMAIL, Count: 3}},
	}, nil
}

func (s *inboxWireStub) BulkMarkRead(_ context.Context, r *inboxv1.BulkMarkReadRequest) (*inboxv1.BulkMarkReadResponse, error) {
	s.bulkRead = r
	return &inboxv1.BulkMarkReadResponse{UpdatedCount: int32(len(r.MessageIds))}, nil
}

func (s *inboxWireStub) BulkArchive(_ context.Context, r *inboxv1.BulkArchiveRequest) (*inboxv1.BulkArchiveResponse, error) {
	s.bulkArchive = r
	return &inboxv1.BulkArchiveResponse{UpdatedCount: int32(len(r.MessageIds))}, nil
}

func (s *inboxWireStub) ListTeamInboxes(context.Context, *inboxv1.ListTeamInboxesRequest) (*inboxv1.ListTeamInboxesResponse, error) {
	return &inboxv1.ListTeamInboxesResponse{TeamInboxes: []*inboxv1.TeamInboxInfo{wireTeam()}}, nil
}

func (s *inboxWireStub) CreateTeamInbox(context.Context, *inboxv1.CreateTeamInboxRequest) (*inboxv1.CreateTeamInboxResponse, error) {
	return &inboxv1.CreateTeamInboxResponse{TeamInbox: wireTeam()}, nil
}

func (s *inboxWireStub) UpdateTeamInbox(context.Context, *inboxv1.UpdateTeamInboxRequest) (*inboxv1.UpdateTeamInboxResponse, error) {
	return &inboxv1.UpdateTeamInboxResponse{TeamInbox: wireTeam()}, nil
}

func (s *inboxWireStub) ListTeamMembers(context.Context, *inboxv1.ListTeamMembersRequest) (*inboxv1.ListTeamMembersResponse, error) {
	return &inboxv1.ListTeamMembersResponse{Members: []*inboxv1.TeamMemberInfo{
		{TeamInboxId: wireTeamID, UserId: "u-1", Role: inboxv1.TeamMemberRole_TEAM_ROLE_ADMIN},
	}}, nil
}

func (s *inboxWireStub) AddTeamMember(_ context.Context, r *inboxv1.AddTeamMemberRequest) (*inboxv1.AddTeamMemberResponse, error) {
	s.addMember = r
	return &inboxv1.AddTeamMemberResponse{Member: &inboxv1.TeamMemberInfo{
		TeamInboxId: r.TeamInboxId, UserId: r.MemberUserId, Role: r.Role,
	}}, nil
}

func (s *inboxWireStub) TestRoutingRule(_ context.Context, r *inboxv1.TestRoutingRuleRequest) (*inboxv1.TestRoutingRuleResponse, error) {
	s.testRule = r
	return &inboxv1.TestRoutingRuleResponse{Matches: s.matches}, nil
}

func (s *inboxWireStub) ListRoutingRules(context.Context, *inboxv1.ListRoutingRulesRequest) (*inboxv1.ListRoutingRulesResponse, error) {
	ch := inboxv1.Channel_CHANNEL_CHAT
	return &inboxv1.ListRoutingRulesResponse{Rules: []*inboxv1.RoutingRuleInfo{
		{Id: shapeRuleID, Name: "Chat", Channel: &ch},
	}}, nil
}

// newInboxWireRouter mounts the real inbox routes so the registered paths are
// exercised too, not only the handlers.
func newInboxWireRouter(t *testing.T) (http.Handler, *inboxWireStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &inboxWireStub{}
	srv := grpc.NewServer()
	inboxv1.RegisterInboxServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("notification", lis.Addr().String())
	t.Cleanup(reg.Close)

	r := chi.NewRouter()
	NewInboxRoutes(reg).RegisterRoutes(r, guardTestAuth)
	return r, stub
}

func doInbox(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, jsonBody(t, body))
	}
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withRoles(req, "admin")
	req = withPermissions(req, "inbox:read", "inbox:write")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeObject(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &m), rec.Body.String())
	return m
}

func TestInboxMessages_TagsAlwaysAnArray(t *testing.T) {
	h, _ := newInboxWireRouter(t)

	rec := doInbox(t, h, http.MethodGet, "/api/v1/inbox/messages", nil)
	assertStatus(t, rec, http.StatusOK)
	list := decodeObject(t, rec)
	msgs := list["messages"].([]any)
	require.Len(t, msgs, 1)
	msg := msgs[0].(map[string]any)
	assert.Equal(t, []any{}, msg["tags"], "a message without tags must carry tags:[]")
	assert.Equal(t, false, msg["is_read"])
	assert.EqualValues(t, 1, list["total_count"])

	rec = doInbox(t, h, http.MethodGet, "/api/v1/inbox/messages/"+uuid.New().String(), nil)
	assertStatus(t, rec, http.StatusOK)
	one := decodeObject(t, rec)["message"].(map[string]any)
	assert.Equal(t, []any{}, one["tags"])
}

func TestInboxTeams_BareBodiesAndStringEnums(t *testing.T) {
	h, _ := newInboxWireRouter(t)

	rec := doInbox(t, h, http.MethodGet, "/api/v1/inbox/teams", nil)
	assertStatus(t, rec, http.StatusOK)
	var teams []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &teams), rec.Body.String())
	require.Len(t, teams, 1)
	assert.Equal(t, "round_robin", teams[0]["assignment_mode"])
	assert.Equal(t, "private", teams[0]["visibility"])

	rec = doInbox(t, h, http.MethodPost, "/api/v1/inbox/teams", map[string]any{"name": "Support"})
	assertStatus(t, rec, http.StatusCreated)
	created := decodeObject(t, rec)
	assert.Equal(t, wireTeamID, created["id"], "POST /teams returns the bare team inbox")
	assert.Equal(t, "round_robin", created["assignment_mode"])

	rec = doInbox(t, h, http.MethodPut, "/api/v1/inbox/teams/"+wireTeamID, map[string]any{
		"assignment_mode": "round_robin", "visibility": "private",
	})
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, wireTeamID, decodeObject(t, rec)["id"])

	rec = doInbox(t, h, http.MethodGet, "/api/v1/inbox/teams/"+wireTeamID+"/members", nil)
	assertStatus(t, rec, http.StatusOK)
	var members []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &members), rec.Body.String())
	require.Len(t, members, 1)
	assert.Equal(t, "admin", members[0]["role"])
}

func TestInboxTeams_AddMemberAcceptsUserID(t *testing.T) {
	h, stub := newInboxWireRouter(t)
	member := uuid.New().String()

	rec := doInbox(t, h, http.MethodPost, "/api/v1/inbox/teams/"+wireTeamID+"/members",
		map[string]any{"user_id": member, "role": "admin"})
	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.addMember)
	assert.Equal(t, member, stub.addMember.MemberUserId)
	assert.Equal(t, inboxv1.TeamMemberRole_TEAM_ROLE_ADMIN, stub.addMember.Role)
	assert.Equal(t, "admin", decodeObject(t, rec)["role"])

	other := uuid.New().String()
	rec = doInbox(t, h, http.MethodPost, "/api/v1/inbox/teams/"+wireTeamID+"/members",
		map[string]any{"member_user_id": other, "role": "member"})
	assertStatus(t, rec, http.StatusCreated)
	assert.Equal(t, other, stub.addMember.MemberUserId, "the old field name keeps working")

	stub.addMember = nil
	rec = doInbox(t, h, http.MethodPost, "/api/v1/inbox/teams/"+wireTeamID+"/members",
		map[string]any{"role": "member"})
	assertStatus(t, rec, http.StatusBadRequest)
	assert.Nil(t, stub.addMember, "a request without a member must not reach the RPC")
}

func TestInboxBulk_DesktopPathsReachTheBulkHandlers(t *testing.T) {
	h, stub := newInboxWireRouter(t)
	ids := []string{uuid.New().String(), uuid.New().String()}

	rec := doInbox(t, h, http.MethodPost, "/api/v1/inbox/messages/bulk/read", map[string]any{"ids": ids})
	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.bulkRead)
	assert.Equal(t, ids, stub.bulkRead.MessageIds)

	rec = doInbox(t, h, http.MethodPost, "/api/v1/inbox/messages/bulk/archive", map[string]any{"ids": ids})
	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.bulkArchive)
	assert.Equal(t, ids, stub.bulkArchive.MessageIds)

	// The hyphenated paths stay registered.
	stub.bulkRead = nil
	rec = doInbox(t, h, http.MethodPost, "/api/v1/inbox/messages/bulk-read", map[string]any{"ids": ids})
	assertStatus(t, rec, http.StatusOK)
	assert.NotNil(t, stub.bulkRead)
}

func TestInboxUnreadCount_CountsWithChannelName(t *testing.T) {
	h, _ := newInboxWireRouter(t)
	for _, path := range []string{"/api/v1/inbox/messages/unread-count", "/api/v1/inbox/unread-count"} {
		rec := doInbox(t, h, http.MethodGet, path, nil)
		assertStatus(t, rec, http.StatusOK)
		body := decodeObject(t, rec)
		assert.EqualValues(t, 3, body["total"])
		counts := body["counts"].([]any)
		require.Len(t, counts, 1)
		assert.Equal(t, map[string]any{"channel": "email", "count": float64(3)}, counts[0])
		assert.NotEmpty(t, body["by_channel"], "by_channel stays for existing callers")
	}
}

func TestInboxRulesTest_ReadsDesktopMessage(t *testing.T) {
	h, stub := newInboxWireRouter(t)

	rec := doInbox(t, h, http.MethodPost, "/api/v1/inbox/rules/test", map[string]any{
		"conditions": map[string]any{"field": "subject", "operator": "contains", "value": "Rechnung"},
		"message":    map[string]any{"channel": "email", "sender_name": "Anna", "subject": "Rechnung 42"},
	})
	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.testRule)
	require.NotNil(t, stub.testRule.TestMessage, "the FE sample arrives as `message`")
	assert.Equal(t, inboxv1.Channel_CHANNEL_EMAIL, stub.testRule.TestMessage.Channel)
	assert.Equal(t, "Rechnung 42", stub.testRule.TestMessage.Subject)
	assert.Equal(t, "Anna", stub.testRule.TestMessage.SenderName)
	assert.Equal(t, map[string]any{"matches": false}, decodeObject(t, rec), "matches:false must be written")

	stub.matches = true
	rec = doInbox(t, h, http.MethodPost, "/api/v1/inbox/rules/test", map[string]any{
		"conditions":   map[string]any{},
		"test_message": map[string]any{"channel": 2, "subject": "x"},
	})
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, inboxv1.Channel_CHANNEL_CHAT, stub.testRule.TestMessage.Channel, "test_message with a numeric channel still works")
	assert.Equal(t, true, decodeObject(t, rec)["matches"])
}

func TestInboxRules_ChannelWrittenAsName(t *testing.T) {
	h, _ := newInboxWireRouter(t)
	rec := doInbox(t, h, http.MethodGet, "/api/v1/inbox/rules", nil)
	assertStatus(t, rec, http.StatusOK)
	var rules []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &rules), rec.Body.String())
	require.Len(t, rules, 1)
	assert.Equal(t, "chat", rules[0]["channel"])
	assert.Equal(t, false, rules[0]["is_active"])
	assert.False(t, strings.Contains(rec.Body.String(), `"channel":2`))
}
