package gateway

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// Handler level: a real gRPC server on loopback stands in for the email service,
// so the assertions run against the JSON the handler writes and the addresses
// it forwards over gRPC.

type emailStub struct {
	emailv1.UnimplementedEmailServiceServer
	gotSend    *emailv1.SendEmailRequest
	gotDraft   *emailv1.SaveDraftRequest
	gotForward *emailv1.ForwardEmailRequest
	messages   []*emailv1.EmailMessageInfo
}

func (s *emailStub) SendEmail(_ context.Context, r *emailv1.SendEmailRequest) (*emailv1.SendEmailResponse, error) {
	s.gotSend = r
	return &emailv1.SendEmailResponse{Message: &emailv1.EmailMessageInfo{Id: "sent"}}, nil
}

func (s *emailStub) SaveDraft(_ context.Context, r *emailv1.SaveDraftRequest) (*emailv1.SaveDraftResponse, error) {
	s.gotDraft = r
	return &emailv1.SaveDraftResponse{Message: &emailv1.EmailMessageInfo{Id: "draft", IsDraft: true}}, nil
}

func (s *emailStub) ForwardEmail(_ context.Context, r *emailv1.ForwardEmailRequest) (*emailv1.ForwardEmailResponse, error) {
	s.gotForward = r
	return &emailv1.ForwardEmailResponse{Message: &emailv1.EmailMessageInfo{Id: "fwd"}}, nil
}

func (s *emailStub) ListMessages(context.Context, *emailv1.ListMessagesRequest) (*emailv1.ListMessagesResponse, error) {
	return &emailv1.ListMessagesResponse{Messages: s.messages, Total: int32(len(s.messages))}, nil
}

func (s *emailStub) GetMessage(context.Context, *emailv1.GetMessageRequest) (*emailv1.GetMessageResponse, error) {
	return &emailv1.GetMessageResponse{Message: &emailv1.EmailMessageInfo{Id: "m-1", Uid: 42}}, nil
}

func (s *emailStub) GetThreadMessages(context.Context, *emailv1.GetThreadMessagesRequest) (*emailv1.GetThreadMessagesResponse, error) {
	return &emailv1.GetThreadMessagesResponse{Messages: s.messages}, nil
}

func newEmailStubRoutes(t *testing.T) (*EmailRoutes, *emailStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &emailStub{}
	srv := grpc.NewServer()
	emailv1.RegisterEmailServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("email", lis.Addr().String())
	return NewEmailRoutes(reg), stub
}

func postEmail(t *testing.T, h http.HandlerFunc, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, jsonBody(t, body))
	req = withAuth(req, uuid.New().String(), testTenantID)
	h(rec, req)
	return rec
}

func TestHandleSendEmail_AcceptsObjectAndStringAddresses(t *testing.T) {
	routes, stub := newEmailStubRoutes(t)
	rec := postEmail(t, routes.HandleSendEmail, "/api/v1/email/send/", map[string]any{
		"to":      []any{map[string]string{"name": "Erika Muster", "email": "erika@example.com"}, "plain@example.com"},
		"cc":      []any{map[string]string{"email": "cc@example.com"}},
		"subject": "Hallo",
	})
	assertStatus(t, rec, http.StatusCreated)

	require.NotNil(t, stub.gotSend)
	require.Len(t, stub.gotSend.To, 2)
	assert.Equal(t, "Erika Muster", stub.gotSend.To[0].Name)
	assert.Equal(t, "erika@example.com", stub.gotSend.To[0].Email)
	assert.Equal(t, "plain@example.com", stub.gotSend.To[1].Email)
	assert.Empty(t, stub.gotSend.To[1].Name)
	require.Len(t, stub.gotSend.Cc, 1)
	assert.Equal(t, "cc@example.com", stub.gotSend.Cc[0].Email)
}

func TestHandleSendEmail_ObjectAddressValidation(t *testing.T) {
	routes, stub := newEmailStubRoutes(t)
	rec := postEmail(t, routes.HandleSendEmail, "/api/v1/email/send/", map[string]any{
		"to": []any{map[string]string{"name": "X", "email": "not-an-email"}},
	})
	assertValidationError(t, rec, "to[0]")

	rec = postEmail(t, routes.HandleSendEmail, "/api/v1/email/send/", map[string]any{
		"to": []any{map[string]string{"email": "ok@example.com"}},
		"cc": []any{map[string]string{"email": "bad"}},
	})
	assertValidationError(t, rec, "cc[0]")

	rec = postEmail(t, routes.HandleSendEmail, "/api/v1/email/send/", map[string]any{
		"to": []any{map[string]string{"name": "ohne Adresse"}},
	})
	assertStatus(t, rec, http.StatusBadRequest)
	assert.Nil(t, stub.gotSend, "an invalid request must not reach the RPC")
}

