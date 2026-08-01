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

	workRouter := chi.NewRouter()
	NewWorkRoutes(emptyRegistry()).RegisterRoutes(workRouter, guardTestAuth)

	documentRouter := chi.NewRouter()
	NewDocumentRoutes(emptyRegistry()).RegisterRoutes(documentRouter, guardTestAuth)

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

		// --- work: projects ---
		{"project list, catalogue key only", workRouter, "GET", "/api/v1/projects/", []string{"work:project:read"}, allowed},
		{"project create, legacy key only", workRouter, "POST", "/api/v1/projects/", []string{"projects:write"}, allowed},
		{"project create, catalogue key only", workRouter, "POST", "/api/v1/projects/", []string{"work:project:create"}, allowed},
		{"project create, edit key does not grant create", workRouter, "POST", "/api/v1/projects/", []string{"work:project:edit"}, denied},
		{"project update, catalogue key only", workRouter, "PUT", "/api/v1/projects/" + articleID, []string{"work:project:edit"}, allowed},
		{"project delete, catalogue key only", workRouter, "DELETE", "/api/v1/projects/" + articleID, []string{"work:project:delete"}, allowed},
		{"project delete, edit key does not grant delete", workRouter, "DELETE", "/api/v1/projects/" + articleID, []string{"work:project:edit"}, denied},
		{"remove project member, catalogue key only", workRouter, "DELETE", "/api/v1/projects/" + articleID + "/members/" + articleID, []string{"work:project:manage_members"}, allowed},
		{"kanban status delete, catalogue edit key maps to it", workRouter, "DELETE", "/api/v1/project-statuses/" + articleID, []string{"work:project:edit"}, allowed},

		// --- work: add-member and set-preference have no FE caller, untouched ---
		{"add project member still requires projects:write only", workRouter, "POST", "/api/v1/projects/" + articleID + "/members", []string{"work:project:edit"}, denied},
		{"add project member with projects:write", workRouter, "POST", "/api/v1/projects/" + articleID + "/members", []string{"projects:write"}, allowed},

		// --- work: tasks ---
		{"task list, catalogue key only", workRouter, "GET", "/api/v1/tasks/", []string{"work:task:read"}, allowed},
		{"task create, catalogue key only", workRouter, "POST", "/api/v1/tasks/", []string{"work:task:create"}, allowed},
		{"task update, catalogue key only", workRouter, "PUT", "/api/v1/tasks/" + articleID, []string{"work:task:edit"}, allowed},
		{"task delete, catalogue key only", workRouter, "DELETE", "/api/v1/tasks/" + articleID, []string{"work:task:delete"}, allowed},
		{"task delete, edit key does not grant delete", workRouter, "DELETE", "/api/v1/tasks/" + articleID, []string{"work:task:edit"}, denied},
		{"task comment create, catalogue key only", workRouter, "POST", "/api/v1/tasks/" + articleID + "/comments", []string{"work:task:comment"}, allowed},
		{"task comment create, edit key does not grant it", workRouter, "POST", "/api/v1/tasks/" + articleID + "/comments", []string{"work:task:edit"}, denied},
		{"task custom fields set, catalogue key only", workRouter, "PUT", "/api/v1/tasks/" + articleID + "/custom-fields", []string{"work:task:edit"}, allowed},
		{"manual time entry, catalogue key only", workRouter, "POST", "/api/v1/tasks/" + articleID + "/time-entries", []string{"work:time:log"}, allowed},
		{"manual time entry, edit key does not grant it", workRouter, "POST", "/api/v1/tasks/" + articleID + "/time-entries", []string{"work:task:edit"}, denied},
		{"task labels set, catalogue key only", workRouter, "PUT", "/api/v1/tasks/" + articleID + "/labels", []string{"work:task:edit"}, allowed},
		{"task dependency delete, catalogue edit key maps to it", workRouter, "DELETE", "/api/v1/task-dependencies/" + articleID, []string{"work:task:edit"}, allowed},

		// --- work: move, comment-update and time-entry-update stay isOwn/coarse-only ---
		{"move task still requires tasks:write only", workRouter, "POST", "/api/v1/tasks/" + articleID + "/move", []string{"work:task:edit"}, denied},
		{"move task with tasks:write", workRouter, "POST", "/api/v1/tasks/" + articleID + "/move", []string{"tasks:write"}, allowed},
		{"time entry update still requires tasks:write only", workRouter, "PUT", "/api/v1/time-entries/" + articleID, []string{"work:time:log"}, denied},

		// --- documents: folders and files ---
		{"folder list, catalogue key only", documentRouter, "GET", "/api/v1/documents/folders/", []string{"documents:file:read"}, allowed},
		{"folder create, legacy key only", documentRouter, "POST", "/api/v1/documents/folders/", []string{"documents:write"}, allowed},
		{"folder create, catalogue key only", documentRouter, "POST", "/api/v1/documents/folders/", []string{"documents:file:edit"}, allowed},
		{"folder delete, catalogue key only", documentRouter, "DELETE", "/api/v1/documents/folders/" + articleID, []string{"documents:file:delete"}, allowed},
		{"file upload register, catalogue key only", documentRouter, "POST", "/api/v1/documents/files/", []string{"documents:file:upload"}, allowed},
		{"file upload register, edit key does not grant it", documentRouter, "POST", "/api/v1/documents/files/", []string{"documents:file:edit"}, denied},
		{"file update, catalogue key only", documentRouter, "PUT", "/api/v1/documents/files/" + articleID, []string{"documents:file:edit"}, allowed},
		{"file move, catalogue key only", documentRouter, "POST", "/api/v1/documents/files/" + articleID + "/move", []string{"documents:file:edit"}, allowed},
		{"file delete, catalogue key only", documentRouter, "DELETE", "/api/v1/documents/files/" + articleID, []string{"documents:file:delete"}, allowed},
		{"file download url, catalogue key only", documentRouter, "GET", "/api/v1/documents/files/" + articleID + "/download-url", []string{"documents:file:download"}, allowed},
		{"file download url, edit key does not grant it", documentRouter, "GET", "/api/v1/documents/files/" + articleID + "/download-url", []string{"documents:file:edit"}, denied},

		// --- documents: copy is a read-action in the FE, stays coarse-only ---
		{"file copy still requires documents:write only", documentRouter, "POST", "/api/v1/documents/files/" + articleID + "/copy", []string{"documents:file:edit"}, denied},
		{"file copy with documents:write", documentRouter, "POST", "/api/v1/documents/files/" + articleID + "/copy", []string{"documents:write"}, allowed},

		// --- documents: shares and versions ---
		{"share create, catalogue key only", documentRouter, "POST", "/api/v1/documents/shares/", []string{"documents:share:manage"}, allowed},
		{"share delete, catalogue key only", documentRouter, "DELETE", "/api/v1/documents/shares/", []string{"documents:share:manage"}, allowed},
		{"version list, catalogue key only", documentRouter, "GET", "/api/v1/documents/files/" + articleID + "/versions", []string{"documents:version:restore"}, allowed},
		{"version list, read key does not grant it", documentRouter, "GET", "/api/v1/documents/files/" + articleID + "/versions", []string{"documents:read"}, denied},
		{"version revert, catalogue key only", documentRouter, "POST", "/api/v1/documents/files/" + articleID + "/versions/revert", []string{"documents:version:restore"}, allowed},

		// --- documents: tags/links/search have no catalogue key, untouched ---
		{"tag create still requires documents:write only", documentRouter, "POST", "/api/v1/documents/tags/", []string{"documents:file:edit"}, denied},
		{"tag create with documents:write", documentRouter, "POST", "/api/v1/documents/tags/", []string{"documents:write"}, allowed},
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
