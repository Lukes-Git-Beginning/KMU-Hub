package gateway

// POST /api/v1/documents/files/{id}/versions is the call behind "Neue Version anlegen" in the
// desktop's VersionHistoryPanel (document-client.ts `documentVersionApi.create`). The route
// did not exist (405). The tests read the request the document service actually receives over
// a real gRPC hop.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
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
	testCreateVersionFileID = "880e8400-e29b-41d4-a716-446655440000"
	testCreateVersionUser   = "55555555-5555-5555-5555-555555555555"
)

type createVersionStub struct {
	documentv1.UnimplementedDocumentServiceServer
	calls []*documentv1.CreateFileVersionRequest
	err   error
}

func (s *createVersionStub) CreateFileVersion(_ context.Context, r *documentv1.CreateFileVersionRequest) (*documentv1.CreateFileVersionResponse, error) {
	s.calls = append(s.calls, r)
	if s.err != nil {
		return nil, s.err
	}
	return &documentv1.CreateFileVersionResponse{Version: &documentv1.DocumentFileVersion{
		Id: "990e8400-e29b-41d4-a716-446655440000", FileId: r.FileId, VersionNumber: 2, VersionLabel: r.VersionLabel,
	}}, nil
}

func newCreateVersionRouter(t *testing.T, stub *createVersionStub) http.Handler {
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

func newCreateVersionHTTPRequest(body string, perms ...string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/files/"+testCreateVersionFileID+"/versions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return withPermissions(withAuth(req, testCreateVersionUser, testTenantID), perms...)
}

func TestDocumentWire_CreateVersionSendsOnlyFileAndLabel(t *testing.T) {
	stub := &createVersionStub{}
	router := newCreateVersionRouter(t, stub)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, newCreateVersionHTTPRequest(
		`{"version_label":"Vor Abnahme","storage_key":"documents/fremd","file_size":5,"created_by":"x"}`, "documents:write"))

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"version":{`)
	assert.Contains(t, rec.Body.String(), `"version_label":"Vor Abnahme"`)
	require.Len(t, stub.calls, 1)
	assert.Equal(t, testCreateVersionFileID, stub.calls[0].GetFileId())
	assert.Equal(t, "Vor Abnahme", stub.calls[0].GetVersionLabel())
	assert.Empty(t, stub.calls[0].GetStorageKey(), "a client cannot name the storage key")
	assert.Zero(t, stub.calls[0].GetFileSize())
	assert.Empty(t, stub.calls[0].GetCreatedBy(), "the author comes from the JWT on the service side")
}

func TestDocumentWire_CreateVersionAcceptsAnEmptyObjectAndRejectsBadInput(t *testing.T) {
	stub := &createVersionStub{}
	router := newCreateVersionRouter(t, stub)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, newCreateVersionHTTPRequest(`{}`, "documents:write"))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.Len(t, stub.calls, 1)
	assert.Empty(t, stub.calls[0].GetVersionLabel())

	for name, body := range map[string]string{
		"label over 255": `{"version_label":"` + strings.Repeat("x", 256) + `"}`,
		"invalid json":   `{`,
	} {
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, newCreateVersionHTTPRequest(body, "documents:write"))
		assert.Equal(t, http.StatusBadRequest, rec.Code, name)
	}
	assert.Len(t, stub.calls, 1, "no RPC for rejected bodies")
}

func TestDocumentWire_CreateVersionMapsServiceErrors(t *testing.T) {
	for code, want := range map[codes.Code]int{
		codes.NotFound:           http.StatusNotFound,
		codes.FailedPrecondition: http.StatusConflict,
	} {
		stub := &createVersionStub{err: status.Error(code, "nope")}
		router := newCreateVersionRouter(t, stub)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, newCreateVersionHTTPRequest(`{}`, "documents:write"))
		assert.Equal(t, want, rec.Code, code.String())
	}
}

func TestDocumentWire_CreateVersionKeepsTheAdditiveGuard(t *testing.T) {
	stub := &createVersionStub{}
	router := newCreateVersionRouter(t, stub)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, newCreateVersionHTTPRequest(`{}`, "documents:read"))
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, newCreateVersionHTTPRequest(`{}`, "documents:version:restore"))
	assert.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Len(t, stub.calls, 1)
}
