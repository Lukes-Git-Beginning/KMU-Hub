package gateway

// Global search as the desktop reads it (GlobalSearchResultItem in useGlobalSearch.ts). A loopback
// gRPC server plays CRM and Documents and answers with the raw proto hits.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

type searchWireCRM struct {
	crmv1.UnimplementedCRMServiceServer
	results []*crmv1.SearchResult
}

func (s *searchWireCRM) Search(context.Context, *crmv1.SearchRequest) (*crmv1.SearchResponse, error) {
	return &crmv1.SearchResponse{Results: s.results}, nil
}

type searchWireDocs struct {
	documentv1.UnimplementedDocumentServiceServer
	results []*documentv1.FileSearchResult
}

func (s *searchWireDocs) SearchFiles(context.Context, *documentv1.SearchFilesRequest) (*documentv1.SearchFilesResponse, error) {
	return &documentv1.SearchFilesResponse{Results: s.results, Total: int32(len(s.results))}, nil
}

func newSearchWireRoutes(t *testing.T, crm *searchWireCRM, docs *searchWireDocs) *GlobalSearchRoutes {
	t.Helper()
	reg := NewServiceRegistry(nil)
	t.Cleanup(reg.Close)
	for name, register := range map[string]func(*grpc.Server){
		"crm":      func(s *grpc.Server) { crmv1.RegisterCRMServiceServer(s, crm) },
		"document": func(s *grpc.Server) { documentv1.RegisterDocumentServiceServer(s, docs) },
	} {
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		srv := grpc.NewServer()
		register(srv)
		go func() { _ = srv.Serve(lis) }()
		t.Cleanup(srv.Stop)
		reg.Register(name, lis.Addr().String())
	}
	return NewGlobalSearchRoutes(reg)
}

func searchWireModule(t *testing.T, routes *GlobalSearchRoutes, name string) (raw json.RawMessage, items []map[string]any) {
	t.Helper()
	req := withPermissions(httptest.NewRequest(http.MethodGet, "/api/v1/search/global?q=acme", nil), searchAllSourcesPerms...)
	rec := httptest.NewRecorder()
	routes.HandleGlobalSearch(rec, req)
	assertStatus(t, rec, http.StatusOK)

	var body struct {
		Modules []struct {
			Module  string          `json:"module"`
			Results json.RawMessage `json:"results"`
			Error   string          `json:"error"`
		} `json:"modules"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
	for _, m := range body.Modules {
		if m.Module != name {
			continue
		}
		require.Empty(t, m.Error)
		require.NoError(t, json.Unmarshal(m.Results, &items), string(m.Results))
		return m.Results, items
	}
	t.Fatalf("module %q missing: %s", name, rec.Body.String())
	return nil, nil
}

func TestGlobalSearchWire_CRMHitsAreFlatAndLinkToExistingRoutes(t *testing.T) {
	const id = "550e8400-e29b-41d4-a716-446655440000"
	routes := newSearchWireRoutes(t, &searchWireCRM{results: []*crmv1.SearchResult{
		{Id: id, EntityType: "contact", Title: "Erika Muster", Subtitle: "Muster GmbH - erika@muster.de"},
		{Id: id, EntityType: "company", Title: "Muster GmbH", Subtitle: "Zürich"},
		{Id: id, EntityType: "deal", Title: "Rahmenvertrag", Subtitle: "Verhandlung"},
		{Id: id, EntityType: "activity", Title: "Rückruf", Subtitle: "Anruf"},
	}}, &searchWireDocs{})

	_, items := searchWireModule(t, routes, "crm")
	require.Len(t, items, 4)

	wantURL := map[string]string{
		"contact":  "/kontakte?contact=" + id,
		"company":  "/kontakte/firmen/" + id,
		"deal":     "/kontakte/pipeline/" + id,
		"activity": "/kontakte/aktivitaeten",
	}
	for _, it := range items {
		et := it["entity_type"].(string)
		assert.Equal(t, wantURL[et], it["url"], et)
		assert.Equal(t, id, it["id"], et)
		assert.NotEmpty(t, it["title"], et)
		assert.NotEmpty(t, it["description"], et)
	}
	assert.Equal(t, "Muster GmbH - erika@muster.de", items[0]["description"])
}

func TestGlobalSearchWire_DocumentHitsCarryTheFileFlat(t *testing.T) {
	routes := newSearchWireRoutes(t, &searchWireCRM{}, &searchWireDocs{results: []*documentv1.FileSearchResult{{
		File:    &documentv1.DocumentFile{Id: "f-1", Filename: "angebot.pdf", MimeType: "application/pdf"},
		Rank:    0.7,
		Snippet: "… Angebot für Muster GmbH …",
	}}})

	_, items := searchWireModule(t, routes, "documents")
	require.Len(t, items, 1)
	it := items[0]
	assert.Equal(t, "f-1", it["id"])
	assert.Equal(t, "angebot.pdf", it["title"])
	assert.Equal(t, "angebot.pdf", it["file_name"])
	assert.Equal(t, "application/pdf", it["mime_type"])
	assert.Equal(t, "… Angebot für Muster GmbH …", it["content_text"])
	assert.Equal(t, "/dokumente", it["url"])
	assert.NotContains(t, it, "file", "the raw proto wrapper must not leak")
}

func TestGlobalSearchWire_NoHitsIsAnEmptyListNotNull(t *testing.T) {
	routes := newSearchWireRoutes(t, &searchWireCRM{}, &searchWireDocs{})
	for _, module := range []string{"crm", "documents"} {
		raw, _ := searchWireModule(t, routes, module)
		assert.JSONEq(t, "[]", string(raw), module)
	}
}
