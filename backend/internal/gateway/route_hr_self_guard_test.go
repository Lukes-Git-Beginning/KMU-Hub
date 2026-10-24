package gateway

// hr:self is the employee key of the HR module. These tests drive the real router with token
// shapes taken from the preset roles (member and manager: hr:self at scope own; admin: the coarse
// keys; a read-only coarse token) and pin which routes the new key opens and which it must not.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHRSelfKey_OpensOnlyTheSelfServiceRoutes(t *testing.T) {
	router := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(router, guardTestAuth)

	const (
		allowed = http.StatusServiceUnavailable // guard passed, empty registry stops it
		denied  = http.StatusForbidden
		id      = "11111111-1111-1111-1111-111111111111"
	)

	selfRoutes := []struct{ method, path string }{
		{"POST", "/api/v1/hr/leave/requests"},
		{"GET", "/api/v1/hr/leave/requests"},
		{"GET", "/api/v1/hr/leave/requests/" + id},
		{"POST", "/api/v1/hr/leave/requests/" + id + "/cancel"},
		{"GET", "/api/v1/hr/leave/balance"},
		{"GET", "/api/v1/hr/leave/types"},
		{"GET", "/api/v1/hr/leave/balance/" + id},
		{"GET", "/api/v1/hr/time/entries"},
		{"POST", "/api/v1/hr/time/clock-in"},
		{"POST", "/api/v1/hr/time/clock-out"},
		{"POST", "/api/v1/hr/time/break/start"},
		{"POST", "/api/v1/hr/time/break/end"},
		{"GET", "/api/v1/hr/time/active"},
		{"GET", "/api/v1/hr/time/status"},
		{"POST", "/api/v1/hr/time/entries"},
		{"GET", "/api/v1/hr/time/summary/daily"},
		{"GET", "/api/v1/hr/time/summary/weekly"},
		{"POST", "/api/v1/hr/time/corrections"},
		{"GET", "/api/v1/hr/time/balance"},
		{"GET", "/api/v1/hr/time/analytics"},
		{"GET", "/api/v1/hr/time/weeks/status"},
		{"POST", "/api/v1/hr/time/weeks/submit"},
	}
	// Supervisory routes. The two foreign-id reads open for hr:self but are pinned to the caller.
	notSelfRoutes := []struct{ method, path string }{
		{"POST", "/api/v1/hr/leave/requests/" + id + "/approve"},
		{"POST", "/api/v1/hr/leave/requests/" + id + "/reject"},
		{"POST", "/api/v1/hr/leave/sick"},
		{"GET", "/api/v1/hr/employees"},
		{"GET", "/api/v1/hr/settings"},
	}

	call := func(method, path string, perms []string) int {
		req := withPermissions(httptest.NewRequest(method, path, nil), perms...)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	for _, rt := range selfRoutes {
		assert.Equal(t, allowed, call(rt.method, rt.path, []string{"hr:self"}), "hr:self on %s %s", rt.method, rt.path)
		assert.Equal(t, denied, call(rt.method, rt.path, []string{"files:read"}), "unrelated key on %s %s", rt.method, rt.path)
	}
	for _, rt := range notSelfRoutes {
		assert.Equal(t, denied, call(rt.method, rt.path, []string{"hr:self"}), "hr:self must not open %s %s", rt.method, rt.path)
	}

	// Regression: the coarse keys keep working on the converted routes (still-valid old tokens).
	assert.Equal(t, allowed, call("POST", "/api/v1/hr/leave/requests", []string{"hr:write"}))
	assert.Equal(t, allowed, call("GET", "/api/v1/hr/leave/balance", []string{"hr:read"}))
	assert.Equal(t, allowed, call("POST", "/api/v1/hr/time/clock-in", []string{"hr:write"}))
	assert.Equal(t, allowed, call("GET", "/api/v1/hr/time/status", []string{"hr:read"}))
	assert.Equal(t, denied, call("POST", "/api/v1/hr/leave/requests", []string{"hr:read"}), "read does not grant write")
}

func TestHRSelfKey_OwnScopeNarrowsLeaveReadsLikeTheCoarseKey(t *testing.T) {
	routes, stub := newLeaveScopeRoutes(t)
	router := chi.NewRouter()
	routes.RegisterRoutes(router, guardTestAuth)

	asEmployee := func(target string) *http.Request {
		req := withPermissions(leaveRequest(target, map[string]string{"hr:self": "own"}), "hr:self")
		return req
	}

	for _, query := range []string{"", "?employee_id=" + leaveOtherID, "?user_id=" + leaveOtherID} {
		stub.listed = nil
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, asEmployee("/api/v1/hr/leave/requests"+query))
		assertStatus(t, rec, http.StatusOK)
		require.NotNil(t, stub.listed)
		assert.Equal(t, leaveCallerID, stub.listed.GetEmployeeId(), "query %q must not widen hr:self at own", query)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, asEmployee("/api/v1/hr/leave/requests/44444444-4444-4444-4444-444444444444"))
	assertStatus(t, rec, http.StatusNotFound)

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, asEmployee("/api/v1/hr/leave/requests/11111111-1111-1111-1111-111111111111"))
	assertStatus(t, rec, http.StatusOK)
}
