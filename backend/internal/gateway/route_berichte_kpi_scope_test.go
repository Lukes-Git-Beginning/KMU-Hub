package gateway

// The dashboard KPI endpoint is the one berichte route whose payload is not
// scoped by its own permission: berichte:reports:read says "may open the
// reports module", while the tiles carry revenue, pipeline volume and stock
// figures owned by other modules. These tests pin the server-side cut, because
// the frontend's tile filter (report-module-visibility.ts) is display, not a
// boundary.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// memberPermissions mirrors the seeded `member` preset from migration 000256
// for the keys that matter here: every module a member may see, and pointedly
// NOT finance:module:view.
var memberPermissions = []string{
	"berichte:reports:read",
	"berichte:module:view",
	"crm:module:view",
	"helpdesk:module:view",
	"inventar:module:view",
	"produktion:module:view",
	"dashboard:module:view",
}

func kpiRequest(t *testing.T, query string, perms ...string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/berichte/kpis"+query, nil)
	return withPermissions(withTenantID(req, testTenantID), perms...)
}

func TestVisibleKPIModules_DropsModuleTheCallerCannotSee(t *testing.T) {
	got := visibleKPIModules(kpiRequest(t, "?modules=finanzen", memberPermissions...), []string{"finanzen"})
	if len(got) != 0 {
		t.Errorf("member without finance:module:view kept %v, want nothing", got)
	}
}

func TestVisibleKPIModules_KeepsGrantedModules(t *testing.T) {
	req := kpiRequest(t, "", append(memberPermissions, "finance:module:view")...)
	got := visibleKPIModules(req, []string{"finanzen", "crm"})
	if strings.Join(got, ",") != "finanzen,crm" {
		t.Errorf("got %v, want [finanzen crm] in request order", got)
	}
}

// An empty ?modules= used to mean "all modules". It now means "all modules this
// caller may see" — the list must be explicit, or the RPC falls back to its own
// "empty = all" reading.
func TestVisibleKPIModules_EmptyRequestExpandsToGrantedOnly(t *testing.T) {
	got := visibleKPIModules(kpiRequest(t, "", memberPermissions...), nil)

	want := "crm,cross,helpdesk,inventar,produktion"
	if strings.Join(got, ",") != want {
		t.Errorf("got %v, want [%s]", got, want)
	}
	for _, m := range got {
		if m == "finanzen" {
			t.Fatal("finanzen expanded into the list of a member who cannot see finance")
		}
	}
}

// A module id with no entry in kpiModuleVisibility must fail closed: a new KPI
// source without a mapping should surface as a missing tile in QA, never as an
// unguarded one in production.
func TestVisibleKPIModules_UnknownModuleFailsClosed(t *testing.T) {
	req := kpiRequest(t, "", append(memberPermissions, "finance:module:view")...)
	if got := visibleKPIModules(req, []string{"work", "not-a-module"}); len(got) != 0 {
		t.Errorf("unknown modules survived the cut: %v", got)
	}
}

// cross carries aggregate figures that belong to no single module and stays
// behind the route guard alone — same stance as the frontend's `cross: null`.
func TestVisibleKPIModules_CrossNeedsNoModuleCapability(t *testing.T) {
	if got := visibleKPIModules(kpiRequest(t, "", "berichte:reports:read"), []string{"cross"}); len(got) != 1 {
		t.Errorf("cross was dropped: %v", got)
	}
}

// The end-to-end shape of the fix: a member asking for the finance tile gets an
// empty 200 and the request never reaches the service. The registry is empty,
// so a request that did travel would answer 503 — which is exactly what the
// contrast test below asserts for a caller who may see the module.
func TestHandleGetDashboardKPIs_MemberAskingForFinanzenGetsNothing(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	rec := httptest.NewRecorder()

	routes.HandleGetDashboardKPIs(rec, kpiRequest(t, "?modules=finanzen", memberPermissions...))

	assertStatus(t, rec, http.StatusOK)
	if body := rec.Body.String(); strings.Contains(body, "kpis") {
		t.Errorf("expected a body without KPI tiles, got %s", body)
	}
}

func TestHandleGetDashboardKPIs_GrantedModuleReachesTheService(t *testing.T) {
	routes := NewBerichteRoutes(emptyRegistry(), berichteFlagsON())
	rec := httptest.NewRecorder()

	req := kpiRequest(t, "?modules=finanzen", append(memberPermissions, "finance:module:view")...)
	routes.HandleGetDashboardKPIs(rec, req)

	// 503 = the handler went on to getClient(), i.e. the module survived the
	// cut. Anything 2xx here would mean the early return swallowed a request it
	// was not supposed to.
	assertStatus(t, rec, http.StatusServiceUnavailable)
}
