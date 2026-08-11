package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// This file covers the event and resource-booking handlers in
// route_calendar.go: HandleGetEvent, HandleUpdateEvent, HandleDeleteEvent,
// HandleListEventsInRange, HandleCreateResource, HandleUpdateResource,
// HandleBookResource and HandleCancelBooking.
//
// HandleListEventsInRange rejects a missing start or end query parameter and
// an unparseable timestamp, but never compares the two once parsed -- there
// is no "start after end" check anywhere in the handler or in parseTimestamp
// (route_work.go:256). An inverted range is therefore forwarded to the RPC
// layer unchanged, the same class of gap already documented for other
// handlers in this package (missing local validation, not a security issue
// since the service still enforces server-side). Documented here via
// TestHandleListEventsInRange_InvertedRange_ReachesRPC rather than assumed
// away, since the backlog's "invertierte Von/Bis-Parameter als Fehlerfall"
// expectation does not hold against the actual code.
//
// HandleBookResource/HandleCancelBooking have no local booking-conflict
// check either -- conflict detection is entirely server-side (BookResource/
// CancelBooking RPCs). The *_ReachesRPC tests below document that the
// handlers pass validated input straight through; they cannot exercise the
// actual conflict response without a real CalendarService, consistent with
// every other ReachesRPC test in this package (dummy localhost:0 address,
// no bufconn stub for CalendarServiceClient in this repo).

// ============================================================================
// HandleListEventsInRange
// ============================================================================

func TestHandleListEventsInRange_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events?start=2026-01-01&end=2026-01-02", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListEventsInRange(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListEventsInRange_MissingStart(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events?end=2026-01-02", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListEventsInRange(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "start and end query parameters are required")
}

func TestHandleListEventsInRange_MissingEnd(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events?start=2026-01-01", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListEventsInRange(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "start and end query parameters are required")
}

func TestHandleListEventsInRange_InvalidStartFormat(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events?start=not-a-date&end=2026-01-02", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListEventsInRange(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid start time format")
}

func TestHandleListEventsInRange_InvalidEndFormat(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events?start=2026-01-01&end=not-a-date", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListEventsInRange(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid end time format")
}

