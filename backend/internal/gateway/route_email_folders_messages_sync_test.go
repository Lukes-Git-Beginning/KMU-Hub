package gateway

// route_email_folders_messages_sync_test.go closes the remaining gap in
// route_email.go's "Kernpfad Nachrichten/Ordner/Sync" (unit
// cov-gateway-email-messages-folders-sync). route_email_accounts_test.go and
// route_email_compose_test.go (Lauf 8, commits b46fc88b/071b5bf1) already cover
// every account, send/compose, bulk-action, move/delete/mark-read handler in
// this file -- the Lauf 10 backlog's premise that route_email.go has "keine
// eigene Testdatei" undercounted that prior work. Measured before writing this
// file (go tool cover -func on route_email.go): the only 0%-covered functions
// on the messages/folders/sync/attachments path are the ten covered below, plus
// the trivial ServiceName getter.
//
// This package has no bufconn/fake-client harness for EmailServiceClient, so a
// successful RPC round trip -- and with it "fremder Tenant liefert 404" for a
// message ID -- can't be exercised at the handler level (HandleGetMessage
// forwards chi.URLParam(r, "id") verbatim with no tenant check of its own).
// That boundary is enforced server-side and already has a DB-backed proof:
// internal/email/message/postgres_repository_test.go,
// TestPostgresRepository_CreateAndGetByID/"GetByID is tenant-scoped" seeds a
// message under tenant A and shows GetByID(id, tenantOther) returns
// ErrMessageNotFound -- which the gRPC layer above it translates to NotFound
// and this gateway maps to 404 via respondGRPCError.
//
// Test names are prefixed with "Email" (TestEmailHandleX, not TestHandleX) --
// route_bexio.go, route_document.go and route_wiki.go all declare handler
// methods with the exact same names (TriggerSync, GetSyncStatus, GetFolder,
// ListFolders, UploadAttachment) on their own route structs, and Go test
// function names are package-scoped regardless of receiver type.

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmailRoutes_ServiceName(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	if got := routes.ServiceName(); got != "email" {
		t.Errorf("ServiceName() = %q, want %q", got, "email")
	}
}

// ============================================================================
// Folder Handlers
// ============================================================================

func TestEmailHandleListFolders_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleListFolders)
}

func TestEmailHandleListFolders_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/email/folders?account_id=acc-1", nil)
	routes.HandleListFolders(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleGetFolder_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleGetFolder)
}

func TestEmailHandleGetFolder_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/email/folders/f-1", nil)
	req = withChiURLParam(req, "id", "f-1")
	routes.HandleGetFolder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleSyncFolders_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleSyncFolders)
}

func TestEmailHandleSyncFolders_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/folders/sync", invalidJSON())
	routes.HandleSyncFolders(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestEmailHandleSyncFolders_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/folders/sync", jsonBody(t, map[string]interface{}{
		"account_id": "acc-1",
	}))
	routes.HandleSyncFolders(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// Message Handlers
// ============================================================================

func TestEmailHandleGetMessage_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleGetMessage)
}

func TestEmailHandleGetMessage_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/email/messages/m-1", nil)
	req = withChiURLParam(req, "id", "m-1")
	routes.HandleGetMessage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleGetThreadMessages_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleGetThreadMessages)
}

func TestEmailHandleGetThreadMessages_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/email/messages/thread/t-1", nil)
	req = withChiURLParam(req, "threadId", "t-1")
	routes.HandleGetThreadMessages(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// Sync Handlers
// ============================================================================

func TestEmailHandleTriggerSync_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleTriggerSync)
}

func TestEmailHandleTriggerSync_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/sync/trigger", invalidJSON())
	routes.HandleTriggerSync(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestEmailHandleTriggerSync_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/sync/trigger", jsonBody(t, map[string]interface{}{
		"account_id": "acc-1",
	}))
	routes.HandleTriggerSync(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleGetSyncStatus_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleGetSyncStatus)
}

func TestEmailHandleGetSyncStatus_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/email/sync/status?account_id=acc-1", nil)
	routes.HandleGetSyncStatus(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleSetReadFlag_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleSetReadFlag)
}

func TestEmailHandleSetReadFlag_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/sync/read-flag", invalidJSON())
	routes.HandleSetReadFlag(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestEmailHandleSetReadFlag_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/sync/read-flag", jsonBody(t, map[string]interface{}{
		"message_id": "m-1",
		"is_read":    true,
	}))
	routes.HandleSetReadFlag(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// Attachment Handlers
// ============================================================================

func TestEmailHandleUploadAttachment_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleUploadAttachment)
}

func TestEmailHandleUploadAttachment_InvalidMultipartForm(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/attachments/upload", bytes.NewReader([]byte("not multipart")))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	routes.HandleUploadAttachment(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid multipart form")
}

func TestEmailHandleUploadAttachment_MissingFile(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	body, contentType := multipartBody(t, map[string]string{"account_id": "acc-1"}, "", "", nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/attachments/upload", body)
	req.Header.Set("Content-Type", contentType)
	routes.HandleUploadAttachment(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "file is required")
}

func TestEmailHandleUploadAttachment_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	body, contentType := multipartBody(t, map[string]string{"account_id": "acc-1"}, "file", "invoice.pdf", []byte("%PDF-1.4"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/email/attachments/upload", body)
	req.Header.Set("Content-Type", contentType)
	routes.HandleUploadAttachment(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestEmailHandleGetAttachmentDownloadURL_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	testEmailServiceUnavailable(t, routes.HandleGetAttachmentDownloadURL)
}

func TestEmailHandleGetAttachmentDownloadURL_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/email/attachments/att-1/download", nil)
	req = withChiURLParam(req, "id", "att-1")
	routes.HandleGetAttachmentDownloadURL(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// testEmailServiceUnavailable is like testServiceUnavailable but for
// route_email.go's handlers, which return 502 (not 503) when the service isn't
// registered: every one of the 58 client-fetch-error branches in that file
// hand-writes response.Error(w, http.StatusBadGateway, "email service
// unavailable") instead of calling the shared respondServiceUnavailable
// helper (503) that every other route file uses -- a real, consistent
// deviation, not a test-fixture quirk. See
// fix-gateway-email-service-unavailable-status-code in BACKLOG.yml.
func testEmailServiceUnavailable(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{}))
	handler(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}
