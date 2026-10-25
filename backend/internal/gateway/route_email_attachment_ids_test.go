package gateway

// attachment_ids of the four compose routes used to die in the gateway: the DTOs had no
// field for them, so an uploaded file never reached the email service. A real gRPC server
// on loopback plays the email service.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

type attachmentIDsStub struct {
	emailv1.UnimplementedEmailServiceServer
	send    *emailv1.SendEmailRequest
	draft   *emailv1.SaveDraftRequest
	reply   *emailv1.ReplyEmailRequest
	forward *emailv1.ForwardEmailRequest
}

func (s *attachmentIDsStub) SendEmail(_ context.Context, r *emailv1.SendEmailRequest) (*emailv1.SendEmailResponse, error) {
	s.send = r
	return &emailv1.SendEmailResponse{Message: &emailv1.EmailMessageInfo{}}, nil
}

func (s *attachmentIDsStub) SaveDraft(_ context.Context, r *emailv1.SaveDraftRequest) (*emailv1.SaveDraftResponse, error) {
	s.draft = r
	return &emailv1.SaveDraftResponse{Message: &emailv1.EmailMessageInfo{}}, nil
}

func (s *attachmentIDsStub) ReplyEmail(_ context.Context, r *emailv1.ReplyEmailRequest) (*emailv1.ReplyEmailResponse, error) {
	s.reply = r
	return &emailv1.ReplyEmailResponse{Message: &emailv1.EmailMessageInfo{}}, nil
}

func (s *attachmentIDsStub) ForwardEmail(_ context.Context, r *emailv1.ForwardEmailRequest) (*emailv1.ForwardEmailResponse, error) {
	s.forward = r
	return &emailv1.ForwardEmailResponse{Message: &emailv1.EmailMessageInfo{}}, nil
}

func TestEmailCompose_AttachmentIDsReachTheEmailService(t *testing.T) {
	stub := &attachmentIDsStub{}
	routes := NewEmailRoutes(registryFor(t, "email", func(s *grpc.Server) { emailv1.RegisterEmailServiceServer(s, stub) }))
	ids := []string{uuid.NewString(), uuid.NewString()}
	acct, orig := uuid.NewString(), uuid.NewString()

	call := func(h http.HandlerFunc, body map[string]any) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/email/messages", jsonBody(t, body))
		h(rec, withAuth(req, uuid.NewString(), testTenantID))
		return rec
	}
	to := []string{"kunde@example.com"}

	rec := call(routes.HandleSendEmail, map[string]any{"account_id": acct, "to": to, "attachment_ids": ids})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(t, ids, stub.send.GetAttachmentIds())

	rec = call(routes.HandleSaveDraft, map[string]any{"account_id": acct, "attachment_ids": ids})
	require.Less(t, rec.Code, 300, rec.Body.String())
	assert.Equal(t, ids, stub.draft.GetAttachmentIds())

	rec = call(routes.HandleReplyEmail, map[string]any{"account_id": acct, "original_message_id": orig, "attachment_ids": ids})
	require.Less(t, rec.Code, 300, rec.Body.String())
	assert.Equal(t, ids, stub.reply.GetAttachmentIds())

	rec = call(routes.HandleForwardEmail, map[string]any{"account_id": acct, "original_message_id": orig, "to": to, "attachment_ids": ids})
	require.Less(t, rec.Code, 300, rec.Body.String())
	assert.Equal(t, ids, stub.forward.GetAttachmentIds())

	// A malformed id is a client error at the boundary, not a lookup in the email service.
	stub.send = nil
	rec = call(routes.HandleSendEmail, map[string]any{"account_id": acct, "to": to, "attachment_ids": []string{"not-a-uuid"}})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Nil(t, stub.send)
}
