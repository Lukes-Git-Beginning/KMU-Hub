package gateway

// GET /api/v1/documents/files is called by the desktop's DokumentePage with
// is_favorite and sort_field; the spec documents favorites and sort. The tests
// read the request the document service actually receives over a real gRPC hop.

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

	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

type listFilesStub struct {
	documentv1.UnimplementedDocumentServiceServer
	calls []*documentv1.ListFilesRequest
}

func (s *listFilesStub) ListFiles(_ context.Context, r *documentv1.ListFilesRequest) (*documentv1.ListFilesResponse, error) {
	s.calls = append(s.calls, r)
	return &documentv1.ListFilesResponse{}, nil
}

func listFilesViaRouter(t *testing.T, query string) *documentv1.ListFilesRequest {
	t.Helper()
	stub := &listFilesStub{}
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/files?"+query, nil)
	req = withPermissions(withAuth(req, testUnshareUser, testTenantID), "documents:read")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"files":[],"total":0}`, rec.Body.String())
	require.Len(t, stub.calls, 1)
	return stub.calls[0]
}

func TestDocumentWire_ListFilesAcceptsTheDesktopsFavoriteAndSortNames(t *testing.T) {
	got := listFilesViaRouter(t, "is_favorite=true&sort_field=size&sort_dir=desc")

	assert.True(t, got.GetFilterFavorites())
	assert.True(t, got.GetIsFavorite())
	assert.Equal(t, documentv1.FileSortField_FILE_SORT_SIZE, got.GetSortField())
	assert.Equal(t, documentv1.SortDirection_SORT_DESC, got.GetSortDirection())
}

func TestDocumentWire_ListFilesAcceptsEveryFrontendSortField(t *testing.T) {
	// FileSortField in desktop/src/renderer/src/api/types/document-types.ts
	for field, want := range map[string]documentv1.FileSortField{
		"name": documentv1.FileSortField_FILE_SORT_NAME,
		"size": documentv1.FileSortField_FILE_SORT_SIZE,
		"date": documentv1.FileSortField_FILE_SORT_DATE,
		"type": documentv1.FileSortField_FILE_SORT_TYPE,
	} {
		got := listFilesViaRouter(t, "sort_field="+field)
		assert.Equal(t, want, got.GetSortField(), field)
	}
}

func TestDocumentWire_ListFilesKeepsTheDocumentedNamesAndTheirPrecedence(t *testing.T) {
	got := listFilesViaRouter(t, "favorites=true&sort=name")
	assert.True(t, got.GetFilterFavorites())
	assert.Equal(t, documentv1.FileSortField_FILE_SORT_NAME, got.GetSortField())

	// the documented name wins when both are sent
	got = listFilesViaRouter(t, "sort=name&sort_field=size&favorites=false&is_favorite=true")
	assert.Equal(t, documentv1.FileSortField_FILE_SORT_NAME, got.GetSortField())
	assert.False(t, got.GetFilterFavorites())
}

func TestDocumentWire_ListFilesDropsAnUnknownSortField(t *testing.T) {
	got := listFilesViaRouter(t, "sort_field=password_hash;drop")
	assert.Equal(t, documentv1.FileSortField_FILE_SORT_UNSPECIFIED, got.GetSortField())
}

func TestDocumentWire_ListFilesWithoutParamsFiltersNothing(t *testing.T) {
	got := listFilesViaRouter(t, "is_favorite=false")
	assert.False(t, got.GetFilterFavorites())
	assert.Equal(t, documentv1.FileSortField_FILE_SORT_UNSPECIFIED, got.GetSortField())
}
