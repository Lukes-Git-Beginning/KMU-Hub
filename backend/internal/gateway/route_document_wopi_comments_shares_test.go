package gateway

// route_document_wopi_comments_shares_test.go covers the nineteen route_document.go
// handlers that had zero test coverage as of the cov-gateway-document-wopi-comments-shares
// backlog unit (2026-08-24): folder path/space init, direct multipart upload, file
// activity/comments, entity link deletion, entity shares, tags, search, virtual files
// and the two WOPI handlers. Like the other document-route test files, this package has
// no fake DocumentServiceClient — "ReachesRPC" asserts the handler clears its own
// validation and hits the (unreachable) dummy connection, which manifests as 503.
//
// WOPI token findings (scope question 1, "Laufzeit, Dateibindung, Tenant-Bindung, gilt
// der Token nach Entzug der Freigabe noch?"), verified by reading
// internal/document/wopi/token.go and internal/server/document_grpc.go:1382-1408, not by
// a new test in this gateway-only package (no reachable token service here):
//   - TTL is fixed at 10h (wopiTokenTTL, token.go:11), matching the WOPI spec
//     recommendation.
//   - file_id and tenant_id are embedded as JWT claims at issuance (document_grpc.go:1397)
//     and re-checked by every WOPI content handler (wopi/handler.go: claims.FileID vs the
//     URL file_id, and tenantID from claims scopes the fileService.GetByID lookup — a
//     cross-tenant file_id fails via RLS, not a Go-side filter, same pattern already
//     verified for internal/work/recording in Iteration 24).
//   - There is no per-file share/grant to "entzogen" (revoke) here: WOPI editing is
//     gated only by the coarse tenant-level "documents:write" permission
//     (route_document.go:187), the same gate every other file-write action in this
//     module uses. Unlike the external share-link flow (which DOES have an explicit
//     revoke, RevokeShareLink), there is no revocable per-file WOPI grant to test —
//     the token is stateless and valid for its full 10h TTL by design, consistent with
//     the rest of the module. Not a new finding; no fix-unit raised.
//   - GenerateWOPIToken never checks the file actually exists before issuing a token
//     (internal/server/document_grpc.go:1382-1408 does no fileService lookup at all) —
//     a token for a nonexistent or already-deleted file_id is issued anyway and only
//     fails downstream at CheckFileInfo/GetFile time. Low severity (no data exposure,
//     matches the "existence not verified at issuance" shape already accepted for
//     share links pre-fix) — tracked as neue-units, see JOURNAL.md.

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
)

// multipartUploadBody builds a multipart/form-data body for HandleUploadFile: regular
// form fields plus one file part with an explicit, caller-chosen Content-Type (unlike
// multipart.Writer.CreateFormFile, which hardcodes application/octet-stream and can
// never exercise the allowedDocumentUploadMimeTypes rejection path).
func multipartUploadBody(t *testing.T, fields map[string]string, fileField, fileName, fileContentType string, fileContent []byte) (*bytes.Buffer, string) {
	t.Helper()
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatalf("write field %q: %v", k, err)
		}
	}
	if fileField != "" {
		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition", `form-data; name="`+fileField+`"; filename="`+fileName+`"`)
		if fileContentType != "" {
			header.Set("Content-Type", fileContentType)
		}
		fw, err := w.CreatePart(header)
		if err != nil {
			t.Fatalf("create form part: %v", err)
		}
		if _, err := fw.Write(fileContent); err != nil {
			t.Fatalf("write file content: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return buf, w.FormDataContentType()
}

const testFolderID = "550e8400-e29b-41d4-a716-446655440000"

// --- HandleGetFolderPath ---

func TestHandleGetFolderPath_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/folders/"+testFolderID+"/path", nil)
	req = withChiURLParam(req, "id", testFolderID)
	routes.HandleGetFolderPath(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetFolderPath_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/folders/"+testFolderID+"/path", nil)
	req = withChiURLParam(req, "id", testFolderID)
	routes.HandleGetFolderPath(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleInitializeUserSpace ---

func TestHandleInitializeUserSpace_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleInitializeUserSpace)
}

// TestHandleInitializeUserSpace_EmptyBodyFallsBackToJWTIdentity pins the
// lean-tolerate-empty-body path: ContentLength 0 must not be treated as
// invalid JSON, and the handler must still reach the RPC layer using the
// JWT-derived user id.
func TestHandleInitializeUserSpace_EmptyBodyFallsBackToJWTIdentity(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/folders/initialize-user", nil)
	req = withUserID(req, "user-123")
	routes.HandleInitializeUserSpace(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleInitializeUserSpace_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/folders/initialize-user", invalidJSON())
	routes.HandleInitializeUserSpace(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleInitializeUserSpace_ExplicitUserIDReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/folders/initialize-user", jsonBody(t, map[string]interface{}{
		"user_id": "550e8400-e29b-41d4-a716-446655440001",
	}))
	routes.HandleInitializeUserSpace(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleInitializeTeamSpace ---

func TestHandleInitializeTeamSpace_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleInitializeTeamSpace)
}

func TestHandleInitializeTeamSpace_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/folders/initialize-team", invalidJSON())
	routes.HandleInitializeTeamSpace(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleInitializeTeamSpace_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/folders/initialize-team", jsonBody(t, map[string]interface{}{
		"team_id": "550e8400-e29b-41d4-a716-446655440001",
	}))
	routes.HandleInitializeTeamSpace(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUploadFile ---

func TestHandleUploadFile_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUploadFile)
}

