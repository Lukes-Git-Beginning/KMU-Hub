package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// This file covers the calendar CRUD (HandleGetCalendar, HandleUpdateCalendar,
// HandleDeleteCalendar) and membership handlers (HandleAddCalendarMember,
// HandleRemoveCalendarMember, HandleUpdateCalendarMemberPermission,
// HandleSubscribeToCalendar, HandleUnsubscribeFromCalendar) in
// route_calendar.go.
//
// HandleGetCalendar/HandleUpdateCalendar/HandleDeleteCalendar validate the
// "id" path param locally via validateUUIDParam. The membership/subscription
// handlers do not: HandleAddCalendarMember/HandleRemoveCalendarMember/
// HandleUpdateCalendarMemberPermission/HandleSubscribeToCalendar/
// HandleUnsubscribeFromCalendar all read "id" (and "userId" where present) via
// raw chi.URLParam with no validateUUIDParam call, so a syntactically invalid
// UUID passes straight through to the (unreachable, in these tests) RPC layer
// -- the same class of gap already catalogued for route_calendar.go in the
// Iteration 6 (fix-gateway-id-validation-consistency) journal entry as one of
// the 161 remaining raw chi.URLParam(r, "id") sites left for a Lauf-9
// follow-up unit. Documented here via the *_ReachesRPC tests rather than
// fixed, since a coverage unit must not change behaviour.
//
// Similarly, updateCalendarMemberPermissionRequest.Permission only carries
// `validate:"required"` -- there is no enum/oneof check against the
// CalendarPermission values, so a syntactically present but semantically
// invalid permission level is not rejected locally either; it reaches the RPC
// layer the same way an invalid UUID would.

// ============================================================================
// HandleGetCalendar / HandleUpdateCalendar / HandleDeleteCalendar
// ============================================================================

