package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	inboxv1 "github.com/kmuhub/kmuhub/proto/inbox/v1"
)

func newInboxRoutes(registry *ServiceRegistry) *InboxRoutes {
	return NewInboxRoutes(registry)
}

// ============================================================================
// ServiceName
// ============================================================================

func TestInboxRoutes_ServiceName(t *testing.T) {
	routes := newInboxRoutes(emptyRegistry())
	if routes.ServiceName() != "notification" {
		t.Errorf("ServiceName() = %q, want %q (inbox is co-hosted in the notification binary)", routes.ServiceName(), "notification")
	}
}

// ============================================================================
// ServiceUnavailable — every handler checks the gRPC client before doing
// anything else, so a single generic request (no URL params, no valid body
// needed) proves the 503 path for all of them.
// ============================================================================

func TestInboxRoutes_ServiceUnavailable(t *testing.T) {
	routes := newInboxRoutes(emptyRegistry())

	handlers := map[string]http.HandlerFunc{
		"HandleListMessages":         routes.HandleListMessages,
		"HandleGetMessage":           routes.HandleGetMessage,
		"HandleMarkRead":             routes.HandleMarkRead,
		"HandleMarkUnread":           routes.HandleMarkUnread,
		"HandleToggleStar":           routes.HandleToggleStar,
		"HandleSetMessageStatus":     routes.HandleSetMessageStatus,
		"HandleAddMessageTag":        routes.HandleAddMessageTag,
		"HandleRemoveMessageTag":     routes.HandleRemoveMessageTag,
		"HandleForwardMessage":       routes.HandleForwardMessage,
		"HandleArchiveMessage":       routes.HandleArchiveMessage,
		"HandleUnarchiveMessage":     routes.HandleUnarchiveMessage,
		"HandleSnoozeMessage":        routes.HandleSnoozeMessage,
		"HandleUnsnoozeMessage":      routes.HandleUnsnoozeMessage,
		"HandleReplyToMessage":       routes.HandleReplyToMessage,
		"HandleAssignMessage":        routes.HandleAssignMessage,
		"HandleGetUnreadCount":       routes.HandleGetUnreadCount,
		"HandleBulkMarkRead":         routes.HandleBulkMarkRead,
		"HandleBulkArchive":          routes.HandleBulkArchive,
		"HandleListThreadMessages":   routes.HandleListThreadMessages,
		"HandleListCannedResponses":  routes.HandleListCannedResponses,
		"HandleCreateCannedResponse": routes.HandleCreateCannedResponse,
		"HandleUpdateCannedResponse": routes.HandleUpdateCannedResponse,
		"HandleDeleteCannedResponse": routes.HandleDeleteCannedResponse,
		"HandleCreateTeamInbox":      routes.HandleCreateTeamInbox,
		"HandleUpdateTeamInbox":      routes.HandleUpdateTeamInbox,
		"HandleDeleteTeamInbox":      routes.HandleDeleteTeamInbox,
		"HandleListTeamInboxes":      routes.HandleListTeamInboxes,
		"HandleAddTeamMember":        routes.HandleAddTeamMember,
		"HandleRemoveTeamMember":     routes.HandleRemoveTeamMember,
		"HandleListTeamMembers":      routes.HandleListTeamMembers,
		"HandleClaimMessage":         routes.HandleClaimMessage,
		"HandleCreateRoutingRule":    routes.HandleCreateRoutingRule,
		"HandleUpdateRoutingRule":    routes.HandleUpdateRoutingRule,
		"HandleDeleteRoutingRule":    routes.HandleDeleteRoutingRule,
		"HandleListRoutingRules":     routes.HandleListRoutingRules,
		"HandleTestRoutingRule":      routes.HandleTestRoutingRule,
	}

	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			testServiceUnavailable(t, h)
		})
	}
}

// ============================================================================
// Route registration order — /messages/unread-count must resolve to
// HandleGetUnreadCount, not fall into /messages/{id} and get rejected as an
// invalid UUID. With registryWithService the gRPC client construction
// succeeds, so a correctly routed request dies later with 503 (RPC
// Unavailable) while a misrouted one dies immediately with 400 "invalid id".
// ============================================================================

