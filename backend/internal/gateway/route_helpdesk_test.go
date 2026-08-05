package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
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
	req = withAuth(req, "user-123", testTenantID)
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
	req = withAuth(req, "user-123", testTenantID)
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
	req = withAuth(req, "user-123", testTenantID)
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
	req = withAuth(req, "user-123", testTenantID)
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
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTicket(rec, req)
	assertValidationError(t, rec, "assignee_id")
}

// TestHandleCreateTicket_IntakeFieldsPassValidation sends all five intake
// fields (channel, requester_email, requester_name, requester_is_external,
// custom_fields) alongside subject. None of them carry a `validate` tag --
// that check lives in TicketIntake.normalize on the service side (B2) -- so
// the handler must decode and forward them without 400ing on its own. The
// dummy registry address means the RPC itself fails with Unavailable, which
// is exactly what proves decode+validate got past the new fields: a 400
// here would mean the handler rejected them before ever reaching the client.
func TestHandleCreateTicket_IntakeFieldsPassValidation(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets", jsonBody(t, map[string]interface{}{
		"subject":               "Test ticket",
		"channel":               "external",
		"requester_email":       "kunde@example.com",
		"requester_name":        "Externe Kundin",
		"requester_is_external": true,
		"custom_fields": map[string]interface{}{
			"order_id": float64(4711),
			"vip":      true,
		},
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTicket(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleCreateTicket_CustomFieldsNotObject sends custom_fields as a JSON
// array instead of an object. json.Decode into map[string]any fails at the
// type level, so this is a 400 "invalid request body" before the handler
// ever builds a structpb.Struct -- distinct from the service-side
// ErrInvalidCustomFields path (nested/empty keys), which needs a live RPC.
func TestHandleCreateTicket_CustomFieldsNotObject(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets", strings.NewReader(
		`{"subject":"Test ticket","custom_fields":["not","an","object"]}`,
	))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTicket(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// --- HandleUpdateTicket ---

// TestHandleUpdateTicket_IntakeFieldsPassValidation sends status and
// custom_fields, the two fields updateTicketRequest silently dropped before
// this unit (BACKLOG B4). Neither carries a `validate` tag -- status is
// checked against ValidTicketStatuses and custom_fields against
// ErrInvalidCustomFields on the service side -- so the handler must decode
// and forward them without 400ing on its own. The dummy registry address
// means the RPC itself fails with Unavailable, which is exactly what proves
// decode+validate got past the new fields: a 400 here would mean the handler
// rejected them before ever reaching the client.
func TestHandleUpdateTicket_IntakeFieldsPassValidation(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/", jsonBody(t, map[string]interface{}{
		"status": "pending",
		"custom_fields": map[string]interface{}{
			"standort": "Bern",
		},
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTicket(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleUpdateTicket_CustomFieldsNotObject mirrors
// TestHandleCreateTicket_CustomFieldsNotObject: custom_fields as a JSON array
// fails json.Decode into map[string]any at the type level, a 400 before the
// handler ever builds a structpb.Struct.
func TestHandleUpdateTicket_CustomFieldsNotObject(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/", strings.NewReader(
		`{"custom_fields":["not","an","object"]}`,
	))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateTicket(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
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
