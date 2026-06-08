package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
