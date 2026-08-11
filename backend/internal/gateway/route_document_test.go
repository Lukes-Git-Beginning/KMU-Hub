package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
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
	withAuthRequired(routes.HandleCreateFolder)(rec, req)
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

// --- Validation gap tests (previously unchecked required fields) ---

func TestHandleCreateFolder_MissingName(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/folders", jsonBody(t, map[string]interface{}{
		"space_type": "personal",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateFolder(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateTag_MissingName(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/tags", jsonBody(t, map[string]interface{}{
		"color": "#FF0000",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateTag(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleShareEntity_MissingEntityID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/shares", jsonBody(t, map[string]interface{}{
		"entity_type":         "file",
		"shared_with_user_id": "550e8400-e29b-41d4-a716-446655440001",
		"permission":          "read",
		// entity_id deliberately omitted
	}))
	req = withUserID(req, "user-123")
	routes.HandleShareEntity(rec, req)
	assertValidationError(t, rec, "entity_id")
}

func TestHandleShareEntity_MissingPermission(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/shares", jsonBody(t, map[string]interface{}{
		"entity_type":         "file",
		"entity_id":           "550e8400-e29b-41d4-a716-446655440000",
		"shared_with_user_id": "550e8400-e29b-41d4-a716-446655440001",
		// permission deliberately omitted
	}))
	req = withUserID(req, "user-123")
	routes.HandleShareEntity(rec, req)
	assertValidationError(t, rec, "permission")
}

// --- Share Links (external, unauthenticated read/download links) ---

func TestHandleListShareLinks_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/files/123/share-links", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListShareLinks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListShareLinks_InvalidIDReachesRPC documents a real gap:
// route_document.go's share-link handlers never call validateUUIDParam on
// the file id (route_document.go is one of the 161 remaining raw
// chi.URLParam(r, "id") sites left after the fix-gateway-id-validation-
// consistency unit, Iteration 6). A syntactically unusable id passes
// straight through to the RPC layer instead of getting a 400 at the
// boundary — same class of finding, not fixed here (coverage units don't
// change behavior).
func TestHandleListShareLinks_InvalidIDReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/files/not-a-uuid/share-links", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListShareLinks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateShareLink_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateShareLink)
}

