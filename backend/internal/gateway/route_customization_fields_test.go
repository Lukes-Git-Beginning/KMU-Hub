package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// --- pure helpers ---

func TestSlugifyFieldKey(t *testing.T) {
	cases := map[string]string{
		"ERP-Ticketnummer":  "erp_ticketnummer",
		"Geschätzter Wert":  "geschaetzter_wert",
		"  Straße  ":        "strasse",
		"Kunde/Lieferant!!": "kunde_lieferant",
	}
	for in, want := range cases {
		if got := slugifyFieldKey(in); got != want {
			t.Errorf("slugifyFieldKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFeToCRMFieldType(t *testing.T) {
	if got, ok := feToCRMFieldType("multi_select"); !ok || got != "multiselect" {
		t.Errorf("multi_select -> %q, %v; want multiselect, true", got, ok)
	}
	for _, t2 := range []string{"text", "number", "date", "boolean", "select"} {
		if got, ok := feToCRMFieldType(t2); !ok || got != t2 {
			t.Errorf("%s -> %q, %v; want %s, true", t2, got, ok, t2)
		}
	}
	for _, unsupported := range []string{"url", "email", "phone"} {
		if _, ok := feToCRMFieldType(unsupported); ok {
			t.Errorf("feToCRMFieldType(%q) = ok, want unsupported (false) — CRM has no column for it", unsupported)
		}
	}
}

func TestCrmToFEFieldType(t *testing.T) {
	if got := crmToFEFieldType("multiselect"); got != "multi_select" {
		t.Errorf("crmToFEFieldType(multiselect) = %q, want multi_select", got)
	}
	if got := crmToFEFieldType("text"); got != "text" {
		t.Errorf("crmToFEFieldType(text) = %q, want text (passthrough)", got)
	}
}

func TestCrmFieldToJSON_WireShape(t *testing.T) {
	def := "42"
	f := &crmv1.CustomFieldInfo{
		Id:           "f1",
		EntityType:   "deal",
		FieldName:    "erp_ref",
		FieldLabel:   "ERP-Referenz",
		FieldType:    "multiselect",
		Options:      []string{"a", "b"},
		DefaultValue: &def,
		IsRequired:   true,
		SortOrder:    3,
	}
	got := crmFieldToJSON(f)
	want := customFieldJSON{
		ID: "f1", Entity: "crm_deal", Key: "erp_ref", Label: "ERP-Referenz",
		Type: "multi_select", Required: true, Options: []string{"a", "b"},
		DefaultValue: &def, Visible: true, Order: 3, InUse: false,
	}
	if got.ID != want.ID || got.Entity != want.Entity || got.Key != want.Key ||
		got.Label != want.Label || got.Type != want.Type || got.Required != want.Required ||
		got.Visible != want.Visible || got.Order != want.Order || got.InUse != want.InUse ||
		*got.DefaultValue != *want.DefaultValue {
		t.Errorf("crmFieldToJSON = %+v, want %+v", got, want)
	}
}

func TestCrmFieldToJSON_NilOptionsBecomeEmptySlice(t *testing.T) {
	got := crmFieldToJSON(&crmv1.CustomFieldInfo{Id: "f1", EntityType: "contact"})
	if got.Options == nil {
		t.Error("Options must be a non-nil empty slice, not nil — the FE house rule forbids null for lists")
	}
}

// --- rejectUnsupportedCustomFieldFields (the no-silent-drop guarantee) ---

func TestRejectUnsupportedCustomFieldFields(t *testing.T) {
	valueSetID := "vs1"
	falseVal := false
	trueVal := true

	cases := []struct {
		name       string
		valueSetID *string
		validation *customFieldValidationRequest
		visible    *bool
		wantOK     bool
	}{
		{"all absent", nil, nil, nil, true},
		{"visible true is fine", nil, nil, &trueVal, true},
		{"valueSetId present rejected", &valueSetID, nil, nil, false},
		{"validation present rejected", nil, &customFieldValidationRequest{}, nil, false},
		{"visible false rejected", nil, nil, &falseVal, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ok := rejectUnsupportedCustomFieldFields(rec, c.valueSetID, c.validation, c.visible)
			if ok != c.wantOK {
				t.Errorf("ok = %v, want %v; body = %s", ok, c.wantOK, rec.Body.String())
			}
			if !c.wantOK {
				assertStatus(t, rec, http.StatusBadRequest)
			}
		})
	}
}

// --- handlers ---

func TestHandleListCustomFields_ServiceUnavailable(t *testing.T) {
	routes := NewCustomizationRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/customization/fields", nil)
	routes.HandleListCustomFields(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

func TestHandleListCustomFields_UnknownEntity(t *testing.T) {
	routes := NewCustomizationRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/customization/fields?entity=bogus", nil)
	routes.HandleListCustomFields(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleCreateCustomField_HelpdeskTicketRejected(t *testing.T) {
	routes := NewCustomizationRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/customization/fields", jsonBody(t, map[string]any{
		"entity": "helpdesk_ticket",
		"label":  "SLA-Stufe",
		"type":   "select",
	}))
	routes.HandleCreateCustomField(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "helpdesk_ticket")
}

func TestHandleCreateCustomField_WorkTaskRequiredTrueRejected(t *testing.T) {
	routes := NewCustomizationRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/customization/fields", jsonBody(t, map[string]any{
		"entity":   "work_task",
		"label":    "Aufwand",
		"type":     "number",
		"required": true,
	}))
	routes.HandleCreateCustomField(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "required")
}

func TestHandleCreateCustomField_ValueSetIdRejected(t *testing.T) {
	routes := NewCustomizationRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/customization/fields", jsonBody(t, map[string]any{
		"entity":     "crm_deal",
		"label":      "Prioritaet",
		"type":       "select",
		"valueSetId": "deal_priority",
	}))
	routes.HandleCreateCustomField(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
	assertErrorContains(t, rec, "valueSetId")
}

func TestHandleCreateCustomField_UnsupportedCRMType(t *testing.T) {
	routes := NewCustomizationRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/customization/fields", jsonBody(t, map[string]any{
		"entity": "crm_contact",
		"label":  "Zweit-Mail",
		"type":   "email",
	}))
	routes.HandleCreateCustomField(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestHandleGetCustomizationField_InvalidUUID(t *testing.T) {
	routes := NewCustomizationRoutes(emptyRegistry())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/customization/fields/bad", nil)
	req = withChiURLParam(req, "id", "not-a-uuid")
	routes.HandleGetCustomField(rec, req)
	assertStatus(t, rec, http.StatusBadRequest)
}