func TestHandleUploadFile_InvalidMultipartForm(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/upload", strings.NewReader("not-multipart"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=broken")
	req = withUserID(req, "user-123")
	routes.HandleUploadFile(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "failed to parse multipart form")
}

func TestHandleUploadFile_MissingFolderID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	body, contentType := multipartUploadBody(t, nil, "file", "report.pdf", "application/pdf", []byte("%PDF-1.4"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/upload", body)
	req.Header.Set("Content-Type", contentType)
	req = withUserID(req, "user-123")
	routes.HandleUploadFile(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "folder_id")
}

func TestHandleUploadFile_MissingFile(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	body, contentType := multipartUploadBody(t, map[string]string{"folder_id": testFolderID}, "", "", "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/upload", body)
	req.Header.Set("Content-Type", contentType)
	req = withUserID(req, "user-123")
	routes.HandleUploadFile(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "file")
}

func TestHandleUploadFile_DisallowedMimeType(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	body, contentType := multipartUploadBody(t, map[string]string{"folder_id": testFolderID}, "file", "script.exe", "application/x-msdownload", []byte("MZ"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/upload", body)
	req.Header.Set("Content-Type", contentType)
	req = withUserID(req, "user-123")
	routes.HandleUploadFile(rec, req)
	assertStatus(t, rec, http.StatusUnsupportedMediaType)
	assertErrorContains(t, rec, "content type not allowed")
}

// TestHandleUploadFile_SVGRejected pins that image/svg+xml — deliberately left
// out of allowedDocumentUploadMimeTypes per the doc comment above the map — is
// actually rejected, since an SVG can carry a <script> and this upload path
// has no sanitization step before the WOPI viewer can open it inline.
func TestHandleUploadFile_SVGRejected(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	body, contentType := multipartUploadBody(t, map[string]string{"folder_id": testFolderID}, "file", "image.svg", "image/svg+xml", []byte("<svg/>"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/upload", body)
	req.Header.Set("Content-Type", contentType)
	req = withUserID(req, "user-123")
	routes.HandleUploadFile(rec, req)
	assertStatus(t, rec, http.StatusUnsupportedMediaType)
}

func TestHandleUploadFile_EmptyFile(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	body, contentType := multipartUploadBody(t, map[string]string{"folder_id": testFolderID}, "file", "empty.txt", "text/plain", []byte{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/upload", body)
	req.Header.Set("Content-Type", contentType)
	req = withUserID(req, "user-123")
	routes.HandleUploadFile(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "file is empty")
}

// TestHandleUploadFile_ExceedsSizeLimit pins the 50 MiB ceiling
// (maxDocumentUploadBytes) shared with the presign upload path.
func TestHandleUploadFile_ExceedsSizeLimit(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	oversized := bytes.Repeat([]byte("a"), maxDocumentUploadBytes+1)
	body, contentType := multipartUploadBody(t, map[string]string{"folder_id": testFolderID}, "file", "huge.txt", "text/plain", oversized)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/upload", body)
	req.Header.Set("Content-Type", contentType)
	req = withUserID(req, "user-123")
	routes.HandleUploadFile(rec, req)
	assertStatus(t, rec, http.StatusRequestEntityTooLarge)
	assertErrorContains(t, rec, "50 MiB")
}

func TestHandleUploadFile_ValidRequestReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	body, contentType := multipartUploadBody(t, map[string]string{
		"folder_id": testFolderID,
		"tag_id":    "550e8400-e29b-41d4-a716-446655440002",
	}, "file", "report.pdf", "application/pdf", []byte("%PDF-1.4 content"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/upload", body)
	req.Header.Set("Content-Type", contentType)
	req = withUserID(req, "user-123")
	routes.HandleUploadFile(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListFileActivity ---

func TestHandleListFileActivity_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/files/"+testFolderID+"/activity", nil)
	req = withChiURLParam(req, "id", testFolderID)
	routes.HandleListFileActivity(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListFileComments / Create / Update / Delete ---

func TestHandleListFileComments_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/files/"+testFolderID+"/comments", nil)
	req = withChiURLParam(req, "id", testFolderID)
	routes.HandleListFileComments(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateFileComment_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateFileComment)
}

func TestHandleCreateFileComment_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/"+testFolderID+"/comments", invalidJSON())
	req = withChiURLParam(req, "id", testFolderID)
	routes.HandleCreateFileComment(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateFileComment_MissingContent(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/"+testFolderID+"/comments", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", testFolderID)
	routes.HandleCreateFileComment(rec, req)
	assertValidationError(t, rec, "content")
}

func TestHandleCreateFileComment_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/files/"+testFolderID+"/comments", jsonBody(t, map[string]interface{}{
		"content": "looks good",
	}))
	req = withChiURLParam(req, "id", testFolderID)
	routes.HandleCreateFileComment(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateFileComment_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdateFileComment)
}

func TestHandleUpdateFileComment_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/documents/comments/"+testFolderID, invalidJSON())
	req = withChiURLParam(req, "id", testFolderID)
	routes.HandleUpdateFileComment(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateFileComment_MissingContent(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/documents/comments/"+testFolderID, jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", testFolderID)
	routes.HandleUpdateFileComment(rec, req)
	assertValidationError(t, rec, "content")
}

func TestHandleUpdateFileComment_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/documents/comments/"+testFolderID, jsonBody(t, map[string]interface{}{
		"content": "edited",
	}))
	req = withChiURLParam(req, "id", testFolderID)
	routes.HandleUpdateFileComment(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteFileComment_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/comments/"+testFolderID, nil)
	req = withChiURLParam(req, "id", testFolderID)
	routes.HandleDeleteFileComment(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteFileComment_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/comments/"+testFolderID, nil)
	req = withChiURLParam(req, "id", testFolderID)
	routes.HandleDeleteFileComment(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetFileVersionDownloadURL ---

func TestHandleGetFileVersionDownloadURL_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/files/"+testFolderID+"/versions/"+testFolderID+"/download", nil)
	req = withChiURLParam(req, "id", testFolderID)
	req = withChiURLParam(req, "versionId", testFolderID)
	routes.HandleGetFileVersionDownloadURL(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteEntityLink ---

func TestHandleDeleteEntityLink_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/links/"+testFolderID, nil)
	req = withChiURLParam(req, "id", testFolderID)
	routes.HandleDeleteEntityLink(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteEntityLink_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/links/"+testFolderID, nil)
	req = withChiURLParam(req, "id", testFolderID)
	routes.HandleDeleteEntityLink(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUnshareEntity ---

func TestHandleUnshareEntity_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUnshareEntity)
}

func TestHandleUnshareEntity_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/shares", invalidJSON())
	routes.HandleUnshareEntity(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUnshareEntity_MissingEntityID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/shares", jsonBody(t, map[string]interface{}{
		"entity_type":         "contact",
		"shared_with_user_id": "550e8400-e29b-41d4-a716-446655440001",
	}))
	routes.HandleUnshareEntity(rec, req)
	assertValidationError(t, rec, "entity_id")
}

func TestHandleUnshareEntity_MissingSharedWithUserID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/shares", jsonBody(t, map[string]interface{}{
		"entity_type": "contact",
		"entity_id":   testFolderID,
	}))
	routes.HandleUnshareEntity(rec, req)
	assertValidationError(t, rec, "shared_with_user_id")
}

