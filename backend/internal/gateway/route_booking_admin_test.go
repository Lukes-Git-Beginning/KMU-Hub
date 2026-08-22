package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/security"
)

// This file covers the ADMIN half of route_booking.go: HandleListBookingPages,
// HandleCreateBookingPage, HandleGetBookingPage, HandleUpdateBookingPage and
// HandleDeleteBookingPage (booking-page CRUD, authenticated, "booking-pages"
// permission). The PUBLIC half — HandleGetPublicBookingPage,
// HandleGetAvailability and HandleCreatePublicBooking, mounted via
// RegisterPublicRoutes outside the registrar loop — is part of the parked
// public-web-surface (BACKLOG-PARKED.yml) and stays untouched here.
//
// The backlog note for this unit names "Doppelbuchung desselben Zeitfensters"
// as the admin part's core business case. That case does not exist at this
// layer: slot-collision detection (ErrBookingSlotUnavailable, bookedSet,
// overlapsCalendarEvents) lives entirely in BookingService.CreatePublicBooking
// and GetAvailability (internal/work/calendar/booking_service.go), both public
// and out of scope. The admin CRUD handlers persist page configuration
// (slug, company name, services, the availability-rules JSON blob) with no
// slot arithmetic of their own — CreateBookingPage/UpdateBookingPage in
// booking_service.go only validate the JSON parses and required fields are
// set. There is also no page-level uniqueness error surfaced to the gateway:
// errToStatus in calendar_grpc.go maps only ErrBookingPageNotFound and the
// three validation errors for booking pages, no AlreadyExists case. The
// closest honest equivalent — a config change reaching the RPC layer
// unvalidated for cross-field conflicts — is documented below via
// TestHandleUpdateBookingPage_ChangedAvailabilityRules_ReachesRPC, following
// the *_ReachesRPC convention already established in
// route_calendar_events_resources_test.go for the same "no bufconn stub for
// CalendarServiceClient in this repo" limitation.
//
// Second finding while writing this file, fixed by
// fix-gateway-booking-page-services-no-dive: createBookingPageRequest.Services
// and updateBookingPageRequest.Services now carry "dive" on their validate
// tag, so go-playground/validator descends into the slice elements and
// bookingServiceRequestItem's own "required"/"gt=0" tags on id/name/
// duration_min/price run for every item (compare
// route_customization.go's `validate:"required,min=1,dive"` on a similar
// nested slice). A service item with an empty name, empty price or
// duration_min <= 0 is now rejected with 400 before it reaches the RPC.

func testBookingRoutes(reg *ServiceRegistry) *BookingRoutes {
	return NewBookingRoutes(reg, security.NewCaptchaVerifier("", ""))
}

func TestBookingRoutes_ServiceName(t *testing.T) {
	routes := testBookingRoutes(emptyRegistry())
	if routes.ServiceName() != "work" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "work")
	}
}

// ============================================================================
// HandleListBookingPages
// ============================================================================

