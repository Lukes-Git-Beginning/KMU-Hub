package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- allowForgotAttempt ---

func TestAllowForgotAttempt_RateLimit(t *testing.T) {
	routes := NewAuthRoutes(nil)
	email := "ratelimit@test.local"

	for i := 1; i <= forgotRateLimitMax; i++ {
		if !routes.allowForgotAttempt(email) {
			t.Fatalf("attempt %d: expected allowed, got denied", i)
		}
	}
	if routes.allowForgotAttempt(email) {
		t.Fatal("attempt beyond the limit: expected denied, got allowed")
	}
}

func TestAllowForgotAttempt_Normalization(t *testing.T) {
	routes := NewAuthRoutes(nil)

	for i := 1; i <= forgotRateLimitMax; i++ {
		if !routes.allowForgotAttempt("User@X.de") {
			t.Fatalf("attempt %d via User@X.de: expected allowed, got denied", i)
		}
	}
	// Same bucket under a different case/whitespace variant — the shared
	// window is already exhausted.
	if routes.allowForgotAttempt(" user@x.de ") {
		t.Fatal("expected the normalized variant to share the same bucket and be denied")
	}
}

func TestAllowForgotAttempt_WindowReset(t *testing.T) {
	routes := NewAuthRoutes(nil)
	email := "windowreset@test.local"

	for i := 1; i <= forgotRateLimitMax; i++ {
		if !routes.allowForgotAttempt(email) {
			t.Fatalf("attempt %d: expected allowed, got denied", i)
		}
	}
	if routes.allowForgotAttempt(email) {
		t.Fatal("expected denied before the window resets")
	}

	actual, ok := routes.forgotLimiter.Load(email)
	if !ok {
		t.Fatal("expected a bucket to exist for the email")
	}
	b := actual.(*forgotBucket)
	b.mu.Lock()
	b.windowEnd = time.Now().Add(-time.Second)
	b.mu.Unlock()

	if !routes.allowForgotAttempt(email) {
		t.Fatal("expected the expired window to reset the counter and allow the attempt")
	}
	b.mu.Lock()
	count := b.count
	b.mu.Unlock()
	if count != 1 {
		t.Errorf("count after reset = %d, want 1", count)
	}
}

// --- HandleForgotPassword ---

func TestHandleForgotPassword_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/forgot-password", jsonBody(t, map[string]interface{}{
		"email": "user@test.local",
	}))
	routes.HandleForgotPassword(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleForgotPassword_InvalidJSON(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/forgot-password", invalidJSON())
	routes.HandleForgotPassword(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleForgotPassword_MissingEmail(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/forgot-password", jsonBody(t, map[string]interface{}{}))
	routes.HandleForgotPassword(rec, req)
	assertValidationError(t, rec, "email")
}

func TestHandleForgotPassword_RateLimited(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	email := "throttled@test.local"
	for i := 1; i <= forgotRateLimitMax; i++ {
		if !routes.allowForgotAttempt(email) {
			t.Fatalf("attempt %d: expected allowed, got denied", i)
		}
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/forgot-password", jsonBody(t, map[string]interface{}{
		"email": email,
	}))
	routes.HandleForgotPassword(rec, req)
	assertStatus(t, rec, http.StatusTooManyRequests)
	assertErrorContains(t, rec, "too many password reset requests")
	if got := rec.Header().Get("Retry-After"); got != "600" {
		t.Errorf("Retry-After = %q, want %q", got, "600")
	}
}

func TestHandleForgotPassword_AlwaysOK(t *testing.T) {
	// Enumeration-safe: 200 whether or not the account exists — the dummy
	// registered client's RPC error is deliberately swallowed.
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/forgot-password", jsonBody(t, map[string]interface{}{
		"email": "nobody@test.local",
	}))
	routes.HandleForgotPassword(rec, req)
	assertStatus(t, rec, http.StatusOK)
}

// --- HandleResetPassword ---

func TestHandleResetPassword_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/reset-password", jsonBody(t, map[string]interface{}{}))
	routes.HandleResetPassword(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleResetPassword_InvalidJSON(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/reset-password", invalidJSON())
	routes.HandleResetPassword(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleResetPassword_MissingToken(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/reset-password", jsonBody(t, map[string]interface{}{
		"new_password": "longenough1",
	}))
	routes.HandleResetPassword(rec, req)
	assertValidationError(t, rec, "token")
}

func TestHandleResetPassword_ShortPassword(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/reset-password", jsonBody(t, map[string]interface{}{
		"token":        "some-token",
		"new_password": "short",
	}))
	routes.HandleResetPassword(rec, req)
	assertValidationError(t, rec, "new_password")
}

// --- HandleUpdateProfile ---

func TestHandleUpdateProfile_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/api/v1/auth/profile", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	routes.HandleUpdateProfile(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateProfile_AvatarURLTooLong(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	longURL := "https://example.test/" + strings.Repeat("a", 512)
	req := httptest.NewRequest("PATCH", "/api/v1/auth/profile", jsonBody(t, map[string]interface{}{
		"avatar_url": longURL,
	}))
	req = withUserID(req, "user-123")
	routes.HandleUpdateProfile(rec, req)
	assertValidationError(t, rec, "avatar_url")
}

// --- HandleUpdateUser ---

func TestHandleUpdateUser_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/users/123", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleUpdateUser(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleUpdateUser_InvalidUUID(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/users/not-a-uuid", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleUpdateUser(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// --- HandleProvisionTenant ---

func TestHandleProvisionTenant_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tenants", jsonBody(t, map[string]interface{}{}))
	routes.HandleProvisionTenant(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleProvisionTenant_MissingAdminEmail(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tenants", jsonBody(t, map[string]interface{}{
		"name": "Acme GmbH",
	}))
	routes.HandleProvisionTenant(rec, req)
	assertValidationError(t, rec, "admin_email")
}

func TestHandleProvisionTenant_InvalidSeatLimit(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/tenants", jsonBody(t, map[string]interface{}{
		"name":        "Acme GmbH",
		"admin_email": "admin@acme.test",
		"seat_limit":  0,
	}))
	routes.HandleProvisionTenant(rec, req)
	assertValidationError(t, rec, "seat_limit")
}

// --- HandleCreateInvitation, HandleListInvitations ---

func TestHandleListInvitations_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/invitations", nil)
	routes.HandleListInvitations(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleAcceptInvitation ---

func TestHandleAcceptInvitation_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/invitations/tok-123/accept", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "token", "tok-123")
	routes.HandleAcceptInvitation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleAcceptInvitation_MissingToken(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/invitations//accept", jsonBody(t, map[string]interface{}{}))
	req = withChiURLParam(req, "token", "")
	routes.HandleAcceptInvitation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "token is required")
}

func TestHandleAcceptInvitation_MissingPassword(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/invitations/tok-123/accept", jsonBody(t, map[string]interface{}{
		"first_name": "Ada",
		"last_name":  "Lovelace",
	}))
	req = withChiURLParam(req, "token", "tok-123")
	routes.HandleAcceptInvitation(rec, req)
	assertValidationError(t, rec, "password")
}

// --- HandleCancelInvitation ---

func TestHandleCancelInvitation_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/invitations/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleCancelInvitation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCancelInvitation_InvalidUUID(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/invitations/not-a-uuid", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleCancelInvitation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}
