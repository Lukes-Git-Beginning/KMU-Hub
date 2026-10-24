package gateway

// DELETE /api/v1/wiki/attachments/{id} is what the desktop's "Anhang löschen" calls
// (wiki-client.ts `deleteAttachment`); only the article-scoped route existed (405). The tests
// read the request the wiki service actually receives over a real gRPC hop.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	wikiv1 "github.com/kmuhub/kmuhub/proto/wiki/v1"
)

const (
	testWikiAttachmentID   = "aa0e8400-e29b-41d4-a716-446655440000"
	testWikiAttachmentUser = "55555555-5555-5555-5555-555555555555"
)

type wikiAttachmentStub struct {
	wikiv1.UnimplementedWikiServiceServer
	calls []*wikiv1.DeleteAttachmentRequest
	err   error
}

func (s *wikiAttachmentStub) DeleteAttachment(_ context.Context, r *wikiv1.DeleteAttachmentRequest) (*wikiv1.DeleteAttachmentResponse, error) {
	s.calls = append(s.calls, r)
	if s.err != nil {
		return nil, s.err
	}
	return &wikiv1.DeleteAttachmentResponse{}, nil
}

func newWikiAttachmentRouter(t *testing.T, stub *wikiAttachmentStub) http.Handler {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	wikiv1.RegisterWikiServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("wiki", lis.Addr().String())
	t.Cleanup(reg.Close)

	r := chi.NewRouter()
	newWikiRoutes(reg).RegisterRoutes(r, func(next http.Handler) http.Handler { return next })
	return r
}

func newWikiAttachmentRequest(path string, perms ...string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	return withPermissions(withAuth(req, testWikiAttachmentUser, testTenantID), perms...)
}

func TestWikiWire_DeleteAttachmentByItsOwnID(t *testing.T) {
	stub := &wikiAttachmentStub{}
	router := newWikiAttachmentRouter(t, stub)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, newWikiAttachmentRequest("/api/v1/wiki/attachments/"+testWikiAttachmentID, "wiki:articles:write"))
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	require.Len(t, stub.calls, 1)
	assert.Equal(t, testWikiAttachmentID, stub.calls[0].GetAttachmentId())
	assert.Equal(t, testTenantID.String(), stub.calls[0].GetTenantId(), "the tenant comes from the token")

	// The article-scoped route keeps working and reaches the same RPC.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, newWikiAttachmentRequest(
		"/api/v1/wiki/articles/"+wikiTestArticleID+"/attachments/"+testWikiAttachmentID, "wiki:articles:write"))
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	require.Len(t, stub.calls, 2)
	assert.Equal(t, testWikiAttachmentID, stub.calls[1].GetAttachmentId())
}

func TestWikiWire_DeleteAttachmentRejectsABadIDWithoutAnRPC(t *testing.T) {
	stub := &wikiAttachmentStub{}
	router := newWikiAttachmentRouter(t, stub)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, newWikiAttachmentRequest("/api/v1/wiki/attachments/not-a-uuid", "wiki:articles:write"))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, stub.calls)
}

func TestWikiWire_DeleteAttachmentMapsNotFound(t *testing.T) {
	// The wiki service answers NotFound for an attachment of another tenant (tenant-scoped
	// DELETE, see wiki/tenant_write_test.go); the gateway must pass that on as 404.
	stub := &wikiAttachmentStub{err: status.Error(codes.NotFound, "attachment not found")}
	router := newWikiAttachmentRouter(t, stub)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, newWikiAttachmentRequest("/api/v1/wiki/attachments/"+testWikiAttachmentID, "wiki:articles:write"))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWikiWire_DeleteAttachmentKeepsTheAdditiveGuard(t *testing.T) {
	stub := &wikiAttachmentStub{}
	router := newWikiAttachmentRouter(t, stub)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, newWikiAttachmentRequest("/api/v1/wiki/attachments/"+testWikiAttachmentID, "wiki:articles:read"))
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Empty(t, stub.calls)

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, newWikiAttachmentRequest("/api/v1/wiki/attachments/"+testWikiAttachmentID, "wiki:article:delete"))
	assert.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	assert.Len(t, stub.calls, 1)
}
