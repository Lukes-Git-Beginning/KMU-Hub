package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/server/response"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// The Account handlers below are proto-direct: they json.Decode straight
// into the proto request struct (no local DTO, no validate tags -- see the
// "proto-direct: no local DTO (S4.1 boundary)" comments in route_email.go)
// and pass IDs from chi.URLParam straight to the RPC without a
// validateUUIDParam call. There is therefore no local "missing required
// field" or "invalid UUID" error path to exercise here, unlike e.g.
// route_video.go's meeting handlers -- the closest available error paths
// are a malformed JSON body and the RPC layer itself once the request
// reaches the (unreachable in this package) Email service.

// ============================================================================
// HandleCreateAccount
// ============================================================================

func TestHandleCreateAccount_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/accounts/", jsonBody(t, map[string]interface{}{
		"email_address": "user@example.com",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateAccount(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleCreateAccount_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/accounts/", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateAccount(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateAccount_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/accounts/", jsonBody(t, map[string]interface{}{
		"email_address": "user@example.com",
		"imap_host":     "imap.example.com",
		"smtp_host":     "smtp.example.com",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateAccount(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetAccount
// ============================================================================

func TestHandleGetAccount_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/email/accounts/?user_id="+uuid.New().String(), nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleGetAccount(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleGetAccount_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/email/accounts/?user_id="+uuid.New().String(), nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleGetAccount(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListAccounts
// ============================================================================

func TestHandleListAccounts_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/email/accounts/list", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListAccounts(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleListAccounts_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/email/accounts/list", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListAccounts(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestListEmailAccountsResponse_EmptyWireShape locks the actual wire shape
// HandleListAccounts produces via response.JSON(w, status, resp): unlike the
// route_video.go list handlers, this response is *not* run through
// response.Proto/ProtoList -- it is encoding/json.Marshal on the raw proto
// struct. ListEmailAccountsResponse.Accounts carries a `json:"...,omitempty"`
// tag (protoc-gen-go default), so a nil/empty slice is not serialised as
// "accounts":[] and not as "accounts":null either -- the key is omitted
// entirely. The frontend (desktop/src/renderer/src/modules/mails/MailsPage.tsx,
// MailsSettingsPanel.tsx) already guards with `accountsData?.accounts ?? []`,
// so this is not a blocker, but it does mean the done_when expectation of
// "leer -> [] statt null" does not hold for this handler -- it's a third,
// undocumented option (key absent).
func TestListEmailAccountsResponse_EmptyWireShape(t *testing.T) {
	rec := httptest.NewRecorder()
	resp := &emailv1.ListEmailAccountsResponse{}
	if resp.Accounts != nil {
		t.Fatalf("expected zero-value Accounts to be nil, got %v", resp.Accounts)
	}

	response.JSON(rec, http.StatusOK, resp)

	body := rec.Body.String()
	if body != "{}\n" {
		t.Errorf("empty ListEmailAccountsResponse serialised as %q, want %q (accounts key omitted, not [] or null)", body, "{}\n")
	}
	var sink map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &sink); err != nil {
		t.Fatalf("response is not valid JSON object: %v", err)
	}
	if _, present := sink["accounts"]; present {
		t.Errorf("expected \"accounts\" key to be absent for an empty list, but it was present")
	}
}

// ============================================================================
// HandleUpdateAccount
// ============================================================================

func TestHandleUpdateAccount_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/email/accounts/"+id, jsonBody(t, map[string]interface{}{
		"display_name": "New Name",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateAccount(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleUpdateAccount_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/email/accounts/"+id, invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateAccount(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

// TestHandleUpdateAccount_InvalidIDUUID documents the actual (missing)
// behaviour: there is no validateUUIDParam call in HandleUpdateAccount, so a
// non-UUID path segment is not rejected locally -- it is forwarded verbatim
// as UpdateEmailAccountRequest.Id and only fails once the request reaches
// the (unreachable) RPC layer, same as a valid UUID would.
func TestHandleUpdateAccount_InvalidIDUUID(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/email/accounts/not-a-uuid", jsonBody(t, map[string]interface{}{
		"display_name": "New Name",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateAccount(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateAccount_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/email/accounts/"+id, jsonBody(t, map[string]interface{}{
		"display_name": "New Name",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateAccount(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteAccount
// ============================================================================

func TestHandleDeleteAccount_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/email/accounts/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteAccount(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleDeleteAccount_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/email/accounts/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteAccount(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleSetDefaultAccount
// ============================================================================

func TestHandleSetDefaultAccount_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/accounts/"+id+"/default", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleSetDefaultAccount(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleSetDefaultAccount_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/accounts/"+id+"/default", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleSetDefaultAccount(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleTestConnection
// ============================================================================

func TestHandleTestConnection_ServiceUnavailable(t *testing.T) {
	routes := NewEmailRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/accounts/test", jsonBody(t, map[string]interface{}{
		"imap_host": "imap.example.com",
		"smtp_host": "smtp.example.com",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleTestConnection(rec, req)
	assertStatus(t, rec, http.StatusBadGateway)
}

func TestHandleTestConnection_InvalidJSON(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/accounts/test", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleTestConnection(rec, req)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleTestConnection_ReachesRPC(t *testing.T) {
	routes := NewEmailRoutes(registryWithService("email"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/email/accounts/test", jsonBody(t, map[string]interface{}{
		"imap_host": "imap.example.com",
		"smtp_host": "smtp.example.com",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleTestConnection(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
