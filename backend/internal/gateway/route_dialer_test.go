package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDialerRoutes_ServiceName(t *testing.T) {
	routes := NewDialerRoutes(emptyRegistry())
	if routes.ServiceName() != "dialer" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "dialer")
	}
}

// --- Campaigns ---

func TestHandleListCampaigns_ServiceUnavailable(t *testing.T) {
	routes := NewDialerRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListCampaigns)
}

func TestHandleCreateCampaign_ServiceUnavailable(t *testing.T) {
	routes := NewDialerRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateCampaign)
}

func TestHandleCreateCampaign_NoUserID(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/dialer/campaigns", jsonBody(t, map[string]interface{}{}))
	routes.HandleCreateCampaign(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
	assertErrorContains(t, rec, "not authenticated")
}

func TestHandleCreateCampaign_InvalidJSON(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/dialer/campaigns", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleCreateCampaign(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleGetCampaign_ServiceUnavailable(t *testing.T) {
	routes := NewDialerRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/dialer/campaigns/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetCampaign(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetCampaign_InvalidUUID(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/dialer/campaigns/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetCampaign(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid campaign id")
}

func TestHandleUpdateCampaign_InvalidUUID(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/dialer/campaigns/bad", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateCampaign(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid campaign id")
}

func TestHandleUpdateCampaign_InvalidJSON(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/dialer/campaigns/550e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateCampaign(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleStartCampaign_InvalidUUID(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/dialer/campaigns/bad/start", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleStartCampaign(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid campaign id")
}

func TestHandleStartCampaign_ServiceUnavailable(t *testing.T) {
	routes := NewDialerRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/dialer/campaigns/550e8400-e29b-41d4-a716-446655440000/start", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleStartCampaign(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandlePauseCampaign_InvalidUUID(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/dialer/campaigns/bad/pause", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandlePauseCampaign(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid campaign id")
}

func TestHandlePauseCampaign_ServiceUnavailable(t *testing.T) {
	routes := NewDialerRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/dialer/campaigns/550e8400-e29b-41d4-a716-446655440000/pause", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandlePauseCampaign(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleArchiveCampaign_InvalidUUID(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/dialer/campaigns/bad", nil)
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleArchiveCampaign(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid campaign id")
}

// --- Campaign Contacts ---

func TestHandleAddContactsToCampaign_ServiceUnavailable(t *testing.T) {
	routes := NewDialerRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/dialer/campaigns/550e8400-e29b-41d4-a716-446655440000/contacts", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddContactsToCampaign(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleAddContactsToCampaign_InvalidUUID(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/dialer/campaigns/bad/contacts", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleAddContactsToCampaign(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid campaign id")
}

func TestHandleAddContactsToCampaign_InvalidJSON(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/dialer/campaigns/550e8400-e29b-41d4-a716-446655440000/contacts", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAddContactsToCampaign(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleGetNextContact_InvalidUUID(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/dialer/campaigns/bad/next", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetNextContact(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid campaign id")
}

// --- Calls ---

func TestHandleInitiateDialerCall_ServiceUnavailable(t *testing.T) {
	routes := NewDialerRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleInitiateDialerCall)
}

func TestHandleInitiateDialerCall_InvalidJSON(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/dialer/calls", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleInitiateDialerCall(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleLogCallOutcome_InvalidUUID(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/dialer/calls/bad/outcome", jsonBody(t, map[string]interface{}{"outcome_id": "x"}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleLogCallOutcome(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid call session id")
}

func TestHandleLogCallOutcome_InvalidJSON(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/dialer/calls/550e8400-e29b-41d4-a716-446655440000/outcome", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleLogCallOutcome(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCompleteWrapUp_ServiceUnavailable(t *testing.T) {
	routes := NewDialerRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/dialer/calls/550e8400-e29b-41d4-a716-446655440000/complete", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCompleteWrapUp(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCompleteWrapUp_InvalidUUID(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/dialer/calls/bad/complete", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCompleteWrapUp(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid call session id")
}

// --- Agent Status ---

func TestHandleSetAgentStatus_InvalidJSON(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/dialer/agent-status", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleSetAgentStatus(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSetAgentStatus_ServiceUnavailable(t *testing.T) {
	routes := NewDialerRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleSetAgentStatus)
}

// --- Call Outcomes ---

func TestHandleCreateCallOutcome_ServiceUnavailable(t *testing.T) {
	routes := NewDialerRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateCallOutcome)
}

func TestHandleCreateCallOutcome_InvalidJSON(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/dialer/outcomes", invalidJSON())
	routes.HandleCreateCallOutcome(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateCallOutcome_InvalidUUID(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/dialer/outcomes/bad", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateCallOutcome(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid outcome id")
}

func TestHandleUpdateCallOutcome_InvalidJSON(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/dialer/outcomes/550e8400-e29b-41d4-a716-446655440000", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateCallOutcome(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleDeleteCallOutcome_InvalidUUID(t *testing.T) {
	routes := NewDialerRoutes(registryWithService("dialer"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/dialer/outcomes/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteCallOutcome(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid outcome id")
}
