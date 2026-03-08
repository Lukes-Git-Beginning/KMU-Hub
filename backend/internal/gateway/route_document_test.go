package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDocumentRoutes_ServiceName(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	if routes.ServiceName() != "document" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "document")
	}
}

// --- Folders ---

func TestHandleCreateFolder_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateFolder)
}

func TestHandleCreateFolder_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/folders", invalidJSON())
	routes.HandleCreateFolder(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateFolder_NoUserID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/folders", jsonBody(t, map[string]interface{}{
		"name": "Test Folder",
	}))
	routes.HandleCreateFolder(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "not authenticated")
}

func TestHandleGetFolder_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/folders/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetFolder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListFolders_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/folders", nil)
	routes.HandleListFolders(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateFolder_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/documents/folders/123", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateFolder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteFolder_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/folders/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteFolder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- Files ---

func TestHandleListFiles_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/files", nil)
	routes.HandleListFiles(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetFile_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/files/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetFile(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateFile_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/documents/files/123", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateFile(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateFile_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/documents/files/123", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateFile(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// --- Shares ---

func TestHandleShareEntity_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleShareEntity)
}

func TestHandleShareEntity_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/shares", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleShareEntity(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListSharedWithMe_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/shares/shared-with-me", nil)
	req = withUserID(req, "user-123")
	routes.HandleListSharedWithMe(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