func TestHandleCreateShareLink_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/123/share-links", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCreateShareLink(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// TestHandleCreateShareLink_InvalidIDReachesRPC — same finding as
// TestHandleListShareLinks_InvalidIDReachesRPC, for the create path.
func TestHandleCreateShareLink_InvalidIDReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/not-a-uuid/share-links", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withUserID(req, "user-123")
	routes.HandleCreateShareLink(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleCreateShareLink_NoCreatedByWhenAnonymous pins that CreatedBy is
// only forwarded when a user id is actually present in the request context —
// an anonymous caller must not crash the handler or forge an actor id.
func TestHandleCreateShareLink_NoCreatedByWhenAnonymous(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/documents/files/"+id+"/share-links", jsonBody(t, map[string]interface{}{
		"expires_in_days": 7,
	}))
	req = withChiURLParam(req, "id", id)
	routes.HandleCreateShareLink(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRevokeShareLink_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/share-links/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRevokeShareLink(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleRevokeShareLink_InvalidIDReachesRPC — same finding, for the
// standalone-by-link-id revoke route.
func TestHandleRevokeShareLink_InvalidIDReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/share-links/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleRevokeShareLink(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetSharedFile (unauthenticated public route) ---

func TestHandleGetSharedFile_NoAuthNeeded(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/documents/share/sometoken", nil)
	req = withChiURLParam(req, "token", "sometoken")
	routes.HandleGetSharedFile(rec, req)
	// No document backend reachable in this test — the honest answer is 503,
	// never 401/403, which would mean the public handler asked for auth.
	if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
		t.Fatalf("public share handler demanded auth: %d, body = %s", rec.Code, rec.Body.String())
	}
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetSharedFile_EmptyToken(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/documents/share/", nil)
	req = withChiURLParam(req, "token", "")
	routes.HandleGetSharedFile(rec, req)
	assertStatus(t, rec, http.StatusNotFound)
}

// TestHandleGetSharedFile_MalformedTokenLooksLikeUnknown pins that an
// over-length token gets the same 404 an unknown one gets, and never reaches
// the backend — a scraper cannot distinguish "malformed" from "no such link".
func TestHandleGetSharedFile_MalformedTokenLooksLikeUnknown(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	longToken := strings.Repeat("x", 200)
	req := httptest.NewRequest("POST", "/api/v1/public/documents/share/"+longToken, nil)
	req = withChiURLParam(req, "token", longToken)
	routes.HandleGetSharedFile(rec, req)
	assertStatus(t, rec, http.StatusNotFound)
}

func TestHandleGetSharedFile_BodyIsOptional(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))

	for name, body := range map[string]string{
		"absent": "",
		"blank":  "   ",
		"empty":  "{}",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/public/documents/share/sometoken", strings.NewReader(body))
		req = withChiURLParam(req, "token", "sometoken")
		routes.HandleGetSharedFile(rec, req)
		if rec.Code == http.StatusBadRequest {
			t.Fatalf("%s body rejected as malformed: %s", name, rec.Body.String())
		}
	}
}

func TestHandleGetSharedFile_InvalidJSONBody(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/documents/share/sometoken", strings.NewReader("{not json"))
	req = withChiURLParam(req, "token", "sometoken")
	routes.HandleGetSharedFile(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// --- Entity Links (file <-> other-entity associations) ---

func TestHandleLinkFileToEntity_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleLinkFileToEntity)
}

func TestHandleLinkFileToEntity_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/123/links", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleLinkFileToEntity(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleLinkFileToEntity_MissingEntityID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/documents/files/"+id+"/links", jsonBody(t, map[string]interface{}{
		"entity_type": "contact",
		// entity_id deliberately omitted
	}))
	req = withChiURLParam(req, "id", id)
	req = withUserID(req, "user-123")
	routes.HandleLinkFileToEntity(rec, req)
	assertValidationError(t, rec, "entity_id")
}

func TestHandleLinkFileToEntity_MissingEntityType(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/documents/files/"+id+"/links", jsonBody(t, map[string]interface{}{
		"entity_id": "550e8400-e29b-41d4-a716-446655440001",
		// entity_type deliberately omitted
	}))
	req = withChiURLParam(req, "id", id)
	req = withUserID(req, "user-123")
	routes.HandleLinkFileToEntity(rec, req)
	assertValidationError(t, rec, "entity_type")
}

func TestHandleUnlinkFileFromEntity_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUnlinkFileFromEntity)
}

func TestHandleUnlinkFileFromEntity_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/files/123/links", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUnlinkFileFromEntity(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUnlinkFileFromEntity_MissingEntityID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("DELETE", "/api/v1/documents/files/"+id+"/links", jsonBody(t, map[string]interface{}{
		"entity_type": "contact",
		// entity_id deliberately omitted
	}))
	req = withChiURLParam(req, "id", id)
	routes.HandleUnlinkFileFromEntity(rec, req)
	assertValidationError(t, rec, "entity_id")
}

func TestHandleListFileEntityLinks_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/files/123/links", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListFileEntityLinks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListFileEntityLinks_InvalidIDReachesRPC — same finding as the
// share-link handlers above: no local UUID validation on the file id.
func TestHandleListFileEntityLinks_InvalidIDReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/files/not-a-uuid/links", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListFileEntityLinks(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- File lifecycle (register/delete/copy/move/download-url/versions) ---

func TestHandleRegisterUploadedFile_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleRegisterUploadedFile)
}

func TestHandleRegisterUploadedFile_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files", invalidJSON())
	routes.HandleRegisterUploadedFile(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleRegisterUploadedFile_MissingFolderID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files", jsonBody(t, map[string]interface{}{
		"filename":    "report.pdf",
		"file_size":   1024,
		"storage_key": "uploads/report.pdf",
		// folder_id deliberately omitted
	}))
	req = withUserID(req, "user-123")
	routes.HandleRegisterUploadedFile(rec, req)
	assertValidationError(t, rec, "folder_id")
}