func TestHandleListBookingPages_ServiceUnavailable(t *testing.T) {
	routes := testBookingRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/booking-pages", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListBookingPages(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListBookingPages_ReachesRPC(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/booking-pages", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListBookingPages(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListBookingPages_IncludeInactive_ReachesRPC(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/booking-pages?include_inactive=true", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleListBookingPages(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleCreateBookingPage
// ============================================================================

func validCreateBookingPageBody() map[string]interface{} {
	return map[string]interface{}{
		"calendar_id":        uuid.New().String(),
		"slug":               "acme-consulting",
		"company_name":       "Acme Consulting",
		"availability_rules": `{"weekdays":[1,2,3,4,5]}`,
		"services": []map[string]interface{}{
			{"id": "svc-1", "name": "Erstberatung", "duration_min": 30, "price": "0.00"},
		},
	}
}

func TestHandleCreateBookingPage_ServiceUnavailable(t *testing.T) {
	routes := testBookingRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/booking-pages", jsonBody(t, validCreateBookingPageBody()))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateBookingPage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateBookingPage_InvalidJSON(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/booking-pages", invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateBookingPage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateBookingPage_MissingCalendarID(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	body := validCreateBookingPageBody()
	delete(body, "calendar_id")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/booking-pages", jsonBody(t, body))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateBookingPage(rec, req)
	assertValidationError(t, rec, "calendar_id")
}

func TestHandleCreateBookingPage_InvalidCalendarIDUUID(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	body := validCreateBookingPageBody()
	body["calendar_id"] = "not-a-uuid"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/booking-pages", jsonBody(t, body))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateBookingPage(rec, req)
	assertValidationError(t, rec, "calendar_id")
}

func TestHandleCreateBookingPage_MissingSlug(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	body := validCreateBookingPageBody()
	delete(body, "slug")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/booking-pages", jsonBody(t, body))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateBookingPage(rec, req)
	assertValidationError(t, rec, "slug")
}

func TestHandleCreateBookingPage_MissingCompanyName(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	body := validCreateBookingPageBody()
	delete(body, "company_name")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/booking-pages", jsonBody(t, body))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateBookingPage(rec, req)
	assertValidationError(t, rec, "company_name")
}

func TestHandleCreateBookingPage_MissingAvailabilityRules(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	body := validCreateBookingPageBody()
	delete(body, "availability_rules")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/booking-pages", jsonBody(t, body))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateBookingPage(rec, req)
	assertValidationError(t, rec, "availability_rules")
}

// TestHandleCreateBookingPage_ServiceItemMissingName_Rejected proves the fix
// for fix-gateway-booking-page-services-no-dive: bookingServiceRequestItem's
// "required" tag on Name now runs for every slice element, so an item with
// no name is rejected with 400 before it reaches the RPC layer.
func TestHandleCreateBookingPage_ServiceItemMissingName_Rejected(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	body := validCreateBookingPageBody()
	body["services"] = []map[string]interface{}{
		{"id": "svc-1", "duration_min": 30, "price": "0.00"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/booking-pages", jsonBody(t, body))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateBookingPage(rec, req)
	assertValidationError(t, rec, "name")
}

// TestHandleCreateBookingPage_ServiceItemZeroDuration_Rejected — same fix,
// for the "gt=0" tag on DurationMin.
func TestHandleCreateBookingPage_ServiceItemZeroDuration_Rejected(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	body := validCreateBookingPageBody()
	body["services"] = []map[string]interface{}{
		{"id": "svc-1", "name": "Erstberatung", "duration_min": 0, "price": "0.00"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/booking-pages", jsonBody(t, body))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateBookingPage(rec, req)
	assertValidationError(t, rec, "duration_min")
}

// TestHandleCreateBookingPage_ServiceItemMissingPrice_Rejected — same fix,
// for the "required" tag on Price.
func TestHandleCreateBookingPage_ServiceItemMissingPrice_Rejected(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	body := validCreateBookingPageBody()
	body["services"] = []map[string]interface{}{
		{"id": "svc-1", "name": "Erstberatung", "duration_min": 30},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/booking-pages", jsonBody(t, body))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateBookingPage(rec, req)
	assertValidationError(t, rec, "price")
}

func TestHandleCreateBookingPage_ReachesRPC(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/booking-pages", jsonBody(t, validCreateBookingPageBody()))
	req = withAuth(req, uuid.New().String(), testTenantID)
	routes.HandleCreateBookingPage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleGetBookingPage
// ============================================================================

func TestHandleGetBookingPage_ServiceUnavailable(t *testing.T) {
	routes := testBookingRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/booking-pages/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetBookingPage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetBookingPage_InvalidIDUUID(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/booking-pages/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetBookingPage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleGetBookingPage_ReachesRPC(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/booking-pages/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleGetBookingPage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleUpdateBookingPage
// ============================================================================

func TestHandleUpdateBookingPage_ServiceUnavailable(t *testing.T) {
	routes := testBookingRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/booking-pages/"+id, jsonBody(t, map[string]interface{}{
		"company_name": "Renamed GmbH",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateBookingPage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateBookingPage_InvalidIDUUID(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/booking-pages/not-a-uuid", jsonBody(t, map[string]interface{}{
		"company_name": "Renamed GmbH",
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateBookingPage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleUpdateBookingPage_InvalidJSON(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/booking-pages/"+id, invalidJSON())
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateBookingPage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// TestHandleUpdateBookingPage_ServiceItemZeroDuration_Rejected — same fix as
// the create-side tests above, on the update path.
func TestHandleUpdateBookingPage_ServiceItemZeroDuration_Rejected(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/booking-pages/"+id, jsonBody(t, map[string]interface{}{
		"services": []map[string]interface{}{
			{"id": "svc-1", "name": "Erstberatung", "duration_min": 0, "price": "0.00"},
		},
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateBookingPage(rec, req)
	assertValidationError(t, rec, "duration_min")
}

// TestHandleUpdateBookingPage_EmptyBody_ReachesRPC documents that every field
// on updateBookingPageRequest is optional (a pointer, "no change" when nil):
// a bare {} is not rejected locally, it reaches the RPC layer unchanged, same
// as a full partial update.
func TestHandleUpdateBookingPage_EmptyBody_ReachesRPC(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/booking-pages/"+id, jsonBody(t, map[string]interface{}{}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateBookingPage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandleUpdateBookingPage_ChangedAvailabilityRules_ReachesRPC documents
// the file-level finding above: a syntactically valid availability-rules
// change carries no local slot-conflict check (there is none at this layer)
// and reaches the RPC layer unchanged, mirroring the *_ReachesRPC convention
// in route_calendar_events_resources_test.go for the same "no bufconn stub"
// limitation. Real slot-collision detection is exercised entirely on the
// public CreatePublicBooking/GetAvailability path, out of scope here.
func TestHandleUpdateBookingPage_ChangedAvailabilityRules_ReachesRPC(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/booking-pages/"+id, jsonBody(t, map[string]interface{}{
		"availability_rules": `{"weekdays":[1,2,3,4,5,6,0]}`,
		"active":             true,
	}))
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleUpdateBookingPage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ============================================================================
// HandleDeleteBookingPage
// ============================================================================

func TestHandleDeleteBookingPage_ServiceUnavailable(t *testing.T) {
	routes := testBookingRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/booking-pages/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteBookingPage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleDeleteBookingPage_InvalidIDUUID(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/booking-pages/not-a-uuid", nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleDeleteBookingPage(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

func TestHandleDeleteBookingPage_ReachesRPC(t *testing.T) {
	routes := testBookingRoutes(registryWithService("work"))
	rec := httptest.NewRecorder()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/booking-pages/"+id, nil)
	req = withAuth(req, uuid.New().String(), testTenantID)
	req = withChiURLParam(req, "id", id)
	routes.HandleDeleteBookingPage(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