// TestHandleListEventsInRange_InvertedRange_ReachesRPC documents that a
// syntactically valid but inverted range (start after end) is not rejected
// locally -- see the file-level comment above.
func TestHandleListEventsInRange_InvertedRange_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events?start=2026-06-01&end=2026-01-01", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListEventsInRange(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListEventsInRange_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events?start=2026-01-01&end=2026-01-02", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListEventsInRange(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetEvent / HandleUpdateEvent / HandleDeleteEvent
// ============================================================================

func TestHandleGetEvent_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetEvent(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetEvent_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetEvent(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetEvent_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetEvent(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateEvent_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/events/"+id, jsonBody(t, map[string]interface{}{
		"title": "Renamed",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateEvent(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateEvent_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/events/not-a-uuid", jsonBody(t, map[string]interface{}{
		"title": "Renamed",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateEvent(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateEvent_InvalidJSON(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/events/"+id, invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateEvent(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateEvent_InvalidStartTimeFormat(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/events/"+id, jsonBody(t, map[string]interface{}{
		"start_time": "not-a-date",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateEvent(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid start_time format")
}

func TestHandleUpdateEvent_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/events/"+id, jsonBody(t, map[string]interface{}{
		"title": "Renamed",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateEvent(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteEvent_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/events/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteEvent(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteEvent_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/events/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteEvent(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteEvent_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/events/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteEvent(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCreateResource / HandleUpdateResource
// ============================================================================

func TestHandleCreateResource_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/resources", jsonBody(t, map[string]interface{}{
		"name":          "Meeting Room A",
		"resource_type": "room",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateResource(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateResource_InvalidJSON(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/resources", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateResource(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateResource_MissingName(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/resources", jsonBody(t, map[string]interface{}{
		"resource_type": "room",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateResource(rec, req)
	assertValidationError(t, rec, "name")
}

func TestHandleCreateResource_MissingResourceType(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/resources", jsonBody(t, map[string]interface{}{
		"name": "Meeting Room A",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateResource(rec, req)
	assertValidationError(t, rec, "resource_type")
}

func TestHandleCreateResource_InvalidCapacity(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/resources", jsonBody(t, map[string]interface{}{
		"name":          "Meeting Room A",
		"resource_type": "room",
		"capacity":      0,
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateResource(rec, req)
	assertValidationError(t, rec, "capacity")
}

func TestHandleCreateResource_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/resources", jsonBody(t, map[string]interface{}{
		"name":          "Meeting Room A",
		"resource_type": "room",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateResource(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateResource_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/resources/"+id, jsonBody(t, map[string]interface{}{
		"name": "Renamed Room",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateResource(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateResource_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/resources/not-a-uuid", jsonBody(t, map[string]interface{}{
		"name": "Renamed Room",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateResource(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateResource_InvalidJSON(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/resources/"+id, invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateResource(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleUpdateResource_InvalidCapacity(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/resources/"+id, jsonBody(t, map[string]interface{}{
		"capacity": -1,
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateResource(rec, req)
	assertValidationError(t, rec, "capacity")
}

func TestHandleUpdateResource_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/resources/"+id, jsonBody(t, map[string]interface{}{
		"name": "Renamed Room",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateResource(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleBookResource / HandleCancelBooking
// ============================================================================

func TestHandleBookResource_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/bookings", jsonBody(t, map[string]interface{}{
		"resource_id": uuid.New().String(),
		"event_id":    uuid.New().String(),
		"start_time":  "2026-01-01T10:00:00Z",
		"end_time":    "2026-01-01T11:00:00Z",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleBookResource(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleBookResource_InvalidJSON(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/bookings", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleBookResource(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleBookResource_InvalidResourceIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/bookings", jsonBody(t, map[string]interface{}{
		"resource_id": "not-a-uuid",
		"event_id":    uuid.New().String(),
		"start_time":  "2026-01-01T10:00:00Z",
		"end_time":    "2026-01-01T11:00:00Z",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleBookResource(rec, req)
	assertValidationError(t, rec, "resource_id")
}

func TestHandleBookResource_InvalidEventIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/bookings", jsonBody(t, map[string]interface{}{
		"resource_id": uuid.New().String(),
		"event_id":    "not-a-uuid",
		"start_time":  "2026-01-01T10:00:00Z",
		"end_time":    "2026-01-01T11:00:00Z",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleBookResource(rec, req)
	assertValidationError(t, rec, "event_id")
}

func TestHandleBookResource_MissingStartTime(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/bookings", jsonBody(t, map[string]interface{}{
		"resource_id": uuid.New().String(),
		"event_id":    uuid.New().String(),
		"end_time":    "2026-01-01T11:00:00Z",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleBookResource(rec, req)
	assertValidationError(t, rec, "start_time")
}

func TestHandleBookResource_InvalidStartTimeFormat(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/bookings", jsonBody(t, map[string]interface{}{
		"resource_id": uuid.New().String(),
		"event_id":    uuid.New().String(),
		"start_time":  "not-a-date",
		"end_time":    "2026-01-01T11:00:00Z",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleBookResource(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid start_time format")
}

// TestHandleBookResource_ConflictingRange_ReachesRPC documents that booking
// conflict detection is entirely server-side: the handler performs no local
// overlap check against existing bookings, so a range that collides with an
// existing booking is forwarded to the RPC layer like any other
// well-formed request -- see the file-level comment above.
func TestHandleBookResource_ConflictingRange_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/bookings", jsonBody(t, map[string]interface{}{
		"resource_id": uuid.New().String(),
		"event_id":    uuid.New().String(),
		"start_time":  "2026-01-01T10:00:00Z",
		"end_time":    "2026-01-01T11:00:00Z",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleBookResource(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleBookResource_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/bookings", jsonBody(t, map[string]interface{}{
		"resource_id": uuid.New().String(),
		"event_id":    uuid.New().String(),
		"start_time":  "2026-01-01T10:00:00Z",
		"end_time":    "2026-01-01T11:00:00Z",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleBookResource(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCancelBooking_ServiceUnavailable(t *testing.T) {
	routes := NewCalendarRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/bookings/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleCancelBooking(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCancelBooking_InvalidIDUUID(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/bookings/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCancelBooking(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleCancelBooking_ReachesRPC(t *testing.T) {
	routes := NewCalendarRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/bookings/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleCancelBooking(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