func TestHandleUnshareEntity_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/shares", jsonBody(t, map[string]interface{}{
		"entity_type":         "contact",
		"entity_id":           testFolderID,
		"shared_with_user_id": "550e8400-e29b-41d4-a716-446655440001",
	}))
	routes.HandleUnshareEntity(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListShares ---

func TestHandleListShares_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/shares/entity?entity_type=contact&entity_id="+testFolderID, nil)
	routes.HandleListShares(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListShares_MissingEntityType(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/shares/entity?entity_id="+testFolderID, nil)
	routes.HandleListShares(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "entity_type and entity_id are required")
}

func TestHandleListShares_MissingEntityID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/shares/entity?entity_type=contact", nil)
	routes.HandleListShares(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "entity_type and entity_id are required")
}

func TestHandleListShares_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/shares/entity?entity_type=contact&entity_id="+testFolderID, nil)
	routes.HandleListShares(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleTagFile / HandleUntagFile ---

func TestHandleTagFile_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleTagFile)
}

func TestHandleTagFile_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/tags/file", invalidJSON())
	routes.HandleTagFile(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleTagFile_MissingFileID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/tags/file", jsonBody(t, map[string]interface{}{
		"tag_id": testFolderID,
	}))
	routes.HandleTagFile(rec, req)
	assertValidationError(t, rec, "file_id")
}

