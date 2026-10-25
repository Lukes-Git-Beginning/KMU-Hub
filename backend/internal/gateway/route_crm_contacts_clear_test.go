package gateway

// The contact form sends "" for every profile field the user left empty, and
// the service documents "" as "clear the column". These tests post the body
// the desktop really builds and hold the gateway to that contract: an empty
// string passes, a non-empty invalid value is still a 400.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// feContactBody is desktop/src/renderer/src/modules/kontakte/adapters.ts
// (coreContactFields + uiFormToWriteRequest, non-demo branch) for a form with
// only the two name fields filled in.
const feContactBody = `{"first_name":"Anna","last_name":"Berg","position":"","salutation":"","title":"",` +
	`"mobile":"","department":"","address_street":"","address_zip":"","address_city":"",` +
	`"address_country":"","website":"","linkedin":"","xing":"","category":"","status":"",` +
	`"custom_fields":[]}`

func decodeWireBody[T any](t *testing.T, body string) (T, int) {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	got, ok := decodeAndValidate[T](w, r)
	if ok {
		return got, http.StatusOK
	}
	return got, w.Code
}

func TestContactWrite_FormBodyWithEmptyProfileFieldsIsAccepted(t *testing.T) {
	t.Parallel()

	create, code := decodeWireBody[createContactRequest](t, feContactBody)
	if code != http.StatusOK {
		t.Fatalf("create with the desktop form body: status %d, want 200", code)
	}
	if create.Salutation == nil || *create.Salutation != "" {
		t.Errorf("empty salutation must reach the service as \"\" (clear), got %v", create.Salutation)
	}

	if _, code := decodeWireBody[updateContactRequest](t, feContactBody); code != http.StatusOK {
		t.Fatalf("update with the desktop form body: status %d, want 200", code)
	}
}

func TestContactWrite_InvalidNonEmptyProfileValueStaysRejected(t *testing.T) {
	t.Parallel()

	for _, field := range []string{
		`"salutation":"Dr"`,
		`"mobile":"abc"`,
		`"category":"vip"`,
		`"status":"gone"`,
	} {
		body := `{"first_name":"Anna","last_name":"Berg",` + field + `}`
		if _, code := decodeWireBody[createContactRequest](t, body); code != http.StatusBadRequest {
			t.Errorf("create %s: status %d, want 400", field, code)
		}
		if _, code := decodeWireBody[updateContactRequest](t, body); code != http.StatusBadRequest {
			t.Errorf("update %s: status %d, want 400", field, code)
		}
	}
}

// useUpdateLead sends temperature "" to hand the badge back to the computed
// score (route_crm_leads.go, updateLeadRequest).
func TestUpdateLead_EmptyTemperatureClearsOverride(t *testing.T) {
	t.Parallel()

	got, code := decodeWireBody[updateLeadRequest](t, `{"temperature":""}`)
	if code != http.StatusOK {
		t.Fatalf("temperature \"\": status %d, want 200", code)
	}
	if got.Temperature == nil || *got.Temperature != "" {
		t.Errorf("temperature \"\" must reach the service as \"\", got %v", got.Temperature)
	}
	if _, code := decodeWireBody[updateLeadRequest](t, `{"temperature":"lukewarm"}`); code != http.StatusBadRequest {
		t.Errorf("temperature lukewarm: status %d, want 400", code)
	}
}
