package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kmuhub/kmuhub/internal/security"
)

// newBookingRoutes creates a BookingRoutes with no captcha (disabled) for tests
// that only exercise validation / honeypot logic (no gRPC calls needed there).
func newBookingRoutesForTest(registry *ServiceRegistry) *BookingRoutes {
	captcha := security.NewCaptchaVerifier("", "https://example.com/siteverify")
	return NewBookingRoutes(registry, captcha)
}

// --- HandleCreatePublicBooking: honeypot ---

func TestHandleCreatePublicBooking_HoneypotTriggered(t *testing.T) {
	// A submission with the "website" honeypot field populated must be rejected
	// with 400 and a generic error message that leaks no information about the honeypot.
	routes := newBookingRoutesForTest(registryWithService("work"))

	body := jsonBody(t, map[string]any{
		"slug":       "test-slug",
		"service_id": "550e8400-e29b-41d4-a716-446655440001",
		"date":       "2026-07-01",
		"time_slot":  "10:00",
		"customer": map[string]any{
			"name":  "Bot User",
			"email": "bot@example.com",
		},
		// Honeypot field — bots fill this in.
		"website": "http://spam.example.com",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/bookings", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	routes.HandleCreatePublicBooking(rec, req)

	assertStatus(t, rec, http.StatusBadRequest)
	// The error must NOT mention "honeypot" or any implementation detail.
	assertErrorContains(t, rec, "invalid request")
}

func TestHandleCreatePublicBooking_HoneypotEmpty_Proceeds(t *testing.T) {
	// When the honeypot field is absent the request should NOT be rejected for that
	// reason. It will fail at the gRPC call (service not reachable in test), which
	// yields 503 — that means the honeypot guard was passed successfully.
	routes := newBookingRoutesForTest(registryWithService("work"))

	body := jsonBody(t, map[string]any{
		"slug":       "test-slug",
		"service_id": "550e8400-e29b-41d4-a716-446655440001",
		"date":       "2026-07-01",
		"time_slot":  "10:00",
		"customer": map[string]any{
			"name":  "Real User",
			"email": "real@example.com",
		},
		// "website" omitted — legitimate client behaviour.
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/bookings", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	routes.HandleCreatePublicBooking(rec, req)

	// 503 = passed honeypot + captcha checks, failed at gRPC (expected in unit test).
	// Any 4xx other than a known validation shape would indicate a false-positive reject.
	if rec.Code == http.StatusBadRequest {
		t.Errorf("honeypot check false-positive: got 400, body = %s", rec.Body.String())
	}
}

func TestHandleCreatePublicBooking_InvalidJSON(t *testing.T) {
	routes := newBookingRoutesForTest(registryWithService("work"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/bookings", invalidJSON())
	req.Header.Set("Content-Type", "application/json")
	routes.HandleCreatePublicBooking(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}
