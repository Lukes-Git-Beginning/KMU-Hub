package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kmuhub/kmuhub/internal/featureflag"
)

func newHelpdeskRoutes(registry *ServiceRegistry) *HelpdeskRoutes {
	flags := featureflag.NewRegistry().Load(func(key string) string {
		if key == "COSMI_MODULE_HELPDESK_ENABLED" {
			return "true"
		}
		return ""
	})
	return NewHelpdeskRoutes(registry, flags)
}

// --- ServiceName ---

func TestHelpdeskRoutes_ServiceName(t *testing.T) {
	routes := newHelpdeskRoutes(emptyRegistry())
	if routes.ServiceName() != "helpdesk" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "helpdesk")
	}
}

// --- HandleCreateTicket ---

func TestHandleCreateTicket_ServiceUnavailable(t *testing.T) {
	routes := newHelpdeskRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateTicket)
}

func TestHandleCreateTicket_InvalidJSON(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleCreateTicket(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateTicket_MissingSubject(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets", jsonBody(t, map[string]interface{}{
		"priority": "normal",
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateTicket(rec, req)
	assertValidationError(t, rec, "subject")
}

func TestHandleCreateTicket_SubjectTooLong(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	longSubject := make([]byte, 201)
	for i := range longSubject {
		longSubject[i] = 'a'
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets", jsonBody(t, map[string]interface{}{
		"subject": string(longSubject),
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateTicket(rec, req)
	assertValidationError(t, rec, "subject")
}

func TestHandleCreateTicket_InvalidPriority(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets", jsonBody(t, map[string]interface{}{
		"subject":  "Test ticket",
		"priority": "critical", // not a valid value
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateTicket(rec, req)
	assertValidationError(t, rec, "priority")
}

func TestHandleCreateTicket_InvalidAssigneeID(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	assignee := "not-a-uuid"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets", jsonBody(t, map[string]interface{}{
		"subject":     "Test ticket",
		"assignee_id": assignee,
	}))
	req = withUserID(req, "user-123")
	routes.HandleCreateTicket(rec, req)
	assertValidationError(t, rec, "assignee_id")
}

// --- HandleAssignTicket ---

func TestHandleAssignTicket_ServiceUnavailable(t *testing.T) {
	routes := newHelpdeskRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAssignTicket(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleAssignTicket_InvalidJSON(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAssignTicket(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleAssignTicket_MissingAssigneeID(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAssignTicket(rec, req)
	assertValidationError(t, rec, "assignee_id")
}

func TestHandleAssignTicket_InvalidAssigneeIDFormat(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{
		"assignee_id": "not-a-uuid",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleAssignTicket(rec, req)
	assertValidationError(t, rec, "assignee_id")
}

// --- HandleMergeTickets ---

func TestHandleMergeTickets_MissingTargetID(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleMergeTickets(rec, req)
	assertValidationError(t, rec, "target_ticket_id")
}

func TestHandleMergeTickets_InvalidTargetIDFormat(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{
		"target_ticket_id": "not-a-uuid",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleMergeTickets(rec, req)
	assertValidationError(t, rec, "target_ticket_id")
}

// --- HandleAddMessage ---

func TestHandleAddMessage_MissingBody(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	req = withUserID(req, "user-123")
	routes.HandleAddMessage(rec, req)
	assertValidationError(t, rec, "body")
}

// --- HandleCreateQueue ---

func TestHandleCreateQueue_ServiceUnavailable(t *testing.T) {
	routes := newHelpdeskRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateQueue)
}

func TestHandleCreateQueue_MissingName(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{}))
	routes.HandleCreateQueue(rec, req)
	assertValidationError(t, rec, "name")
}

// --- HandleCreateCannedResponse ---

func TestHandleCreateCannedResponse_ServiceUnavailable(t *testing.T) {
	routes := newHelpdeskRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleCreateCannedResponse)
}

func TestHandleCreateCannedResponse_MissingName(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{
		"body": "some body text",
	}))
	routes.HandleCreateCannedResponse(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateCannedResponse_MissingBody(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{
		"name": "Quick reply",
	}))
	routes.HandleCreateCannedResponse(rec, req)
	assertValidationError(t, rec, "body")
}

// --- HandleCreateSLAPolicy ---

func TestHandleCreateSLAPolicy_MissingName(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{
		"first_response_mins": 30,
	}))
	routes.HandleCreateSLAPolicy(rec, req)
	assertValidationError(t, rec, "name")
}

// --- HandleApplySLAPolicy ---

func TestHandleApplySLAPolicy_MissingSLAPolicyID(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleApplySLAPolicy(rec, req)
	assertValidationError(t, rec, "sla_policy_id")
}

func TestHandleApplySLAPolicy_InvalidSLAPolicyIDFormat(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", jsonBody(t, map[string]interface{}{
		"sla_policy_id": "not-a-uuid",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleApplySLAPolicy(rec, req)
	assertValidationError(t, rec, "sla_policy_id")
}
