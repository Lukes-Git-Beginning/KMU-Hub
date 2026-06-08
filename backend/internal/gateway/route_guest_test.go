package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// newGuestRoutes creates a GuestRoutes with a nil guestService.
// This is sufficient for tests that only exercise validation/decode
// (which fail before the service is called).
func newGuestRoutes(registry *ServiceRegistry) *GuestRoutes {
	return NewGuestRoutes(nil, registry)
}

// --- HandleCreateSession ---

func TestHandleCreateSession_InvalidJSON(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/sessions", invalidJSON())
	routes.HandleCreateSession(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleCreateSession_MissingChannelID(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/sessions", jsonBody(t, map[string]interface{}{
		"display_name": "Alice",
	}))
	routes.HandleCreateSession(rec, req)
	assertValidationError(t, rec, "channel_id")
}

func TestHandleCreateSession_InvalidChannelIDFormat(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/sessions", jsonBody(t, map[string]interface{}{
		"channel_id":   "not-a-uuid",
		"display_name": "Alice",
	}))
	routes.HandleCreateSession(rec, req)
	assertValidationError(t, rec, "channel_id")
}

func TestHandleCreateSession_MissingDisplayName(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/sessions", jsonBody(t, map[string]interface{}{
		"channel_id": "550e8400-e29b-41d4-a716-446655440000",
	}))
	routes.HandleCreateSession(rec, req)
	assertValidationError(t, rec, "display_name")
}

func TestHandleCreateSession_InvalidEmailFormat(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/sessions", jsonBody(t, map[string]interface{}{
		"channel_id":   "550e8400-e29b-41d4-a716-446655440000",
		"display_name": "Alice",
		"email":        "not-an-email",
	}))
	routes.HandleCreateSession(rec, req)
	assertValidationError(t, rec, "email")
}

// --- HandleValidateToken ---

func TestHandleValidateToken_InvalidJSON(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/sessions/validate", invalidJSON())
	routes.HandleValidateToken(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleValidateToken_MissingToken(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/sessions/validate", jsonBody(t, map[string]interface{}{}))
	routes.HandleValidateToken(rec, req)
	assertValidationError(t, rec, "token")
}

// --- HandleSendMessage (guest) ---
// Note: "send message" also exists on chat routes; these tests are prefixed
// with "Guest" to avoid redeclaration within the gateway test package.

func TestGuestHandleSendMessage_NoSession(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/channels/xxx/messages", invalidJSON())
	// Without a session in context the handler returns 401 before body decode.
	routes.HandleSendMessage(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestGuestHandleSendMessage_NoSession_EmptyBody(t *testing.T) {
	routes := newGuestRoutes(registryWithService("chat"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/guest/channels/xxx/messages", jsonBody(t, map[string]interface{}{}))
	// Handler guards on session before body decode — verifying 401 not validation error.
	routes.HandleSendMessage(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}