func TestHandleTagFile_MissingTagID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/tags/file", jsonBody(t, map[string]interface{}{
		"file_id": testFolderID,
	}))
	routes.HandleTagFile(rec, req)
	assertValidationError(t, rec, "tag_id")
}

func TestHandleTagFile_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/tags/file", jsonBody(t, map[string]interface{}{
		"file_id": testFolderID,
		"tag_id":  "550e8400-e29b-41d4-a716-446655440001",
	}))
	routes.HandleTagFile(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUntagFile_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUntagFile)
}

func TestHandleUntagFile_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/tags/file", invalidJSON())
	routes.HandleUntagFile(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUntagFile_MissingFileID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/tags/file", jsonBody(t, map[string]interface{}{
		"tag_id": testFolderID,
	}))
	routes.HandleUntagFile(rec, req)
	assertValidationError(t, rec, "file_id")
}

func TestHandleUntagFile_MissingTagID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/tags/file", jsonBody(t, map[string]interface{}{
		"file_id": testFolderID,
	}))
	routes.HandleUntagFile(rec, req)
	assertValidationError(t, rec, "tag_id")
}

func TestHandleUntagFile_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/documents/tags/file", jsonBody(t, map[string]interface{}{
		"file_id": testFolderID,
		"tag_id":  "550e8400-e29b-41d4-a716-446655440001",
	}))
	routes.HandleUntagFile(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleSearchFiles ---

func TestHandleSearchFiles_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/search?q=invoice", nil)
	routes.HandleSearchFiles(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSearchFiles_MissingQuery(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/search", nil)
	routes.HandleSearchFiles(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "query parameter 'q' is required")
}

func TestHandleSearchFiles_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/search?q=invoice&folder_id="+testFolderID+"&tag_ids=a&tag_ids=b", nil)
	routes.HandleSearchFiles(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListVirtualFiles ---

func TestHandleListVirtualFiles_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/virtual", nil)
	routes.HandleListVirtualFiles(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListVirtualFiles_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/virtual?source=favorites&page=2&page_size=10", nil)
	routes.HandleListVirtualFiles(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGenerateWOPIToken ---

func TestHandleGenerateWOPIToken_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGenerateWOPIToken)
}

func TestHandleGenerateWOPIToken_InvalidJSON(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/wopi/token", invalidJSON())
	routes.HandleGenerateWOPIToken(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGenerateWOPIToken_MissingFileID(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/wopi/token", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	routes.HandleGenerateWOPIToken(rec, req)
	assertValidationError(t, rec, "file_id")
}

func TestHandleGenerateWOPIToken_InvalidFileIDFormat(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/wopi/token", jsonBody(t, map[string]interface{}{
		"file_id": "not-a-uuid",
	}))
	req = withUserID(req, "user-123")
	routes.HandleGenerateWOPIToken(rec, req)
	assertValidationError(t, rec, "file_id")
}

func TestHandleGenerateWOPIToken_ReachesRPC(t *testing.T) {
	routes := NewDocumentRoutes(registryWithService("document"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/documents/wopi/token", jsonBody(t, map[string]interface{}{
		"file_id": testFolderID,
	}))
	req = withUserID(req, "user-123")
	routes.HandleGenerateWOPIToken(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetWOPIDiscovery ---

func TestHandleGetWOPIDiscovery_ServiceUnavailable(t *testing.T) {
	routes := NewDocumentRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/documents/wopi/discovery", nil)
	routes.HandleGetWOPIDiscovery(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
