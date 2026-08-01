package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/security"
)

// guardTestAuth stands in for the JWT auth middleware. The guards under test
// read the permission list straight from the request context, which each test
// case fills in itself.
func guardTestAuth(next http.Handler) http.Handler { return next }

// withPermissions puts a permission list into the context the way the auth
// middleware does after decoding an access token.
func withPermissions(r *http.Request, perms ...string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), middleware.UserPermsKey, perms))
}

// TestCapabilityGuards_AdditiveWiring checks the guard wiring of the modules
// that were tightened onto capability-catalogue keys: every route must accept
// BOTH its legacy coarse key and its catalogue key, and must still reject a
// token that carries neither.
//
// The registry is empty, so a request that passes the guard dies further down
// with 503 (service not registered). The expected status is spelled out rather
// than "anything but 403" on purpose: a mistyped path would return 404, which
// would satisfy a loose assertion without any guard ever running.
func TestCapabilityGuards_AdditiveWiring(t *testing.T) {
	wikiRouter := chi.NewRouter()
	newWikiRoutes(emptyRegistry()).RegisterRoutes(wikiRouter, guardTestAuth)

	hrRouter := chi.NewRouter()
	NewHRRoutes(emptyRegistry(), nil).RegisterRoutes(hrRouter, guardTestAuth)

	helpdeskRouter := chi.NewRouter()
	newHelpdeskRoutes(emptyRegistry()).RegisterRoutes(helpdeskRouter, guardTestAuth)

	chatRouter := chi.NewRouter()
	NewChatRoutes(emptyRegistry()).RegisterRoutes(chatRouter, guardTestAuth)

	inboxRouter := chi.NewRouter()
	NewInboxRoutes(emptyRegistry()).RegisterRoutes(inboxRouter, guardTestAuth)

	calendarRouter := chi.NewRouter()
	NewCalendarRoutes(emptyRegistry()).RegisterRoutes(calendarRouter, guardTestAuth)

	bookingRouter := chi.NewRouter()
	NewBookingRoutes(emptyRegistry(), security.NewCaptchaVerifier("", "")).RegisterRoutes(bookingRouter, guardTestAuth)

	const articleID = "11111111-1111-1111-1111-111111111111"

	const (
		allowed = http.StatusServiceUnavailable // guard passed, empty registry stops it
		denied  = http.StatusForbidden
	)

	tests := []struct {
		name       string
		router     chi.Router
		method     string
		path       string
		perms      []string
		wantStatus int
	}{
		// --- wiki: read ---
		{"wiki list, legacy key only", wikiRouter, "GET", "/api/v1/wiki/articles/", []string{"wiki:articles:read"}, allowed},
		{"wiki list, catalogue key only", wikiRouter, "GET", "/api/v1/wiki/articles/", []string{"wiki:article:read"}, allowed},
		{"wiki list, neither key", wikiRouter, "GET", "/api/v1/wiki/articles/", []string{"wiki:categories:read"}, denied},
		{"wiki list, no permissions at all", wikiRouter, "GET", "/api/v1/wiki/articles/", nil, denied},

		// --- wiki: write actions are distinct capabilities ---
		{"wiki create, legacy key only", wikiRouter, "POST", "/api/v1/wiki/articles/", []string{"wiki:articles:write"}, allowed},
		{"wiki create, catalogue key only", wikiRouter, "POST", "/api/v1/wiki/articles/", []string{"wiki:article:create"}, allowed},
		{"wiki create, read key does not grant write", wikiRouter, "POST", "/api/v1/wiki/articles/", []string{"wiki:article:read"}, denied},
		{"wiki edit, catalogue key only", wikiRouter, "PATCH", "/api/v1/wiki/articles/" + articleID + "/", []string{"wiki:article:edit"}, allowed},
		{"wiki edit, delete key does not grant edit", wikiRouter, "PATCH", "/api/v1/wiki/articles/" + articleID + "/", []string{"wiki:article:delete"}, denied},
		{"wiki delete, catalogue key only", wikiRouter, "DELETE", "/api/v1/wiki/articles/" + articleID + "/", []string{"wiki:article:delete"}, allowed},

		// --- wiki: share tokens and categories ---
		{"wiki share create, catalogue key only", wikiRouter, "POST", "/api/v1/wiki/articles/" + articleID + "/share", []string{"wiki:share_token:create"}, allowed},
		{"wiki share revoke, catalogue key only", wikiRouter, "DELETE", "/api/v1/wiki/share/" + articleID, []string{"wiki:share_token:create"}, allowed},
		{"wiki category manage, legacy key only", wikiRouter, "POST", "/api/v1/wiki/categories/", []string{"wiki:categories:write"}, allowed},
		{"wiki category manage, catalogue key only", wikiRouter, "POST", "/api/v1/wiki/categories/", []string{"wiki:category:manage"}, allowed},
		{"wiki category manage, article key does not grant it", wikiRouter, "POST", "/api/v1/wiki/categories/", []string{"wiki:article:edit"}, denied},

		// --- zeiterfassung: supervisory routes ---
		{"week approve, legacy key only", hrRouter, "POST", "/api/v1/hr/time/weeks/approve", []string{"hr:write"}, allowed},
		{"week approve, catalogue key only", hrRouter, "POST", "/api/v1/hr/time/weeks/approve", []string{"zeiterfassung:week:approve"}, allowed},
		{"week reject, catalogue key only", hrRouter, "POST", "/api/v1/hr/time/weeks/reject", []string{"zeiterfassung:week:approve"}, allowed},
		{"week approve, team key does not grant it", hrRouter, "POST", "/api/v1/hr/time/weeks/approve", []string{"zeiterfassung:team:view"}, denied},
		{"team view, catalogue key only", hrRouter, "GET", "/api/v1/hr/time/team", []string{"zeiterfassung:team:view"}, allowed},
		{"correction approve, catalogue key only", hrRouter, "POST", "/api/v1/hr/time/corrections/" + articleID + "/approve", []string{"zeiterfassung:corrections:approve"}, allowed},

		// --- zeiterfassung: personal daily use stays on the coarse key ---
		// The catalogue deliberately carries no capability for clocking in, so
		// a supervisory key must NOT open it.
		{"clock-in still requires hr:write", hrRouter, "POST", "/api/v1/hr/time/clock-in", []string{"zeiterfassung:week:approve"}, denied},
		{"clock-in with hr:write", hrRouter, "POST", "/api/v1/hr/time/clock-in", []string{"hr:write"}, allowed},

		// --- helpdesk: tickets ---
		{"ticket create, legacy key only", helpdeskRouter, "POST", "/api/v1/helpdesk/tickets", []string{"helpdesk:write"}, allowed},
		{"ticket create, catalogue key only", helpdeskRouter, "POST", "/api/v1/helpdesk/tickets", []string{"helpdesk:ticket:create"}, allowed},
		{"ticket list, catalogue key only", helpdeskRouter, "GET", "/api/v1/helpdesk/tickets", []string{"helpdesk:ticket:read"}, allowed},
		{"ticket edit, catalogue key only", helpdeskRouter, "PUT", "/api/v1/helpdesk/tickets/" + articleID, []string{"helpdesk:ticket:edit"}, allowed},
		{"ticket close, catalogue edit key maps to it", helpdeskRouter, "POST", "/api/v1/helpdesk/tickets/" + articleID + "/close", []string{"helpdesk:ticket:edit"}, allowed},
		{"ticket close, reply key does not grant it", helpdeskRouter, "POST", "/api/v1/helpdesk/tickets/" + articleID + "/close", []string{"helpdesk:ticket:reply"}, denied},
		{"ticket reply, catalogue key only", helpdeskRouter, "POST", "/api/v1/helpdesk/tickets/" + articleID + "/messages", []string{"helpdesk:ticket:reply"}, allowed},
		{"ticket reply, edit key does not grant it", helpdeskRouter, "POST", "/api/v1/helpdesk/tickets/" + articleID + "/messages", []string{"helpdesk:ticket:edit"}, denied},

		// --- helpdesk: kb, canned responses, stats ---
		{"kb article create, catalogue key only", helpdeskRouter, "POST", "/api/v1/helpdesk/kb-articles", []string{"helpdesk:kb:manage"}, allowed},
		{"canned response create, catalogue key only", helpdeskRouter, "POST", "/api/v1/helpdesk/canned-responses", []string{"helpdesk:canned:manage"}, allowed},
		{"stats, catalogue key only", helpdeskRouter, "GET", "/api/v1/helpdesk/stats", []string{"helpdesk:stats:view"}, allowed},

		// --- helpdesk: queues have no catalogue key, coarse key still required ---
		{"queue create still requires helpdesk:write", helpdeskRouter, "POST", "/api/v1/helpdesk/queues", []string{"helpdesk:ticket:edit"}, denied},
		{"queue create with helpdesk:write", helpdeskRouter, "POST", "/api/v1/helpdesk/queues", []string{"helpdesk:write"}, allowed},

		// --- kommunikation: channel administration ---
		{"channel create, legacy key only", chatRouter, "POST", "/api/v1/channels/", []string{"channels:write"}, allowed},
		{"channel create, catalogue key only", chatRouter, "POST", "/api/v1/channels/", []string{"kommunikation:channel:manage"}, allowed},
		{"channel delete, catalogue key only", chatRouter, "DELETE", "/api/v1/channels/" + articleID, []string{"kommunikation:channel:manage"}, allowed},
		{"channel member role, catalogue key only", chatRouter, "PUT", "/api/v1/channels/" + articleID + "/members/" + articleID + "/role", []string{"kommunikation:channel:manage"}, allowed},

		// --- kommunikation: personal channel use is not widened by the new key ---
		{"channel join still requires channels:write", chatRouter, "POST", "/api/v1/channels/" + articleID + "/join", []string{"kommunikation:channel:manage"}, denied},
		{"channel join with channels:write", chatRouter, "POST", "/api/v1/channels/" + articleID + "/join", []string{"channels:write"}, allowed},

		// --- kommunikation: inbox canned responses and team-inbox routes ---
		{"inbox canned response create, catalogue key only", inboxRouter, "POST", "/api/v1/inbox/canned-responses", []string{"kommunikation:canned:manage"}, allowed},
		{"team inbox update, catalogue key only", inboxRouter, "PUT", "/api/v1/inbox/teams/" + articleID, []string{"kommunikation:team_inbox:manage"}, allowed},
		{"team inbox add member, catalogue key only", inboxRouter, "POST", "/api/v1/inbox/teams/" + articleID + "/members", []string{"kommunikation:team_inbox:manage"}, allowed},

		// --- kalender: event categories and booking pages ---
		{"category create, legacy key only", calendarRouter, "POST", "/api/v1/calendar/categories", []string{"calendars:write"}, allowed},
		{"category create, catalogue key only", calendarRouter, "POST", "/api/v1/calendar/categories", []string{"kalender:category:manage"}, allowed},
		{"category delete, catalogue key only", calendarRouter, "DELETE", "/api/v1/calendar/categories/" + articleID, []string{"kalender:category:manage"}, allowed},
		{"booking page create, catalogue key only", bookingRouter, "POST", "/api/v1/calendar/booking-pages/", []string{"kalender:booking_page:manage"}, allowed},
		{"booking page delete, catalogue key only", bookingRouter, "DELETE", "/api/v1/calendar/booking-pages/" + articleID, []string{"kalender:booking_page:manage"}, allowed},

		// --- kalender: calendars/events/resources have no catalogue key, untouched ---
		{"calendar create still requires calendars:write only", calendarRouter, "POST", "/api/v1/calendar/calendars", []string{"kalender:category:manage"}, denied},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req = withAuth(req, "user-123", testTenantID)
			req = withPermissions(req, tt.perms...)

			rec := httptest.NewRecorder()
			tt.router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
