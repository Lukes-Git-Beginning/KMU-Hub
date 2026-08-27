package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Covers the ten vertraege handler group route_vertraege_test.go leaves
// untested: HandleGetContract, HandleListContracts, HandleExportContract,
// HandleListContractEvents, HandleListParties, HandleRemoveParty,
// HandleListReminders, HandleUpdateReminder, HandleDeleteReminder,
// HandleSaveContractSignature.

const testVertraegeContractID = "550e8400-e29b-41d4-a716-446655440000"
const testVertraegePartyID = "660e8400-e29b-41d4-a716-446655440001"
const testVertraegeReminderID = "770e8400-e29b-41d4-a716-446655440002"

// --- HandleGetContract ---

func TestHandleGetContract_ServiceUnavailable(t *testing.T) {
	routes := newVertraegeRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleGetContract)
}

func TestHandleGetContract_NoTenantID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts/"+testVertraegeContractID, nil)
	req = withChiURLParam(req, "id", testVertraegeContractID)
	routes.HandleGetContract(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleGetContract_InvalidUUID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetContract(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetContract_ReachesRPC(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts/"+testVertraegeContractID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVertraegeContractID)
	routes.HandleGetContract(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListContracts ---

func TestHandleListContracts_ServiceUnavailable(t *testing.T) {
	routes := newVertraegeRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListContracts)
}

func TestHandleListContracts_NoTenantID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts", nil)
	routes.HandleListContracts(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListContracts_InvalidContactID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts?contact_id=not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListContracts(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid contact_id")
}

func TestHandleListContracts_ReachesRPC(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts?status=active&contract_type=service", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListContracts(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleExportContract ---

func TestHandleExportContract_ServiceUnavailable(t *testing.T) {
	routes := newVertraegeRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleExportContract)
}

func TestHandleExportContract_NoTenantID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/export", nil)
	req = withChiURLParam(req, "id", testVertraegeContractID)
	routes.HandleExportContract(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleExportContract_InvalidUUID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts/not-a-uuid/export", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleExportContract(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleExportContract_ReachesRPC(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/export", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVertraegeContractID)
	routes.HandleExportContract(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListContractEvents ---

func TestHandleListContractEvents_ServiceUnavailable(t *testing.T) {
	routes := newVertraegeRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListContractEvents)
}

func TestHandleListContractEvents_NoTenantID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/events", nil)
	req = withChiURLParam(req, "id", testVertraegeContractID)
	routes.HandleListContractEvents(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListContractEvents_InvalidUUID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts/not-a-uuid/events", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListContractEvents(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListContractEvents_ReachesRPC(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/events?page=2&page_size=10", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVertraegeContractID)
	routes.HandleListContractEvents(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListParties ---

func TestHandleListParties_ServiceUnavailable(t *testing.T) {
	routes := newVertraegeRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListParties)
}

func TestHandleListParties_NoTenantID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/parties", nil)
	req = withChiURLParam(req, "id", testVertraegeContractID)
	routes.HandleListParties(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListParties_InvalidUUID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts/not-a-uuid/parties", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListParties(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListParties_ReachesRPC(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/parties", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVertraegeContractID)
	routes.HandleListParties(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleRemoveParty ---

func TestHandleRemoveParty_ServiceUnavailable(t *testing.T) {
	routes := newVertraegeRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleRemoveParty)
}

func TestHandleRemoveParty_NoTenantID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/parties/"+testVertraegePartyID, nil)
	req = withChiURLParam(req, "partyId", testVertraegePartyID)
	routes.HandleRemoveParty(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleRemoveParty_InvalidUUID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/parties/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "partyId", "not-a-uuid")
	routes.HandleRemoveParty(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleRemoveParty_ReachesRPC(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/parties/"+testVertraegePartyID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "partyId", testVertraegePartyID)
	routes.HandleRemoveParty(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleListReminders ---

func TestHandleListReminders_ServiceUnavailable(t *testing.T) {
	routes := newVertraegeRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleListReminders)
}

func TestHandleListReminders_NoTenantID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/reminders", nil)
	req = withChiURLParam(req, "id", testVertraegeContractID)
	routes.HandleListReminders(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleListReminders_InvalidUUID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts/not-a-uuid/reminders", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListReminders(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleListReminders_ReachesRPCWithOnlyPending(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/reminders?only_pending=true", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVertraegeContractID)
	routes.HandleListReminders(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleUpdateReminder ---

func TestHandleUpdateReminder_ServiceUnavailable(t *testing.T) {
	routes := newVertraegeRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleUpdateReminder)
}

func TestHandleUpdateReminder_NoTenantID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/reminders/"+testVertraegeReminderID,
		jsonBody(t, map[string]interface{}{"subject": "Neu"}))
	req = withChiURLParam(req, "reminderId", testVertraegeReminderID)
	routes.HandleUpdateReminder(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleUpdateReminder_InvalidUUID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/reminders/not-a-uuid",
		jsonBody(t, map[string]interface{}{"subject": "Neu"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "reminderId", "not-a-uuid")
	routes.HandleUpdateReminder(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleUpdateReminder_InvalidJSON(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/reminders/"+testVertraegeReminderID, invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "reminderId", testVertraegeReminderID)
	routes.HandleUpdateReminder(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateReminder_ReachesRPC(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/reminders/"+testVertraegeReminderID,
		jsonBody(t, map[string]interface{}{"subject": "Neuer Betreff"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "reminderId", testVertraegeReminderID)
	routes.HandleUpdateReminder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleDeleteReminder ---

func TestHandleDeleteReminder_ServiceUnavailable(t *testing.T) {
	routes := newVertraegeRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleDeleteReminder)
}

func TestHandleDeleteReminder_NoTenantID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/reminders/"+testVertraegeReminderID, nil)
	req = withChiURLParam(req, "reminderId", testVertraegeReminderID)
	routes.HandleDeleteReminder(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleDeleteReminder_InvalidUUID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/reminders/not-a-uuid", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "reminderId", "not-a-uuid")
	routes.HandleDeleteReminder(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleDeleteReminder_ReachesRPC(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/reminders/"+testVertraegeReminderID, nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "reminderId", testVertraegeReminderID)
	routes.HandleDeleteReminder(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleSaveContractSignature ---

func TestHandleSaveContractSignature_ServiceUnavailable(t *testing.T) {
	routes := newVertraegeRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleSaveContractSignature)
}

func TestHandleSaveContractSignature_NoTenantID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/signature",
		jsonBody(t, map[string]interface{}{"signature_data": "data:image/png;base64,AAAA", "signed_by": "Anna"}))
	req = withChiURLParam(req, "id", testVertraegeContractID)
	routes.HandleSaveContractSignature(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleSaveContractSignature_InvalidUUID(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/vertraege/contracts/not-a-uuid/signature",
		jsonBody(t, map[string]interface{}{"signature_data": "data:image/png;base64,AAAA", "signed_by": "Anna"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleSaveContractSignature(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleSaveContractSignature_MissingSignatureData(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/signature",
		jsonBody(t, map[string]interface{}{"signed_by": "Anna"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVertraegeContractID)
	routes.HandleSaveContractSignature(rec, req)
	assertValidationError(t, rec, "signature_data")
}

func TestHandleSaveContractSignature_MissingSignedBy(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/signature",
		jsonBody(t, map[string]interface{}{"signature_data": "data:image/png;base64,AAAA"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVertraegeContractID)
	routes.HandleSaveContractSignature(rec, req)
	assertValidationError(t, rec, "signed_by")
}

func TestHandleSaveContractSignature_ReachesRPC(t *testing.T) {
	routes := newVertraegeRoutes(registryWithService("vertraege"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/vertraege/contracts/"+testVertraegeContractID+"/signature",
		jsonBody(t, map[string]interface{}{"signature_data": "data:image/png;base64,AAAA", "signed_by": "Anna Muster"}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", testVertraegeContractID)
	routes.HandleSaveContractSignature(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