func TestInboxRoutes_UnreadCountRouteOrder(t *testing.T) {
	r := chi.NewRouter()
	routes := newInboxRoutes(registryWithService("notification"))
	routes.RegisterRoutes(r, guardTestAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inbox/messages/unread-count", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withPermissions(req, "inbox:read")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code == http.StatusBadRequest {
		t.Fatalf("unread-count was routed into /messages/{id} and rejected as an invalid UUID; body = %s", rec.Body.String())
	}
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// UUID param validation — handlers that only need a message/team/rule id
// ============================================================================

func TestInboxRoutes_InvalidIDParam(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))

	cases := map[string]http.HandlerFunc{
		"HandleGetMessage":           routes.HandleGetMessage,
		"HandleMarkRead":             routes.HandleMarkRead,
		"HandleMarkUnread":           routes.HandleMarkUnread,
		"HandleToggleStar":           routes.HandleToggleStar,
		"HandleArchiveMessage":       routes.HandleArchiveMessage,
		"HandleUnarchiveMessage":     routes.HandleUnarchiveMessage,
		"HandleUnsnoozeMessage":      routes.HandleUnsnoozeMessage,
		"HandleListThreadMessages":   routes.HandleListThreadMessages,
		"HandleClaimMessage":         routes.HandleClaimMessage,
		"HandleUpdateTeamInbox":      routes.HandleUpdateTeamInbox,
		"HandleDeleteTeamInbox":      routes.HandleDeleteTeamInbox,
		"HandleListTeamMembers":      routes.HandleListTeamMembers,
		"HandleUpdateCannedResponse": routes.HandleUpdateCannedResponse,
		"HandleDeleteCannedResponse": routes.HandleDeleteCannedResponse,
		"HandleDeleteRoutingRule":    routes.HandleDeleteRoutingRule,
	}

	for name, h := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req = withAuth(req, "user-123", testTenantID)
			req = withChiURLParam(req, "id", "not-a-uuid")
			h(rec, req)
			assertStatus(t, rec, http.StatusBadRequest)
			assertErrorContains(t, rec, "invalid id")
		})
	}
}