func TestHandleGetCalendar_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/calendars/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetCalendar(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetCalendar_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/calendars/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetCalendar(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetCalendar_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/calendars/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetCalendar(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateCalendar_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/calendars/"+id, jsonBody(t, map[string]interface{}{
		"name": "Renamed",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateCalendar(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateCalendar_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/calendars/not-a-uuid", jsonBody(t, map[string]interface{}{
		"name": "Renamed",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateCalendar(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateCalendar_InvalidJSON(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/calendars/"+id, invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateCalendar(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateCalendar_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/calendars/"+id, jsonBody(t, map[string]interface{}{
		"name": "Renamed",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateCalendar(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteCalendar_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/calendars/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteCalendar(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteCalendar_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/calendars/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteCalendar(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteCalendar_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/calendars/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteCalendar(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleAddCalendarMember / HandleRemoveCalendarMember
// ============================================================================

func TestHandleAddCalendarMember_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/calendars/"+calID+"/members", jsonBody(t, map[string]interface{}{
		"user_id":    uuid.New().String(),
		"permission": "write",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	routes.HandleAddCalendarMember(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleAddCalendarMember_InvalidJSON(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/calendars/"+calID+"/members", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	routes.HandleAddCalendarMember(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleAddCalendarMember_MissingUserID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/calendars/"+calID+"/members", jsonBody(t, map[string]interface{}{
		"permission": "write",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	routes.HandleAddCalendarMember(rec, req)
	assertValidationError(t, rec, "user_id")
}

func TestHandleAddCalendarMember_InvalidUserIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/calendars/"+calID+"/members", jsonBody(t, map[string]interface{}{
		"user_id":    "not-a-uuid",
		"permission": "write",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	routes.HandleAddCalendarMember(rec, req)
	assertValidationError(t, rec, "user_id")
}

func TestHandleAddCalendarMember_MissingPermission(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/calendars/"+calID+"/members", jsonBody(t, map[string]interface{}{
		"user_id": uuid.New().String(),
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	routes.HandleAddCalendarMember(rec, req)
	assertValidationError(t, rec, "permission")
}

// TestHandleAddCalendarMember_InvalidCalendarIDUUID_ReachesRPC documents that
// the "id" path param has no local validateUUIDParam call (unlike
// HandleGetCalendar/HandleUpdateCalendar/HandleDeleteCalendar in the same
// file) -- a structurally invalid calendar id is forwarded to the RPC layer
// unchanged instead of being rejected at the gateway boundary.
func TestHandleAddCalendarMember_InvalidCalendarIDUUID_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/calendars/not-a-uuid/members", jsonBody(t, map[string]interface{}{
		"user_id":    uuid.New().String(),
		"permission": "write",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleAddCalendarMember(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleAddCalendarMember_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/calendars/"+calID+"/members", jsonBody(t, map[string]interface{}{
		"user_id":    uuid.New().String(),
		"permission": "write",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	routes.HandleAddCalendarMember(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRemoveCalendarMember_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	userID := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/calendars/"+calID+"/members/"+userID, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	req = withChiURLParam(req, "userId", userID)
	routes.HandleRemoveCalendarMember(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleRemoveCalendarMember_InvalidCalendarIDUUID_ReachesRPC documents
// the same missing-local-validation gap as HandleAddCalendarMember above,
// for the calendar id half of the pair.
func TestHandleRemoveCalendarMember_InvalidCalendarIDUUID_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	userID := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/calendars/not-a-uuid/members/"+userID, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	req = withChiURLParam(req, "userId", userID)
	routes.HandleRemoveCalendarMember(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleRemoveCalendarMember_InvalidUserIDUUID_ReachesRPC documents the
// same gap for the userId half of the pair.
func TestHandleRemoveCalendarMember_InvalidUserIDUUID_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/calendars/"+calID+"/members/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	req = withChiURLParam(req, "userId", "not-a-uuid")
	routes.HandleRemoveCalendarMember(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleRemoveCalendarMember_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	userID := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/calendars/"+calID+"/members/"+userID, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	req = withChiURLParam(req, "userId", userID)
	routes.HandleRemoveCalendarMember(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateCalendarMemberPermission
// ============================================================================

func TestHandleUpdateCalendarMemberPermission_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	userID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/calendars/"+calID+"/members/"+userID+"/permission", jsonBody(t, map[string]interface{}{
		"permission": "admin",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	req = withChiURLParam(req, "userId", userID)
	routes.HandleUpdateCalendarMemberPermission(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateCalendarMemberPermission_InvalidJSON(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	userID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/calendars/"+calID+"/members/"+userID+"/permission", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	req = withChiURLParam(req, "userId", userID)
	routes.HandleUpdateCalendarMemberPermission(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateCalendarMemberPermission_MissingPermission(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	userID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/calendars/"+calID+"/members/"+userID+"/permission", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	req = withChiURLParam(req, "userId", userID)
	routes.HandleUpdateCalendarMemberPermission(rec, req)
	assertValidationError(t, rec, "permission")
}

// TestHandleUpdateCalendarMemberPermission_InvalidPermissionLevel_ReachesRPC
// documents that updateCalendarMemberPermissionRequest.Permission only
// carries `validate:"required"` -- a syntactically present but semantically
// unknown permission level (not one of the CalendarPermission values) is not
// rejected locally, unlike e.g. HandleRSVPToEvent's status field elsewhere in
// this package. It is forwarded to the RPC layer for the service to reject.
func TestHandleUpdateCalendarMemberPermission_InvalidPermissionLevel_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	userID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/calendars/"+calID+"/members/"+userID+"/permission", jsonBody(t, map[string]interface{}{
		"permission": "not-a-real-level",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	req = withChiURLParam(req, "userId", userID)
	routes.HandleUpdateCalendarMemberPermission(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateCalendarMemberPermission_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	userID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/calendars/"+calID+"/members/"+userID+"/permission", jsonBody(t, map[string]interface{}{
		"permission": "admin",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	req = withChiURLParam(req, "userId", userID)
	routes.HandleUpdateCalendarMemberPermission(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleSubscribeToCalendar / HandleUnsubscribeFromCalendar
// ============================================================================

func TestHandleSubscribeToCalendar_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/calendars/"+calID+"/subscribe", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	routes.HandleSubscribeToCalendar(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSubscribeToCalendar_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/calendars/"+calID+"/subscribe", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	routes.HandleSubscribeToCalendar(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUnsubscribeFromCalendar_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/calendars/"+calID+"/subscribe", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	routes.HandleUnsubscribeFromCalendar(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUnsubscribeFromCalendar_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	calID := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/calendars/"+calID+"/subscribe", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", calID)
	routes.HandleUnsubscribeFromCalendar(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
