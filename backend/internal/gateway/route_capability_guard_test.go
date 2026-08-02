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

	crmRouter := chi.NewRouter()
	NewCRMRoutes(emptyRegistry(), nil).RegisterRoutes(crmRouter, guardTestAuth)

	bizRouter := chi.NewRouter()
	NewBizRoutes(emptyRegistry()).RegisterRoutes(bizRouter, guardTestAuth)

	inventarRouter := chi.NewRouter()
	newInventarRoutes(emptyRegistry()).RegisterRoutes(inventarRouter, guardTestAuth)

	einkaufRouter := chi.NewRouter()
	newEinkaufRoutes(emptyRegistry()).RegisterRoutes(einkaufRouter, guardTestAuth)

	produktionRouter := chi.NewRouter()
	newProduktionRoutes(emptyRegistry()).RegisterRoutes(produktionRouter, guardTestAuth)

	vertraegeRouter := chi.NewRouter()
	newVertraegeRoutes(emptyRegistry()).RegisterRoutes(vertraegeRouter, guardTestAuth)

	schichtenRouter := chi.NewRouter()
	newSchichtenRoutes(emptyRegistry()).RegisterRoutes(schichtenRouter, guardTestAuth)

	fuhrparkRouter := chi.NewRouter()
	newFuhrparkRoutes(emptyRegistry()).RegisterRoutes(fuhrparkRouter, guardTestAuth)

	vermietungRouter := chi.NewRouter()
	newVermietungRoutes(emptyRegistry()).RegisterRoutes(vermietungRouter, guardTestAuth)

	rapporteRouter := chi.NewRouter()
	newRapporteRoutes(emptyRegistry()).RegisterRoutes(rapporteRouter, guardTestAuth)

	dialerRouter := chi.NewRouter()
	NewDialerRoutes(emptyRegistry()).RegisterRoutes(dialerRouter, guardTestAuth)

	berichteRouter := chi.NewRouter()
	NewBerichteRoutes(emptyRegistry(), berichteFlagsON()).RegisterRoutes(berichteRouter, guardTestAuth)

	formulareRouter := chi.NewRouter()
	newFormulareRoutes(emptyRegistry()).RegisterRoutes(formulareRouter, guardTestAuth)

	automationRouter := chi.NewRouter()
	NewAutomationRoutes(emptyRegistry()).RegisterRoutes(automationRouter, guardTestAuth)

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
		{"file activity, catalogue key only", documentRouter, "GET", "/api/v1/documents/files/" + articleID + "/activity", []string{"documents:file:read"}, allowed},
		{"file activity, edit key does not grant it", documentRouter, "GET", "/api/v1/documents/files/" + articleID + "/activity", []string{"documents:file:edit"}, denied},

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
		{"entity link delete by ID still requires documents:write only", documentRouter, "DELETE", "/api/v1/documents/links/" + articleID, []string{"documents:file:edit"}, denied},
		{"entity link delete by ID with documents:write", documentRouter, "DELETE", "/api/v1/documents/links/" + articleID, []string{"documents:write"}, allowed},

		// --- crm: contacts ---
		{"contact list, legacy key only", crmRouter, "GET", "/api/v1/contacts", []string{"contacts:read"}, allowed},
		{"contact list, catalogue key only", crmRouter, "GET", "/api/v1/contacts", []string{"crm:contact:read"}, allowed},
		{"contact list, neither key", crmRouter, "GET", "/api/v1/contacts", []string{"crm:deal:read"}, denied},
		{"contact create, legacy key only", crmRouter, "POST", "/api/v1/contacts", []string{"contacts:write"}, allowed},
		{"contact create, catalogue key only", crmRouter, "POST", "/api/v1/contacts", []string{"crm:contact:create"}, allowed},
		{"contact create, edit key does not grant create", crmRouter, "POST", "/api/v1/contacts", []string{"crm:contact:edit"}, denied},
		{"contact update, catalogue key only", crmRouter, "PUT", "/api/v1/contacts/" + articleID, []string{"crm:contact:edit"}, allowed},
		{"contact delete, catalogue key only", crmRouter, "DELETE", "/api/v1/contacts/" + articleID, []string{"crm:contact:delete"}, allowed},
		{"contact delete, edit key does not grant delete", crmRouter, "DELETE", "/api/v1/contacts/" + articleID, []string{"crm:contact:edit"}, denied},
		{"contact tags add, catalogue edit key maps to it", crmRouter, "POST", "/api/v1/contacts/" + articleID + "/tags", []string{"crm:contact:edit"}, allowed},
		{"contact import csv, catalogue key only", crmRouter, "POST", "/api/v1/contacts/import/csv", []string{"crm:import:run"}, allowed},
		{"contact export csv, catalogue key only", crmRouter, "POST", "/api/v1/contacts/export/csv", []string{"crm:contact:export"}, allowed},

		// --- crm: contact visibility has no FE caller, untouched ---
		{"contact visibility still requires contacts:write only", crmRouter, "PUT", "/api/v1/contacts/" + articleID + "/visibility", []string{"crm:contact:edit"}, denied},
		{"contact visibility with contacts:write", crmRouter, "PUT", "/api/v1/contacts/" + articleID + "/visibility", []string{"contacts:write"}, allowed},

		// --- crm: companies share the crm:contact:* fine keys (no dedicated company key) ---
		{"company list, catalogue contact key maps to it", crmRouter, "GET", "/api/v1/companies", []string{"crm:contact:read"}, allowed},
		{"company create, catalogue contact key maps to it", crmRouter, "POST", "/api/v1/companies", []string{"crm:contact:create"}, allowed},
		{"company delete, catalogue contact key maps to it", crmRouter, "DELETE", "/api/v1/companies/" + articleID, []string{"crm:contact:delete"}, allowed},

		// --- crm: pipeline stages have no FE gate call, untouched ---
		{"pipeline stage create still requires pipeline_stages:write only", crmRouter, "POST", "/api/v1/pipeline-stages", []string{"crm:pipeline:manage"}, denied},
		{"pipeline stage create with pipeline_stages:write", crmRouter, "POST", "/api/v1/pipeline-stages", []string{"pipeline_stages:write"}, allowed},

		// --- crm: deals ---
		{"deal list, catalogue key only", crmRouter, "GET", "/api/v1/deals", []string{"crm:deal:read"}, allowed},
		{"deal create, catalogue key only", crmRouter, "POST", "/api/v1/deals", []string{"crm:deal:create"}, allowed},
		{"deal create, edit key does not grant create", crmRouter, "POST", "/api/v1/deals", []string{"crm:deal:edit"}, denied},
		{"deal update, catalogue key only", crmRouter, "PUT", "/api/v1/deals/" + articleID, []string{"crm:deal:edit"}, allowed},
		{"deal stage move, catalogue edit key maps to it", crmRouter, "POST", "/api/v1/deals/" + articleID + "/stage", []string{"crm:deal:edit"}, allowed},
		{"deal delete, catalogue key only", crmRouter, "DELETE", "/api/v1/deals/" + articleID, []string{"crm:deal:delete"}, allowed},

		// --- crm: deal tags have no FE caller, untouched. HandleAddDealTags is a
		// permanent 501 stub that decodes the body before ever reaching a gRPC
		// client, so a passed guard surfaces as 400 (nil body) here, not 503.
		{"deal tags add still requires deals:write only", crmRouter, "POST", "/api/v1/deals/" + articleID + "/tags", []string{"crm:deal:edit"}, denied},
		{"deal tags add with deals:write", crmRouter, "POST", "/api/v1/deals/" + articleID + "/tags", []string{"deals:write"}, http.StatusBadRequest},

		// --- crm: activities share the crm:contact:* fine keys (nav-gated with contacts) ---
		{"activity list, catalogue contact key maps to it", crmRouter, "GET", "/api/v1/activities", []string{"crm:contact:read"}, allowed},
		{"activity create, catalogue contact key maps to it", crmRouter, "POST", "/api/v1/activities", []string{"crm:contact:edit"}, allowed},
		{"activity delete, catalogue contact key maps to it", crmRouter, "DELETE", "/api/v1/activities/" + articleID, []string{"crm:contact:delete"}, allowed},
		{"activity delete, edit key does not grant delete", crmRouter, "DELETE", "/api/v1/activities/" + articleID, []string{"crm:contact:edit"}, denied},

		// --- crm: reports are gated by crm:deal:read in KontakteLayout.tsx, not contact:read ---
		{"pipeline report, catalogue deal key only", crmRouter, "GET", "/api/v1/reports/pipeline", []string{"crm:deal:read"}, allowed},
		{"pipeline report, contact key does not grant it", crmRouter, "GET", "/api/v1/reports/pipeline", []string{"crm:contact:read"}, denied},

		// --- finance: invoices ---
		{"invoice list, legacy key only", bizRouter, "GET", "/api/v1/finance/invoices", []string{"finance:read"}, allowed},
		{"invoice list, catalogue key only", bizRouter, "GET", "/api/v1/finance/invoices", []string{"finance:invoice:read"}, allowed},
		{"invoice list, neither key", bizRouter, "GET", "/api/v1/finance/invoices", []string{"crm:deal:read"}, denied},
		{"invoice create, catalogue key only", bizRouter, "POST", "/api/v1/finance/invoices", []string{"finance:invoice:create"}, allowed},
		{"invoice create, edit key does not grant create", bizRouter, "POST", "/api/v1/finance/invoices", []string{"finance:invoice:edit"}, denied},
		{"invoice update, catalogue key only", bizRouter, "PUT", "/api/v1/finance/invoices/" + articleID, []string{"finance:invoice:edit"}, allowed},
		{"invoice send, catalogue key only", bizRouter, "POST", "/api/v1/finance/invoices/" + articleID + "/send", []string{"finance:invoice:send"}, allowed},
		{"invoice pay, catalogue edit key maps to it", bizRouter, "POST", "/api/v1/finance/invoices/" + articleID + "/pay", []string{"finance:invoice:edit"}, allowed},
		{"invoice cancel, catalogue delete key maps to it", bizRouter, "POST", "/api/v1/finance/invoices/" + articleID + "/cancel", []string{"finance:invoice:delete"}, allowed},
		{"invoice cancel, edit key does not grant it", bizRouter, "POST", "/api/v1/finance/invoices/" + articleID + "/cancel", []string{"finance:invoice:edit"}, denied},

		// --- finance: invoice import/lock/validate-number have no FE caller, untouched ---
		{"invoice import still requires finance:write only", bizRouter, "POST", "/api/v1/finance/invoices/import", []string{"finance:invoice:create"}, denied},
		{"invoice import with finance:write", bizRouter, "POST", "/api/v1/finance/invoices/import", []string{"finance:write"}, allowed},

		// --- finance: recurring invoices share the finance:invoice:* fine keys ---
		{"recurring list, catalogue key only", bizRouter, "GET", "/api/v1/finance/recurring", []string{"finance:invoice:read"}, allowed},
		{"recurring generate, catalogue create key maps to it", bizRouter, "POST", "/api/v1/finance/recurring/" + articleID + "/generate", []string{"finance:invoice:create"}, allowed},
		{"recurring pause, catalogue edit key maps to it", bizRouter, "POST", "/api/v1/finance/recurring/" + articleID + "/pause", []string{"finance:invoice:edit"}, allowed},
		{"recurring delete, catalogue key only", bizRouter, "DELETE", "/api/v1/finance/recurring/" + articleID, []string{"finance:invoice:delete"}, allowed},

		// --- finance: credit notes create is reachable by either create or delete (storno) ---
		{"credit note create, catalogue create key only", bizRouter, "POST", "/api/v1/finance/credit-notes", []string{"finance:invoice:create"}, allowed},
		{"credit note create, catalogue delete key also grants it (storno)", bizRouter, "POST", "/api/v1/finance/credit-notes", []string{"finance:invoice:delete"}, allowed},
		{"credit note create, read key does not grant it", bizRouter, "POST", "/api/v1/finance/credit-notes", []string{"finance:invoice:read"}, denied},
		{"credit note send, catalogue key only", bizRouter, "POST", "/api/v1/finance/credit-notes/" + articleID + "/send", []string{"finance:invoice:send"}, allowed},

		// --- finance: quotes ---
		{"quote list, catalogue key only", bizRouter, "GET", "/api/v1/finance/quotes", []string{"finance:quote:read"}, allowed},
		{"quote create, catalogue key only", bizRouter, "POST", "/api/v1/finance/quotes", []string{"finance:quote:create"}, allowed},
		{"quote edit, catalogue create key maps to it (no separate edit key)", bizRouter, "PUT", "/api/v1/finance/quotes/" + articleID, []string{"finance:quote:create"}, allowed},
		{"quote accept, catalogue create key maps to it", bizRouter, "POST", "/api/v1/finance/quotes/" + articleID + "/accept", []string{"finance:quote:create"}, allowed},
		{"quote send, catalogue key only", bizRouter, "POST", "/api/v1/finance/quotes/" + articleID + "/send", []string{"finance:quote:send"}, allowed},
		{"quote send, create key does not grant send", bizRouter, "POST", "/api/v1/finance/quotes/" + articleID + "/send", []string{"finance:quote:create"}, denied},
		{"quote convert, catalogue invoice-create key maps to it", bizRouter, "POST", "/api/v1/finance/quotes/" + articleID + "/convert", []string{"finance:invoice:create"}, allowed},
		{"quote convert, quote-create key does not grant it", bizRouter, "POST", "/api/v1/finance/quotes/" + articleID + "/convert", []string{"finance:quote:create"}, denied},
		{"deal-to-quote create, catalogue quote-create key maps to it", bizRouter, "POST", "/api/v1/finance/deals/" + articleID + "/quote", []string{"finance:quote:create"}, allowed},

		// --- finance: quote delete has no catalogue key, untouched ---
		{"quote delete still requires finance:delete only", bizRouter, "DELETE", "/api/v1/finance/quotes/" + articleID, []string{"finance:quote:create"}, denied},
		{"quote delete with finance:delete", bizRouter, "DELETE", "/api/v1/finance/quotes/" + articleID, []string{"finance:delete"}, allowed},

		// --- finance: dunning ---
		{"dunning list, catalogue key only", bizRouter, "GET", "/api/v1/finance/dunning", []string{"finance:dunning:run"}, allowed},
		{"dunning detect, catalogue key only", bizRouter, "POST", "/api/v1/finance/dunning/detect", []string{"finance:dunning:run"}, allowed},
		{"dunning config get, catalogue settings key only", bizRouter, "GET", "/api/v1/finance/dunning/config", []string{"finance:settings:manage"}, allowed},
		{"dunning config put, catalogue settings key only", bizRouter, "PUT", "/api/v1/finance/dunning/config", []string{"finance:settings:manage"}, allowed},

		// --- finance: dunning status/notice (GoBD gaps) have no FE caller, untouched ---
		{"dunning status still requires finance:admin only", bizRouter, "PUT", "/api/v1/finance/dunning/" + articleID + "/status", []string{"finance:settings:manage"}, denied},
		{"dunning status with finance:admin", bizRouter, "PUT", "/api/v1/finance/dunning/" + articleID + "/status", []string{"finance:admin"}, allowed},

		// --- finance: export ---
		{"datev export, catalogue key only", bizRouter, "POST", "/api/v1/finance/export/datev", []string{"finance:export:run"}, allowed},

		// --- finance: gobd export has no FE caller, untouched ---
		{"gobd export still requires finance:admin only", bizRouter, "POST", "/api/v1/finance/export/gobd", []string{"finance:export:run"}, denied},
		{"gobd export with finance:admin", bizRouter, "POST", "/api/v1/finance/export/gobd", []string{"finance:admin"}, allowed},

		// --- finance: company settings, bank-statements, incoming-invoices, open-items are
		// FE-mock-only today (BankingWidget/TransactionsTab/ExpensesTab all read from a Zustand
		// mock store, see JOURNAL.md) — legacy-only, untouched ---
		{"company settings update still requires finance:admin only", bizRouter, "PUT", "/api/v1/finance/settings", []string{"finance:settings:manage"}, denied},
		{"company settings update with finance:admin", bizRouter, "PUT", "/api/v1/finance/settings", []string{"finance:admin"}, allowed},
		{"bank statements list still requires finance:read only", bizRouter, "GET", "/api/v1/finance/bank-statements", []string{"finance:incoming:book"}, denied},
		{"bank statements list with finance:read", bizRouter, "GET", "/api/v1/finance/bank-statements", []string{"finance:read"}, allowed},
		{"incoming invoices list still requires finance:read only", bizRouter, "GET", "/api/v1/finance/incoming-invoices", []string{"finance:incoming:review"}, denied},
		{"incoming invoices list with finance:read", bizRouter, "GET", "/api/v1/finance/incoming-invoices", []string{"finance:read"}, allowed},
		{"open items list still requires finance:read only", bizRouter, "GET", "/api/v1/finance/open-items", []string{"finance:amounts:view"}, denied},
		{"open items list with finance:read", bizRouter, "GET", "/api/v1/finance/open-items", []string{"finance:read"}, allowed},

		// --- inventar: items ---
		{"item create, legacy key only", inventarRouter, "POST", "/api/v1/inventar/items", []string{"inventar:item:write"}, allowed},
		{"item create, catalogue key only", inventarRouter, "POST", "/api/v1/inventar/items", []string{"inventar:item:create"}, allowed},
		{"item create, edit key does not grant create", inventarRouter, "POST", "/api/v1/inventar/items", []string{"inventar:item:edit"}, denied},
		{"item edit, catalogue key only", inventarRouter, "PATCH", "/api/v1/inventar/items/" + articleID, []string{"inventar:item:edit"}, allowed},
		{"item delete, catalogue key only", inventarRouter, "DELETE", "/api/v1/inventar/items/" + articleID, []string{"inventar:item:delete"}, allowed},
		{"item delete, edit key does not grant delete", inventarRouter, "DELETE", "/api/v1/inventar/items/" + articleID, []string{"inventar:item:edit"}, denied},

		// --- inventar: movements — adjust is a distinct fine switch from recording a normal movement ---
		{"stock adjust, legacy item:write still grants it", inventarRouter, "POST", "/api/v1/inventar/items/" + articleID + "/adjust", []string{"inventar:item:write"}, allowed},
		{"stock adjust, catalogue movement:adjust key only", inventarRouter, "POST", "/api/v1/inventar/items/" + articleID + "/adjust", []string{"inventar:movement:adjust"}, allowed},
		{"stock adjust, movement:create does not grant adjust", inventarRouter, "POST", "/api/v1/inventar/items/" + articleID + "/adjust", []string{"inventar:movement:create"}, denied},
		{"movement record, catalogue key only", inventarRouter, "POST", "/api/v1/inventar/items/" + articleID + "/movements", []string{"inventar:movement:create"}, allowed},

		// --- inventar: attachments ---
		{"attachment create, catalogue key only", inventarRouter, "POST", "/api/v1/inventar/items/" + articleID + "/attachments", []string{"inventar:attachment:manage"}, allowed},
		{"attachment delete, catalogue key only", inventarRouter, "DELETE", "/api/v1/inventar/attachments/" + articleID, []string{"inventar:attachment:manage"}, allowed},

		// --- inventar: inventur — count backs both the counting UI and the finish-counting
		// (open->review) status transition; book is the separate stock-write step ---
		{"inventur create, catalogue key only", inventarRouter, "POST", "/api/v1/inventar/inventur", []string{"inventar:inventur:create"}, allowed},
		{"inventur status update, catalogue count key only", inventarRouter, "PATCH", "/api/v1/inventar/inventur/" + articleID + "/status", []string{"inventar:inventur:count"}, allowed},
		{"inventur count upsert, catalogue count key only", inventarRouter, "POST", "/api/v1/inventar/inventur/" + articleID + "/counts", []string{"inventar:inventur:count"}, allowed},
		{"inventur count upsert, book key does not grant it", inventarRouter, "POST", "/api/v1/inventar/inventur/" + articleID + "/counts", []string{"inventar:inventur:book"}, denied},
		{"inventur book differences, catalogue key only", inventarRouter, "POST", "/api/v1/inventar/inventur/" + articleID + "/book", []string{"inventar:inventur:book"}, allowed},
		{"inventur book differences, count key does not grant it", inventarRouter, "POST", "/api/v1/inventar/inventur/" + articleID + "/book", []string{"inventar:inventur:count"}, denied},

		// --- inventar: transfer, warnings, session deletion have no FE caller, untouched ---
		{"stock transfer still requires item:write only", inventarRouter, "POST", "/api/v1/inventar/transfer", []string{"inventar:movement:adjust"}, denied},
		{"stock transfer with item:write", inventarRouter, "POST", "/api/v1/inventar/transfer", []string{"inventar:item:write"}, allowed},
		{"warning create still requires warning:write only", inventarRouter, "POST", "/api/v1/inventar/warnings", []string{"inventar:item:create"}, denied},
		{"warning create with warning:write", inventarRouter, "POST", "/api/v1/inventar/warnings", []string{"inventar:warning:write"}, allowed},
		{"inventur session delete still requires inventur:write only", inventarRouter, "DELETE", "/api/v1/inventar/inventur/" + articleID, []string{"inventar:inventur:count"}, denied},
		{"inventur session delete with inventur:write", inventarRouter, "DELETE", "/api/v1/inventar/inventur/" + articleID, []string{"inventar:inventur:write"}, allowed},

		// --- einkauf: purchase orders ---
		{"po create, legacy key only", einkaufRouter, "POST", "/api/v1/einkauf/pos", []string{"einkauf:po:write"}, allowed},
		{"po create, catalogue key only", einkaufRouter, "POST", "/api/v1/einkauf/pos", []string{"einkauf:po:create"}, allowed},
		{"po create, edit key does not grant create", einkaufRouter, "POST", "/api/v1/einkauf/pos", []string{"einkauf:po:edit"}, denied},
		{"po edit, catalogue key only", einkaufRouter, "PATCH", "/api/v1/einkauf/pos/" + articleID, []string{"einkauf:po:edit"}, allowed},
		{"po delete, catalogue key only", einkaufRouter, "DELETE", "/api/v1/einkauf/pos/" + articleID, []string{"einkauf:po:delete"}, allowed},
		{"po delete, edit key does not grant delete", einkaufRouter, "DELETE", "/api/v1/einkauf/pos/" + articleID, []string{"einkauf:po:edit"}, denied},

		// --- einkauf: submit backs both "send" and the approval-banner "approve" button ---
		{"po submit, catalogue send key only", einkaufRouter, "POST", "/api/v1/einkauf/pos/" + articleID + "/submit", []string{"einkauf:po:send"}, allowed},
		{"po submit, catalogue approve key also grants it", einkaufRouter, "POST", "/api/v1/einkauf/pos/" + articleID + "/submit", []string{"einkauf:po:approve"}, allowed},
		{"po submit, cancel key does not grant it", einkaufRouter, "POST", "/api/v1/einkauf/pos/" + articleID + "/submit", []string{"einkauf:po:cancel"}, denied},
		{"po cancel, catalogue key only", einkaufRouter, "POST", "/api/v1/einkauf/pos/" + articleID + "/cancel", []string{"einkauf:po:cancel"}, allowed},
		{"po receive, catalogue key only", einkaufRouter, "POST", "/api/v1/einkauf/pos/" + articleID + "/receive", []string{"einkauf:po:receive"}, allowed},
		{"po partial-receive, catalogue key only", einkaufRouter, "POST", "/api/v1/einkauf/pos/" + articleID + "/partial-receive", []string{"einkauf:po:receive"}, allowed},

		// --- einkauf: suppliers — "deactivate" in EinkaufPage.tsx is a soft label over DeleteSupplier ---
		{"supplier create, catalogue key only", einkaufRouter, "POST", "/api/v1/einkauf/suppliers", []string{"einkauf:supplier:create"}, allowed},
		{"supplier edit, catalogue key only", einkaufRouter, "PATCH", "/api/v1/einkauf/suppliers/" + articleID, []string{"einkauf:supplier:edit"}, allowed},
		{"supplier deactivate, catalogue key only", einkaufRouter, "DELETE", "/api/v1/einkauf/suppliers/" + articleID, []string{"einkauf:supplier:deactivate"}, allowed},
		{"supplier deactivate, edit key does not grant it", einkaufRouter, "DELETE", "/api/v1/einkauf/suppliers/" + articleID, []string{"einkauf:supplier:edit"}, denied},

		// --- einkauf: rating and contract call cross into a different resource than their legacy guard ---
		{"rating create, legacy supplier:write still grants it", einkaufRouter, "POST", "/api/v1/einkauf/suppliers/" + articleID + "/ratings", []string{"einkauf:supplier:write"}, allowed},
		{"rating create, catalogue key only", einkaufRouter, "POST", "/api/v1/einkauf/suppliers/" + articleID + "/ratings", []string{"einkauf:rating:create"}, allowed},
		{"contract call create, legacy contract:write still grants it", einkaufRouter, "POST", "/api/v1/einkauf/contracts/" + articleID + "/calls", []string{"einkauf:contract:write"}, allowed},
		{"contract call create, catalogue key only", einkaufRouter, "POST", "/api/v1/einkauf/contracts/" + articleID + "/calls", []string{"einkauf:contract:call"}, allowed},

		// --- einkauf: PO lines, catalog CRUD, framework contract CRUD, rating deletion have no
		// FE caller, untouched ---
		{"po line add still requires line:write only", einkaufRouter, "POST", "/api/v1/einkauf/pos/" + articleID + "/lines", []string{"einkauf:po:edit"}, denied},
		{"po line add with line:write", einkaufRouter, "POST", "/api/v1/einkauf/pos/" + articleID + "/lines", []string{"einkauf:line:write"}, allowed},
		{"catalog item create still requires catalog:write only", einkaufRouter, "POST", "/api/v1/einkauf/catalog", []string{"einkauf:catalog:read"}, denied},
		{"catalog item create with catalog:write", einkaufRouter, "POST", "/api/v1/einkauf/catalog", []string{"einkauf:catalog:write"}, allowed},
		{"framework contract create still requires contract:write only", einkaufRouter, "POST", "/api/v1/einkauf/contracts", []string{"einkauf:contract:call"}, denied},
		{"framework contract create with contract:write", einkaufRouter, "POST", "/api/v1/einkauf/contracts", []string{"einkauf:contract:write"}, allowed},
		{"rating delete still requires supplier:write only", einkaufRouter, "DELETE", "/api/v1/einkauf/suppliers/" + articleID + "/ratings/" + articleID, []string{"einkauf:rating:create"}, denied},
		{"rating delete with supplier:write", einkaufRouter, "DELETE", "/api/v1/einkauf/suppliers/" + articleID + "/ratings/" + articleID, []string{"einkauf:supplier:write"}, allowed},

		// --- produktion: orders ---
		{"order create, legacy key only", produktionRouter, "POST", "/api/v1/produktion/orders", []string{"produktion:order:write"}, allowed},
		{"order create, catalogue key only", produktionRouter, "POST", "/api/v1/produktion/orders", []string{"produktion:order:create"}, allowed},
		{"order create, edit key does not grant create", produktionRouter, "POST", "/api/v1/produktion/orders", []string{"produktion:order:edit"}, denied},
		{"order edit, catalogue key only", produktionRouter, "PATCH", "/api/v1/produktion/orders/" + articleID, []string{"produktion:order:edit"}, allowed},
		{"order delete, catalogue key only", produktionRouter, "DELETE", "/api/v1/produktion/orders/" + articleID, []string{"produktion:order:delete"}, allowed},
		{"order start, catalogue key only", produktionRouter, "POST", "/api/v1/produktion/orders/" + articleID + "/start", []string{"produktion:order:start"}, allowed},
		{"order start, complete key does not grant start", produktionRouter, "POST", "/api/v1/produktion/orders/" + articleID + "/start", []string{"produktion:order:complete"}, denied},
		{"order complete, catalogue key only", produktionRouter, "POST", "/api/v1/produktion/orders/" + articleID + "/complete", []string{"produktion:order:complete"}, allowed},
		{"order cancel, catalogue key only", produktionRouter, "POST", "/api/v1/produktion/orders/" + articleID + "/cancel", []string{"produktion:order:cancel"}, allowed},

		// --- produktion: BOMs, machines, quality, work steps ---
		{"bom create, catalogue key only", produktionRouter, "POST", "/api/v1/produktion/boms", []string{"produktion:bom:create"}, allowed},
		{"bom edit, catalogue key only", produktionRouter, "PATCH", "/api/v1/produktion/boms/" + articleID, []string{"produktion:bom:edit"}, allowed},
		{"bom delete still requires bom:write only", produktionRouter, "DELETE", "/api/v1/produktion/boms/" + articleID, []string{"produktion:bom:edit"}, denied},
		{"bom delete with bom:write", produktionRouter, "DELETE", "/api/v1/produktion/boms/" + articleID, []string{"produktion:bom:write"}, allowed},
		{"machine create, catalogue manage key only", produktionRouter, "POST", "/api/v1/produktion/machines", []string{"produktion:machine:manage"}, allowed},
		{"machine edit, catalogue manage key only", produktionRouter, "PATCH", "/api/v1/produktion/machines/" + articleID, []string{"produktion:machine:manage"}, allowed},
		{"machine delete still requires machine:write only", produktionRouter, "DELETE", "/api/v1/produktion/machines/" + articleID, []string{"produktion:machine:manage"}, denied},
		{"machine delete with machine:write", produktionRouter, "DELETE", "/api/v1/produktion/machines/" + articleID, []string{"produktion:machine:write"}, allowed},
		{"quality check create, catalogue key only", produktionRouter, "POST", "/api/v1/produktion/quality", []string{"produktion:quality:create"}, allowed},
		{"workstep edit, catalogue key only", produktionRouter, "PATCH", "/api/v1/produktion/orders/" + articleID + "/steps/" + articleID, []string{"produktion:workstep:edit"}, allowed},
		{"workstep create still requires workstep:write only", produktionRouter, "POST", "/api/v1/produktion/orders/" + articleID + "/steps", []string{"produktion:workstep:edit"}, denied},
		{"workstep create with workstep:write", produktionRouter, "POST", "/api/v1/produktion/orders/" + articleID + "/steps", []string{"produktion:workstep:write"}, allowed},

		// --- produktion: machine bookings and production plans have no FE caller, untouched ---
		{"machine booking create still requires booking:write only", produktionRouter, "POST", "/api/v1/produktion/bookings", []string{"produktion:machine:manage"}, denied},
		{"machine booking create with booking:write", produktionRouter, "POST", "/api/v1/produktion/bookings", []string{"produktion:booking:write"}, allowed},

		// --- vertraege: contracts ---
		{"contract create, catalogue key only", vertraegeRouter, "POST", "/api/v1/vertraege/contracts", []string{"vertraege:contract:create"}, allowed},
		{"contract create, legacy key only", vertraegeRouter, "POST", "/api/v1/vertraege/contracts", []string{"vertraege:contract:write"}, allowed},
		{"contract edit, catalogue edit key only", vertraegeRouter, "PATCH", "/api/v1/vertraege/contracts/" + articleID, []string{"vertraege:contract:edit"}, allowed},
		{"contract edit (termination dialog), catalogue terminate key also grants it", vertraegeRouter, "PATCH", "/api/v1/vertraege/contracts/" + articleID, []string{"vertraege:contract:terminate"}, allowed},
		{"contract edit, delete key does not grant it", vertraegeRouter, "PATCH", "/api/v1/vertraege/contracts/" + articleID, []string{"vertraege:contract:delete"}, denied},
		{"contract delete, catalogue key only", vertraegeRouter, "DELETE", "/api/v1/vertraege/contracts/" + articleID, []string{"vertraege:contract:delete"}, allowed},
		{"contract delete, edit key does not grant it", vertraegeRouter, "DELETE", "/api/v1/vertraege/contracts/" + articleID, []string{"vertraege:contract:edit"}, denied},

		// --- vertraege: parties, reminders, export, signature have no catalogue key, untouched ---
		{"party add still requires party:write only", vertraegeRouter, "POST", "/api/v1/vertraege/contracts/" + articleID + "/parties", []string{"vertraege:contract:edit"}, denied},
		{"party add with party:write", vertraegeRouter, "POST", "/api/v1/vertraege/contracts/" + articleID + "/parties", []string{"vertraege:party:write"}, allowed},
		{"reminder create still requires reminder:write only", vertraegeRouter, "POST", "/api/v1/vertraege/contracts/" + articleID + "/reminders", []string{"vertraege:contract:edit"}, denied},
		{"reminder create with reminder:write", vertraegeRouter, "POST", "/api/v1/vertraege/contracts/" + articleID + "/reminders", []string{"vertraege:reminder:write"}, allowed},

		// --- schichten: shifts (create is also reachable via assignment:manage — see route_schichten.go) ---
		{"shift create, legacy key only", schichtenRouter, "POST", "/api/v1/schichten/shifts", []string{"schichten:shift:write"}, allowed},
		{"shift create, catalogue assignment:manage key also grants it", schichtenRouter, "POST", "/api/v1/schichten/shifts", []string{"schichten:assignment:manage"}, allowed},
		{"shift create, template:manage does not grant it", schichtenRouter, "POST", "/api/v1/schichten/shifts", []string{"schichten:template:manage"}, denied},
		{"shift publish, catalogue key only", schichtenRouter, "POST", "/api/v1/schichten/shifts/publish", []string{"schichten:shift:publish"}, allowed},
		{"shift publish, assignment:manage does not grant it", schichtenRouter, "POST", "/api/v1/schichten/shifts/publish", []string{"schichten:assignment:manage"}, denied},

		// --- schichten: shift update/delete have no FE caller, untouched ---
		{"shift update still requires shift:write only", schichtenRouter, "PATCH", "/api/v1/schichten/shifts/" + articleID, []string{"schichten:shift:publish"}, denied},
		{"shift update with shift:write", schichtenRouter, "PATCH", "/api/v1/schichten/shifts/" + articleID, []string{"schichten:shift:write"}, allowed},

		// --- schichten: assignments ---
		{"assignment create, catalogue key only", schichtenRouter, "POST", "/api/v1/schichten/shifts/" + articleID + "/assignments", []string{"schichten:assignment:manage"}, allowed},
		{"assignment delete, catalogue key only", schichtenRouter, "DELETE", "/api/v1/schichten/shifts/" + articleID + "/assignments/" + articleID, []string{"schichten:assignment:manage"}, allowed},
		{"assignment delete, denied without either key", schichtenRouter, "DELETE", "/api/v1/schichten/shifts/" + articleID + "/assignments/" + articleID, []string{"schichten:shift:publish"}, denied},

		// --- schichten: templates (apply is also reachable via template:manage only, see route_schichten.go) ---
		{"template create, catalogue key only", schichtenRouter, "POST", "/api/v1/schichten/templates", []string{"schichten:template:manage"}, allowed},
		{"template edit, catalogue key only", schichtenRouter, "PATCH", "/api/v1/schichten/templates/" + articleID, []string{"schichten:template:manage"}, allowed},
		{"template delete, catalogue key only", schichtenRouter, "DELETE", "/api/v1/schichten/templates/" + articleID, []string{"schichten:template:manage"}, allowed},
		{"template apply, catalogue template:manage key only", schichtenRouter, "POST", "/api/v1/schichten/templates/" + articleID + "/apply", []string{"schichten:template:manage"}, allowed},
		{"template apply, legacy shift:write also grants it", schichtenRouter, "POST", "/api/v1/schichten/templates/" + articleID + "/apply", []string{"schichten:shift:write"}, allowed},
		{"template apply, assignment:manage does not grant it", schichtenRouter, "POST", "/api/v1/schichten/templates/" + articleID + "/apply", []string{"schichten:assignment:manage"}, denied},

		// --- schichten: swap requests already use fine keys directly, untouched ---
		{"swap create requires the fine create key", schichtenRouter, "POST", "/api/v1/schichten/swap-requests", []string{"schichten:swap:read"}, denied},
		{"swap create with fine create key", schichtenRouter, "POST", "/api/v1/schichten/swap-requests", []string{"schichten:swap:create"}, allowed},
		{"swap approve with fine approve key", schichtenRouter, "POST", "/api/v1/schichten/swap-requests/" + articleID + "/approve", []string{"schichten:swap:approve"}, allowed},

		// --- fuhrpark: vehicles ---
		{"vehicle create, legacy key only", fuhrparkRouter, "POST", "/api/v1/fuhrpark/vehicles", []string{"fuhrpark:vehicle:write"}, allowed},
		{"vehicle create, catalogue manage key only", fuhrparkRouter, "POST", "/api/v1/fuhrpark/vehicles", []string{"fuhrpark:vehicle:manage"}, allowed},
		{"vehicle create, denied without either key", fuhrparkRouter, "POST", "/api/v1/fuhrpark/vehicles", []string{"fuhrpark:service:create"}, denied},

		// --- fuhrpark: vehicle update/delete have no FE caller, untouched ---
		{"vehicle update still requires vehicle:write only", fuhrparkRouter, "PATCH", "/api/v1/fuhrpark/vehicles/" + articleID, []string{"fuhrpark:vehicle:manage"}, denied},
		{"vehicle update with vehicle:write", fuhrparkRouter, "PATCH", "/api/v1/fuhrpark/vehicles/" + articleID, []string{"fuhrpark:vehicle:write"}, allowed},

		// --- fuhrpark: service/damage/fuel/trip create ---
		{"service schedule, catalogue key only", fuhrparkRouter, "POST", "/api/v1/fuhrpark/vehicles/" + articleID + "/services", []string{"fuhrpark:service:create"}, allowed},
		{"damage report, catalogue key only", fuhrparkRouter, "POST", "/api/v1/fuhrpark/vehicles/" + articleID + "/damages", []string{"fuhrpark:damage:create"}, allowed},
		{"fuel log create, catalogue key only", fuhrparkRouter, "POST", "/api/v1/fuhrpark/vehicles/" + articleID + "/fuel-logs", []string{"fuhrpark:fuel:create"}, allowed},
		{"trip log create, catalogue key only", fuhrparkRouter, "POST", "/api/v1/fuhrpark/vehicles/" + articleID + "/trip-logs", []string{"fuhrpark:trip:create"}, allowed},

		// --- fuhrpark: service/damage/fuel/trip update+delete have no catalogue key, untouched ---
		{"service update still requires service:write only", fuhrparkRouter, "PATCH", "/api/v1/fuhrpark/services/" + articleID, []string{"fuhrpark:service:create"}, denied},
		{"service update with service:write", fuhrparkRouter, "PATCH", "/api/v1/fuhrpark/services/" + articleID, []string{"fuhrpark:service:write"}, allowed},

		// --- vermietung: objects ---
		{"object create, legacy key only", vermietungRouter, "POST", "/api/v1/vermietung/objects", []string{"vermietung:object:write"}, allowed},
		{"object create, catalogue key only", vermietungRouter, "POST", "/api/v1/vermietung/objects", []string{"vermietung:object:create"}, allowed},
		{"object create, edit key does not grant create", vermietungRouter, "POST", "/api/v1/vermietung/objects", []string{"vermietung:object:edit"}, denied},
		{"object edit, catalogue key only", vermietungRouter, "PATCH", "/api/v1/vermietung/objects/" + articleID, []string{"vermietung:object:edit"}, allowed},
		{"object delete, catalogue key only", vermietungRouter, "DELETE", "/api/v1/vermietung/objects/" + articleID, []string{"vermietung:object:delete"}, allowed},
		{"object delete, edit key does not grant delete", vermietungRouter, "DELETE", "/api/v1/vermietung/objects/" + articleID, []string{"vermietung:object:edit"}, denied},

		// --- vermietung: rentals ---
		{"rental create, catalogue key only", vermietungRouter, "POST", "/api/v1/vermietung/rentals", []string{"vermietung:rental:create"}, allowed},
		{"rental edit, catalogue key only", vermietungRouter, "PATCH", "/api/v1/vermietung/rentals/" + articleID, []string{"vermietung:rental:edit"}, allowed},
		{"rental cancel (stornieren), catalogue key only", vermietungRouter, "DELETE", "/api/v1/vermietung/rentals/" + articleID, []string{"vermietung:rental:cancel"}, allowed},
		{"rental cancel, edit key does not grant it", vermietungRouter, "DELETE", "/api/v1/vermietung/rentals/" + articleID, []string{"vermietung:rental:edit"}, denied},
		{"rental start (handover), catalogue key only", vermietungRouter, "POST", "/api/v1/vermietung/rentals/" + articleID + "/start", []string{"vermietung:rental:handover"}, allowed},
		{"rental end (handover), catalogue key only", vermietungRouter, "POST", "/api/v1/vermietung/rentals/" + articleID + "/end", []string{"vermietung:rental:handover"}, allowed},
		{"rental start, edit key does not grant it", vermietungRouter, "POST", "/api/v1/vermietung/rentals/" + articleID + "/start", []string{"vermietung:rental:edit"}, denied},

		// --- vermietung: signature has no catalogue key, untouched ---
		{"rental signature still requires rental:write only", vermietungRouter, "PUT", "/api/v1/vermietung/rentals/" + articleID + "/signature", []string{"vermietung:rental:edit"}, denied},
		{"rental signature with rental:write", vermietungRouter, "PUT", "/api/v1/vermietung/rentals/" + articleID + "/signature", []string{"vermietung:rental:write"}, allowed},

		// --- vermietung: inspections ---
		{"inspection create, catalogue key only", vermietungRouter, "POST", "/api/v1/vermietung/rentals/" + articleID + "/inspections", []string{"vermietung:inspection:create"}, allowed},

		// --- rapporte: reports ---
		{"report create, legacy key only", rapporteRouter, "POST", "/api/v1/rapporte/reports", []string{"rapporte:report:write"}, allowed},
		{"report create, catalogue key only", rapporteRouter, "POST", "/api/v1/rapporte/reports", []string{"rapporte:report:create"}, allowed},
		{"report create, edit key does not grant create", rapporteRouter, "POST", "/api/v1/rapporte/reports", []string{"rapporte:report:edit"}, denied},
		{"report edit, catalogue key only", rapporteRouter, "PATCH", "/api/v1/rapporte/reports/" + articleID, []string{"rapporte:report:edit"}, allowed},
		{"report delete, catalogue key only", rapporteRouter, "DELETE", "/api/v1/rapporte/reports/" + articleID, []string{"rapporte:report:delete"}, allowed},
		{"report delete, edit key does not grant delete", rapporteRouter, "DELETE", "/api/v1/rapporte/reports/" + articleID, []string{"rapporte:report:edit"}, denied},

		// --- rapporte: approve/reject already use the fine key directly, untouched ---
		{"report approve requires the fine approve key", rapporteRouter, "POST", "/api/v1/rapporte/reports/" + articleID + "/approve", []string{"rapporte:report:edit"}, denied},
		{"report approve with fine approve key", rapporteRouter, "POST", "/api/v1/rapporte/reports/" + articleID + "/approve", []string{"rapporte:report:approve"}, allowed},

		// --- rapporte: submit/lines/attachments/signature have no catalogue key, untouched ---
		{"report submit still requires report:write only", rapporteRouter, "POST", "/api/v1/rapporte/reports/" + articleID + "/submit", []string{"rapporte:report:edit"}, denied},
		{"report submit with report:write", rapporteRouter, "POST", "/api/v1/rapporte/reports/" + articleID + "/submit", []string{"rapporte:report:write"}, allowed},

		// --- rapporte: measurements have no FE caller (local mock store), untouched ---
		{"measurement create still requires measurement:write only", rapporteRouter, "POST", "/api/v1/rapporte/measurements", []string{"rapporte:measurement:manage"}, denied},
		{"measurement create with measurement:write", rapporteRouter, "POST", "/api/v1/rapporte/measurements", []string{"rapporte:measurement:write"}, allowed},

		// --- dialer: campaigns (legacy resource already lives in module:subject
		// shape, so read/calls/agent/outcomes concatenate to their catalogue key
		// 1:1 already -- only the "write" vs. "manage" verb mismatch needs wiring) ---
		{"campaign create, legacy key only", dialerRouter, "POST", "/api/v1/dialer/campaigns/", []string{"dialer:campaigns:write"}, allowed},
		{"campaign create, catalogue manage key only", dialerRouter, "POST", "/api/v1/dialer/campaigns/", []string{"dialer:campaigns:manage"}, allowed},
		{"campaign create, denied without either key", dialerRouter, "POST", "/api/v1/dialer/campaigns/", []string{"dialer:calls:write"}, denied},
		{"campaign update, legacy key only", dialerRouter, "PUT", "/api/v1/dialer/campaigns/" + articleID, []string{"dialer:campaigns:write"}, allowed},
		{"campaign update, catalogue manage key only", dialerRouter, "PUT", "/api/v1/dialer/campaigns/" + articleID, []string{"dialer:campaigns:manage"}, allowed},

		// --- dialer: skip/requeue are reachable from two FE surfaces sharing one
		// handler (DialerWorkspacePage's live queue, calls:write; and
		// ContactQueueTable.tsx in the supervisor's campaign-detail view, gated
		// on campaigns:manage) -- both keys must open it, next-contact stays
		// calls:write only since only the workspace calls it ---
		{"contact skip, legacy calls:write", dialerRouter, "POST", "/api/v1/dialer/campaigns/" + articleID + "/contacts/" + articleID + "/skip", []string{"dialer:calls:write"}, allowed},
		{"contact skip, campaigns:manage also grants it", dialerRouter, "POST", "/api/v1/dialer/campaigns/" + articleID + "/contacts/" + articleID + "/skip", []string{"dialer:campaigns:manage"}, allowed},
		{"contact skip, denied without either key", dialerRouter, "POST", "/api/v1/dialer/campaigns/" + articleID + "/contacts/" + articleID + "/skip", []string{"dialer:outcomes:manage"}, denied},
		{"contact requeue, campaigns:manage also grants it", dialerRouter, "POST", "/api/v1/dialer/campaigns/" + articleID + "/contacts/" + articleID + "/requeue", []string{"dialer:campaigns:manage"}, allowed},
		{"get next contact still requires calls:write only", dialerRouter, "GET", "/api/v1/dialer/campaigns/" + articleID + "/next", []string{"dialer:campaigns:manage"}, denied},
		{"get next contact with calls:write", dialerRouter, "GET", "/api/v1/dialer/campaigns/" + articleID + "/next", []string{"dialer:calls:write"}, allowed},

		// --- berichte: definitions ---
		{"definition create, legacy key only", berichteRouter, "POST", "/api/v1/berichte/definitions/", []string{"berichte:reports:write"}, allowed},
		{"definition create, catalogue key only", berichteRouter, "POST", "/api/v1/berichte/definitions/", []string{"berichte:reports:create"}, allowed},
		{"definition create, edit key does not grant create", berichteRouter, "POST", "/api/v1/berichte/definitions/", []string{"berichte:reports:edit"}, denied},
		{"definition edit, catalogue key only", berichteRouter, "PATCH", "/api/v1/berichte/definitions/" + articleID, []string{"berichte:reports:edit"}, allowed},
		{"definition delete, catalogue key only", berichteRouter, "DELETE", "/api/v1/berichte/definitions/" + articleID, []string{"berichte:reports:delete"}, allowed},

		// --- berichte: export is a real endpoint behind DatevView.tsx's canExport,
		// but MyReportsLibrary.tsx calls the same route with no capability check at
		// all -- so the legacy "read" key stays as-is (additive-only, cannot be
		// narrowed) and export:run is wired in as a second, real anchor ---
		{"definition export, catalogue export:run key also grants it", berichteRouter, "POST", "/api/v1/berichte/definitions/" + articleID + "/export", []string{"berichte:export:run"}, allowed},
		{"definition export, legacy read still grants it", berichteRouter, "POST", "/api/v1/berichte/definitions/" + articleID + "/export", []string{"berichte:reports:read"}, allowed},

		// --- berichte: cache invalidation has no catalogue verb, untouched ---
		{"definition cache invalidate still requires reports:write only", berichteRouter, "DELETE", "/api/v1/berichte/definitions/" + articleID + "/cache", []string{"berichte:reports:edit"}, denied},
		{"definition cache invalidate with reports:write", berichteRouter, "DELETE", "/api/v1/berichte/definitions/" + articleID + "/cache", []string{"berichte:reports:write"}, allowed},

		// --- berichte: schedules ---
		{"schedule create, catalogue schedule:manage key only", berichteRouter, "POST", "/api/v1/berichte/schedules/", []string{"berichte:schedule:manage"}, allowed},
		{"schedule create, denied without either key", berichteRouter, "POST", "/api/v1/berichte/schedules/", []string{"berichte:reports:read"}, denied},
		{"schedule toggle, catalogue schedule:manage key only", berichteRouter, "POST", "/api/v1/berichte/schedules/" + articleID + "/toggle", []string{"berichte:schedule:manage"}, allowed},

		// --- berichte: documents (multi-page authoring) ---
		{"document create, catalogue key only", berichteRouter, "POST", "/api/v1/berichte/documents/", []string{"berichte:reports:create"}, allowed},
		// PATCH carries both content edits and the draft->final->released->archived
		// lifecycle transition (ReportDocumentEditor.tsx transitionTo()), so it
		// needs both fine keys.
		{"document update, edit key only", berichteRouter, "PATCH", "/api/v1/berichte/documents/" + articleID, []string{"berichte:reports:edit"}, allowed},
		{"document update, publish key also grants it (lifecycle transition)", berichteRouter, "PATCH", "/api/v1/berichte/documents/" + articleID, []string{"berichte:reports:publish"}, allowed},
		{"document update, denied without write/edit/publish", berichteRouter, "PATCH", "/api/v1/berichte/documents/" + articleID, []string{"berichte:reports:delete"}, denied},
		{"document delete, catalogue key only", berichteRouter, "DELETE", "/api/v1/berichte/documents/" + articleID, []string{"berichte:reports:delete"}, allowed},
		{"document share create, catalogue share:manage key only", berichteRouter, "POST", "/api/v1/berichte/documents/" + articleID + "/shares", []string{"berichte:share:manage"}, allowed},
		{"document share revoke, catalogue share:manage key only", berichteRouter, "DELETE", "/api/v1/berichte/shares/" + articleID, []string{"berichte:share:manage"}, allowed},

		// --- formulare: schemas ---
		{"form schema create, legacy key only", formulareRouter, "POST", "/api/v1/formulare/schemas/", []string{"formulare:schemas:write"}, allowed},
		{"form schema create, catalogue key only", formulareRouter, "POST", "/api/v1/formulare/schemas/", []string{"formulare:schemas:create"}, allowed},
		{"form schema create, denied without either key", formulareRouter, "POST", "/api/v1/formulare/schemas/", []string{"formulare:submissions:read"}, denied},
		// PATCH carries both content edits and the draft/active/closed/archived
		// lifecycle toggle (FormularePage.tsx canSchemasPublish gates the same
		// updateSchema call as canSchemasEdit), so it needs both fine keys.
		{"form schema update, edit key only", formulareRouter, "PATCH", "/api/v1/formulare/schemas/" + articleID, []string{"formulare:schemas:edit"}, allowed},
		{"form schema update, publish key also grants it (lifecycle toggle)", formulareRouter, "PATCH", "/api/v1/formulare/schemas/" + articleID, []string{"formulare:schemas:publish"}, allowed},
		{"form schema delete, catalogue key only", formulareRouter, "DELETE", "/api/v1/formulare/schemas/" + articleID, []string{"formulare:schemas:delete"}, allowed},
		// Duplicate is FE-gated by canSchemasCreate, not schemas:edit.
		{"form schema duplicate, catalogue create key only", formulareRouter, "POST", "/api/v1/formulare/schemas/" + articleID + "/duplicate", []string{"formulare:schemas:create"}, allowed},
		{"form schema duplicate, edit key does not grant it", formulareRouter, "POST", "/api/v1/formulare/schemas/" + articleID + "/duplicate", []string{"formulare:schemas:edit"}, denied},

		// --- formulare: submissions export (real endpoint behind FormularePage.tsx
		// canExportRun; legacy "read" stays as-is, additive-only) ---
		{"submission export, catalogue export:run key also grants it", formulareRouter, "GET", "/api/v1/formulare/schemas/" + articleID + "/submissions/export", []string{"formulare:export:run"}, allowed},
		{"submission export, legacy submissions:read still grants it", formulareRouter, "GET", "/api/v1/formulare/schemas/" + articleID + "/submissions/export", []string{"formulare:submissions:read"}, allowed},

		// --- formulare: webhooks have no catalogue key (FE UI is an unmounted
		// stub), untouched ---
		{"webhook list still requires the legacy key only", formulareRouter, "GET", "/api/v1/formulare/schemas/" + articleID + "/webhooks/", []string{"formulare:schemas:read"}, denied},
		{"webhook list with legacy webhooks:read", formulareRouter, "GET", "/api/v1/formulare/schemas/" + articleID + "/webhooks/", []string{"formulare:webhooks:read"}, allowed},

		// --- automatisierung: legacy resource is the bare "automations" (no
		// module prefix, pre-000129 seed), so unlike every other module in this
		// loop NONE of the legacy keys concatenate to a catalogue key by
		// coincidence -- every route needs the additive wrapper ---
		{"automation create, legacy bare key only", automationRouter, "POST", "/api/v1/automations/", []string{"automations:write"}, allowed},
		{"automation create, catalogue key only", automationRouter, "POST", "/api/v1/automations/", []string{"automatisierung:automations:create"}, allowed},
		{"automation create, denied without either key", automationRouter, "POST", "/api/v1/automations/", []string{"automatisierung:automations:edit"}, denied},
		{"automation list, legacy bare key only", automationRouter, "GET", "/api/v1/automations/", []string{"automations:read"}, allowed},
		{"automation list, catalogue key only", automationRouter, "GET", "/api/v1/automations/", []string{"automatisierung:automations:read"}, allowed},
		{"automation update, catalogue edit key only", automationRouter, "PUT", "/api/v1/automations/" + articleID, []string{"automatisierung:automations:edit"}, allowed},
		{"automation delete, catalogue delete key only", automationRouter, "DELETE", "/api/v1/automations/" + articleID, []string{"automatisierung:automations:delete"}, allowed},
		{"automation enable, catalogue toggle key only", automationRouter, "POST", "/api/v1/automations/" + articleID + "/enable", []string{"automatisierung:automations:toggle"}, allowed},
		{"automation disable, catalogue toggle key only", automationRouter, "POST", "/api/v1/automations/" + articleID + "/disable", []string{"automatisierung:automations:toggle"}, allowed},
		{"execution list, catalogue executions:read key only", automationRouter, "GET", "/api/v1/automations/" + articleID + "/executions", []string{"automatisierung:executions:read"}, allowed},
		{"execution list, automations:read does not grant it", automationRouter, "GET", "/api/v1/automations/" + articleID + "/executions", []string{"automatisierung:automations:read"}, denied},

		// --- automatisierung: templates tab is gated on automations:create in
		// AutomatisierungPage.tsx (browsing templates is only useful if you can
		// instantiate one) ---
		{"templates list, catalogue create key only", automationRouter, "GET", "/api/v1/automations/templates", []string{"automatisierung:automations:create"}, allowed},
		{"templates list, read key does not grant it", automationRouter, "GET", "/api/v1/automations/templates", []string{"automatisierung:automations:read"}, denied},

		// --- automatisierung: test-condition/dry-run run from AutomationWizard,
		// reused for both the create flow and the "Bearbeiten" edit hand-off ---
		{"dry-run, catalogue create key grants it (create-flow wizard)", automationRouter, "POST", "/api/v1/automations/dry-run", []string{"automatisierung:automations:create"}, allowed},
		{"dry-run, catalogue edit key also grants it (edit-flow wizard)", automationRouter, "POST", "/api/v1/automations/dry-run", []string{"automatisierung:automations:edit"}, allowed},
		{"dry-run, denied without create or edit", automationRouter, "POST", "/api/v1/automations/dry-run", []string{"automatisierung:automations:read"}, denied},

		// --- automatisierung: stats stays on RequireRole("admin"), a different
		// mechanism entirely -- no permission key should grant it ---
		{"stats stays role-gated, permission key alone does not grant it", automationRouter, "GET", "/api/v1/automations/stats", []string{"automatisierung:automations:read"}, denied},
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
