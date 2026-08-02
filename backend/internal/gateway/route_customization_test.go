package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/structpb"
)

// --- resolveLabels ---

// TestResolveLabels_DefaultAndTenantProvenance covers the two provenances this
// endpoint actually produces: a stored, non-empty override wins as "tenant",
// everything else — absent or an empty-string leftover — falls back to
// "default" with an empty value (the FE i18n bundle already holds the text).
func TestResolveLabels_DefaultAndTenantProvenance(t *testing.T) {
	out := resolveLabels(map[string]string{
		"crm.deals.title":  "Aufträge",
		"work.tasks.title": "",
		"not.in.whitelist": "should never surface",
	})

	if len(out) != len(labelWhitelist) {
		t.Fatalf("got %d entries, want exactly the %d whitelist keys", len(out), len(labelWhitelist))
	}
	if _, ok := out["not.in.whitelist"]; ok {
		t.Error("non-whitelisted key leaked into the resolved map")
	}

	got := out["crm.deals.title"]
	want := resolvedLabelJSON{Value: "Aufträge", Provenance: "tenant", Key: "crm.deals.title"}
	if got != want {
		t.Errorf("crm.deals.title = %+v, want %+v", got, want)
	}

	gotEmpty := out["work.tasks.title"]
	wantEmpty := resolvedLabelJSON{Value: "", Provenance: "default", Key: "work.tasks.title"}
	if gotEmpty != wantEmpty {
		t.Errorf("work.tasks.title (empty override) = %+v, want %+v", gotEmpty, wantEmpty)
	}

	gotAbsent := out["crm.contacts.title"]
	wantAbsent := resolvedLabelJSON{Value: "", Provenance: "default", Key: "crm.contacts.title"}
	if gotAbsent != wantAbsent {
		t.Errorf("crm.contacts.title (no override) = %+v, want %+v", gotAbsent, wantAbsent)
	}
}

