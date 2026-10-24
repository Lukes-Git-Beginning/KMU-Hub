package gateway

// fix-email-module-for-members: the email:mailbox keys open the routes a mailbox owner uses on
// their own mail (the email server confines each to the caller's accounts); everything that
// reaches beyond one mailbox stays on the coarse email:* keys. A real gRPC server on loopback
// plays the email service where the handler's own behaviour matters.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

func TestEmailMailboxGuards(t *testing.T) {
	router := chi.NewRouter()
	NewEmailRoutes(emptyRegistry()).RegisterRoutes(router, guardTestAuth)

	const id = "11111111-1111-1111-1111-111111111111"
	member := []string{"email:mailbox:read", "email:mailbox:write", "email:mailbox:delete"}

	cases := []struct {
		method, path string
		open         bool
	}{
		{"GET", "/api/v1/email/accounts/list", true},
		{"GET", "/api/v1/email/accounts/", true},
		{"POST", "/api/v1/email/accounts/" + id + "/default", true},
		{"GET", "/api/v1/email/folders/", true},
		{"GET", "/api/v1/email/folders/" + id, true},
		{"POST", "/api/v1/email/folders/sync", true},
		{"GET", "/api/v1/email/messages/", true},
		{"GET", "/api/v1/email/messages/" + id, true},
		{"GET", "/api/v1/email/messages/thread/" + id, true},
		{"POST", "/api/v1/email/messages/" + id + "/read", true},
		{"POST", "/api/v1/email/messages/" + id + "/move", true},
		{"POST", "/api/v1/email/messages/" + id + "/labels", true},
		{"DELETE", "/api/v1/email/messages/" + id, true},
		{"POST", "/api/v1/email/messages/bulk", true},
		{"POST", "/api/v1/email/send/", true},
		{"POST", "/api/v1/email/send/reply", true},
		{"GET", "/api/v1/email/signatures/", true},
		{"POST", "/api/v1/email/signatures/", true},
		{"DELETE", "/api/v1/email/signatures/" + id, true},
		{"GET", "/api/v1/email/labels", true},
		{"GET", "/api/v1/email/templates", true},
		{"POST", "/api/v1/email/templates", true},
		{"POST", "/api/v1/email/templates/" + id + "/render", true},
		{"GET", "/api/v1/email/links/message/" + id, true},
		{"POST", "/api/v1/email/links/", true},
		{"POST", "/api/v1/email/sync/trigger", true},
		{"GET", "/api/v1/email/sync/status", true},
		{"POST", "/api/v1/email/attachments/upload", true},
		{"GET", "/api/v1/email/attachments/" + id + "/download", true},
		// Beyond one mailbox: admin only.
		{"POST", "/api/v1/email/accounts/", false},
		{"PUT", "/api/v1/email/accounts/" + id, false},
		{"DELETE", "/api/v1/email/accounts/" + id, false},
		{"POST", "/api/v1/email/accounts/test", false},
		{"GET", "/api/v1/email/rules", false},
		{"POST", "/api/v1/email/rules/apply", false},
		{"POST", "/api/v1/email/labels", false},
		{"DELETE", "/api/v1/email/labels/" + id, false},
		{"PUT", "/api/v1/email/templates/" + id, false},
		{"DELETE", "/api/v1/email/templates/" + id, false},
		{"GET", "/api/v1/email/links/contact/" + id, false},
		{"POST", "/api/v1/email/contacts/export/csv", false},
		{"POST", "/api/v1/email/contacts/import/csv", false},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, tokenShape{perms: member}.request(c.method, c.path))
			if c.open {
				assert.NotEqual(t, http.StatusForbidden, rec.Code, "mailbox keys must open the owner route")
			} else {
				assert.Equal(t, http.StatusForbidden, rec.Code, "mailbox keys must not open a route beyond one mailbox")
			}
		})
	}

	t.Run("coarse keys still open every mailbox route", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, tokenShape{perms: []string{"email:read"}}.request("GET", "/api/v1/email/messages/"+id))
		assert.NotEqual(t, http.StatusForbidden, rec.Code)
	})
	t.Run("no key, no mailbox", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, tokenShape{perms: []string{"mail:module:view"}}.request("GET", "/api/v1/email/accounts/list"))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

type emailMailboxStub struct {
	emailv1.UnimplementedEmailServiceServer
	account *emailv1.GetEmailAccountRequest
	bulk    *emailv1.BulkMessageActionRequest
}

func (s *emailMailboxStub) GetEmailAccount(_ context.Context, r *emailv1.GetEmailAccountRequest) (*emailv1.GetEmailAccountResponse, error) {
	s.account = r
	return &emailv1.GetEmailAccountResponse{}, nil
}

func (s *emailMailboxStub) BulkMessageAction(_ context.Context, r *emailv1.BulkMessageActionRequest) (*emailv1.BulkMessageActionResponse, error) {
	s.bulk = r
	return &emailv1.BulkMessageActionResponse{}, nil
}

func TestEmailMailbox_HandlerBoundaries(t *testing.T) {
	stub := &emailMailboxStub{}
	router := chi.NewRouter()
	NewEmailRoutes(registryFor(t, "email", func(s *grpc.Server) { emailv1.RegisterEmailServiceServer(s, stub) })).
		RegisterRoutes(router, guardTestAuth)

	const other = "33333333-3333-3333-3333-333333333333" // anyone but leaveCallerID, the token's user
	member := tokenShape{perms: []string{"email:mailbox:read", "email:mailbox:write", "email:mailbox:delete"}}
	admin := tokenShape{perms: []string{"email:read", "email:write", "email:delete"}}

	t.Run("member asking for another user's account gets their own", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, member.request(http.MethodGet, "/api/v1/email/accounts/?user_id="+other))
		assertStatus(t, rec, http.StatusOK)
		require.NotNil(t, stub.account)
		assert.Equal(t, leaveCallerID, stub.account.GetUserId())
	})
	t.Run("admin may look up another user's account", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, admin.request(http.MethodGet, "/api/v1/email/accounts/?user_id="+other))
		assertStatus(t, rec, http.StatusOK)
		assert.Equal(t, other, stub.account.GetUserId())
	})

	bulkDelete := func(tk tokenShape) int {
		stub.bulk = nil
		req := tk.request(http.MethodPost, "/api/v1/email/messages/bulk")
		req.Body = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ids":["`+other+`"],"action":"delete"}`)).Body
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}
	t.Run("bulk delete takes the mailbox delete key", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, bulkDelete(member))
		require.NotNil(t, stub.bulk)
	})
	t.Run("bulk delete without a delete key is refused", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, bulkDelete(tokenShape{perms: []string{"email:mailbox:write"}}))
		assert.Nil(t, stub.bulk)
	})
}