func TestHandleRemoveTeamMember_InvalidTeamID(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withChiURLParam(req, "user_id", "22222222-2222-2222-2222-222222222222")
	routes.HandleRemoveTeamMember(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleRemoveTeamMember_InvalidMemberUserID(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	req = withChiURLParam(req, "user_id", "not-a-uuid")
	routes.HandleRemoveTeamMember(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid user_id")
}

// ============================================================================
// HandleSetMessageStatus — body validation
// ============================================================================

func TestHandleSetMessageStatus_InvalidJSON(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleSetMessageStatus(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSetMessageStatus_MissingStatus(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleSetMessageStatus(rec, req)
	assertValidationError(t, rec, "status")
}

func TestHandleSetMessageStatus_InvalidStatus(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"status": "made-up-status",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleSetMessageStatus(rec, req)
	assertValidationError(t, rec, "status")
}

// ============================================================================
// HandleAddMessageTag / HandleRemoveMessageTag — body validation
// ============================================================================

func TestHandleAddMessageTag_MissingTag(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleAddMessageTag(rec, req)
	assertValidationError(t, rec, "tag")
}

func TestHandleRemoveMessageTag_MissingTag(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleRemoveMessageTag(rec, req)
	assertValidationError(t, rec, "tag")
}

// ============================================================================
// HandleForwardMessage — body validation
// ============================================================================

func TestHandleForwardMessage_MissingTo(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleForwardMessage(rec, req)
	assertValidationError(t, rec, "to")
}

// ============================================================================
// HandleSnoozeMessage — body validation, including the RFC3339 parse that
// happens after decodeAndValidate (no `validate` tag can express "parses as
// a timestamp").
// ============================================================================

func TestHandleSnoozeMessage_MissingSnoozeUntil(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleSnoozeMessage(rec, req)
	assertValidationError(t, rec, "snooze_until")
}

func TestHandleSnoozeMessage_InvalidTimestamp(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"snooze_until": "not a timestamp",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleSnoozeMessage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid snooze_until timestamp")
}

// ============================================================================
// HandleReplyToMessage — body validation
// ============================================================================

func TestHandleReplyToMessage_MissingBody(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleReplyToMessage(rec, req)
	assertValidationError(t, rec, "body")
}

// ============================================================================
// HandleAssignMessage — body validation
// ============================================================================

func TestHandleAssignMessage_MissingAssigneeID(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleAssignMessage(rec, req)
	assertValidationError(t, rec, "assignee_id")
}

func TestHandleAssignMessage_InvalidAssigneeID(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"assignee_id": "not-a-uuid",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleAssignMessage(rec, req)
	assertValidationError(t, rec, "assignee_id")
}

// ============================================================================
// HandleBulkMarkRead / HandleBulkArchive — body validation, incl. the
// `dive,uuid` element-level rule.
// ============================================================================

func TestHandleBulkMarkRead_EmptyIDs(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"ids": []string{},
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleBulkMarkRead(rec, req)
	assertValidationError(t, rec, "ids")
}

func TestHandleBulkMarkRead_InvalidElementUUID(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"ids": []string{"not-a-uuid"},
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleBulkMarkRead(rec, req)
	assertValidationError(t, rec, "ids[0]")
}

func TestHandleBulkArchive_EmptyIDs(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"ids": []string{},
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleBulkArchive(rec, req)
	assertValidationError(t, rec, "ids")
}

// ============================================================================
// Canned-response handlers — body validation
// ============================================================================

func TestHandleInboxCreateCannedResponse_MissingName(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"body": "some body",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateCannedResponse(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleInboxCreateCannedResponse_MissingBody(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"name": "Greeting",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateCannedResponse(rec, req)
	assertValidationError(t, rec, "body")
}

func TestHandleInboxCreateCannedResponse_NameTooLong(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	longName := make([]byte, 201)
	for i := range longName {
		longName[i] = 'a'
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"name": string(longName),
		"body": "some body",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateCannedResponse(rec, req)
	assertValidationError(t, rec, "name")
}

// ============================================================================
// Team inbox handlers — body validation
// ============================================================================

func TestHandleCreateTeamInbox_MissingName(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTeamInbox(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateTeamInbox_InvalidAssignmentMode(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"name":            "Support",
		"assignment_mode": "lottery",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTeamInbox(rec, req)
	assertValidationError(t, rec, "assignment_mode")
}

func TestHandleCreateTeamInbox_InvalidVisibility(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"name":       "Support",
		"visibility": "secret",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateTeamInbox(rec, req)
	assertValidationError(t, rec, "visibility")
}

// ============================================================================
// Team member handlers — body validation
// ============================================================================

func TestHandleAddTeamMember_MissingFields(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleAddTeamMember(rec, req)
	assertValidationError(t, rec, "member_user_id")
}

func TestHandleAddTeamMember_InvalidMemberUserID(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"member_user_id": "not-a-uuid",
		"role":           "member",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleAddTeamMember(rec, req)
	assertValidationError(t, rec, "member_user_id")
}

// ============================================================================
// Routing rule handlers — body + JSON-fragment validation
// ============================================================================

func TestHandleCreateRoutingRule_MissingName(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"conditions": map[string]interface{}{},
		"actions":    map[string]interface{}{},
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateRoutingRule(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateRoutingRule_InvalidConditionsJSON(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"name":       "VIP escalation",
		"conditions": "not an object",
		"actions":    map[string]interface{}{},
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateRoutingRule(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid conditions JSON")
}

func TestHandleCreateRoutingRule_InvalidActionsJSON(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"name":       "VIP escalation",
		"conditions": map[string]interface{}{},
		"actions":    "not an object",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleCreateRoutingRule(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid actions JSON")
}

func TestHandleUpdateRoutingRule_InvalidConditionsJSON(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/", jsonBody(t, map[string]interface{}{
		"conditions": "not an object",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleUpdateRoutingRule(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid conditions JSON")
}

func TestHandleUpdateRoutingRule_InvalidActionsJSON(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/", jsonBody(t, map[string]interface{}{
		"actions": "not an object",
	}))
	req = withAuth(req, "user-123", testTenantID)
	req = withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
	routes.HandleUpdateRoutingRule(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid actions JSON")
}

func TestHandleTestRoutingRule_InvalidJSONBody(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", invalidJSON())
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleTestRoutingRule(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleTestRoutingRule_InvalidConditionsJSON(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"conditions": "not an object",
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleTestRoutingRule(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid conditions JSON")
}

func TestHandleTestRoutingRule_ValidPassesToClient(t *testing.T) {
	routes := newInboxRoutes(registryWithService("notification"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", jsonBody(t, map[string]interface{}{
		"conditions": map[string]interface{}{"channel": "email"},
		"test_message": map[string]interface{}{
			"id": "11111111-1111-1111-1111-111111111111",
		},
	}))
	req = withAuth(req, "user-123", testTenantID)
	routes.HandleTestRoutingRule(rec, req)
	// A well-formed body must reach the RPC call, not be rejected locally —
	// the dummy registry address means the RPC itself fails Unavailable.
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// Enum parsers — every branch, including the default fallback
// ============================================================================

func TestParseChannelQuery(t *testing.T) {
	cases := map[string]inboxv1.Channel{
		"email":        inboxv1.Channel_CHANNEL_EMAIL,
		"chat":         inboxv1.Channel_CHANNEL_CHAT,
		"notification": inboxv1.Channel_CHANNEL_NOTIFICATION,
		"garbage":      inboxv1.Channel_CHANNEL_UNSPECIFIED,
		"":             inboxv1.Channel_CHANNEL_UNSPECIFIED,
	}
	for in, want := range cases {
		if got := parseChannelQuery(in); got != want {
			t.Errorf("parseChannelQuery(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseAssignmentMode(t *testing.T) {
	cases := map[string]inboxv1.AssignmentMode{
		"manual":      inboxv1.AssignmentMode_ASSIGNMENT_MODE_MANUAL,
		"round_robin": inboxv1.AssignmentMode_ASSIGNMENT_MODE_ROUND_ROBIN,
		"garbage":     inboxv1.AssignmentMode_ASSIGNMENT_MODE_MANUAL,
	}
	for in, want := range cases {
		if got := parseAssignmentMode(in); got != want {
			t.Errorf("parseAssignmentMode(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseVisibility(t *testing.T) {
	cases := map[string]inboxv1.TeamInboxVisibility{
		"open":    inboxv1.TeamInboxVisibility_VISIBILITY_OPEN,
		"private": inboxv1.TeamInboxVisibility_VISIBILITY_PRIVATE,
		"garbage": inboxv1.TeamInboxVisibility_VISIBILITY_OPEN,
	}
	for in, want := range cases {
		if got := parseVisibility(in); got != want {
			t.Errorf("parseVisibility(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseTeamMemberRole(t *testing.T) {
	cases := map[string]inboxv1.TeamMemberRole{
		"admin":   inboxv1.TeamMemberRole_TEAM_ROLE_ADMIN,
		"member":  inboxv1.TeamMemberRole_TEAM_ROLE_MEMBER,
		"garbage": inboxv1.TeamMemberRole_TEAM_ROLE_MEMBER,
	}
	for in, want := range cases {
		if got := parseTeamMemberRole(in); got != want {
			t.Errorf("parseTeamMemberRole(%q) = %v, want %v", in, got, want)
		}
	}
}

// ============================================================================
// rawJSONToStruct
// ============================================================================

func TestRawJSONToStruct_Empty(t *testing.T) {
	s, err := rawJSONToStruct(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != nil {
		t.Errorf("got %v, want nil struct for empty input", s)
	}
}

func TestRawJSONToStruct_Valid(t *testing.T) {
	s, err := rawJSONToStruct([]byte(`{"channel":"email"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil || s.Fields["channel"].GetStringValue() != "email" {
		t.Errorf("got %v, want struct with channel=email", s)
	}
}

func TestRawJSONToStruct_InvalidNotAnObject(t *testing.T) {
	if _, err := rawJSONToStruct([]byte(`"just a string"`)); err == nil {
		t.Error("expected error for non-object JSON, got nil")
	}
}

func TestRawJSONToStruct_MalformedJSON(t *testing.T) {
	if _, err := rawJSONToStruct([]byte(`{not valid`)); err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}
