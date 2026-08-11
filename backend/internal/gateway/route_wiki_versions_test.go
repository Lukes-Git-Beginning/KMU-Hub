package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Covers the article CRUD and version handlers in route_wiki.go that
// route_wiki_test.go does not yet exercise: HandleGetArticle,
// HandleUpdateArticle, HandleDeleteArticle, HandleListVersions,
// HandleRestoreVersion and HandleGetVersion. All six are thin passthroughs
// with no gateway-local business logic of their own — the only thing this
// package can prove is the UUID/JSON validation ahead of the RPC call and
// that a valid request reaches the RPC layer (same *_ReachesRPC boundary
// used throughout this run, e.g. TestHandleListOrders_ReachesRPC in
// route_produktion_orders_test.go: registryWithService points the client at
// localhost:0, so a request that clears local validation still ends in a
// connection-refused 503, not a 200).

const wikiTestArticleID = "550e8400-e29b-41d4-a716-446655440000"
const wikiTestVersionID = "660e8400-e29b-41d4-a716-446655440000"

// --- HandleGetArticle ---

func TestHandleGetArticle_ServiceUnavailable(t *testing.T) {
	routes := newWikiRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/wiki/articles/"+wikiTestArticleID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", wikiTestArticleID)
	routes.HandleGetArticle(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetArticle_InvalidUUID(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/wiki/articles/bad", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetArticle(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetArticle_ReachesRPC(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/wiki/articles/"+wikiTestArticleID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", wikiTestArticleID)
	routes.HandleGetArticle(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateArticle ---

func TestHandleUpdateArticle_ServiceUnavailable(t *testing.T) {
	routes := newWikiRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/wiki/articles/"+wikiTestArticleID, jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", wikiTestArticleID)
	routes.HandleUpdateArticle(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateArticle_InvalidUUID(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/wiki/articles/bad", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateArticle(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateArticle_InvalidJSON(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/wiki/articles/"+wikiTestArticleID, invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", wikiTestArticleID)
	routes.HandleUpdateArticle(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateArticle_ReachesRPC(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/wiki/articles/"+wikiTestArticleID, jsonBody(t, map[string]interface{}{
		"title": "Updated title",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", wikiTestArticleID)
	routes.HandleUpdateArticle(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteArticle ---

func TestHandleDeleteArticle_ServiceUnavailable(t *testing.T) {
	routes := newWikiRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/wiki/articles/"+wikiTestArticleID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", wikiTestArticleID)
	routes.HandleDeleteArticle(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteArticle_InvalidUUID(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/wiki/articles/bad", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteArticle(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteArticle_ReachesRPC(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/wiki/articles/"+wikiTestArticleID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", wikiTestArticleID)
	routes.HandleDeleteArticle(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListVersions ---

func TestHandleListVersions_ServiceUnavailable(t *testing.T) {
	routes := newWikiRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/wiki/articles/"+wikiTestArticleID+"/versions", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", wikiTestArticleID)
	routes.HandleListVersions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListVersions_InvalidUUID(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/wiki/articles/bad/versions", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListVersions(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListVersions_ReachesRPC(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/wiki/articles/"+wikiTestArticleID+"/versions", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", wikiTestArticleID)
	routes.HandleListVersions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleRestoreVersion ---
//
// Two ids are validated in sequence — the article id first, then the version
// id (route_wiki.go:432-439) — so an invalid version id only surfaces once
// the article id has already cleared validation.

func TestHandleRestoreVersion_ServiceUnavailable(t *testing.T) {
	routes := newWikiRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/wiki/articles/"+wikiTestArticleID+"/versions/"+wikiTestVersionID+"/restore", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", wikiTestArticleID)
	req = withChiURLParam(req, "versionId", wikiTestVersionID)
	routes.HandleRestoreVersion(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRestoreVersion_InvalidArticleUUID(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/wiki/articles/bad/versions/"+wikiTestVersionID+"/restore", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withChiURLParam(req, "versionId", wikiTestVersionID)
	routes.HandleRestoreVersion(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleRestoreVersion_InvalidVersionUUID(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/wiki/articles/"+wikiTestArticleID+"/versions/bad/restore", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", wikiTestArticleID)
	req = withChiURLParam(req, "versionId", "not-a-uuid")
	routes.HandleRestoreVersion(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid versionId")
}

func TestHandleRestoreVersion_MissingVersionID(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/wiki/articles/"+wikiTestArticleID+"/versions/restore", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", wikiTestArticleID)
	routes.HandleRestoreVersion(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid versionId")
}

func TestHandleRestoreVersion_ReachesRPC(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/wiki/articles/"+wikiTestArticleID+"/versions/"+wikiTestVersionID+"/restore", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", wikiTestArticleID)
	req = withChiURLParam(req, "versionId", wikiTestVersionID)
	routes.HandleRestoreVersion(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetVersion ---
//
// This is the standalone GET /wiki/versions/{id} route (route_wiki.go:98-99):
// "id" here names a version id, not an article id — the request it builds is
// wikiv1.GetVersionRequest{VersionId: id}, unlike every other handler in this
// file where "id" is the article.

func TestHandleGetVersion_ServiceUnavailable(t *testing.T) {
	routes := newWikiRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/wiki/versions/"+wikiTestVersionID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", wikiTestVersionID)
	routes.HandleGetVersion(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetVersion_InvalidUUID(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/wiki/versions/bad", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetVersion(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetVersion_ReachesRPC(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/wiki/versions/"+wikiTestVersionID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", wikiTestVersionID)
	routes.HandleGetVersion(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
