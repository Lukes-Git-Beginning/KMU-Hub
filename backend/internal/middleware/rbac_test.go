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