// TestLabelOverridesResponseJSON_WireShape pins the exact camelCase shape the
// frontend's LabelOverridesResponse type expects.
func TestLabelOverridesResponseJSON_WireShape(t *testing.T) {
	body := labelOverridesResponseJSON{
		Locale: "de",
		Labels: map[string]resolvedLabelJSON{
			"crm.deals.title": {Value: "Aufträge", Provenance: "tenant", Key: "crm.deals.title"},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"locale":"de","labels":{"crm.deals.title":{"value":"Aufträge","provenance":"tenant","key":"crm.deals.title"}}}`
	if string(raw) != want {
		t.Errorf("wire shape drifted\n got: %s\nwant: %s", raw, want)
	}
}

// --- applyLabelOverrides ---

func TestApplyLabelOverrides_UnknownKeyDropped(t *testing.T) {
	merged := applyLabelOverrides(nil, map[string]string{"not.in.whitelist": "x"})
	if len(merged) != 0 {
		t.Errorf("merged = %+v, want empty — non-whitelisted key must not be stored", merged)
	}
}

func TestApplyLabelOverrides_EmptyValueClears(t *testing.T) {
	existing := map[string]string{"crm.deals.title": "Aufträge"}
	merged := applyLabelOverrides(existing, map[string]string{"crm.deals.title": ""})
	if _, ok := merged["crm.deals.title"]; ok {
		t.Errorf("merged = %+v, want crm.deals.title cleared", merged)
	}
}

func TestApplyLabelOverrides_OverwritesAndPreservesUntouchedKeys(t *testing.T) {
	existing := map[string]string{
		"crm.deals.title":     "Aufträge",
		"work.projects.title": "Mandate",
	}
	merged := applyLabelOverrides(existing, map[string]string{"crm.deals.title": "Kontrakte"})

	if merged["crm.deals.title"] != "Kontrakte" {
		t.Errorf("crm.deals.title = %q, want overwritten to Kontrakte", merged["crm.deals.title"])
	}
	if merged["work.projects.title"] != "Mandate" {
		t.Errorf("work.projects.title = %q, want untouched Mandate", merged["work.projects.title"])
	}
}

// --- structValueToStringMap ---

func TestStructValueToStringMap_RoundTripsObject(t *testing.T) {
	v, err := structpb.NewValue(map[string]any{"crm.deals.title": "Aufträge", "count": float64(3)})
	if err != nil {
		t.Fatalf("NewValue: %v", err)
	}
	got := structValueToStringMap(v)
	if got["crm.deals.title"] != "Aufträge" {
		t.Errorf("crm.deals.title = %q, want Aufträge", got["crm.deals.title"])
	}
	if _, ok := got["count"]; ok {
		t.Error("non-string entry 'count' should have been dropped, not silently stringified")
	}
}

func TestStructValueToStringMap_NilAndNonObject(t *testing.T) {
	if got := structValueToStringMap(nil); got != nil {
		t.Errorf("nil Value = %+v, want nil", got)
	}

	arr, err := structpb.NewValue([]any{"a", "b"})
	if err != nil {
		t.Fatalf("NewValue: %v", err)
	}
	if got := structValueToStringMap(arr); got != nil {
		t.Errorf("array Value = %+v, want nil (not a JSON object)", got)
	}
}

// --- HTTP wiring ---

func newCustomizationTestRouter(registry *ServiceRegistry) chi.Router {
	r := chi.NewRouter()
	NewCustomizationRoutes(registry).RegisterRoutes(r, guardTestAuth)
	return r
}

// TestHandleGetLabels_NoGuard confirms the read side has no RequirePermission
// wrapper — a token with zero permissions still reaches the handler (and dies
// at the empty registry, not at a 403).
func TestHandleGetLabels_NoGuard(t *testing.T) {
	router := newCustomizationTestRouter(emptyRegistry())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/customization/labels", nil)
	req = withAuth(req, "user-1", testTenantID)
	req = withPermissions(req)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandlePutLabels_RequiresPermission is the guard test for the write side:
// a caller without admin:customization:manage is rejected before the handler
// ever runs (empty registry would otherwise mask a missing guard as 503).
func TestHandlePutLabels_RequiresPermission(t *testing.T) {
	router := newCustomizationTestRouter(emptyRegistry())

	body := jsonBody(t, updateLabelOverridesRequest{Locale: "de", Overrides: map[string]string{"crm.deals.title": "Aufträge"}})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/customization/labels", body)
	req = withAuth(req, "user-1", testTenantID)
	req = withPermissions(req)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusForbidden)
}

// TestHandlePutLabels_PermissionGrantedReachesHandler is the mirror case:
// holding admin:customization:manage clears the guard, so the request dies
// further down at the empty registry (503), not at 403.
func TestHandlePutLabels_PermissionGrantedReachesHandler(t *testing.T) {
	router := newCustomizationTestRouter(emptyRegistry())

	body := jsonBody(t, updateLabelOverridesRequest{Locale: "de", Overrides: map[string]string{"crm.deals.title": "Aufträge"}})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/customization/labels", body)
	req = withAuth(req, "user-1", testTenantID)
	req = withPermissions(req, "admin:customization:manage")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// TestHandlePutLabels_VendorLayerRejected pins the scope decision documented
// in the JOURNAL: the vendor overlay has no writer yet, so a request naming it
// explicitly must fail loudly instead of silently landing in the tenant layer.
func TestHandlePutLabels_VendorLayerRejected(t *testing.T) {
	router := newCustomizationTestRouter(registryWithService("auth"))

	body := jsonBody(t, updateLabelOverridesRequest{Locale: "de", Layer: "vendor", Overrides: map[string]string{"crm.deals.title": "x"}})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/customization/labels", body)
	req = withAuth(req, "user-1", testTenantID)
	req = withPermissions(req, "admin:customization:manage")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "vendor")
}

// TestHandleGetLabels_InvalidLocale rejects a garbage locale query param
// before it ever reaches the settings client.
func TestHandleGetLabels_InvalidLocale(t *testing.T) {
	router := newCustomizationTestRouter(registryWithService("auth"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/customization/labels?locale=not-a-locale", nil)
	req = withAuth(req, "user-1", testTenantID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusBadRequest)
}

// TestHandleGetLabels_BaseSkipsTenantLookup pins useLabelDefaults' contract
// (desktop/src/renderer/src/api/hooks/useLabelOverrides.ts): base=1 must
// return every whitelist key as provenance "default" without ever resolving
// the tenant's stored overrides — it succeeds even though the fake "auth"
// connection would fail any real RPC.
func TestHandleGetLabels_BaseSkipsTenantLookup(t *testing.T) {
	router := newCustomizationTestRouter(registryWithService("auth"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/customization/labels?locale=de&base=1", nil)
	req = withAuth(req, "user-1", testTenantID)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)

	var body labelOverridesResponseJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for key, entry := range body.Labels {
		if entry.Provenance != "default" || entry.Value != "" {
			t.Errorf("base=1: %s = %+v, want provenance=default value=\"\"", key, entry)
		}
	}
}
