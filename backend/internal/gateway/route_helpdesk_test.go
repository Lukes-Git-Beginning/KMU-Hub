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

// --- HandleCreateKBArticle / HandleUpdateKBArticle ---

func TestHandleCreateKBArticle_ContentTooLong(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	tooLong := strings.Repeat("a", 500_001)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/kb-articles", jsonBody(t, map[string]interface{}{
		"title":   "Test article",
		"content": tooLong,
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateKBArticle(rec, req)
	assertValidationError(t, rec, "content")
}

func TestHandleUpdateKBArticle_ContentTooLong(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	tooLong := strings.Repeat("a", 500_001)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/", jsonBody(t, map[string]interface{}{
		"content": tooLong,
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateKBArticle(rec, req)
	assertValidationError(t, rec, "content")
}

// --- HandleListTickets ---

func TestHandleListTickets_ServiceUnavailable(t *testing.T) {
	routes := newHelpdeskRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/helpdesk/tickets", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListTickets(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListTickets_MissingTenant(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/helpdesk/tickets", nil)
	routes.HandleListTickets(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// TestHandleListTickets_ReachesRPC documents that ListTickets is a
// response.Proto passthrough of the RPC's own response -- no gateway-owned
// marshaling to assert a wire shape against, the same documented boundary as
// every other list handler in this run without a bufconn stub for
// HelpdeskServiceClient (see route_rapporte_test.go
// TestHandleListReports_ReachesRPC for the same limit).
func TestHandleListTickets_ReachesRPC(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/helpdesk/tickets?status=open&contact_id=550e8400-e29b-41d4-a716-446655440000", nil)
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleListTickets(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleListTickets_OwnScopeWithoutUserIsRejected covers the handler's
// own integration with ownerFilterForScope (not just the helper in
// own_scope_filter_test.go): a caller narrowed to "own" on
// helpdesk:ticket:read with no user id in the token must be refused, not
// fall through to an unfiltered list.
func TestHandleListTickets_OwnScopeWithoutUserIsRejected(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/helpdesk/tickets", nil)
	req = withTenantID(req, testTenantID)
	req = withScopes(req, map[string]string{"helpdesk:ticket:read": "own"})
	routes.HandleListTickets(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// --- HandleGetTicket ---

func TestHandleGetTicket_InvalidIDUUID(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/helpdesk/tickets/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetTicket(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetTicket_ServiceUnavailable(t *testing.T) {
	routes := newHelpdeskRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/helpdesk/tickets/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetTicket(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetTicket_ReachesRPC(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/helpdesk/tickets/550e8400-e29b-41d4-a716-446655440000", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetTicket(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleCloseTicket ---

func TestHandleCloseTicket_InvalidIDUUID(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets/not-a-uuid/close", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCloseTicket(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCloseTicket_ServiceUnavailable(t *testing.T) {
	routes := newHelpdeskRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets/550e8400-e29b-41d4-a716-446655440000/close", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCloseTicket(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCloseTicket_ReachesRPC(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets/550e8400-e29b-41d4-a716-446655440000/close", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCloseTicket(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleReopenTicket ---

func TestHandleReopenTicket_InvalidIDUUID(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets/not-a-uuid/reopen", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleReopenTicket(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleReopenTicket_ServiceUnavailable(t *testing.T) {
	routes := newHelpdeskRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets/550e8400-e29b-41d4-a716-446655440000/reopen", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleReopenTicket(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleReopenTicket_ReachesRPC(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets/550e8400-e29b-41d4-a716-446655440000/reopen", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleReopenTicket(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleSubmitCsat ---

func TestHandleSubmitCsat_ServiceUnavailable(t *testing.T) {
	routes := newHelpdeskRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets/550e8400-e29b-41d4-a716-446655440000/csat", jsonBody(t, map[string]interface{}{
		"rating": 5,
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSubmitCsat(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSubmitCsat_InvalidIDUUID(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets/not-a-uuid/csat", jsonBody(t, map[string]interface{}{
		"rating": 5,
	}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleSubmitCsat(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleSubmitCsat_InvalidJSON(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets/550e8400-e29b-41d4-a716-446655440000/csat", invalidJSON())
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSubmitCsat(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSubmitCsat_RatingOutOfRange(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets/550e8400-e29b-41d4-a716-446655440000/csat", jsonBody(t, map[string]interface{}{
		"rating": 6,
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSubmitCsat(rec, req)
	assertValidationError(t, rec, "rating")
}

func TestHandleSubmitCsat_ReachesRPC(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/helpdesk/tickets/550e8400-e29b-41d4-a716-446655440000/csat", jsonBody(t, map[string]interface{}{
		"rating":  4,
		"comment": "gut geloest",
	}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleSubmitCsat(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleSubmitCsatByToken ---
//
// This is the one unauthenticated route in this file: RegisterPublicRoutes
// mounts it outside authMiddleware, the token is the whole credential. Tests
// therefore never call withAuth/withTenantID here -- doing so would hide a
// regression where the handler started depending on auth context it must
// never see.

func TestHandleSubmitCsatByToken_ServiceUnavailable(t *testing.T) {
	routes := newHelpdeskRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/helpdesk/csat/tok-123", jsonBody(t, map[string]interface{}{
		"rating": 5,
	}))
	req = withChiURLParam(req, "token", "tok-123")
	routes.HandleSubmitCsatByToken(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSubmitCsatByToken_EmptyToken(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/helpdesk/csat/", jsonBody(t, map[string]interface{}{
		"rating": 5,
	}))
	req = withChiURLParam(req, "token", "")
	routes.HandleSubmitCsatByToken(rec, req)
	assertStatus(t, rec, http.StatusNotFound)
	assertErrorContains(t, rec, "survey link not found")
}

// TestHandleSubmitCsatByToken_TokenTooLong covers the >128 char guard. Every
// rejection about the token itself -- missing, over-long, unknown, expired,
// revoked, already redeemed -- comes back as the same 404 per the handler's
// doc comment, so a caller cannot use the response to learn which tokens
// exist. This is the only one of those cases resolvable locally without a
// bufconn stub for HelpdeskServiceClient; unknown/expired/revoked/redeemed
// all require a scripted RPC response and are the RPC layer's concern.
func TestHandleSubmitCsatByToken_TokenTooLong(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	tooLong := strings.Repeat("a", 129)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/helpdesk/csat/"+tooLong, jsonBody(t, map[string]interface{}{
		"rating": 5,
	}))
	req = withChiURLParam(req, "token", tooLong)
	routes.HandleSubmitCsatByToken(rec, req)
	assertStatus(t, rec, http.StatusNotFound)
	assertErrorContains(t, rec, "survey link not found")
}

func TestHandleSubmitCsatByToken_InvalidJSON(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/helpdesk/csat/tok-123", invalidJSON())
	req = withChiURLParam(req, "token", "tok-123")
	routes.HandleSubmitCsatByToken(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSubmitCsatByToken_RatingOutOfRange(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/helpdesk/csat/tok-123", jsonBody(t, map[string]interface{}{
		"rating": 0,
	}))
	req = withChiURLParam(req, "token", "tok-123")
	routes.HandleSubmitCsatByToken(rec, req)
	assertValidationError(t, rec, "rating")
}

func TestHandleSubmitCsatByToken_ReachesRPC(t *testing.T) {
	routes := newHelpdeskRoutes(registryWithService("helpdesk"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/public/helpdesk/csat/tok-123", jsonBody(t, map[string]interface{}{
		"rating":  5,
		"comment": "danke",
	}))
	req = withChiURLParam(req, "token", "tok-123")
	routes.HandleSubmitCsatByToken(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
