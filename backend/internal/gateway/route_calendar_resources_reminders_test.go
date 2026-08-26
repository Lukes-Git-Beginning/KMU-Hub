package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// This file covers the remaining resource, reminder and LiveKit handlers in
// route_calendar.go: HandleListResources, HandleGetResource,
// HandleDeleteResource, HandleListResourceAvailability,
// HandleSetEventReminders, HandleListEventReminders and
// HandleGenerateJoinToken.
//
// Race-condition check (backlog scope point 1): HandleListResourceAvailability
// itself has no write path -- it forwards to the ListResourceAvailability RPC,
// which the server backs with resource.Service.ListAvailability (a plain read).
// The actual booking write path, resource.Service.Book (internal/work/resource/
// service.go:199), does NOT re-check availability with its own read before
// writing: it inserts the booking directly via CreateBooking and relies on the
// `resource_bookings` table's EXCLUDE USING GIST constraint (postgres_repository.go:212-219,
// pg error 23P01 -> ErrBookingConflict) to reject overlapping ranges atomically
// at the database level. That is exactly the pattern
// harden-quote-conversion-unique-index established as safe -- there is no
// read-then-write gap here, so no fix-unit is raised for this point.
//
// HandleGenerateJoinToken finding (backlog scope point 2, real bug, NOT fixed
// in this coverage unit): the handler and the gRPC server behind it
// (CalendarGRPCServer.GenerateJoinToken, internal/server/calendar_grpc.go:1293)
// never verify that event_id refers to an event that exists, belongs to the
// caller's tenant, or that the caller has any relationship to it (attendee,
// creator, calendar member). Any authenticated user holding the tenant-wide
// "calendars:write" permission can POST an arbitrary UUID as {id} and receive a
// valid, signed 24h LiveKit join token for the room derived from it
// (livekit.Service.GenerateRoomName truncates to the first 8 hex chars of the
// UUID, internal/work/livekit/service.go:74-77) -- including UUIDs belonging to
// events in OTHER TENANTS, since nothing scopes the lookup by tenant_id (there
// is no lookup at all). Filed as fix-generatejointoken-missing-event-tenant-check
// at the end of BACKLOG.yml; not fixed here because a coverage unit must not
// change behaviour.

// ============================================================================
// HandleListResources
// ============================================================================

func TestHandleListResources_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListResources(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListResources_InvalidMinCapacity(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources?min_capacity=not-a-number", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListResources(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid min_capacity value")
}

func TestHandleListResources_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources?include_inactive=true&type=room&min_capacity=4&tags=a,b&floor=2", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListResources(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetResource
// ============================================================================

func TestHandleGetResource_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetResource(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetResource_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetResource(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetResource_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetResource(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteResource
// ============================================================================

func TestHandleDeleteResource_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/resources/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteResource(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteResource_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/resources/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteResource(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteResource_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/resources/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteResource(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListResourceAvailability
// ============================================================================

func TestHandleListResourceAvailability_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources/"+id+"/availability?start=2026-01-01&end=2026-01-02", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleListResourceAvailability(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListResourceAvailability_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources/not-a-uuid/availability?start=2026-01-01&end=2026-01-02", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListResourceAvailability(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListResourceAvailability_MissingStart(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources/"+id+"/availability?end=2026-01-02", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleListResourceAvailability(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "start and end query parameters are required")
}

func TestHandleListResourceAvailability_MissingEnd(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources/"+id+"/availability?start=2026-01-01", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleListResourceAvailability(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "start and end query parameters are required")
}

func TestHandleListResourceAvailability_InvalidStartFormat(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources/"+id+"/availability?start=not-a-date&end=2026-01-02", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleListResourceAvailability(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid start time format")
}

func TestHandleListResourceAvailability_InvalidEndFormat(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources/"+id+"/availability?start=2026-01-01&end=not-a-date", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleListResourceAvailability(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid end time format")
}

func TestHandleListResourceAvailability_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/resources/"+id+"/availability?start=2026-01-01&end=2026-01-02", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleListResourceAvailability(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleSetEventReminders
// ============================================================================

func TestHandleSetEventReminders_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/events/"+id+"/reminders", jsonBody(t, map[string]interface{}{
		"minutes_before": []int32{10, 30},
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleSetEventReminders(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleSetEventReminders_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/events/not-a-uuid/reminders", jsonBody(t, map[string]interface{}{
		"minutes_before": []int32{10},
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleSetEventReminders(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleSetEventReminders_InvalidJSON(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/events/"+id+"/reminders", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleSetEventReminders(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleSetEventReminders_NegativeMinutesBefore(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/events/"+id+"/reminders", jsonBody(t, map[string]interface{}{
		"minutes_before": []int32{-5},
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleSetEventReminders(rec, req)
	assertValidationError(t, rec, "minutes_before[0]")
}

func TestHandleSetEventReminders_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/events/"+id+"/reminders", jsonBody(t, map[string]interface{}{
		"minutes_before": []int32{10, 30},
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleSetEventReminders(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleListEventReminders
// ============================================================================

func TestHandleListEventReminders_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events/"+id+"/reminders", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleListEventReminders(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListEventReminders_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events/not-a-uuid/reminders", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleListEventReminders(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleListEventReminders_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events/"+id+"/reminders", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleListEventReminders(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGenerateJoinToken
// ============================================================================

func TestHandleGenerateJoinToken_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/events/"+id+"/join-token", jsonBody(t, map[string]interface{}{
		"display_name": "Jane Doe",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGenerateJoinToken(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGenerateJoinToken_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/events/not-a-uuid/join-token", jsonBody(t, map[string]interface{}{
		"display_name": "Jane Doe",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGenerateJoinToken(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGenerateJoinToken_InvalidJSON(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/events/"+id+"/join-token", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGenerateJoinToken(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleGenerateJoinToken_MissingDisplayName(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/events/"+id+"/join-token", jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGenerateJoinToken(rec, req)
	assertValidationError(t, rec, "display_name")
}

func TestHandleGenerateJoinToken_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/events/"+id+"/join-token", jsonBody(t, map[string]interface{}{
		"display_name": "Jane Doe",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGenerateJoinToken(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