func TestHandleSaveDraft_AcceptsObjectAddressesAndEmptyRecipients(t *testing.T) {
	routes, stub := newEmailStubRoutes(t)
	rec := postEmail(t, routes.HandleSaveDraft, "/api/v1/email/send/draft", map[string]any{
		"to": []any{map[string]string{"name": "Erika", "email": "erika@example.com"}},
	})
	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.gotDraft)
	assert.Equal(t, "Erika", stub.gotDraft.To[0].Name)

	rec = postEmail(t, routes.HandleSaveDraft, "/api/v1/email/send/draft", map[string]any{"subject": "leer"})
	assertStatus(t, rec, http.StatusCreated)
}

func TestHandleForwardEmail_AcceptsObjectAddresses(t *testing.T) {
	routes, stub := newEmailStubRoutes(t)
	rec := postEmail(t, routes.HandleForwardEmail, "/api/v1/email/send/forward", map[string]any{
		"original_message_id": uuid.New().String(),
		"to":                  []any{map[string]string{"name": "Erika", "email": "erika@example.com"}},
	})
	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.gotForward)
	assert.Equal(t, "erika@example.com", stub.gotForward.To[0].Email)
	assert.Equal(t, "Erika", stub.gotForward.To[0].Name)
}

func decodeMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return got
}

func assertMessageLists(t *testing.T, msg any) {
	t.Helper()
	m, ok := msg.(map[string]any)
	require.True(t, ok, "message must be an object, got %T", msg)
	for _, f := range emailMessageListFields {
		v, present := m[f]
		require.True(t, present, "%s missing", f)
		assert.IsType(t, []any{}, v, "%s must be an array", f)
	}
}

func TestHandleGetMessage_WritesEmptyListsAndNumericUID(t *testing.T) {
	routes, _ := newEmailStubRoutes(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/email/messages/m-1", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleGetMessage(rec, req)
	assertStatus(t, rec, http.StatusOK)

	got := decodeMap(t, rec)
	assertMessageLists(t, got["message"])
	assert.EqualValues(t, 42, got["message"].(map[string]any)["uid"], "int64 stays a JSON number")
}

func TestHandleSendEmail_ResponseWritesEmptyLists(t *testing.T) {
	routes, _ := newEmailStubRoutes(t)
	rec := postEmail(t, routes.HandleSendEmail, "/api/v1/email/send/", map[string]any{"to": []string{"a@example.com"}})
	assertStatus(t, rec, http.StatusCreated)
	assertMessageLists(t, decodeMap(t, rec)["message"])
}

func TestHandleListMessages_WritesEmptyListsAndEmptyPage(t *testing.T) {
	routes, stub := newEmailStubRoutes(t)
	stub.messages = []*emailv1.EmailMessageInfo{{Id: "m-1"}, {Id: "m-2", Cc: []*emailv1.EmailAddress{{Email: "c@example.com"}}}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/email/messages/?folder_id="+uuid.New().String(), nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListMessages(rec, req)
	assertStatus(t, rec, http.StatusOK)

	got := decodeMap(t, rec)
	items, ok := got["messages"].([]any)
	require.True(t, ok)
	require.Len(t, items, 2)
	for _, it := range items {
		assertMessageLists(t, it)
	}
	assert.Len(t, items[1].(map[string]any)["cc"], 1, "a filled list is left alone")

	stub.messages = nil
	rec = httptest.NewRecorder()
	routes.HandleListMessages(rec, req)
	assertStatus(t, rec, http.StatusOK)
	got = decodeMap(t, rec)
	assert.Equal(t, []any{}, got["messages"], "empty page is [] not missing")
	assert.EqualValues(t, 0, got["total"])
}

func TestHandleGetThreadMessages_EmptyThreadIsEmptyArray(t *testing.T) {
	routes, stub := newEmailStubRoutes(t)
	stub.messages = nil
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/email/messages/thread/t-1", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleGetThreadMessages(rec, req)
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, []any{}, decodeMap(t, rec)["messages"])
	assert.NotContains(t, decodeMap(t, rec), "total", "thread responses carry no total")
}

func TestEmailAddr_UnmarshalJSON(t *testing.T) {
	var l addrList
	require.NoError(t, json.Unmarshal([]byte(`["a@b.de", {"name":"N","email":"c@d.de"}]`), &l))
	assert.Equal(t, addrList{{Email: "a@b.de"}, {Name: "N", Email: "c@d.de"}}, l)

	assert.Error(t, json.Unmarshal([]byte(`[{"name":"only"}]`), &l))
	assert.Error(t, json.Unmarshal([]byte(`[42]`), &l))
}
