package gateway

// DELETE /api/v1/documents/shares/{id} is the path the desktop's ShareDialog calls
// (document-client.ts `unshare`). The tests read the request the document service
// actually receives over a real gRPC hop.

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

	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

const (
	testUnshareShareID = "770e8400-e29b-41d4-a716-446655440000"
	testUnshareUser    = "44444444-4444-4444-4444-444444444444"
)

type unshareByIDStub struct {
	documentv1.UnimplementedDocumentServiceServer
	calls []*documentv1.UnshareEntityRequest
	err   error
}

func (s *unshareByIDStub) UnshareEntity(_ context.Context, r *documentv1.UnshareEntityRequest) (*documentv1.UnshareEntityResponse, error) {
	s.calls = append(s.calls, r)
	if s.err != nil {
		return nil, s.err
	}
	return &documentv1.UnshareEntityResponse{}, nil
}

func newUnshareByIDRouter(t *testing.T, stub *unshareByIDStub) http.Handler {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	documentv1.RegisterDocumentServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("document", lis.Addr().String())
	t.Cleanup(reg.Close)

	r := chi.NewRouter()
	NewDocumentRoutes(reg).RegisterRoutes(r, func(next http.Handler) http.Handler { return next })
	return r
}

func unshareByIDRequest(id string, perms ...string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/documents/shares/"+id, nil)
	return withPermissions(withAuth(req, testUnshareUser, testTenantID), perms...)
}

func TestDocumentWire_UnshareByIDSendsTheShareID(t *testing.T) {
	stub := &unshareByIDStub{}
	router := newUnshareByIDRouter(t, stub)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, unshareByIDRequest(testUnshareShareID, "documents:write"))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"unshared":true}`, rec.Body.String())
	require.Len(t, stub.calls, 1)
	assert.Equal(t, testUnshareShareID, stub.calls[0].GetShareId())
	assert.Empty(t, stub.calls[0].GetEntityId())
}

func TestDocumentWire_UnshareByIDRejectsAnInvalidID(t *testing.T) {
	stub := &unshareByIDStub{}
	router := newUnshareByIDRouter(t, stub)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, unshareByIDRequest("not-a-uuid", "documents:write"))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, stub.calls, "no RPC for an invalid ID")
}

func TestDocumentWire_UnshareByIDMapsServiceErrors(t *testing.T) {
	for code, want := range map[codes.Code]int{
		codes.NotFound:         http.StatusNotFound,
		codes.PermissionDenied: http.StatusForbidden,
	} {
		stub := &unshareByIDStub{err: status.Error(code, "nope")}
		router := newUnshareByIDRouter(t, stub)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, unshareByIDRequest(testUnshareShareID, "documents:write"))
		assert.Equal(t, want, rec.Code, code.String())
	}
}

func TestDocumentWire_UnshareByIDKeepsTheAdditiveGuard(t *testing.T) {
	stub := &unshareByIDStub{}
	router := newUnshareByIDRouter(t, stub)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, unshareByIDRequest(testUnshareShareID, "documents:read"))
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, unshareByIDRequest(testUnshareShareID, "documents:share:manage"))
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Len(t, stub.calls, 1)
}
