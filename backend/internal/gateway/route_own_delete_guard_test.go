package gateway

// The four own-item delete routes sit behind the coarse delete keys (calendars:delete,
// messages:delete, files:delete). Migration 000336 grants them to member and manager; the services
// refuse anyone but the owner/author. These tests drive the real routers with that token shape and
// pin that the guard lets the key through and keeps every unrelated key out.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

func TestOwnDeleteRoutes_AcceptTheDeleteKeyAndNothingElse(t *testing.T) {
	const (
		allowed = http.StatusServiceUnavailable // guard passed, empty registry stops it
		denied  = http.StatusForbidden
		id      = "11111111-1111-1111-1111-111111111111"
	)

	router := chi.NewRouter()
	NewCalendarRoutes(emptyRegistry()).RegisterRoutes(router, guardTestAuth)
	NewChatRoutes(emptyRegistry()).RegisterRoutes(router, guardTestAuth)

	routes := []struct{ key, path string }{
		{"calendars:delete", "/api/v1/calendar/events/" + id},
		{"calendars:delete", "/api/v1/calendar/calendars/" + id},
		{"messages:delete", "/api/v1/messages/" + id},
		{"files:delete", "/api/v1/files/" + id},
	}

	call := func(path string, perms ...string) int {
		req := withPermissions(httptest.NewRequest(http.MethodDelete, path, nil), perms...)
		// The chat routers sit behind RequireAuthenticated, which wants a user id.
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, id))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	for _, rt := range routes {
		assert.Equal(t, allowed, call(rt.path, rt.key), "%s on DELETE %s", rt.key, rt.path)
		// The write key every member already holds must not delete.
		assert.Equal(t, denied, call(rt.path, "calendars:write", "messages:write", "files:write"),
			"write keys must not delete %s", rt.path)
	}
}
