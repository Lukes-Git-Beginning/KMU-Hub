package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func withUserContext(r *http.Request, userID string, roles, perms []string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, UserIDKey, userID)
	ctx = context.WithValue(ctx, UserRolesKey, roles)
	ctx = context.WithValue(ctx, UserPermsKey, perms)
	return r.WithContext(ctx)
}

func TestRequireRole(t *testing.T) {
	handler := RequireRole("admin", "manager")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		roles      []string
		wantStatus int
	}{
		{"admin allowed", []string{"admin"}, http.StatusOK},
		{"manager allowed", []string{"manager"}, http.StatusOK},
		{"member denied", []string{"member"}, http.StatusForbidden},
		{"no roles denied", nil, http.StatusForbidden},
		{"multiple roles with one match", []string{"member", "admin"}, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = withUserContext(req, "user-123", tt.roles, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// TestRequirePermissionAny covers the compatibility guarantee the helper
// exists for: a token minted before a guard was tightened (coarse key only)
// must still pass, and so must a freshly minted one (fine-grained key).
func TestRequirePermissionAny(t *testing.T) {
	handler := RequirePermissionAny(
		[2]string{"files", "write"},
		[2]string{"documents:file", "upload"},
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		perms      []string
		wantStatus int
	}{
		{"legacy token, coarse key only", []string{"files:write"}, http.StatusOK},
		{"fresh token, fine-grained key only", []string{"documents:file:upload"}, http.StatusOK},
		{"both keys", []string{"files:write", "documents:file:upload"}, http.StatusOK},
		{"neither key", []string{"files:read", "documents:file:read"}, http.StatusForbidden},
		{"no permissions", nil, http.StatusForbidden},
		{"resource matches but action does not", []string{"documents:file:delete"}, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = withUserContext(req, "user-123", nil, tt.perms)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestRequirePermission(t *testing.T) {
	handler := RequirePermission("contacts", "write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		perms      []string
		wantStatus int
	}{
		{"has permission", []string{"contacts:read", "contacts:write"}, http.StatusOK},
		{"missing permission", []string{"contacts:read"}, http.StatusForbidden},
		{"no permissions", nil, http.StatusForbidden},
		{"wrong resource", []string{"deals:write"}, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = withUserContext(req, "user-123", nil, tt.perms)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