func TestHandleRegisterUploadedFile_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files", jsonBody(t, map[string]interface{}{
		"folder_id":   "550e8400-e29b-41d4-a716-446655440000",
		"filename":    "report.pdf",
		"file_size":   1024,
		"storage_key": "uploads/report.pdf",
	}))
	req = withUserID(req, "user-123")
	routes.HandleRegisterUploadedFile(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteFile_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/files/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleDeleteFile(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleDeleteFile_InvalidIDReachesRPC documents the same gap as the
// share-link handlers: route_document.go's file lifecycle handlers never
// call validateUUIDParam on the file id, so a syntactically unusable id
// reaches the RPC layer instead of getting a 400 at the boundary.
func TestHandleDeleteFile_InvalidIDReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/files/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteFile(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCopyFile_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCopyFile)
}

func TestHandleCopyFile_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/123/copy", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCopyFile(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCopyFile_MissingTargetFolderID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/documents/files/"+id+"/copy", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", id)
	routes.HandleCopyFile(rec, req)
	assertValidationError(t, rec, "target_folder_id")
}

func TestHandleCopyFile_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/documents/files/"+id+"/copy", jsonBody(t, map[string]interface{}{
		"target_folder_id": "550e8400-e29b-41d4-a716-446655440001",
	}))
	req = withChiURLParam(req, "id", id)
	routes.HandleCopyFile(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleMoveFile_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleMoveFile)
}

func TestHandleMoveFile_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/123/move", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleMoveFile(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleMoveFile_MissingTargetFolderID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/documents/files/"+id+"/move", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", id)
	routes.HandleMoveFile(rec, req)
	assertValidationError(t, rec, "target_folder_id")
}

func TestHandleMoveFile_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/documents/files/"+id+"/move", jsonBody(t, map[string]interface{}{
		"target_folder_id": "550e8400-e29b-41d4-a716-446655440001",
	}))
	req = withChiURLParam(req, "id", id)
	routes.HandleMoveFile(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetFileDownloadURL_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/files/123/download-url", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetFileDownloadURL(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleGetFileDownloadURL_InvalidIDReachesRPC — same known gap as the
// other file lifecycle handlers: no local UUID validation on the file id.
func TestHandleGetFileDownloadURL_InvalidIDReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/files/not-a-uuid/download-url", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetFileDownloadURL(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListFileVersions_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/files/123/versions", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleListFileVersions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListFileVersions_InvalidIDReachesRPC — same known gap as the
// other file lifecycle handlers: no local UUID validation on the file id.
func TestHandleListFileVersions_InvalidIDReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/files/not-a-uuid/versions", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListFileVersions(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRevertFileVersion_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleRevertFileVersion)
}

func TestHandleRevertFileVersion_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/123/versions/revert", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleRevertFileVersion(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// TestHandleRevertFileVersion_InvalidVersionNumber documents the actual
// request shape: revertVersionRequest.VersionNumber is an int32 (validate
// "gt=0"), not a version UUID — the backlog unit's done_when assumed an
// "invalid version id (UUID)" check, but the handler takes no version id
// parameter at all; the target version is addressed by number in the JSON
// body. A zero/negative value is what a client can actually get wrong here.
func TestHandleRevertFileVersion_InvalidVersionNumber(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/documents/files/"+id+"/versions/revert", jsonBody(t, map[string]interface{}{
		"version_number": 0,
	}))
	req = withChiURLParam(req, "id", id)
	routes.HandleRevertFileVersion(rec, req)
	assertValidationError(t, rec, "version_number")
}

func TestHandleRevertFileVersion_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	id := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest("POST", "/api/v1/documents/files/"+id+"/versions/revert", jsonBody(t, map[string]interface{}{
		"version_number": 2,
	}))
	req = withChiURLParam(req, "id", id)
	routes.HandleRevertFileVersion(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
