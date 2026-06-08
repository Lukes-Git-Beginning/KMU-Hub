package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthRoutes_ServiceName(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	if routes.ServiceName() != "auth" {
		t.Errorf("ServiceName() = %q, want %q", routes.ServiceName(), "auth")
	}
}

// --- HandleRegister ---

func TestHandleRegister_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleRegister)
}

func TestHandleRegister_InvalidJSON(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/register", invalidJSON())
	routes.HandleRegister(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleRegister_MissingFields(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))

	tests := []struct {
		name      string
		body      map[string]interface{}
		wantField string
	}{
		{"empty", map[string]interface{}{}, "email"},
		{"no_password", map[string]interface{}{"email": "test@example.com"}, "password"},
		{"no_email", map[string]interface{}{"password": "secret123"}, "email"},
		{"empty_strings", map[string]interface{}{"email": "", "password": ""}, "email"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/v1/auth/register", jsonBody(t, tt.body))
			routes.HandleRegister(rec, req)
			assertValidationError(t, rec, tt.wantField)
		})
	}
}

func TestHandleRegister_InvalidEmailFormat(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/register", jsonBody(t, map[string]interface{}{
		"email": "not-an-email", "password": "longenough1",
	}))
	routes.HandleRegister(rec, req)
	assertValidationError(t, rec, "email")
}

func TestHandleRegister_ShortPassword(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/register", jsonBody(t, map[string]interface{}{
		"email": "valid@example.com", "password": "short",
	}))
	routes.HandleRegister(rec, req)
	assertValidationError(t, rec, "password")
}

// --- HandleLogin ---

func TestHandleLogin_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleLogin)
}

func TestHandleLogin_InvalidJSON(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/login", invalidJSON())
	routes.HandleLogin(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

func TestHandleLogin_MissingFields(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/login", jsonBody(t, map[string]interface{}{}))
	routes.HandleLogin(rec, req)
	assertValidationError(t, rec, "email")
}

func TestHandleLogin_InvalidEmailFormat(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/login", jsonBody(t, map[string]interface{}{
		"email": "nope", "password": "whatever",
	}))
	routes.HandleLogin(rec, req)
	assertValidationError(t, rec, "email")
}

// --- HandleRefresh ---

func TestHandleRefresh_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleRefresh)
}

func TestHandleRefresh_InvalidJSON(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/refresh", invalidJSON())
	routes.HandleRefresh(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// --- HandleLogout ---

func TestHandleLogout_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	testServiceUnavailable(t, routes.HandleLogout)
}

func TestHandleLogout_InvalidJSON(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/logout", invalidJSON())
	routes.HandleLogout(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid request body")
}

// --- HandleGetProfile ---

func TestHandleGetProfile_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req = withUserID(req, "user-123")
	routes.HandleGetProfile(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetProfile_NoUserID(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	withAuthRequired(routes.HandleGetProfile)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

// --- HandleChangePassword ---

func TestHandleChangePassword_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/change-password", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "user-123")
	routes.HandleChangePassword(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleChangePassword_NoUserID(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/change-password", jsonBody(t, map[string]interface{}{}))
	withAuthRequired(routes.HandleChangePassword)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleChangePassword_InvalidJSON(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/change-password", invalidJSON())
	req = withUserID(req, "user-123")
	routes.HandleChangePassword(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleChangePassword_MissingFields(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/change-password", jsonBody(t, map[string]interface{}{
		"old_password": "old",
	}))
	req = withUserID(req, "user-123")
	routes.HandleChangePassword(rec, req)
	assertValidationError(t, rec, "new_password")
}

// --- HandleListUsers ---

func TestHandleListUsers_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	routes.HandleListUsers(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// --- HandleGetUser ---

func TestHandleGetUser_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/123", nil)
	req = withChiURLParam(req, "id", "550e8400-e29b-41d4-a716-446655440000")
	routes.HandleGetUser(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleGetUser_InvalidUUID(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/bad-id", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetUser(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "invalid id")
}

// --- HandleCreateInvitation ---

func TestHandleCreateInvitation_ServiceUnavailable(t *testing.T) {
	routes := NewAuthRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/invitations", jsonBody(t, map[string]interface{}{}))
	req = withUserID(req, "admin-123")
	routes.HandleCreateInvitation(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleCreateInvitation_NoUserID(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/invitations", jsonBody(t, map[string]interface{}{}))
	withAuthRequired(routes.HandleCreateInvitation)(rec, req)
	assertStatus(t, rec, http.StatusUnauthorized)
}

func TestHandleCreateInvitation_InvalidJSON(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/invitations", invalidJSON())
	req = withUserID(req, "admin-123")
	routes.HandleCreateInvitation(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateInvitation_MissingEmail(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/invitations", jsonBody(t, map[string]interface{}{
		"role": "member",
	}))
	req = withUserID(req, "admin-123")
	routes.HandleCreateInvitation(rec, req)
	assertValidationError(t, rec, "email")
}

func TestHandleCreateInvitation_InvalidRole(t *testing.T) {
	routes := NewAuthRoutes(registryWithService("auth"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/invitations", jsonBody(t, map[string]interface{}{
		"email": "new@example.com", "role": "superuser",
	}))
	req = withUserID(req, "admin-123")
	routes.HandleCreateInvitation(rec, req)
	assertValidationError(t, rec, "role")
}
