package gateway

// route_crm_contact_files_test.go covers route_crm_contact_files.go (unit
// cov-gateway-crm-contact-files-wopi), which had no test file. Every handler
// calls verifyContactOwnership first (CRM GetContact) before touching the
// Document service -- and, like every prior gateway coverage unit this run,
// this package has no bufconn/fake-client harness for CRMServiceClient or
// DocumentServiceClient, so a *successful* ownership check (and therefore
// the folder-resolution/file-registration branches behind it,
// resolveContactFilesFolder/findContactFilesFolder/getDocumentClient) can't
// be exercised here. verifyContactOwnership's actual cross-tenant
// enforcement lives server-side in CRM's GetContact RPC and is proven at
// internal/crm/contact/service_test.go
// TestService_CrossTenant_GetByID_DifferentTenantGetsNotFound and
// TestService_GetByID_InvalidTenant: a foreign or missing tenant contact
// there returns ErrNotFound, which this file's handlers translate via
// respondGRPCError into 404 (grpcStatusToHTTP maps codes.NotFound), never a
// stranger's contact ID succeeding.
//
// What IS provable at this level is proven below: the ownership check runs
// BEFORE anything else, including request body parsing -- an invalid JSON
// body on HandleCreateContactFile still yields 503 (ownership check fails
// first against the fake CRM connection), not 400.

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

const contactFilesTestID = "10101010-1010-1010-1010-101010101010"

// ============================================================================
// ServiceUnavailable — both handlers resolve the CRM client first.
// ============================================================================

func TestContactFilesRoutes_ServiceUnavailable(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)

	idReq := func(method string) *http.Request {
		r := httptest.NewRequest(method, "/", nil)
		return withChiURLParam(r, "id", contactFilesTestID)
	}

	handlers := map[string]http.HandlerFunc{
		"HandleListContactFiles": func(w http.ResponseWriter, r *http.Request) {
			routes.HandleListContactFiles(w, idReq(http.MethodGet))
		},
		"HandleCreateContactFile": func(w http.ResponseWriter, r *http.Request) {
			routes.HandleCreateContactFile(w, idReq(http.MethodPost))
		},
	}

	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			testServiceUnavailable(t, h)
		})
	}
}

// ============================================================================
// Invalid contact ID — rejected before any RPC.
// ============================================================================

func TestHandleListContactFiles_InvalidContactID(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/not-a-uuid/files", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListContactFiles(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateContactFile_InvalidContactID(t *testing.T) {
	routes := NewCRMRoutes(emptyRegistry(), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/not-a-uuid/files", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCreateContactFile(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

// ============================================================================
// HandleListContactFiles — reaches the CRM ownership-check RPC.
// ============================================================================

func TestHandleListContactFiles_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/"+contactFilesTestID+"/files", nil)
	req = withChiURLParam(req, "id", contactFilesTestID)
	routes.HandleListContactFiles(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCreateContactFile — the ownership check runs before body parsing.
// ============================================================================

func TestHandleCreateContactFile_ReachesRPC(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/"+contactFilesTestID+"/files", jsonBody(t, map[string]any{
		"filename":    "vertrag.pdf",
		"storage_key": "documents/abc123",
		"file_size":   1024,
	}))
	req = withChiURLParam(req, "id", contactFilesTestID)
	routes.HandleCreateContactFile(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateContactFile_OwnershipCheckedBeforeBodyValidation(t *testing.T) {
	routes := NewCRMRoutes(registryWithService("crm"), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts/"+contactFilesTestID+"/files", invalidJSON())
	req = withChiURLParam(req, "id", contactFilesTestID)
	routes.HandleCreateContactFile(rec, req)
	// If body validation ran first, this would be 400 ("invalid request
	// body"). It is 503 instead: the ownership check against the fake CRM
	// connection fails before the body is ever decoded.
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// toContactFileResponse — the flat wire shape mirrors the desktop client's
// ContactFile type, not the nested DocumentFile proto message.
// ============================================================================

func TestToContactFileResponse_FlatShape(t *testing.T) {
	createdAt := time.Date(2026, 8, 20, 10, 30, 0, 0, time.UTC)
	f := &documentv1.DocumentFile{
		Id:         "file-1",
		Filename:   "vertrag.pdf",
		MimeType:   "application/pdf",
		StorageKey: "documents/abc123",
		CreatedAt:  timestamppb.New(createdAt),
	}

	resp := toContactFileResponse(contactFilesTestID, f)

	if resp.ID != "file-1" {
		t.Errorf("ID = %q, want %q", resp.ID, "file-1")
	}
	if resp.ContactID != contactFilesTestID {
		t.Errorf("ContactID = %q, want %q", resp.ContactID, contactFilesTestID)
	}
	if resp.Filename != "vertrag.pdf" {
		t.Errorf("Filename = %q, want %q", resp.Filename, "vertrag.pdf")
	}
	if resp.MimeType != "application/pdf" {
		t.Errorf("MimeType = %q, want %q", resp.MimeType, "application/pdf")
	}
	if resp.StorageKey != "documents/abc123" {
		t.Errorf("StorageKey = %q, want %q", resp.StorageKey, "documents/abc123")
	}
	if want := createdAt.Format(time.RFC3339); resp.CreatedAt != want {
		t.Errorf("CreatedAt = %q, want %q", resp.CreatedAt, want)
	}
}
