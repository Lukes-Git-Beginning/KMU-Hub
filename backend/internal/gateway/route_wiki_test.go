package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/featureflag"
)

func newWikiRoutes(registry *ServiceRegistry) *WikiRoutes {
	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_WIKI_ENABLED" {
			return "true"
		}
		return ""
	})
	return NewWikiRoutes(registry, flags)
}

// --- ServiceName ---

func TestWikiRoutes_ServiceName(t *testing.T) {
	routes := newWikiRoutes(emptyRegistry())
	if routes.ServiceName() != "wiki" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "wiki")
	}
}

// --- HandleCreateArticle ---

func TestHandleCreateArticle_ServiceUnavailable(t *testing.T) {
	routes := newWikiRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/wiki/articles/", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateArticle(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateArticle_InvalidJSON(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/wiki/articles/", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateArticle(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateArticle_MissingTitle(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/wiki/articles/", jsonBody(t, map[string]interface{}{
		"published": false,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateArticle(rec, req)
	assertValidationError(t, rec, "title")
}

func TestHandleCreateArticle_InvalidCategoryID(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/wiki/articles/", jsonBody(t, map[string]interface{}{
		"title":       "My article",
		"category_id": "not-a-uuid",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateArticle(rec, req)
	assertValidationError(t, rec, "category_id")
}

// --- HandleUploadAttachment ---

func TestHandleUploadAttachment_ServiceUnavailable(t *testing.T) {
	routes := newWikiRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUploadAttachment(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUploadAttachment_InvalidJSON(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUploadAttachment(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUploadAttachment_MissingFileRef(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{
		"mime": "image/png",
		"size": 1024,
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUploadAttachment(rec, req)
	assertValidationError(t, rec, "file_ref")
}

// --- HandleCreateCategory ---

func TestHandleCreateCategory_ServiceUnavailable(t *testing.T) {
	routes := newWikiRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateCategory(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateCategory_MissingName(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{
		"position": 1,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateCategory(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateCategory_InvalidParentID(t *testing.T) {
	routes := newWikiRoutes(registryWithService("wiki"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{
		"name":      "Sub-category",
		"parent_id": "not-a-uuid",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateCategory(rec, req)
	assertValidationError(t, rec, "parent_id")
}

// --- Public share redemption ---

// wikiFlagsOFF is a registry with modules.wiki disabled.
func wikiFlagsOFF() *featureflag.Registry {
	return featureflag.NewRegistry().Load(func(string) string { return "" })
}

// TestWikiPublicShareRoute_Registered pins that the redemption is mounted under
// /api/v1/public/ as a POST. The verb is not cosmetic: a GET would leave the
// token in access logs, browser history and Referer headers, and let a prefetch
// redeem the link.
func TestWikiPublicShareRoute_Registered(t *testing.T) {
	r := chi.NewRouter()
	newWikiRoutes(emptyRegistry()).RegisterPublicRoutes(r, passthroughLimiter)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/wiki/articles/sometoken", nil)
	r.ServeHTTP(rec, req)
	if rec.Code == http.StatusNotFound {
		t.Fatalf("public share route not registered; got 404, body = %s", rec.Body.String())
	}

	// The same path as a GET must not answer at all.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/public/wiki/articles/sometoken", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound {
		t.Fatalf("GET on the public share route answered %d; it must not be a GET", rec.Code)
	}
}

// TestWikiPublicShareRoute_FlagOFF pins that the module flag also gates the
// public path — an unauthenticated route surviving a disabled module would be
// the worst one to forget.
func TestWikiPublicShareRoute_FlagOFF(t *testing.T) {
	r := chi.NewRouter()
	NewWikiRoutes(emptyRegistry(), wikiFlagsOFF()).RegisterPublicRoutes(r, passthroughLimiter)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/wiki/articles/sometoken", nil)
	r.ServeHTTP(rec, req)
	assertStatus(t, rec, http.StatusNotFound)
}

// TestHandleRedeemShareToken_NoAuthNeeded pins that the public handler never
// reads a tenant or user from the request: it must work with a context that
// carries neither, because a visitor has neither.
func TestHandleRedeemShareToken_NoAuthNeeded(t *testing.T) {
	r := chi.NewRouter()
	newWikiRoutes(emptyRegistry()).RegisterPublicRoutes(r, passthroughLimiter)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/wiki/articles/sometoken", nil)
	r.ServeHTTP(rec, req)

	// No wiki backend is reachable in this test, so the honest answer is 503 —
	// but never 401/403, which would mean the handler asked for auth.
	if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
		t.Fatalf("public share handler demanded auth: %d, body = %s", rec.Code, rec.Body.String())
	}
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// An over-long token is answered before the service is ever reached, and with
// the same 404 an unknown token gets — a probe must not cost a round trip, and
// "malformed" must not be distinguishable from "no such link".
func TestHandleRedeemShareToken_OverlongTokenIsNotFound(t *testing.T) {
	r := chi.NewRouter()
	newWikiRoutes(registryWithService("wiki")).RegisterPublicRoutes(r, passthroughLimiter)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/public/wiki/articles/"+strings.Repeat("a", 129), nil)
	r.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusNotFound)
}
